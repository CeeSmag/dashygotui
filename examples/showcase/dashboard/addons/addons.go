// Package addons holds the showcase's features, one per file, and the map that
// places them. A feature file says what the feature does; only the map says where
// it goes. The map is split by page: this file lists the tabs, the shortcuts and
// the hooks, and each <name>_map.go maps one page or window. Rows are numbered by
// position; a panel's Number sets where its count starts.
package addons

import "github.com/CeeSmag/dashygotui/examples/showcase/dashboard"

// Map is what the showcase's dashboard contains and where.
func Map() dashboard.Map {
	return dashboard.Map{
		Shortcuts: []dashboard.Shortcut{refreshShortcut, stopShortcut, clearlogsShortcut, helpShortcut},
		Hooks:     []dashboard.Hook{alertHook, exportResult, execResult, streamTick},
		Pages:     []dashboard.Page{monitorPage(), chartsPage(), actionsPage(), popupsPage(), viewsPage(), themePage()},
	}
}
