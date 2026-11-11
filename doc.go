// Package dashygotui is a generic framework for building terminal dashboards
// on top of Bubble Tea v2.
//
// An application declares its pages and section slots once, in a single
// dashboard file, and owns one [Registry] for its state type. Options, widgets,
// views and shortcuts are then added by dropping one self-registering file
// into an addons directory; no other file is edited.
//
// See RULES.md for the architecture and coding rules that every contribution
// must follow.
package dashygotui
