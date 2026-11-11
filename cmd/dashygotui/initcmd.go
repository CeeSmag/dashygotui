package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var versionRe = regexp.MustCompile(`(?m)github\.com/CeeSmag/dashygotui\s+(v\S+)`)

// agentsText is the file that tells a person or an AI agent how to work on the
// application. %s is the dashygotui command with the pinned version.
const agentsText = `# Working on this application

This application is built on the **dashygotui** library. Its design rules matter
more here than anywhere: the library does the work, the application stays small.

Before changing the dashboard, read the rules. They match the library version this
application uses:

    %[1]s rules

(topics: design, app-rules, cookbook; ` + "`rules -list`" + ` shows the checks.)

The short version:

1. **One feature, one file** in ` + "`addons/`" + `. Add a file; edit no other. Start from
   ` + "`%[1]s new <opt|widget|view|modal|shortcut> <name>`" + `.
2. **Use the short forms** from the cookbook (` + "`Opens`, `Do`, `Entry`, `stdaddons.Toggle`" + `...).
   If an addon is long, something is missing from the library.
3. **If the library cannot do it, stop and record a gap** in GAPS.md. Do not wrap,
   copy or fork library code, and do not override a default to get around a limit.
   A workaround that cannot wait is marked ` + "`// dashygotui:gap <where it is recorded>`" + `.
4. **Do only what was asked.** Smallest diff; no files created, renamed or deleted
   that were not requested.
5. **Run the checks before declaring a task done:**

       %[1]s check -changed
       go test ./...

   A deliberate exception is ` + "`//dashygotui:allow <rule> <reason>`" + ` on the line.
6. Commit or push only when asked.
`

const gapsText = `# Gaps

Things this application needed that the dashygotui library does not provide yet.
Each is also a candidate issue for the library (use its gap template). Mark the
workaround in code with ` + "`// dashygotui:gap GAPS.md#<heading>`" + `, and delete both when
a release closes the gap.

Template for an entry:

## <short name>

- **Needed:** what the application tried to do.
- **Did instead:** the workaround, and where it is (file).
- **Shortest API that would have avoided it:** ...
- **Would a second application use it?** ...

(No gaps recorded yet.)
`

func initCmd(args []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("init", flag.ContinueOnError)
	fl.SetOutput(stderr)
	force := fl.Bool("force", false, "overwrite files that already exist")
	version := fl.String("version", "", "the dashygotui version to pin in the instructions (default: read from go.mod)")
	claude := fl.Bool("claude", false, "also write a Claude Code skill (.claude/skills/dashygotui/SKILL.md) and a hook in .claude/settings.json that runs the check after every edit")
	if err := fl.Parse(args); err != nil {
		return exitUsage
	}
	v := *version
	if v == "" {
		if b, err := os.ReadFile("go.mod"); err == nil {
			if m := versionRe.FindSubmatch(b); m != nil {
				v = string(m[1])
			}
		}
	}
	if v == "" {
		v = "latest"
		fmt.Fprintln(stderr, "dashygotui init: no dashygotui version in go.mod; pinning \"latest\" (edit AGENTS.md once you depend on a version)")
	}
	cmd := "go run github.com/CeeSmag/dashygotui/cmd/dashygotui@" + v
	agents := fmt.Sprintf(agentsText, cmd)

	files := []struct{ name, body string }{
		{"AGENTS.md", agents},
		{"CLAUDE.md", strings.Replace(agents, "# Working on this application", "# CLAUDE.md", 1)},
		{"GAPS.md", gapsText},
	}
	code := exitOK
	for _, f := range files {
		if _, err := os.Stat(f.name); err == nil && !*force {
			fmt.Fprintf(stdout, "kept    %s (exists; -force overwrites)\n", f.name)
			continue
		}
		if err := os.WriteFile(f.name, []byte(f.body), 0o644); err != nil { //nolint:gosec // documentation, readable by design
			fmt.Fprintf(stderr, "dashygotui init: writing %s: %v\n", f.name, err)
			code = exitUsage
			continue
		}
		fmt.Fprintf(stdout, "wrote   %s\n", f.name)
	}
	if *claude {
		if c := claudeSetup(cmd, *force, stdout, stderr); c != exitOK {
			code = c
		}
	}
	fmt.Fprintf(stdout, "\nAdd the check to your build, for example in a Makefile:\n\n  dashygotui-check:\n\t%s check\n", cmd)
	return code
}

