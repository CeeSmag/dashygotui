// Command dashygotui is the companion tool of the dashygotui library. It prints
// the library's design rules, checks an application's addons against them,
// writes new addons from templates, and sets an application up so that people
// and AI agents working on it find the rules.
//
// Run it with the version of the library the application uses, so the rules
// match the API it compiles against:
//
//	go run github.com/CeeSmag/dashygotui/cmd/dashygotui@v0.3.0 rules
//
// Usage:
//
//	dashygotui rules [topic]            print the design rules (design, app-rules, cookbook, addons, reference, architecture)
//	dashygotui check [-changed] [dir]   lint the addons under dir (default .); exits 1 on findings; -changed also applies the one-file rule to the git change set
//	dashygotui new <kind> <name>        write addons/<kind>_<name>.go from a template (opt, widget, view, modal, shortcut)
//	dashygotui init [-claude]           write AGENTS.md, CLAUDE.md and GAPS.md for the application; -claude adds a Claude Code skill and a check-after-edit hook
package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// Exit codes.
const (
	exitOK       = 0
	exitFindings = 1
	exitUsage    = 2
)

const usage = `usage: dashygotui <command> [arguments]

commands:
  rules [topic]          print the design rules; topics: design, app-rules, cookbook, addons, reference, architecture
  check [-changed] [dir] lint the addons under dir (default "."); exit 1 when a rule is broken;
                         -changed also checks the git change set (adding an addon edits no other file)
  new <kind> <name>      write a new addon file from a template; kinds: opt, widget, view, modal, shortcut
  init [-claude]         write AGENTS.md, CLAUDE.md and GAPS.md; -claude adds a Claude Code skill and hook

Run a command with -h for its flags.
`

// run executes one command and returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return exitUsage
	}
	switch cmd, rest := args[0], args[1:]; cmd {
	case "rules":
		return rulesCmd(rest, stdout, stderr)
	case "check":
		return checkCmd(rest, stdout, stderr)
	case "new":
		return newCmd(rest, stdout, stderr)
	case "init":
		return initCmd(rest, stdout, stderr)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return exitOK
	default:
		fmt.Fprintf(stderr, "dashygotui: unknown command %q\n\n%s", cmd, usage)
		return exitUsage
	}
}
