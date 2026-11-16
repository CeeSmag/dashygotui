// Package dashygotui is a generic framework for building terminal dashboards
// on top of Bubble Tea v2.
//
// An application's features (options, widgets, views, sub-pages, shortcuts,
// hooks) are values, one file per feature in an addons directory, and each says
// only what it does. One hand-written file in that directory, the map
// (addons/addons.go), says where everything goes: the pages, the panels on each
// page and the rows in each panel. The application passes the map to [New] in
// [Config.Map]. Adding a feature adds its file and its line in the map.
//
// See RULES.md for the architecture and coding rules that every contribution
// must follow.
package dashygotui