// skillText is the Claude Code skill: when to load the rules, and the loop.
const skillText = `---
name: dashygotui
description: Use when adding or changing anything on this application's dashboard (an option, widget, view, shortcut, pop-up or setting), or when something the dashboard library should provide seems to be missing. The application is built on dashygotui and its rules are strict.
---

# Working on a dashygotui application

1. Read the rules for the library version this application uses:

       %[1]s rules

2. Look in the cookbook part of that output for the shortest form. Most addons are a few
   lines (` + "`Opens`, `Do`, `Modal`, `Entry`, `stdaddons.Toggle/Number/Pick/Text`" + `).
3. Add **one new file** in ` + "`addons/`" + ` (` + "`%[1]s new <kind> <name>`" + ` writes a starter).
   Edit no other file. If another file really must change, say why in your reply.
4. Keep an addon's own state with ` + "`dashygotui.Local`" + `, never a variable captured in ` + "`init`" + `.
5. If the library cannot do what is needed: **stop.** Do not wrap, copy or fork library code, and do not
   override a default to get around a limit. Record the gap in GAPS.md and tell the user.
6. Before saying you are done:

       %[1]s check -changed
       go test ./...
`

// hookCommand runs the check after an edit and hands its findings to the agent
// (exit status 2 reports a hook's stderr back).
func hookCommand(cmd string) string { return "sh -c '" + cmd + " check -q 1>&2 || exit 2'" }

// claudeSetup writes the skill and merges the hook into .claude/settings.json,
// leaving everything else in that file alone.
func claudeSetup(cmd string, force bool, stdout, stderr io.Writer) int {
	skill := filepath.Join(".claude", "skills", "dashygotui", "SKILL.md")
	if _, err := os.Stat(skill); err == nil && !force {
		fmt.Fprintf(stdout, "kept    %s (exists; -force overwrites)\n", skill)
	} else {
		if err := os.MkdirAll(filepath.Dir(skill), 0o755); err != nil { //nolint:gosec // a project directory, readable by design
			fmt.Fprintf(stderr, "dashygotui init: creating %s: %v\n", filepath.Dir(skill), err)
			return exitUsage
		}
		if err := os.WriteFile(skill, []byte(fmt.Sprintf(skillText, cmd)), 0o644); err != nil { //nolint:gosec // documentation, readable by design
			fmt.Fprintf(stderr, "dashygotui init: writing %s: %v\n", skill, err)
			return exitUsage
		}
		fmt.Fprintf(stdout, "wrote   %s\n", skill)
	}

	settings := filepath.Join(".claude", "settings.json")
	changed, err := mergeHook(settings, hookCommand(cmd))
	switch {
	case err != nil:
		fmt.Fprintf(stderr, "dashygotui init: %s: %v\n", settings, err)
		return exitUsage
	case changed:
		fmt.Fprintf(stdout, "wrote   %s (a hook that runs the check after every edit)\n", settings)
	default:
		fmt.Fprintf(stdout, "kept    %s (the hook is already there)\n", settings)
	}
	return exitOK
}

// mergeHook adds a PostToolUse hook running command to the settings file, creating
// it if need be, and reports whether anything changed. A file that is not a JSON
// object is an error and is never overwritten.
func mergeHook(path, command string) (bool, error) {
	doc := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		if len(strings.TrimSpace(string(b))) > 0 {
			if err := json.Unmarshal(b, &doc); err != nil {
				return false, fmt.Errorf("not a JSON object, left alone: %w", err)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("reading: %w", err)
	}

	hooks, _ := doc["hooks"].(map[string]any)
	if hooks == nil {
		if _, exists := doc["hooks"]; exists {
			return false, errors.New(`"hooks" is not an object, left alone`)
		}
		hooks = map[string]any{}
	}
	post, _ := hooks["PostToolUse"].([]any)
	if _, exists := hooks["PostToolUse"]; exists && post == nil {
		return false, errors.New(`"hooks.PostToolUse" is not a list, left alone`)
	}
	for _, e := range post {
		if b, err := json.Marshal(e); err == nil && strings.Contains(string(b), "cmd/dashygotui") {
			return false, nil // already installed, possibly for another version
		}
	}
	post = append(post, map[string]any{
		"matcher": "Edit|Write|MultiEdit",
		"hooks":   []any{map[string]any{"type": "command", "command": command}},
	})
	hooks["PostToolUse"] = post
	doc["hooks"] = hooks

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // keep ">" and "&" readable in the command
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return false, fmt.Errorf("encoding: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // a project directory, readable by design
		return false, fmt.Errorf("creating directory: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil { //nolint:gosec // settings, readable by design
		return false, fmt.Errorf("writing: %w", err)
	}
	return true, nil
}
