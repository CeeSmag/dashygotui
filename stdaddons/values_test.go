package stdaddons_test

import (
	"strings"
	"testing"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/logs"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

type valState struct{}

func valueDash(t *testing.T, addons ...dashygotui.Addon[valState]) *dashtest.Driver {
	t.Helper()
	return dashtest.New(t, dashygotui.Config[valState]{
		Title: "Values",
		Map:   valueMap(addons...),
	}, 140, 40)
}

// valueMap places the addons as the rows of one panel.
func valueMap(addons ...dashygotui.Addon[valState]) dashygotui.Map[valState] {
	return dashygotui.Map[valState]{Pages: []dashygotui.Page[valState]{{ID: "p", Title: "Page",
		Panels: []dashygotui.Panel[valState]{{Title: "Settings", Items: addons}}}}}
}

func TestValueFormsDrawTheSameRows(t *testing.T) {
	t.Parallel()
	buf := logs.NewBuffer(10)
	d := valueDash(t,
		dashygotui.Hotkey[valState]{Key: "t", Item: stdaddons.ThemePicker[valState]{}},
		stdaddons.RefreshRatePicker[valState]{},
		stdaddons.SpacingMenu[valState]{},
		stdaddons.DisplayPicker[valState]{},
		stdaddons.BackgroundPicker[valState]{},
		stdaddons.TransparencyToggle[valState]{},
		dashygotui.Hotkey[valState]{Key: "g", Item: stdaddons.LogView[valState]{Buffer: buf}},
	)
	d.RequireContains("Color Theme", "Refresh Rate", "Section Gap", "Display Mode", "Background", "Transparent Background", "Logs")
	d.Press("t")
	d.RequireContains("Move to preview a palette")
	d.Press("esc", "g")
	d.RequireContains("LOGS")
}

func TestLogViewNeedsABuffer(t *testing.T) {
	t.Parallel()
	_, err := dashygotui.New(dashygotui.Config[valState]{
		Map: valueMap(stdaddons.LogView[valState]{}),
	})
	if err == nil || !strings.Contains(err.Error(), "Page > Settings > row 1: log view has no Buffer") {
		t.Fatalf("got %v", err)
	}
}
