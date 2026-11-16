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
		Title:    "Values",
		Pages:    []dashygotui.Page[valState]{{ID: "p", Title: "Page"}},
		Sections: []dashygotui.SectionSpec[valState]{{ID: "s", Page: "p", Title: "Settings"}},
		Addons:   addons,
	}, 140, 40)
}

func TestValueFormsDrawTheSameRows(t *testing.T) {
	t.Parallel()
	buf := logs.NewBuffer(10)
	d := valueDash(t,
		stdaddons.ThemePicker[valState]{Section: "s", Number: 1, Hotkey: "t"},
		stdaddons.RefreshRatePicker[valState]{Section: "s", Number: 2},
		stdaddons.SpacingMenu[valState]{Section: "s", Number: 3},
		stdaddons.DisplayPicker[valState]{Section: "s", Number: 4},
		stdaddons.BackgroundPicker[valState]{Section: "s", Number: 5},
		stdaddons.TransparencyToggle[valState]{Section: "s", Number: 6},
		stdaddons.LogView[valState]{Buffer: buf, Section: "s", Number: 7, Hotkey: "g"},
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
		Pages:    []dashygotui.Page[valState]{{ID: "p", Title: "Page"}},
		Sections: []dashygotui.SectionSpec[valState]{{ID: "s", Page: "p", Title: "Settings"}},
		Addons:   []dashygotui.Addon[valState]{dashygotui.At[valState]("opt_logs.go:3", stdaddons.LogView[valState]{Section: "s", Number: 1})},
	})
	if err == nil || !strings.Contains(err.Error(), "opt_logs.go:3: log view has no Buffer") {
		t.Fatalf("got %v", err)
	}
}
