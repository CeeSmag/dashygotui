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
	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
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
// size and modified time in a left panel (~75%), with key hints on the right
// (~25%). Ctrl+S opens a one-line search bar above both panels. Enter opens,
// Ctrl+Enter selects the highlighted directory, Ctrl+B goes to the parent,
// Ctrl+H toggles dot-directories, letter and digit keys jump, and Ctrl+D
// creates a directory. Dot-directories are hidden by default.
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
	// view has already closed. path is the absolute path of the highlighted
	// directory. Optional.
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
			return dirBrowserBody(ctx.Theme(), navOf(ctx), title, size)
		},
		Hints: func(*dashygotui.Context[S]) []dashygotui.Hint { return nil },
		Keys: func(ctx *dashygotui.Context[S], msg tea.KeyPressMsg) (tea.Cmd, bool) {
			nav := navOf(ctx)
			bodyRows := max(ctx.Size().Rows-4, 1)
			act, handled := nav.HandleKey(msg.String(), bodyRows)
			switch act {
			case browse.ActionSelect:
				path, ok := nav.SelectedPath()
				if !ok {
					return nil, true
				}
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

func dirBrowserBody(th *theme.Theme, nav *browse.Navigator, title string, size dashygotui.Size) []string {
	const gap = 1
	if size.Cols < 20 || size.Rows < 4 {
		return panel.Panel(th, title, "", nav.DisplayPath(nav.Dir()), size.Cols, size.Rows, nil)
	}
	barH := 0
	var bar []string
	if nav.Searching() {
		barH = 1
		bar = []string{dirBrowserSearchBar(th, nav, size.Cols)}
	}
	panelsH := max(size.Rows-barH, 1)
	rightW := max(size.Cols/4, 18)
	leftW := size.Cols - rightW - gap
	if leftW < 12 {
		leftW = size.Cols
		rightW = 0
	}
	sub := nav.DisplayPath(nav.Dir())
	if err := nav.Err(); err != "" {
		sub += "  " + err
	}
	listInner := max(leftW-4, 1)
	listRows := max(panelsH-2, 1)
	left := panel.Panel(th, title, "", sub, leftW, panelsH, nav.Rows(th, listInner, listRows))
	var panels []string
	if rightW == 0 {
		panels = left
	} else {
		keysTitle := "Keys"
		hiddenBadge := ""
		if nav.ShowHidden() {
			hiddenBadge = th.Muted.Render("[.]")
		}
		right := panel.Panel(th, keysTitle, hiddenBadge, "", rightW, panelsH, dirBrowserKeyLines(th, nav, max(rightW-4, 1)))
		panels = panel.HJoin(gap, left, right)
	}
	if barH == 0 {
		return panels
	}
	return panel.Frame(panel.VStack(bar, panels), size.Cols, size.Rows)
}

func dirBrowserSearchBar(th *theme.Theme, nav *browse.Navigator, cols int) string {
	prompt := th.Key.Render("Search:") + " " + th.White.Render(nav.Filter()+th.G.EchoChar)
	hint := th.Muted.Render("Esc clear")
	gap := 2
	hintW := text.Width(hint)
	if cols > hintW+gap+10 {
		mid := cols - hintW - gap
		return th.Fit(prompt, mid) + strings.Repeat(" ", gap) + hint
	}
	return th.Fit(prompt, cols)
}

func dirBrowserKeyLines(th *theme.Theme, nav *browse.Navigator, cols int) []string {
	type row struct{ key, label string }
	var rows []row
	if nav.Searching() {
		rows = []row{
			{"type", "filter"},
			{"Esc", "clear"},
			{"Enter/right", "open"},
			{"Ctrl+Enter", "select"},
			{"left", "parent"},
			{"Ctrl+S", "done"},
		}
	} else {
		hidden := "show hidden"
		if nav.ShowHidden() {
			hidden = "hide hidden"
		}
		rows = []row{
			{th.G.UpDown, "move"},
			{"Enter/right", "open"},
			{"Ctrl+Enter", "select"},
			{"Ctrl+B/left", "parent"},
			{"a-z 0-9", "jump"},
			{"Ctrl+S", "search"},
			{"Ctrl+D", "new folder"},
			{"Ctrl+H", hidden},
			{"q", "close"},
		}
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		line := th.Key.Render(r.key) + "  " + th.Muted.Render(r.label)
		out = append(out, th.Fit(line, cols))
	}
	return out
}
