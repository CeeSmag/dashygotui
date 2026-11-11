package modal

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/chart"
	"github.com/CeeSmag/dashygotui/text"
)

// Item is one entry of a [Choice].
type Item struct {
	// Label is the entry's name.
	Label string
	// Badge is optional state shown after the label, already styled.
	Badge string
	// Detail is a muted line of explanation under the label.
	Detail string
}

// Choice is a numbered list the user picks one entry from. Up and down (or
// j and k) move, a digit jumps to that entry, Enter selects, Esc cancels.
type Choice[S any] struct {
	// Title is shown in the pop-up's top border.
	Title string
	// Help is explanatory text shown above the list.
	Help string
	// Items are the entries. At least one is needed.
	Items []Item
	// Cursor is the highlighted entry; set it to the current value.
	Cursor int
	// OnMove is called whenever the highlight moves, for live previews.
	// Optional.
	OnMove func(ctx *dashygotui.Context[S], index int)
	// OnSelect is called, with the modal already closed, when an entry is
	// chosen.
	OnSelect func(ctx *dashygotui.Context[S], index int) tea.Cmd
	// OnCancel is called, with the modal already closed, on Esc, for example to
	// undo a live preview. Optional.
	OnCancel func(ctx *dashygotui.Context[S]) tea.Cmd
}

func (c *Choice[S]) move(ctx *dashygotui.Context[S], to int) {
	n := len(c.Items)
	if n == 0 {
		return
	}
	c.Cursor = (to%n + n) % n
	if c.OnMove != nil {
		c.OnMove(ctx, c.Cursor)
	}
}

// HandleKey implements dashygotui.Modal.
func (c *Choice[S]) HandleKey(ctx *dashygotui.Context[S], msg tea.KeyPressMsg) tea.Cmd {
	name := strings.ToLower(msg.String())
	switch name {
	case "up", "k":
		c.move(ctx, c.Cursor-1)
	case "down", "j":
		c.move(ctx, c.Cursor+1)
	case "home":
		c.move(ctx, 0)
	case "end":
		c.move(ctx, len(c.Items)-1)
	case "enter", "space":
		if c.Cursor < 0 || c.Cursor >= len(c.Items) {
			ctx.CloseModal()
			return nil
		}
		idx, sel := c.Cursor, c.OnSelect
		ctx.CloseModal()
		if sel != nil {
			return sel(ctx, idx)
		}
	case "esc", "q":
		cancel := c.OnCancel
		ctx.CloseModal()
		if cancel != nil {
			return cancel(ctx)
		}
	default:
		if len(name) == 1 && name[0] >= '1' && name[0] <= '9' {
			if i := int(name[0] - '1'); i < len(c.Items) {
				c.move(ctx, i)
			}
		}
	}
	return nil
}

// Render implements dashygotui.Modal.
func (c *Choice[S]) Render(ctx *dashygotui.Context[S], size dashygotui.Size) []string {
	th := ctx.Theme()
	w := width(size.Cols, 72)
	inner := w - 4

	var head []string
	if c.Help != "" {
		head = append(head, text.Wrap(c.Help, inner)...)
		head = append(head, "")
	}
	foot := []string{"", keys(th, "↑↓", "Move", "Enter", "Select", "Esc", "Cancel")}

	// Each entry takes two rows; show a window of them around the cursor when
	// the terminal cannot hold the whole list.
	room := size.Rows - 2 - len(head) - len(foot)
	visible := max(room/2, 1)
	start := 0
	if len(c.Items) > visible {
		start = min(max(c.Cursor-visible/2, 0), len(c.Items)-visible)
	}
	end := min(start+visible, len(c.Items))

	body := head
	for i := start; i < end; i++ {
		it := c.Items[i]
		body = append(body, chart.ChoiceRows(th, i+1, it.Label, it.Badge, it.Detail, inner, i == c.Cursor)...)
	}
	body = append(body, foot...)

	badge := ""
	if len(c.Items) > visible {
		badge = ctx.Count(len(c.Items), "choices")
	}
	return popup(th, c.Title, badge, w, size.Rows, body)
}
