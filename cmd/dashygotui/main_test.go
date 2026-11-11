package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runCLI runs the command in the current directory and returns its output.
func runCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// write creates path (and its directories) under dir.
func write(t *testing.T, dir, path, body string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNoArgumentsAndUnknownCommand(t *testing.T) {
	if code, _, errOut := runCLI(t); code != exitUsage || !strings.Contains(errOut, "usage:") {
		t.Errorf("no args: code %d, stderr %q", code, errOut)
	}
	if code, _, errOut := runCLI(t, "frobnicate"); code != exitUsage || !strings.Contains(errOut, `unknown command "frobnicate"`) {
		t.Errorf("unknown: code %d, stderr %q", code, errOut)
	}
	if code, out, _ := runCLI(t, "help"); code != exitOK || !strings.Contains(out, "commands:") {
		t.Errorf("help: code %d, stdout %q", code, out)
	}
}

func TestRulesPrintsTheCharterByDefault(t *testing.T) {
	code, out, errOut := runCLI(t, "rules")
	if code != exitOK || errOut != "" {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
	for _, want := range []string{"design charter", "The generic gate", "Rules for applications built on dashygotui", "| A5 |", "Cookbook"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q", want)
		}
	}
}

func TestRulesTopicsAndList(t *testing.T) {
	if code, out, _ := runCLI(t, "rules", "cookbook"); code != exitOK || !strings.Contains(out, "I want to...") || strings.Contains(out, "design charter") {
		t.Errorf("cookbook: code %d", code)
	}
	if code, _, errOut := runCLI(t, "rules", "nope"); code != exitUsage || !strings.Contains(errOut, `unknown topic "nope"`) {
		t.Errorf("unknown topic: code %d, stderr %q", code, errOut)
	}
	code, out, _ := runCLI(t, "rules", "-list")
	if code != exitOK || !strings.Contains(out, "app-rules") || !strings.Contains(out, "A9") || !strings.Contains(out, "G1") {
		t.Errorf("list: code %d, out %q", code, out)
	}
}

func TestEveryTopicHasADocument(t *testing.T) {
	for topic := range topics() {
		if code, out, errOut := runCLI(t, "rules", topic); code != exitOK || len(out) < 200 {
			t.Errorf("topic %s: code %d, %d bytes, stderr %q", topic, code, len(out), errOut)
		}
	}
}

func TestCheck(t *testing.T) {
	good := "package addons\nimport \"fmt\"\nfunc init() { _ = fmt.Sprint(1) }\n"
	t.Run("clean", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "ui/addons/opt_a.go", good)
		code, out, _ := runCLI(t, "check", dir)
		if code != exitOK || !strings.Contains(out, "ok: 1 addon file(s)") {
			t.Errorf("code %d, out %q", code, out)
		}
	})
	t.Run("a broken rule fails", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "ui/addons/opt_a.go", "package addons\nimport \"time\"\nfunc init() {\n\ttime.Sleep(1)\n}\n")
		write(t, dir, "ui/addons/helpers.go", "package addons\n")
		code, out, _ := runCLI(t, "check", dir)
		if code != exitFindings {
			t.Fatalf("code %d, out %q", code, out)
		}
		for _, want := range []string{"ui/addons/opt_a.go:4: A4: time.Sleep", "ui/addons/helpers.go:1: A1", "2 finding(s)"} {
			if !strings.Contains(out, want) {
				t.Errorf("output lacks %q:\n%s", want, out)
			}
		}
	})
	t.Run("an allow comment excuses and is listed", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "ui/addons/opt_a.go", "package addons\nimport \"time\"\nfunc init() {\n\ttime.Sleep(1) //dashygotui:allow A4 hardware settle time\n}\n")
		code, out, _ := runCLI(t, "check", dir)
		if code != exitOK || !strings.Contains(out, "excused by //dashygotui:allow") || !strings.Contains(out, "hardware settle time") {
			t.Errorf("code %d, out %q", code, out)
		}
	})
	t.Run("gap markers are listed, a bare one fails", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "ui/addons/opt_a.go", good)
		write(t, dir, "ui/actions.go", "package ui\n// dashygotui:gap GAPS.md#restart needs OnSaved\nfunc f() {}\n")
		code, out, _ := runCLI(t, "check", dir)
		if code != exitOK || !strings.Contains(out, "marked //dashygotui:gap") || !strings.Contains(out, "GAPS.md#restart") {
			t.Errorf("code %d, out %q", code, out)
		}
		write(t, dir, "ui/bare.go", "package ui\n//dashygotui:gap\nfunc g() {}\n")
		if code, out, _ := runCLI(t, "check", dir); code != exitFindings || !strings.Contains(out, "G1") {
			t.Errorf("a bare marker: code %d, out %q", code, out)
		}
	})
	t.Run("quiet hides the lists", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "ui/addons/opt_a.go", good)
		write(t, dir, "ui/actions.go", "package ui\n// dashygotui:gap GAPS.md#x y\nfunc f() {}\n")
		if _, out, _ := runCLI(t, "check", "-q", dir); strings.Contains(out, "marked") {
			t.Errorf("out %q", out)
		}
	})
	t.Run("no addons directory is said, not an error", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "main.go", "package main\n")
		code, _, errOut := runCLI(t, "check", dir)
		if code != exitOK || !strings.Contains(errOut, "no addons directory") {
			t.Errorf("code %d, stderr %q", code, errOut)
		}
	})
	t.Run("a file that does not parse is a usage error", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "ui/addons/opt_a.go", "package addons\nfunc (")
		if code, _, errOut := runCLI(t, "check", dir); code != exitUsage || !strings.Contains(errOut, "parsing") {
			t.Errorf("code %d, stderr %q", code, errOut)
		}
	})
}

