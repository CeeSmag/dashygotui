package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/CeeSmag/dashygotui/internal/guard"
	"github.com/CeeSmag/dashygotui/templates"
)

// kinds are the feature kinds `new` can write, each with a template.
func kinds() []string { return guard.MapKinds() }

var (
	moduleRe  = regexp.MustCompile(`(?m)^module\s+(\S+)`)
	nameRe    = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	pkgNameRe = regexp.MustCompile(`(?m)^package\s+(\w+)`)
)

func newCmd(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("new", flag.ContinueOnError)
	fl.SetOutput(stderr)
	dir := fl.String("dir", "", "the addons directory (default: the only directory named addons under .)")
	dash := fl.String("dashboard", "", "import path of the dashboard package (default: the addons directory's parent)")
	if err := fl.Parse(args); err != nil {
		return exitUsage
	}
	if fl.NArg() != 2 {
		fmt.Fprintf(stderr, "usage: dashygotui new [-dir addons] [-dashboard import/path] <kind> <name>\nkinds: %s\n", strings.Join(kinds(), ", "))
		return exitUsage
	}
	kind, name := fl.Arg(0), fl.Arg(1)
	if !contains(kinds(), kind) {
		fmt.Fprintf(stderr, "dashygotui new: unknown kind %q (want %s)\n", kind, strings.Join(kinds(), ", "))
		return exitUsage
	}
	if !nameRe.MatchString(name) {
		fmt.Fprintf(stderr, "dashygotui new: name %q must be lower case letters, digits and underscores, starting with a letter\n", name)
		return exitUsage
	}

	addons, err := findAddonsDir(".", *dir)
	if err != nil {
		fmt.Fprintf(stderr, "dashygotui new: %v\n", err)
		return exitUsage
	}
	target := filepath.Join(addons, name+"_"+kind+".go")
	if _, err := os.Stat(target); err == nil {
		fmt.Fprintf(stderr, "dashygotui new: %s already exists\n", target)
		return exitUsage
	}

	importPath := *dash
	if importPath == "" {
		if importPath, err = dashboardImport(addons); err != nil {
			fmt.Fprintf(stderr, "dashygotui new: %v (pass -dashboard)\n", err)
			return exitUsage
		}
	}

	raw, err := fs.ReadFile(templates.FS, kind+".go.example")
	if err != nil {
		fmt.Fprintf(stderr, "dashygotui new: reading the %s template: %v\n", kind, err)
		return exitUsage
	}
	src := render(string(raw), importPath, name, packageName(addons))
	if err := os.WriteFile(target, []byte(src), 0o644); err != nil { //nolint:gosec // a source file, readable by design
		fmt.Fprintf(stderr, "dashygotui new: writing %s: %v\n", target, err)
		return exitUsage
	}
	fmt.Fprintf(stdout, "wrote %s\n", target)
	fmt.Fprintf(stdout, "Place it in %s, %s\n", filepath.Join(addons, guard.MapFile), placement(kind, lowerCamel(name), titleCase(name)))
	if _, err := os.Stat(filepath.Join(addons, guard.MapFile)); err != nil {
		fmt.Fprintf(stdout, "(there is no %s yet: it is the map, with func Map() dashboard.Map; see dashygotui rules design)\n", guard.MapFile)
	}
	fmt.Fprintln(stdout, "then run: dashygotui check")
	return exitOK
}

// placement says where a new feature of the kind goes in the map, and how.
func placement(kind, id, label string) string {
	switch kind {
	case "widget":
		return "as a panel of a page: {Widget: " + id + "Widget}"
	case "view":
		return "in the Items of a panel: " + id + "View (or in Map.Views, opened by its ID)"
	case "page":
		return "in the Items of a panel, with its panels: dashboard.Page{Title: \"" + label + "\", Badge: " + id + "Badge, Session: " + id + "Session, Panels: ...}"
	case "modal":
		return "in the Items of a panel: " + id + "ModalOpt"
	case "shortcut":
		return "in Map.Shortcuts: " + id + "Shortcut"
	case "hook":
		return "in Map.Hooks: " + id + "Hook"
	}
	return "in the Items of a panel: " + id + "Opt"
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// render fills a template: the import path, the name and the package name.
func render(tmpl, importPath, name, pkg string) string {
	r := strings.NewReplacer(
		"example.com/yourapp/dashboard", importPath,
		"myFeature", lowerCamel(name), // the Go names: one file's values must not clash with another's
		"my_feature", name,
		"My Feature", titleCase(name),
	)
	out := r.Replace(tmpl)
	if pkg != "" {
		out = pkgNameRe.ReplaceAllString(out, "package "+pkg)
	}
	return out
}

// lowerCamel turns my_feature into myFeature.
func lowerCamel(snake string) string {
	t := strings.ReplaceAll(titleCase(snake), " ", "")
	if t == "" {
		return "addon"
	}
	return strings.ToLower(t[:1]) + t[1:]
}

func titleCase(snake string) string {
	words := strings.Split(snake, "_")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// findAddonsDir returns the addons directory: dir if given, else the only
// directory named addons under root.
func findAddonsDir(root, dir string) (string, error) {
	if dir != "" {
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			return "", fmt.Errorf("%s is not a directory", dir)
		}
		return dir, nil
	}
	var found []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && p != root && (d.Name() == ".git" || d.Name() == "vendor" || d.Name() == "node_modules") {
			return filepath.SkipDir
		}
		if d.IsDir() && d.Name() == "addons" {
			found = append(found, p)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("looking for an addons directory: %w", err)
	}
	sort.Strings(found)
	switch len(found) {
	case 0:
		return "", fmt.Errorf("no directory named addons under %s (pass -dir)", root)
	case 1:
		return found[0], nil
	}
	return "", fmt.Errorf("several addons directories (%s); pass -dir", strings.Join(found, ", "))
}

// dashboardImport guesses the dashboard package's import path: the parent of the
// addons directory, under the module found in the nearest go.mod.
func dashboardImport(addons string) (string, error) {
	abs, err := filepath.Abs(addons)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", addons, err)
	}
	for dir := abs; ; dir = filepath.Dir(dir) {
		if b, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
			m := moduleRe.FindSubmatch(b)
			if m == nil {
				return "", fmt.Errorf("no module line in %s", filepath.Join(dir, "go.mod"))
			}
			rel, err := filepath.Rel(dir, filepath.Dir(abs))
			if err != nil {
				return "", fmt.Errorf("relating %s to the module: %w", abs, err)
			}
			if rel == "." {
				return string(m[1]), nil
			}
			return string(m[1]) + "/" + filepath.ToSlash(rel), nil
		}
		if filepath.Dir(dir) == dir {
			return "", fmt.Errorf("no go.mod above %s", addons)
		}
	}
}

// packageName is the package clause of the files already in the directory.
func packageName(addons string) string {
	entries, _ := os.ReadDir(addons)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		if b, err := os.ReadFile(filepath.Join(addons, e.Name())); err == nil {
			if m := pkgNameRe.FindSubmatch(b); m != nil {
				return string(m[1])
			}
		}
	}
	return ""
}
