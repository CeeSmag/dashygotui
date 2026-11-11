package dashygotui

import (
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

// Exec suspends the dashboard, runs cmd attached to the terminal (for example a
// pager, an editor or an interactive login), and resumes. The outcome arrives
// as a [TaskResult] with the given tag, with Err set when the command failed.
//
// This is the only place the library touches os/exec, and it is only a wrapper
// over Bubble Tea's own suspend-and-resume.
func (c *Context[S]) Exec(cmd *exec.Cmd, tag string) tea.Cmd {
	c.m.token++
	token := c.m.token // captured now: the callback runs off the update loop
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return taskMsg{res: TaskResult{Token: token, Tag: tag, Err: err}}
	})
}
