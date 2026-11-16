package dashygotui_test

import (
	"context"
	"fmt"
	"time"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/panel"
)

// shop is the application state the examples show.
type shop struct{ Orders int }

// A feature is a value, usually alone in its own file under addons/, that says
// only what it does. The map, in addons/addons.go, says where everything goes.
func Example() {
	refund := dashygotui.Option[shop]{
		Label: "Issue Refund",
		Badge: func(c *dashygotui.Context[shop]) string { return c.Count(c.State().Orders, "orders") },
		Do:    func(c *dashygotui.Context[shop]) { c.Flash("Refund issued") },
	}
	orders := dashygotui.Widget[shop]{
		Title: "Orders",
		Render: func(c *dashygotui.Context[shop], _ dashygotui.Size) []string {
			return []string{panel.Row(c.Theme(), "Open", fmt.Sprint(c.State().Orders))}
		},
	}

	dash, err := dashygotui.New(dashygotui.Config[shop]{
		Title: "Shop",
		Map: dashygotui.Map[shop]{Pages: []dashygotui.Page[shop]{ // addons.Map() in an application
			{Title: "Main", Panels: []dashygotui.Panel[shop]{
				{Widget: orders},
				{Title: "Actions", Items: []dashygotui.Addon[shop]{refund}}, // [1]
			}},
		}},
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

// New reports every problem at once, wrapped in ErrInvalidConfig, each with its
// place in the map.
func ExampleNew_invalid() {
	idle := dashygotui.Option[shop]{Label: "Idle"}
	_, err := dashygotui.New(dashygotui.Config[shop]{Map: dashygotui.Map[shop]{Pages: []dashygotui.Page[shop]{
		{Title: "Main", Panels: []dashygotui.Panel[shop]{{Title: "Actions", Items: []dashygotui.Addon[shop]{idle}}}},
	}}})
	fmt.Println(err)
	// Output: invalid dashboard configuration: Main > Actions > row 1: option "Idle" needs exactly one of Run, Do, Opens and Modal, has 0
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
