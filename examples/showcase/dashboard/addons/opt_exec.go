package addons

import (
	"context"
	"os/exec"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// [72] Run External Command: Context.Exec suspends the dashboard, runs a program
// attached to the terminal, and resumes. The outcome arrives as a TaskResult
// tagged "exec", which an OnResult hook turns into a footer message. (This one
// needs a Unix shell; elsewhere it reports the error.)

func init() {
	const tag = "exec"

	dashboard.Registry.Option(dashboard.Option{
		Key: "exec", Section: "hooks", Number: 72, Label: "Run External Command",
		Run: func(c *dashboard.Ctx) tea.Cmd {
			cmd := exec.CommandContext(context.Background(), "sh", "-c", `printf '\nHello from a child process.\n\nThe dashboard is suspended while this runs.\n'; printf 'Press Enter to return... '; read _`)
			return c.Exec(cmd, tag)
		},
	})

	dashboard.Registry.OnResult(func(c *dashboard.Ctx, res dashygotui.TaskResult) (bool, tea.Cmd) {
		if res.Tag != tag {
			return false, nil
		}
		if res.Err != nil {
			c.FlashError("External command failed: " + res.Err.Error())
		} else {
			c.Flash("External command finished")
		}
		return true, nil
	})
}
