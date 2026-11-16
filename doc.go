// Package dashygotui is a generic framework for building terminal dashboards
// on top of Bubble Tea v2.
//
// An application declares its pages and section slots once, in a single
// dashboard file. Options, widgets, views, shortcuts and hooks are values, one
// file per feature in an addons directory; the dashygotui gen command writes
// the list of them (addons.All), which the application passes to [New] in
// [Config.Addons]. Adding a feature adds one file; no other file is edited.
//
// See RULES.md for the architecture and coding rules that every contribution
// must follow.
package dashygotui
