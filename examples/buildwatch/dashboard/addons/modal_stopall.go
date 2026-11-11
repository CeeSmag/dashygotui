package addons

import (
	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/buildwatch/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// Ctrl+X cancels every running build, after asking. A shortcut can show a pop-up
// with Modal; Tone draws its footer hint in the theme's danger colour.

func init() {
	dashboard.Registry.Add(dashboard.Shortcut{
		Key: "ctrl+x", Display: "Ctrl+X", Label: "Cancel all", Tone: dashygotui.ToneDanger,
		Scope: dashygotui.ScopeEverywhere,
		Modal: func(*dashboard.Ctx) dashboard.Modal {
			return &modal.Confirm[dashboard.State]{
				Title: "Cancel every running build?", Lines: []string{"Builds in progress are stopped."}, Accept: "Cancel builds",
				OnAccept: func(c *dashboard.Ctx) tea.Cmd { c.Flash("Cancelled all running builds"); return nil },
			}
		},
	})
}
