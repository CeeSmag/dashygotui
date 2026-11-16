package dashygotui_test

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/panel"
)

// shop is the application state the examples show.
type shop struct{ Orders int }

// An addon is a value, usually alone in its own file under addons/, where the
// generated addons.All lists it. The dashboard file declares the pages and
// sections and passes the list in Config.Addons.
func Example() {
	refund := dashygotui.Option[shop]{
		Key: "refund", Section: "actions", Number: 1, Label: "Issue Refund",
		Badge: func(c *dashygotui.Context[shop]) string { return c.Count(c.State().Orders, "orders") },
		Do:    func(c *dashygotui.Context[shop]) { c.Flash("Refund issued") },
	}
	orders := dashygotui.Widget[shop]{
		Key: "orders", Page: "main", Title: "Orders",
		Render: func(c *dashygotui.Context[shop], _ dashygotui.Size) []string {
			return []string{panel.Row(c.Theme(), "Open", fmt.Sprint(c.State().Orders))}
		},
	}

	// The dashboard file: pages, sections, and how to fetch the state.
	dash, err := dashygotui.New(dashygotui.Config[shop]{
		Title:    "Shop",
		Addons:   []dashygotui.Addon[shop]{refund, orders}, // addons.All() in an application
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

// A mistake in an addon is reported before the UI starts, naming the place
// each offending addon is declared: the place At records, as the generated
// list does for every addon.
func ExampleAt() {
	run := func(*dashygotui.Context[shop]) tea.Cmd { return nil }
	_, err := dashygotui.New(dashygotui.Config[shop]{
		Pages:    []dashygotui.Page[shop]{{ID: "main", Title: "Main"}},
		Sections: []dashygotui.SectionSpec[shop]{{ID: "actions", Page: "main", Title: "Actions"}},
		Addons: []dashygotui.Addon[shop]{
			dashygotui.At[shop]("opt_a.go:5", dashygotui.Option[shop]{Key: "a", Section: "actions", Number: 7, Label: "A", Run: run}),
			dashygotui.At[shop]("opt_b.go:5", dashygotui.Option[shop]{Key: "b", Section: "actions", Number: 7, Label: "B", Run: run}),
		},
	})
	fmt.Println(err)
	// Output: invalid dashboard configuration: opt_b.go:5: option "b" reuses number 7 (first used at opt_a.go:5)
}

// New reports every problem at once, wrapped in ErrInvalidConfig.
func ExampleNew_invalid() {
	ghost := dashygotui.Option[shop]{
		Key: "ghost", Section: "nowhere", Number: 1, Label: "Ghost",
		Do: func(*dashygotui.Context[shop]) {},
	}
	_, err := dashygotui.New(dashygotui.Config[shop]{
		Addons: []dashygotui.Addon[shop]{ghost},
		Pages:  []dashygotui.Page[shop]{{ID: "main", Title: "Main"}},
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
