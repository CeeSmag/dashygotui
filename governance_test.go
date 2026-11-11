package dashygotui_test

// Governance tests: they keep the library's public surface deliberate,
// documented and generic. See docs/DESIGN.md.

import (
	"flag"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui/internal/guard"
)

var updateAPI = flag.Bool("update-api", false, "rewrite api/dashygotui.txt from the code")

const apiFile = "api/dashygotui.txt"

func TestAPISnapshot(t *testing.T) {
	got, err := guard.APISnapshot(".")
	if err != nil {
		t.Fatal(err)
	}
	if *updateAPI {
		if err := os.MkdirAll("api", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(apiFile, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(apiFile)
	if err != nil {
		t.Fatalf("reading %s: %v (create it with: go test -run TestAPISnapshot -update-api .)", apiFile, err)
	}
	if got == string(want) {
		return
	}
	gotLines, wantLines := toSet(got), toSet(string(want))
	var diff []string
	for l := range gotLines {
		if !wantLines[l] {
			diff = append(diff, "+ "+l)
		}
	}
	for l := range wantLines {
		if !gotLines[l] {
			diff = append(diff, "- "+l)
		}
	}
	t.Errorf("the exported API changed. If that is intended, record it in CHANGELOG.md (a breaking change is called\n"+
		"out), then run: go test -run TestAPISnapshot -update-api .\n\n%s", strings.Join(sorted(diff), "\n"))
}

func toSet(s string) map[string]bool {
	m := map[string]bool{}
	for _, l := range strings.Split(s, "\n") {
		if l != "" && !strings.HasPrefix(l, "#") {
			m[l] = true
		}
	}
	return m
}

func sorted(s []string) []string {
	out := append([]string(nil), s...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// referenceFile is generated from the doc comments of the types an addon author
// fills in, so the reference an AI reads can never lag the code.
const referenceFile = "docs/REFERENCE.md"

func referenceText(t *testing.T) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("# Field reference\n\n" +
		"Generated from the doc comments of the types an addon author fills in. Do not edit: change the\n" +
		"comment in the code and run `go test -run TestReferenceIsCurrent -update-api .`.\n" +
		"For when to use what, read `COOKBOOK.md`; for the narrative, `ADDONS.md`.\n\n")
	for _, part := range []struct {
		title, dir string
		types      []string
	}{
		{"Package dashygotui", ".", []string{"Config", "Page", "SectionSpec", "Option", "Entry", "Widget", "View", "ViewPanel", "Shortcut"}},
		{"Package modal", "modal", []string{"Edit", "Confirm", "Typed", "Notice", "Choice", "Slider"}},
		{"Package stdaddons", "stdaddons", []string{"Toggle", "Number", "Pick", "Text"}},
	} {
		body, err := guard.FieldReference(part.dir, part.types...)
		if err != nil {
			t.Fatal(err)
		}
		sb.WriteString("## " + part.title + "\n\n" + body)
	}
	return sb.String()
}

func TestReferenceIsCurrent(t *testing.T) {
	got := referenceText(t)
	if *updateAPI {
		if err := os.WriteFile(referenceFile, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(referenceFile)
	if err != nil {
		t.Fatalf("reading %s: %v (create it with: go test -run TestReferenceIsCurrent -update-api .)", referenceFile, err)
	}
	if got != string(want) {
		t.Errorf("%s is out of date with the doc comments in the code; regenerate it: go test -run TestReferenceIsCurrent -update-api .", referenceFile)
	}
}

func TestLibraryHasNoApplicationVocabulary(t *testing.T) {
	err := filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch {
			case p == ".":
				return nil
			case d.Name() == ".git", d.Name() == "examples", d.Name() == "bin", d.Name() == "testdata",
				p == filepath.Join("internal", "guard"):
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, ".example") {
			return nil
		}
		if p == "governance_test.go" {
			return nil // its fixtures must contain the words
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, v := range guard.VocabularyViolations(filepath.ToSlash(p), src, guard.ApplicationVocabulary()) {
			t.Error(v)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestVocabularyGuardIsWholeWordAndCaseInsensitive(t *testing.T) {
	for src, want := range map[string]int{
		"// built for CloudVault":   1,
		"x := \"run sudo now\"":     1,
		"// multiplexers are fine":  0,
		"// Plex is a product":      1,
		"// complex is fine":        0,
		"// Systemd and journalctl": 1, // a line is reported once
		"nothing to see":            0,
	} {
		if got := guard.VocabularyViolations("f.go", []byte(src), guard.ApplicationVocabulary()); len(got) != want {
			t.Errorf("%q: %d violations %v, want %d", src, len(got), got, want)
		}
	}
}

// goBlocks are the fenced Go samples of a markdown file.
var goBlockRe = regexp.MustCompile("(?s)```go\\n(.*?)\\n```")

// parses reports whether a documentation sample is syntactically Go, as a whole
// file, as declarations, or as statements. "..." stands for code left out.
func parses(sample string) bool {
	sample = strings.ReplaceAll(sample, "...", "nil")
	for _, wrap := range []string{"%s", "package p\n%s", "package p\nfunc _() {\n%s\n}", "package p\nvar _ = T{\n%s\n}"} {
		if _, err := parser.ParseFile(token.NewFileSet(), "sample.go", strings.Replace(wrap, "%s", sample, 1), 0); err == nil {
			return true
		}
	}
	return false
}

// A sample in the docs that is not even Go misleads whoever copies it; an AI
// copies samples. The check is syntactic: it cannot catch a renamed field, which
// the generated reference and the cookbook's own tests are for.
func TestDocumentationSamplesAreGo(t *testing.T) {
	for _, file := range []string{"README.md", "docs/COOKBOOK.md", "docs/ADDONS.md", "docs/DESIGN.md", "docs/APP-RULES.md"} {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for i, m := range goBlockRe.FindAllStringSubmatch(string(b), -1) {
			if !parses(m[1]) {
				t.Errorf("%s: Go sample #%d does not parse:\n%s", file, i+1, m[1])
			}
		}
	}
}

func TestSampleCheckerAcceptsAndRejects(t *testing.T) {
	for sample, want := range map[string]bool{
		"package p\nfunc f() {}":                  true,
		"func f() {}":                             true,
		"x := 1\nc.Flash(\"hi\")":                 true,
		"Run: func(c *Ctx) { ... }":               true,  // fields of a literal, left bare
		"Run: func(c *Ctx { ... }":                false, // unbalanced
		"dashboard.Registry.Add(Option{Do: ...})": true,
		"this is not go at all {":                 false,
	} {
		if got := parses(sample); got != want {
			t.Errorf("parses(%q) = %v, want %v", sample, got, want)
		}
	}
}
