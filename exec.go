package dashygotui

import (
	"io"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Exec suspends the dashboard, runs cmd attached to the terminal (for example a
// pager, an editor or an interactive login), and resumes. The outcome arrives
// as a [TaskResult] with the given tag, with Err set when the command failed.
//
// This is the only place the library touches os/exec, and it is only a wrapper
// over Bubble Tea's own suspend-and-resume. [Config.Exec] runs it.
func (c *Context[S]) Exec(cmd *exec.Cmd, tag string) tea.Cmd {
	c.m.token++
	token := c.m.token // captured now: the callback runs off the update loop
	return c.m.d.cfg.Exec(execCommand{cmd}, func(err error) tea.Msg {
		return taskMsg{res: TaskResult{Token: token, Tag: tag, Err: err}}
	})
}

// execCommand lets an *exec.Cmd run under [tea.Exec], as tea.ExecProcess does.
// String gives its arguments, so a test can see what was run without importing
// os/exec.
type execCommand struct{ cmd *exec.Cmd }

func (e execCommand) Run() error { return e.cmd.Run() }

func (e execCommand) SetStdin(r io.Reader) {
	if e.cmd.Stdin == nil {
		e.cmd.Stdin = r
	}
}

func (e execCommand) SetStdout(w io.Writer) {
	if e.cmd.Stdout == nil {
		e.cmd.Stdout = w
	}
}

func (e execCommand) SetStderr(w io.Writer) {
	if e.cmd.Stderr == nil {
		e.cmd.Stderr = w
	}
}

// String is the command line: the arguments joined by spaces, such as
// "vi notes.txt".
func (e execCommand) String() string { return strings.Join(e.cmd.Args, " ") }
