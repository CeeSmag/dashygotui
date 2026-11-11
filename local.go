package dashygotui

import "reflect"

// localKey identifies one value in a dashboard's local store: the name an addon
// chose, and the type it stores, so two addons cannot trip over each other's
// values by reusing a name for a different type.
type localKey struct {
	key string
	typ reflect.Type
}

// Local returns the value of type T that this dashboard keeps under key, the zero
// value on first use. It is where an addon keeps state of its own: a history it
// samples, a selection, a counter.
//
//	history := dashygotui.Local[[]float64](c, "cpu_trace")
//	*history = append(*history, c.State().CPU)
//
// The value belongs to the dashboard, not to the process: a second dashboard (in
// a test, or embedded in a larger program) starts with its own. That is the
// difference from a variable captured in an addon's init function, which is shared
// by every dashboard in the process. Use a key that names the addon, such as its
// option Key; the same key with another type is a different value.
//
// Call it on the update loop, which is where hooks, Render, Body and the other
// callbacks run. A function started by [Context.RunTask] runs elsewhere: read the
// value before starting it and hand the task what it needs. The pointer stays
// valid for the life of the dashboard.
func Local[T, S any](c *Context[S], key string) *T {
	return LocalWith(c, key, func() *T { return new(T) })
}

// LocalWith is [Local] for a value that needs constructing: init is called once,
// on first use, and returns the value to keep.
//
//	stream := dashygotui.LocalWith(c, "load", func() *chart.Stream { return chart.NewStream(chart.StreamOpts{}) })
func LocalWith[T, S any](c *Context[S], key string, init func() *T) *T {
	k := localKey{key: key, typ: reflect.TypeFor[T]()}
	if v, ok := c.m.locals[k].(*T); ok { // the key includes the type, so a hit is always a *T
		return v
	}
	v := init()
	if c.m.locals == nil {
		c.m.locals = map[localKey]any{}
	}
	c.m.locals[k] = v
	return v
}
