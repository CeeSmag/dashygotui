package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/CeeSmag/dashygotui/internal/guard"
)

func checkCmd(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("check", flag.ContinueOnError)
	fl.SetOutput(stderr)
	quiet := fl.Bool("q", false, "print only the findings, not the exceptions and gaps")
	changed := fl.Bool("changed", false, "also check the change set against base: adding an addon should edit no other file (rule C1; advice unless -strict)")
	base := fl.String("base", "HEAD", "with -changed, the git revision to compare the working tree with")
	strict := fl.Bool("strict", false, "with -changed, fail on C1 findings too")
	if err := fl.Parse(args); err != nil {
		return exitUsage
	}
	dir := "."
	switch fl.NArg() {
	case 0:
	case 1:
		dir = fl.Arg(0)
	default:
		fmt.Fprintln(stderr, "dashygotui check: at most one directory")
		return exitUsage
	}

	srcs, err := guard.Load(dir)
	if err != nil {
		fmt.Fprintf(stderr, "dashygotui check: %v\n", err)
		return exitUsage
	}
	addons := 0
	for _, s := range srcs {
		if guard.InAddons(s.Rel) {
			addons++
		}
	}
	if addons == 0 {
		fmt.Fprintf(stderr, "dashygotui check: no addons directory under %s, nothing to check\n", dir)
	}

	rep := guard.CheckApp(srcs)
	for _, f := range rep.Findings {
		fmt.Fprintln(stdout, f)
	}
	if !*quiet {
		if len(rep.Exceptions) > 0 {
			fmt.Fprintf(stdout, "\n%d excused by //dashygotui:allow (review these):\n", len(rep.Exceptions))
			for _, f := range rep.Exceptions {
				fmt.Fprintln(stdout, "  "+f.String())
			}
		}
		if len(rep.Gaps) > 0 {
			fmt.Fprintf(stdout, "\n%d workaround(s) marked //dashygotui:gap (close them when the library catches up):\n", len(rep.Gaps))
			for _, f := range rep.Gaps {
				fmt.Fprintf(stdout, "  %s:%d: %s\n", f.File, f.Line, f.Msg)
			}
		}
	}
	advice := 0
	if *changed {
		files, err := changedFiles(dir, *base)
		if err != nil {
			fmt.Fprintf(stderr, "dashygotui check: %v\n", err)
			return exitUsage
		}
		c1 := guard.ChangeFindings(files)
		for _, f := range c1 {
			fmt.Fprintln(stdout, f)
		}
		advice = len(c1)
		if *strict && advice > 0 {
			rep.Findings = append(rep.Findings, c1...)
		}
		if advice == 0 {
			fmt.Fprintf(stdout, "ok: %d changed file(s) since %s follow the one-file rule\n", len(files), *base)
		}
	}
	if len(rep.Findings) > 0 {
		fmt.Fprintf(stdout, "\n%d finding(s) in %d addon file(s). See: dashygotui rules app-rules\n", len(rep.Findings), addons)
		return exitFindings
	}
	if advice > 0 {
		fmt.Fprintf(stdout, "\n%d note(s) about the change set (advice; -strict makes them fail). See: dashygotui rules app-rules\n", advice)
	}
	fmt.Fprintf(stdout, "ok: %d addon file(s) follow the application rules\n", addons)
	return exitOK
}

// changedFiles lists what differs between the working tree and base, plus the
// untracked files, with paths relative to dir.
func changedFiles(dir, base string) ([]guard.ChangedFile, error) {
	git := func(args ...string) (string, error) {
		out, err := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...).Output() //nolint:gosec // a fixed program with the user's own revision as an argument
		if err != nil {
			return "", fmt.Errorf("git %s: %w (is %s a git checkout, and %q a revision?)", strings.Join(args, " "), err, dir, base)
		}
		return string(out), nil
	}
	var files []guard.ChangedFile
	diff, err := git("diff", "--name-status", "--relative", "-M", base, "--")
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(strings.TrimSpace(diff), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) < 2 || parts[0] == "" {
			continue
		}
		files = append(files, guard.ChangedFile{Path: parts[len(parts)-1], Status: parts[0][0]})
	}
	untracked, err := git("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	for _, p := range strings.Split(strings.TrimSpace(untracked), "\n") {
		if p != "" {
			files = append(files, guard.ChangedFile{Path: p, Status: 'A'})
		}
	}
	return files, nil
}
