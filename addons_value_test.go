package dashygotui_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
)

// valueConfig is baseConfig with its addons passed as values, not registered.
func valueConfig(addons ...dashygotui.Addon[state]) cfg {
	c := baseConfig(nil)
	c.Addons = addons
	return c
}

func TestAddonsAsValues(t *testing.T) {
	t.Parallel()
	ran := 0
	d := dashtest.New(t, valueConfig(
		option{Key: "backup", Section: "actions", Number: 1, Label: "Run Backup", Do: func(*ctx) { ran++ }},
		widget{Key: "stats", Page: "monitor", Title: "Stats", Render: func(*ctx, dashygotui.Size) []string { return []string{"hello"} }},
		dashygotui.View[state]{ID: "notes", Title: "Notes", Body: func(*ctx, dashygotui.Size) []string { return []string{"a note"} },
			Entry: &dashygotui.Entry[state]{Section: "actions", Number: 2}},
	), 120, 30)
	d.RequireContains("Stats", "hello")
	d.Type("1")
	d.Press("enter")
	if ran != 1 {
		t.Fatalf("the option ran %d times, want 1", ran)
	}
	d.Type("2")
	d.Press("enter")
	d.RequireContains("a note")
}

func TestNoAddonsIsAValidDashboard(t *testing.T) {
	t.Parallel()
	d := dashtest.New(t, valueConfig(), 120, 30) // every addon file deleted
	d.RequireContains("TestApp", "Monitor")
}

func TestAddonsAndRegistryTogether(t *testing.T) {
	t.Parallel()
	r := dashygotui.NewRegistry[state]()
	r.Option(validOption("old", 1))
	c := valueConfig(validOption("new", 1))
	c.Registry = r
	msg := errText(t, c)
	if !strings.Contains(msg, `option "new" reuses number 1`) {
		t.Fatalf("a clash between the registry and Addons is not reported: %s", msg)
	}
	c.Addons = []dashygotui.Addon[state]{validOption("new", 2)}
	if msg := errText(t, c); msg != "" {
		t.Fatalf("New with both: %s", msg)
	}
}

func TestNewDoesNotChangeTheRegistry(t *testing.T) {
	t.Parallel()
	r := dashygotui.NewRegistry[state]()
	r.Option(validOption("old", 1))
	c := valueConfig(validOption("new", 2))
	c.Registry = r
	for range 2 { // a second dashboard from the same registry must not see the first one's addons twice
		if msg := errText(t, c); msg != "" {
			t.Fatal(msg)
		}
	}
}

func TestAtNamesTheDeclarationInErrors(t *testing.T) {
	t.Parallel()
	msg := errText(t, valueConfig(
		dashygotui.At[state]("opt_storage.go:11", validOption("storage", 12)),
		dashygotui.At[state]("opt_trash.go:14", validOption("trash", 12)),
		validOption("unsited", 12),
	))
	for _, want := range []string{
		`opt_trash.go:14: option "trash" reuses number 12 (first used at opt_storage.go:11)`,
		`Config.Addons[2]: option "unsited" reuses number 12 (first used at opt_storage.go:11)`,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error does not contain %q:\n%s", want, msg)
		}
	}
}

func TestAtReachesInsideComposites(t *testing.T) {
	t.Parallel()
	msg := errText(t, valueConfig(
		dashygotui.At[state]("opt_a.go:3", pair{validOption("a", 5), validOption("b", 5)}),
	))
	if !strings.Contains(msg, `opt_a.go:3: option "b" reuses number 5 (first used at opt_a.go:3)`) {
		t.Fatalf("a composite's parts do not carry its site:\n%s", msg)
	}
}

// pair is a composite addon of two.
type pair [2]dashygotui.Addon[state]

func (p pair) Addons() []dashygotui.Addon[state] { return p[:] }

func TestHookValue(t *testing.T) {
	t.Parallel()
	var ticks, msgs, results int
	c := valueConfig(
		option{Key: "job", Section: "actions", Number: 1, Label: "Job", Hotkey: "w", Run: func(c *ctx) tea.Cmd {
			return c.RunTask("job", "", func(context.Context) (string, error) { return "done", nil })
		}},
		dashygotui.Hook[state]{
			Tick: func(*ctx, time.Time) tea.Cmd { ticks++; return nil },
			Msg:  func(_ *ctx, m tea.Msg) tea.Cmd { _, ok := m.(customMsg); msgs += map[bool]int{true: 1}[ok]; return nil },
			Result: func(c *ctx, res dashygotui.TaskResult) (bool, tea.Cmd) {
				if res.Tag != "job" {
					return false, nil
				}
				results++
				c.Flash("claimed by the hook")
				return true, nil
			},
		},
	)
	d := dashtest.New(t, c, 120, 30)
	d.Tick()
	d.Send(customMsg{n: 1})
	d.Press("w")
	if ticks == 0 || msgs != 1 || results != 1 {
		t.Fatalf("ticks=%d msgs=%d results=%d, want each hook called", ticks, msgs, results)
	}
	d.RequireContains("claimed by the hook")
	d.RequireAbsent("done")
}

func TestEmptyHookIsAnError(t *testing.T) {
	t.Parallel()
	msg := errText(t, valueConfig(dashygotui.At[state]("opt_x.go:2", dashygotui.Hook[state]{})))
	if !strings.Contains(msg, "opt_x.go:2: hook sets none of Tick, Msg and Result") {
		t.Fatalf("got %q", msg)
	}
}

func TestNilAddonInTheListIsAnError(t *testing.T) {
	t.Parallel()
	_, err := dashygotui.New(valueConfig(nil))
	if !errors.Is(err, dashygotui.ErrInvalidConfig) || !strings.Contains(err.Error(), "Config.Addons[0]") {
		t.Fatalf("got %v", err)
	}
}
