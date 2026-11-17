package stdaddons

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/browse"
	"github.com/CeeSmag/dashygotui/modal"
	"github.com/CeeSmag/dashygotui/panel"
)

// DirBrowser is a full-screen directory picker and the row that opens it. Place
// it in a panel's Items in the map:
//
//	stdaddons.DirBrowser[dashboard.State]{
//		Label: "Choose Folder",
//		Start: func(c *dashboard.Ctx) string { return c.State().Home },
//		OnSelect: func(c *dashboard.Ctx, path string) tea.Cmd {
//			return c.RunTask("folder", "Saving...", func(ctx context.Context) (string, error) {
//				return c.State().SetFolder(ctx, path)
//			})
//		},
//	}
//
// The view starts at Start (or the user home), lists directories with item count,
// size and modified time, and uses its own keys: Enter opens, Ctrl+Enter
// selects the current directory, Ctrl+B goes to the parent, Ctrl+F searches,
// letter and digit keys jump, and Ctrl+D creates a directory.
type DirBrowser[S any] struct {
	// Label is the row text. Required.
	Label string
	// Title is the view title. It defaults to Label.
	Title string
	// ID names the view for Opens. It defaults to "stdaddons.dirbrowser".
	ID string
	// Badge is the row badge. Optional.
	Badge func(*dashygotui.Context[S]) string
	// Start returns the directory to open when the view is entered. Empty or
	// unset uses the user home directory.
	Start func(*dashygotui.Context[S]) string
	// OnSelect runs after the user confirms a directory with Ctrl+Enter. The
	// view has already closed. path is absolute. Optional.
	OnSelect func(ctx *dashygotui.Context[S], path string) tea.Cmd
	// Pending is the footer message while a create-directory Save runs.
	// Defaults to "Creating the folder...".
	Pending string
}

// Addons implements [dashygotui.Addon].
func (a DirBrowser[S]) Addons() []dashygotui.Addon[S] {
	if strings.TrimSpace(a.Label) == "" {
		return invalid[S]("stdaddons.DirBrowser has no Label")
	}
	viewID := a.ID
	if viewID == "" {
		viewID = "stdaddons.dirbrowser"
	}
	title := a.Title
	if title == "" {
		title = a.Label
	}
	pending := a.Pending
	if pending == "" {
		pending = "Creating the folder..."
	}
	startOf := func(ctx *dashygotui.Context[S]) string {
		if a.Start != nil {
			if s := strings.TrimSpace(a.Start(ctx)); s != "" {
				return s
			}
		}
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return "/"
		}
		return home
	}
	navOf := func(ctx *dashygotui.Context[S]) *browse.Navigator {
		return dashygotui.LocalWith(ctx, viewID, func() *browse.Navigator {
			return browse.New(startOf(ctx))
		})
	}

	view := dashygotui.View[S]{
		ID:    viewID,
		Title: title,
		Mode:  "DIR",
		OnEnter: func(ctx *dashygotui.Context[S]) tea.Cmd {
			navOf(ctx).Reset(startOf(ctx))
			return nil
		},
		Body: func(ctx *dashygotui.Context[S], size dashygotui.Size) []string {
			th, nav := ctx.Theme(), navOf(ctx)
			sub := nav.DisplayPath(nav.Dir())
			if nav.Searching() {
				sub += "  Search: " + nav.Filter() + th.G.EchoChar
			}
			if err := nav.Err(); err != "" {
				sub += "  " + err
			}
			body := nav.Rows(th, max(size.Cols-4, 1), max(size.Rows-2, 1))
			return panel.Panel(th, title, "", sub, size.Cols, size.Rows, body)
		},
		Hints: func(ctx *dashygotui.Context[S]) []dashygotui.Hint {
			nav := navOf(ctx)
			if nav.Searching() {
				return []dashygotui.Hint{
					{Key: "type", Label: "filter", Prio: 10},
					{Key: "Esc", Label: "clear", Prio: 20},
					{Key: "Enter", Label: "open", Prio: 30},
					{Key: "Ctrl+Enter", Label: "select", Prio: 40},
				}
			}
			return []dashygotui.Hint{
				{Key: ctx.Theme().G.UpDown, Label: "move", Prio: 10},
				{Key: "Enter", Label: "open", Prio: 20},
				{Key: "Ctrl+Enter", Label: "select", Prio: 30},
				{Key: "Ctrl+B", Label: "back", Prio: 40},
				{Key: "a-z", Label: "jump", Prio: 50},
				{Key: "Ctrl+F", Label: "search", Prio: 60},
				{Key: "Ctrl+D", Label: "new dir", Prio: 70},
			}
		},
		Keys: func(ctx *dashygotui.Context[S], msg tea.KeyPressMsg) (tea.Cmd, bool) {
			nav := navOf(ctx)
			bodyRows := max(ctx.Size().Rows-4, 1)
			act, handled := nav.HandleKey(msg.String(), bodyRows)
			switch act {
			case browse.ActionSelect:
				path := nav.Dir()
				ctx.Back()
				if a.OnSelect != nil {
					return a.OnSelect(ctx, path), true
				}
				return nil, true
			case browse.ActionCreate:
				dir := nav.Dir()
				display := nav.DisplayPath(dir)
				ctx.OpenModal(&modal.Edit[S]{
					Title: "New Directory",
					Help:  "Name for the folder under " + display + ".",
					Validate: func(value string) error {
						return browse.ValidateName(dir, value)
					},
					Pending: pending,
					Save: func(_ context.Context, value string) (string, error) {
						name := strings.TrimSpace(value)
						if err := browse.ValidateName(dir, name); err != nil {
							return "", err
						}
						if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
							return "", err
						}
						return "Created " + name + ".", nil
					},
					OnSaved: func(c *dashygotui.Context[S], message string) tea.Cmd {
						name := strings.TrimSuffix(strings.TrimPrefix(message, "Created "), ".")
						n := navOf(c)
						n.Reload()
						if name != "" {
							n.Focus(name)
						}
						return nil
					},
				})
				return nil, true
			}
			return nil, handled
		},
	}
	option := dashygotui.Option[S]{
		Label: a.Label,
		Badge: a.Badge,
		Opens: viewID,
	}
	return []dashygotui.Addon[S]{view, option}
}
