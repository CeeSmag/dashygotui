package addons

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [14] Alert Volume: a modal.Slider with live OnChange, OnDone to save, and Esc
// restoring the original value.

func init() {
	const volumeKey = "alerts.volume"
	volume := func(c *dashboard.Ctx) int {
		v := 60
		_, _ = c.Store().Get(volumeKey, &v)
		return v
	}

	dashboard.Registry.Add(dashboard.Option{
		Key: "volume", Section: "settings", Number: 14, Label: "Alert Volume",
		Badge: func(c *dashboard.Ctx) string { return c.Badge(fmt.Sprintf("%d%%", volume(c))) },
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Slider[dashboard.State]{
				Title: "Alert Volume", Help: "How loud alerts are.", Unit: "percent",
				Min: 0, Max: 100, Value: volume(c),
				OnChange: func(c *dashboard.Ctx, v int) { c.Flash(fmt.Sprintf("Volume %d%%", v)) },
				OnDone: func(c *dashboard.Ctx, v int) tea.Cmd {
					c.FlashResult(fmt.Sprintf("Volume saved at %d%%.", v), c.Store().Set(volumeKey, v))
					return nil
				},
			}
		},
	})
}
