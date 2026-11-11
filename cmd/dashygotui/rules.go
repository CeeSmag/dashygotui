package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"

	"github.com/CeeSmag/dashygotui/docs"
	"github.com/CeeSmag/dashygotui/internal/guard"
)

// topics maps a rules topic to the document that holds it.
func topics() map[string]string {
	return map[string]string{
		"design":       "DESIGN.md",
		"app-rules":    "APP-RULES.md",
		"cookbook":     "COOKBOOK.md",
		"addons":       "ADDONS.md",
		"architecture": "ARCHITECTURE.md",
		"reference":    "REFERENCE.md",
	}
}

// defaultTopics are printed when no topic is named: what an agent must read
// before writing an addon.
func defaultTopics() []string { return []string{"design", "app-rules", "cookbook"} }

func rulesCmd(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("rules", flag.ContinueOnError)
	fl.SetOutput(stderr)
	list := fl.Bool("list", false, "list the topics and the checked rules")
	if err := fl.Parse(args); err != nil {
		return exitUsage
	}
	if *list {
		names := make([]string, 0)
		for n := range topics() {
			names = append(names, n)
		}
		sort.Strings(names)
		fmt.Fprintln(stdout, "topics: "+strings.Join(names, ", "))
		fmt.Fprintln(stdout, "\nrules checked by `dashygotui check`:")
		rules := guard.Rules()
		ids := make([]string, 0, len(rules))
		for id := range rules {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			fmt.Fprintf(stdout, "  %-3s %s\n", id, rules[id])
		}
		return exitOK
	}

	want := fl.Args()
	if len(want) == 0 {
		want = defaultTopics()
	}
	for i, topic := range want {
		file, ok := topics()[topic]
		if !ok {
			fmt.Fprintf(stderr, "dashygotui: unknown topic %q (try dashygotui rules -list)\n", topic)
			return exitUsage
		}
		body, err := fs.ReadFile(docs.FS, file)
		if err != nil {
			fmt.Fprintf(stderr, "dashygotui: reading %s: %v\n", file, err)
			return exitUsage
		}
		if i > 0 {
			fmt.Fprint(stdout, "\n\n---\n\n")
		}
		fmt.Fprint(stdout, string(body))
	}
	return exitOK
}
