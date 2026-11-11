package modal

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/panel"
)

// Notice is a read-only, scrollable pop-up for a report or a card. It closes on
// Enter, Esc, Space, b or q and scrolls with the arrow, page, home and end keys.
//
// Give it static Title, Badge and Lines, or a Content function when the text
// depends on the width or on live state.
type Notice[S any] struct {
	// Title and Badge are shown in the top border.
	Title string
	Badge string
	// Lines are the body text; they are wrapped to fit.
	Lines []string
	// Width is the desired width. Default 68, or 96 with Content.
	Width int
	// Content, when set, produces the title, badge and lines afresh on every
	// frame for an inner width, and replaces Title, Badge and Lines.
	Content func(ctx *dashygotui.Context[S], inner int) (title, badge string, lines []string)
}

// content resolves the notice to its title, badge, wrapped lines and width.
func (n *Notice[S]) content(ctx *dashygotui.Context[S], cols int) (title, badge string, lines []string, w int) {
	want := n.Width
	if want <= 0 {
		want = 68
		if n.Content != nil {
			want = 96
		}
	}
	w = width(cols, want)
	inner := max(w-4, 1)
	if n.Content != nil {
		title, badge, lines = n.Content(ctx, inner)
		return title, badge, lines, w
	}
	return n.Title, n.Badge, wrapAll(n.Lines, inner), w
}

// ScrollMax implements dashygotui.ScrollableModal.
func (n *Notice[S]) ScrollMax(ctx *dashygotui.Context[S], size dashygotui.Size) int {
	if size.Cols <= 4 || size.Rows <= 4 {
		return 0
	}
	_, _, lines, _ := n.content(ctx, size.Cols)
	h := min(len(lines)+2, max(size.Rows-2, 3))
	return max(len(lines)-max(h-2, 1), 0)
}

// HandleKey implements dashygotui.Modal.
func (n *Notice[S]) HandleKey(ctx *dashygotui.Context[S], msg tea.KeyPressMsg) tea.Cmd {
	switch strings.ToLower(msg.String()) {
	case "esc", "q", "b", "enter", "space":
		ctx.CloseModal()
	}
	return nil
}

// Render implements dashygotui.Modal.
func (n *Notice[S]) Render(ctx *dashygotui.Context[S], size dashygotui.Size) []string {
	title, badge, lines, w := n.content(ctx, size.Cols)
	h := min(len(lines)+2, max(size.Rows-2, 3))
	visible := max(h-2, 1)
	scroll := min(ctx.ModalScroll(), max(len(lines)-visible, 0))
	window := lines
	if len(window) > visible {
		window = window[scroll : scroll+visible]
	}
	th := ctx.Theme()
	enter := th.G.Enter
	sub := fmt.Sprintf("press [%s], [b] or [q] to return", enter)
	if len(lines) > visible {
		sub = fmt.Sprintf("%s scroll %s %d more %s [%s] return", th.G.UpDown, th.G.Sep, len(lines)-scroll-visible, th.G.Sep, enter)
	}
	return panel.Panel(th, title, badge, sub, w, h, window)
}
