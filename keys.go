package dashygotui

import "charm.land/bubbles/v2/key"

// KeyMap lists the keys the dashboard itself reacts to. Zero-value bindings are
// replaced by their defaults, so a partial KeyMap only overrides what it sets.
type KeyMap struct {
	Up       key.Binding // move the cursor up
	Down     key.Binding // move the cursor down
	Left     key.Binding // previous panel
	Right    key.Binding // next panel
	PageUp   key.Binding // move up by several rows
	PageDown key.Binding // move down by several rows
	Home     key.Binding // first row
	End      key.Binding // last row
	Select   key.Binding // open the highlighted option
	NextPage key.Binding // next dashboard page
	PrevPage key.Binding // previous dashboard page
	Back     key.Binding // close a pop-up, leave a view
	Quit     key.Binding // quit from a page
	Force    key.Binding // quit from anywhere, including over a pop-up
	Slower   key.Binding // lower the refresh rate
	Faster   key.Binding // raise the refresh rate
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:     key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "previous panel")),
		Right:    key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "next panel")),
		PageUp:   key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
		PageDown: key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdn", "page down")),
		Home:     key.NewBinding(key.WithKeys("home"), key.WithHelp("home", "first")),
		End:      key.NewBinding(key.WithKeys("end"), key.WithHelp("end", "last")),
		Select:   key.NewBinding(key.WithKeys("enter", "space"), key.WithHelp("↵", "open")),
		NextPage: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next page")),
		PrevPage: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous page")),
		Back:     key.NewBinding(key.WithKeys("esc", "b"), key.WithHelp("esc", "back")),
		Quit:     key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		Force:    key.NewBinding(key.WithKeys("ctrl+q", "ctrl+c"), key.WithHelp("ctrl+q", "quit")),
		Slower:   key.NewBinding(key.WithKeys("-", "_"), key.WithHelp("-", "slower")),
		Faster:   key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "faster")),
	}
}

// withDefaults fills every unset binding from [DefaultKeyMap].
func (k KeyMap) withDefaults() KeyMap {
	d := DefaultKeyMap()
	fill := func(dst *key.Binding, def key.Binding) {
		if len(dst.Keys()) == 0 {
			*dst = def
		}
	}
	fill(&k.Up, d.Up)
	fill(&k.Down, d.Down)
	fill(&k.Left, d.Left)
	fill(&k.Right, d.Right)
	fill(&k.PageUp, d.PageUp)
	fill(&k.PageDown, d.PageDown)
	fill(&k.Home, d.Home)
	fill(&k.End, d.End)
	fill(&k.Select, d.Select)
	fill(&k.NextPage, d.NextPage)
	fill(&k.PrevPage, d.PrevPage)
	fill(&k.Back, d.Back)
	fill(&k.Quit, d.Quit)
	fill(&k.Force, d.Force)
	fill(&k.Slower, d.Slower)
	fill(&k.Faster, d.Faster)
	return k
}

// reserved reports whether name is bound to a navigation or control action, or
// is a digit, and so may not be claimed by a shortcut or hotkey. The force-quit
// keys count only when includeForce is set: a [ScopeEverywhere] shortcut may
// take them over.
func (k KeyMap) reserved(name string, includeForce bool) bool {
	bindings := []key.Binding{
		k.Up, k.Down, k.Left, k.Right, k.PageUp, k.PageDown, k.Home, k.End,
		k.Select, k.NextPage, k.PrevPage, k.Back, k.Quit, k.Slower, k.Faster,
	}
	if includeForce {
		bindings = append(bindings, k.Force)
	}
	for _, b := range bindings {
		for _, kk := range b.Keys() {
			if kk == name {
				return true
			}
		}
	}
	return len(name) == 1 && name[0] >= '0' && name[0] <= '9'
}