// app builds a module with one addon and returns its directory.
func app(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, "go.mod", "module example.com/app\n\ngo 1.27\n\nrequire github.com/CeeSmag/dashygotui v0.3.0\n")
	write(t, dir, "internal/ui/addons/opt_first.go", "package addons\n\nfunc init() {\n\t_ = struct{ Number int }{Number: 7}\n}\n")
	return dir
}

func TestNewWritesAnAddonFromATemplate(t *testing.T) {
	dir := app(t)
	t.Chdir(dir)
	code, out, errOut := runCLI(t, "new", "opt", "storage_limit")
	if code != exitOK {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, "opt_storage_limit.go") {
		t.Errorf("out %q", out)
	}
	b, err := os.ReadFile(filepath.Join(dir, "internal/ui/addons/opt_storage_limit.go"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{"package addons", `"example.com/app/internal/ui"`, `"storage_limit"`, `"Storage Limit"`, "Number: 8"} {
		if !strings.Contains(got, want) {
			t.Errorf("the new file lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "yourapp") || strings.Contains(got, "my_feature") {
		t.Errorf("template placeholders were left behind:\n%s", got)
	}

	if code, _, errOut := runCLI(t, "new", "opt", "storage_limit"); code != exitUsage || !strings.Contains(errOut, "already exists") {
		t.Errorf("overwrite: code %d, stderr %q", code, errOut)
	}
}

func TestNewWritesEveryKind(t *testing.T) {
	dir := app(t)
	t.Chdir(dir)
	for _, kind := range kinds() {
		if code, _, errOut := runCLI(t, "new", kind, "thing"); code != exitOK {
			t.Errorf("%s: code %d, stderr %q", kind, code, errOut)
		}
		if _, err := os.Stat(filepath.Join(dir, "internal/ui/addons", kind+"_thing.go")); err != nil {
			t.Errorf("%s: %v", kind, err)
		}
	}
}

func TestNewRejectsBadInput(t *testing.T) {
	dir := app(t)
	t.Chdir(dir)
	for name, args := range map[string][]string{
		"unknown kind": {"new", "gadget", "x"},
		"capital name": {"new", "opt", "Storage"},
		"dash in name": {"new", "opt", "my-thing"},
		"missing name": {"new", "opt"},
		"bad -dir":     {"new", "-dir", "nope", "opt", "x"},
	} {
		code, _, _ := runCLI(t, args...)
		if code != exitUsage {
			t.Errorf("%s: code %d, want usage error", name, code)
		}
	}
}

func TestNewNeedsExactlyOneAddonsDirectory(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module example.com/app\n")
	write(t, dir, "a/addons/opt_x.go", "package addons\n")
	write(t, dir, "b/addons/opt_y.go", "package addons\n")
	t.Chdir(dir)
	if code, _, errOut := runCLI(t, "new", "opt", "z"); code != exitUsage || !strings.Contains(errOut, "several addons directories") {
		t.Errorf("code %d, stderr %q", code, errOut)
	}
	if code, _, errOut := runCLI(t, "new", "-dir", "a/addons", "opt", "z"); code != exitOK {
		t.Errorf("with -dir: code %d, stderr %q", code, errOut)
	}
}

func TestInitWritesTheInstructionFiles(t *testing.T) {
	dir := app(t)
	t.Chdir(dir)
	code, out, _ := runCLI(t, "init")
	if code != exitOK || !strings.Contains(out, "wrote   AGENTS.md") || !strings.Contains(out, "dashygotui-check") {
		t.Fatalf("code %d, out %q", code, out)
	}
	agents, err := os.ReadFile("AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"cmd/dashygotui@v0.3.0 rules", "cmd/dashygotui@v0.3.0 check -changed", "record a gap", "One feature, one file"} {
		if !strings.Contains(string(agents), want) {
			t.Errorf("AGENTS.md lacks %q:\n%s", want, agents)
		}
	}
	claude, _ := os.ReadFile("CLAUDE.md")
	if !strings.HasPrefix(string(claude), "# CLAUDE.md") || !strings.Contains(string(claude), "dashygotui@v0.3.0") {
		t.Errorf("CLAUDE.md:\n%s", claude)
	}
	if _, err := os.Stat("GAPS.md"); err != nil {
		t.Error(err)
	}

	if err := os.WriteFile("GAPS.md", []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, out, _ := runCLI(t, "init"); !strings.Contains(out, "kept    GAPS.md") {
		t.Errorf("a second init must keep existing files: %q", out)
	}
	if b, _ := os.ReadFile("GAPS.md"); string(b) != "mine" {
		t.Error("init overwrote GAPS.md without -force")
	}
	runCLI(t, "init", "-force")
	if b, _ := os.ReadFile("GAPS.md"); string(b) == "mine" {
		t.Error("-force should overwrite")
	}
}

func TestInitWithoutAVersionPinsLatest(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	code, _, errOut := runCLI(t, "init")
	if code != exitOK || !strings.Contains(errOut, "latest") {
		t.Errorf("code %d, stderr %q", code, errOut)
	}
	if b, _ := os.ReadFile("AGENTS.md"); !strings.Contains(string(b), "dashygotui@latest") {
		t.Errorf("AGENTS.md:\n%s", b)
	}
	runCLI(t, "init", "-force", "-version", "v1.2.3")
	if b, _ := os.ReadFile("AGENTS.md"); !strings.Contains(string(b), "dashygotui@v1.2.3") {
		t.Errorf("-version ignored:\n%s", b)
	}
}

// gitRepo makes a repository with one commit of the given files, or skips the
// test when git is not installed.
func gitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	if _, err := osexec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	for p, body := range files {
		write(t, dir, p, body)
	}
	for _, args := range [][]string{
		{"init", "-q"}, {"add", "-A"},
		{"-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "base"},
	} {
		if out, err := osexec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func TestCheckChangedAppliesTheOneFileRule(t *testing.T) {
	base := map[string]string{
		"ui/dashboard.go":        "package ui\n",
		"ui/addons/opt_first.go": "package addons\n",
	}
	t.Run("a clean tree", func(t *testing.T) {
		dir := gitRepo(t, base)
		code, out, _ := runCLI(t, "check", "-changed", dir)
		if code != exitOK || !strings.Contains(out, "0 changed file(s)") {
			t.Errorf("code %d, out %q", code, out)
		}
	})
	t.Run("only a new addon", func(t *testing.T) {
		dir := gitRepo(t, base)
		write(t, dir, "ui/addons/opt_second.go", "package addons\n")
		code, out, _ := runCLI(t, "check", "-changed", dir)
		if code != exitOK || strings.Contains(out, "C1") || !strings.Contains(out, "follow the one-file rule") {
			t.Errorf("code %d, out %q", code, out)
		}
	})
	t.Run("a new addon that edits the dashboard file is advice", func(t *testing.T) {
		dir := gitRepo(t, base)
		write(t, dir, "ui/addons/opt_second.go", "package addons\n")
		write(t, dir, "ui/dashboard.go", "package ui\n// edited\n")
		code, out, _ := runCLI(t, "check", "-changed", dir)
		if code != exitOK || !strings.Contains(out, "ui/dashboard.go:1: C1: edited while adding ui/addons/opt_second.go") || !strings.Contains(out, "advice") {
			t.Errorf("code %d, out %q", code, out)
		}
	})
	t.Run("-strict makes it fail", func(t *testing.T) {
		dir := gitRepo(t, base)
		write(t, dir, "ui/addons/opt_second.go", "package addons\n")
		write(t, dir, "ui/addons/opt_first.go", "package addons\n// edited\n")
		code, out, _ := runCLI(t, "check", "-changed", "-strict", dir)
		if code != exitFindings || !strings.Contains(out, "C1") {
			t.Errorf("code %d, out %q", code, out)
		}
	})
	t.Run("editing without adding is not this rule", func(t *testing.T) {
		dir := gitRepo(t, base)
		write(t, dir, "ui/dashboard.go", "package ui\n// edited\n")
		if code, out, _ := runCLI(t, "check", "-changed", "-strict", dir); code != exitOK || strings.Contains(out, "C1") {
			t.Errorf("code %d, out %q", code, out)
		}
	})
	t.Run("a base revision to compare with", func(t *testing.T) {
		dir := gitRepo(t, base)
		if code, _, errOut := runCLI(t, "check", "-changed", "-base", "no-such-rev", dir); code != exitUsage || !strings.Contains(errOut, "git diff") {
			t.Errorf("code %d, stderr %q", code, errOut)
		}
	})
	t.Run("not a git checkout", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "ui/addons/opt_a.go", "package addons\n")
		if code, _, errOut := runCLI(t, "check", "-changed", dir); code != exitUsage || !strings.Contains(errOut, "git") {
			t.Errorf("code %d, stderr %q", code, errOut)
		}
	})
}

func TestInitClaudeWritesTheSkillAndMergesTheHook(t *testing.T) {
	dir := app(t)
	t.Chdir(dir)
	write(t, dir, ".claude/settings.json", `{"model": "opus", "hooks": {"PostToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "echo mine"}]}], "Stop": []}}`)

	code, out, errOut := runCLI(t, "init", "-claude")
	if code != exitOK {
		t.Fatalf("code %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, ".claude/skills/dashygotui/SKILL.md") || !strings.Contains(out, "a hook that runs the check") {
		t.Errorf("out %q", out)
	}

	skill, err := os.ReadFile(".claude/skills/dashygotui/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"name: dashygotui", "description: Use when", "cmd/dashygotui@v0.3.0 rules", "check -changed", "dashygotui.Local", "stop."} {
		if !strings.Contains(string(skill), want) {
			t.Errorf("SKILL.md lacks %q:\n%s", want, skill)
		}
	}

	raw, _ := os.ReadFile(".claude/settings.json")
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v\n%s", err, raw)
	}
	if doc["model"] != "opus" {
		t.Error("the user's other settings must survive")
	}
	post := doc["hooks"].(map[string]any)["PostToolUse"].([]any)
	if len(post) != 2 || !strings.Contains(string(raw), "echo mine") {
		t.Errorf("the user's own hook must survive and ours be added: %s", raw)
	}
	if !strings.Contains(string(raw), "dashygotui@v0.3.0 check -q 1>&2 || exit 2") || !strings.Contains(string(raw), "Edit|Write|MultiEdit") {
		t.Errorf("hook not as expected:\n%s", raw)
	}
	if _, ok := doc["hooks"].(map[string]any)["Stop"]; !ok {
		t.Error("other hook events must survive")
	}

	// A second run changes nothing.
	before := string(raw)
	_, out, _ = runCLI(t, "init", "-claude")
	after, _ := os.ReadFile(".claude/settings.json")
	if string(after) != before || !strings.Contains(out, "the hook is already there") {
		t.Errorf("a second init must not add the hook twice:\n%s\nout %q", after, out)
	}
}

func TestInitClaudeCreatesSettingsAndNeverOverwritesBrokenOnes(t *testing.T) {
	t.Run("no settings yet", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if code, _, errOut := runCLI(t, "init", "-claude"); code != exitOK {
			t.Fatalf("code %d, stderr %q", code, errOut)
		}
		raw, err := os.ReadFile(".claude/settings.json")
		if err != nil || !json.Valid(raw) {
			t.Errorf("settings.json: %v, valid %v", err, json.Valid(raw))
		}
	})
	t.Run("a settings file that is not an object is left alone", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		write(t, dir, ".claude/settings.json", `["not", "an", "object"]`)
		code, _, errOut := runCLI(t, "init", "-claude")
		if code != exitUsage || !strings.Contains(errOut, "left alone") {
			t.Errorf("code %d, stderr %q", code, errOut)
		}
		if b, _ := os.ReadFile(".claude/settings.json"); string(b) != `["not", "an", "object"]` {
			t.Errorf("the file was overwritten: %s", b)
		}
	})
	t.Run("hooks of the wrong shape are left alone", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		write(t, dir, ".claude/settings.json", `{"hooks": {"PostToolUse": "nope"}}`)
		if code, _, errOut := runCLI(t, "init", "-claude"); code != exitUsage || !strings.Contains(errOut, "not a list") {
			t.Errorf("code %d, stderr %q", code, errOut)
		}
	})
}
