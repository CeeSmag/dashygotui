package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/CeeSmag/dashygotui/internal/guard"
)

// docTemplate is the doc.go gen writes next to a new generated list. %[1]s is
// the package name, %[2]s the generator command, %[3]s the state type.
const docTemplate = `// Package %[1]s holds the dashboard's features, one per file. Each top-level
// var in an addon file is an addon; All, in the generated addons_gen.go, lists
// them. After adding, renaming or deleting an addon file, run go generate ./...
package %[1]s

//go:generate %[2]s gen -state %[3]s
`

func genCmd(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("gen", flag.ContinueOnError)
	fl.SetOutput(stderr)
	state := fl.String("state", "", "the application's state type as <import path>.<Type>; default: the one the existing addons_gen.go records")
	all := fl.Bool("all", false, "regenerate every addons_gen.go under the directory, each with the state it records")
	tool := fl.String("tool", "go tool dashygotui", "the command the //go:generate line of a new doc.go runs")
	quiet := fl.Bool("q", false, "print nothing when nothing changed")
	if err := fl.Parse(args); err != nil {
		return exitUsage
	}
	dir := "."
	switch fl.NArg() {
	case 0:
	case 1:
		dir = fl.Arg(0)
	default:
		fmt.Fprintln(stderr, "dashygotui gen: at most one directory")
		return exitUsage
	}

	dirs := []string{dir}
	if *all {
		if *state != "" {
			fmt.Fprintln(stderr, "dashygotui gen: -all uses the state each list records; drop -state")
			return exitUsage
		}
		var err error
		if dirs, err = genDirs(dir); err != nil {
			fmt.Fprintf(stderr, "dashygotui gen: %v\n", err)
			return exitUsage
		}
	}
	for _, d := range dirs {
		msg, err := genDir(d, *state, *tool)
		if err != nil {
			fmt.Fprintf(stderr, "dashygotui gen: %s: %v\n", d, err)
			return exitFindings
		}
		if msg != "" || !*quiet {
			fmt.Fprintln(stdout, cmpOr(msg, "ok    "+filepath.Join(d, guard.GenFile)+" is current"))
		}
	}
	return exitOK
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// genDirs lists the directories under root that hold a generated list.
func genDirs(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "testdata", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == guard.GenFile {
			out = append(out, filepath.Dir(p))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("looking for %s under %s: %w", guard.GenFile, root, err)
	}
	return out, nil
}

// genDir writes the generated list of the addons package in dir, and a doc.go
// when the package has none. It returns what it wrote, or "" when the list was
// already current.
func genDir(dir, state, tool string) (string, error) {
	srcs, err := loadDir(dir)
	if err != nil {
		return "", err
	}
	genPath := filepath.Join(dir, guard.GenFile)
	if state == "" {
		for _, s := range srcs {
			if s.Rel == guard.GenFile {
				state, _ = guard.GenState(s.File)
			}
		}
		if state == "" {
			return "", errors.New("no -state given and no " + guard.GenFile + " to read it from")
		}
	}
	var notes []string
	if !hasFile(srcs, guard.DocFile) {
		pkg := packageName(dir)
		if pkg == "" {
			pkg = filepath.Base(dir)
		}
		docPath := filepath.Join(dir, guard.DocFile)
		if err := os.WriteFile(docPath, fmt.Appendf(nil, docTemplate, pkg, tool, state), 0o644); err != nil { //nolint:gosec // source code is world-readable like the rest of the module
			return "", fmt.Errorf("writing %s: %w", docPath, err)
		}
		notes = append(notes, "wrote "+docPath)
		if srcs, err = loadDir(dir); err != nil {
			return "", err
		}
	}
	src, err := guard.AddonList(srcs, state)
	if err != nil {
		return "", err
	}
	old, err := os.ReadFile(genPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("reading %s: %w", genPath, err)
	}
	if bytes.Equal(old, src) {
		return strings.Join(notes, "\n"), nil
	}
	if err := os.WriteFile(genPath, src, 0o644); err != nil { //nolint:gosec // source code is world-readable like the rest of the module
		return "", fmt.Errorf("writing %s: %w", genPath, err)
	}
	return strings.Join(append(notes, "wrote "+genPath), "\n"), nil
}

// loadDir parses the non-test Go files of dir itself, not of its subdirectories.
func loadDir(dir string) ([]guard.Source, error) {
	srcs, err := guard.Load(dir)
	if err != nil {
		return nil, err
	}
	out := srcs[:0]
	for _, s := range srcs {
		if !strings.Contains(s.Rel, "/") {
			out = append(out, s)
		}
	}
	return out, nil
}

func hasFile(srcs []guard.Source, name string) bool {
	for _, s := range srcs {
		if s.Rel == name {
			return true
		}
	}
	return false
}
