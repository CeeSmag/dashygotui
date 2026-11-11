package dashygotui_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/panel"
)

// shop is the application state the examples show.
type shop struct{ Orders int }

// The registry is declared once, in the application's dashboard file. Addons
// register with it from init(); in a test or an example, a fresh one is used.
func Example() {
	reg := dashygotui.NewRegistry[shop]()

	// An addon: one self-contained registration, usually in its own file.
	reg.Option(dashygotui.Option[shop]{
		Key: "refund", Section: "actions", Number: 1, Label: "Issue Refund",
		Badge: func(c *dashygotui.Context[shop]) string { return c.Count(c.State().Orders, "orders") },
		Run: func(c *dashygotui.Context[shop]) tea.Cmd {
			c.Flash("Refund issued")
			return nil
		},
	})
	reg.Widget(dashygotui.Widget[shop]{
		Key: "orders", Page: "main", Title: "Orders",
		Render: func(c *dashygotui.Context[shop], _ dashygotui.Size) []string {
			return []string{panel.Row(c.Theme(), "Open", fmt.Sprint(c.State().Orders))}
		},
	})

	// The dashboard file: pages, sections, and how to fetch the state.
	dash, err := dashygotui.New(dashygotui.Config[shop]{
		Title:    "Shop",
		Registry: reg,
		Pages:    []dashygotui.Page[shop]{{ID: "main", Title: "Main"}},
		Sections: []dashygotui.SectionSpec[shop]{{ID: "actions", Page: "main", Title: "Actions"}},
		Poll: func(context.Context) (shop, error) {
			return shop{Orders: 12}, nil
		},
		Interval: time.Second,
	})
	if err != nil {
		fmt.Println("invalid:", err)
		return
	}
	// dash.Run(ctx) would take over the terminal; see examples/demo.
	_ = dash
	fmt.Println("dashboard ready")
	// Output: dashboard ready
}

// A mistake in an addon is recorded, not panicked on, and reported before the
// UI starts, naming the file and line of each offending registration.
func ExampleRegistry_Validate() {
	reg := dashygotui.NewRegistry[shop]()
	run := func(*dashygotui.Context[shop]) tea.Cmd { return nil }
	reg.Option(dashygotui.Option[shop]{Key: "a", Section: "actions", Number: 7, Label: "A", Run: run})
	reg.Option(dashygotui.Option[shop]{Key: "b", Section: "actions", Number: 7, Label: "B", Run: run})

	err := reg.Validate()
	// The message ends in "(first used at example_test.go:NN)"; print the part
	// that does not depend on line numbers.
	msg := err.Error()
	fmt.Println(strings.Contains(msg, `option "b" reuses number 7`))
	fmt.Println(strings.HasPrefix(msg, "example_test.go:"))
	// Output:
	// true
	// true
}

// New reports every problem at once, wrapped in ErrInvalidConfig.
func ExampleNew_invalid() {
	reg := dashygotui.NewRegistry[shop]()
	reg.Option(dashygotui.Option[shop]{
		Key: "ghost", Section: "nowhere", Number: 1, Label: "Ghost",
		Run: func(*dashygotui.Context[shop]) tea.Cmd { return nil },
	})
	_, err := dashygotui.New(dashygotui.Config[shop]{
		Registry: reg,
		Pages:    []dashygotui.Page[shop]{{ID: "main", Title: "Main"}},
	})
	fmt.Println(err)
	// Output: invalid dashboard configuration: option "ghost" targets unknown section "nowhere"
}

func ExampleDefaultSpacing() {
	sp := dashygotui.DefaultSpacing()
	fmt.Printf("%+v\n", sp)
	fmt.Printf("%+v\n", dashygotui.Spacing{TerminalPage: 99, Section: -3}.Clamp())
	// Output:
	// {TerminalPage:0 PageSection:0 Section:2 SectionOption:0}
	// {TerminalPage:6 PageSection:0 Section:0 SectionOption:0}
}

func ExampleKeyMap() {
	// Override only what you need; everything else keeps its default.
	keys := dashygotui.DefaultKeyMap()
	keys.Quit.SetKeys("x")
	cfg := dashygotui.Config[shop]{Keys: dashygotui.KeyMap{Quit: keys.Quit}}
	fmt.Println(cfg.Keys.Quit.Keys())
	// Output: [x]
}

func ExampleRefreshIntervals() {
	for _, d := range dashygotui.RefreshIntervals()[:3] {
		fmt.Println(d)
	}
	// Output:
	// 100ms
	// 250ms
	// 500ms
}
