package modal

import (
	"cmp"
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
	// Badge is shown in the pop-up's top border, already styled, such as the
	// current value. Optional.
	Badge func(ctx *dashygotui.Context[S]) string
	// Intro returns explanatory lines shown above the list, after Help, for the
	// given inner width. It is called on every frame, so the lines can show
	// live state. Optional.
	Intro func(ctx *dashygotui.Context[S], inner int) []string
	// Items are the entries. At least one is needed, from here or from Rows.
	Items []Item
	// Rows returns the entries instead of Items, called on every frame and key
	// press, so badges can show live state. Optional.
	Rows func(ctx *dashygotui.Context[S]) []Item
	// FooterSelect words the Enter hint; the default is "Select". Optional.
	FooterSelect string
	// FooterCancel words the Esc hint; the default is "Cancel". Optional.
	FooterCancel string
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

// items is the current list of entries.
func (c *Choice[S]) items(ctx *dashygotui.Context[S]) []Item {
	if c.Rows != nil {
		return c.Rows(ctx)
	}
	return c.Items
}

func (c *Choice[S]) move(ctx *dashygotui.Context[S], to int) {
	n := len(c.items(ctx))
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
	n := len(c.items(ctx))
	switch name {
	case "up", "k":
		c.move(ctx, c.Cursor-1)
	case "down", "j":
		c.move(ctx, c.Cursor+1)
	case "home":
		c.move(ctx, 0)
	case "end":
		c.move(ctx, n-1)
	case "enter", "space":
		if c.Cursor < 0 || c.Cursor >= n {
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
			if i := int(name[0] - '1'); i < n {
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

	items := c.items(ctx)
	var head []string
	if c.Help != "" {
		head = append(head, text.Wrap(c.Help, inner)...)
		head = append(head, "")
	}
	if c.Intro != nil {
		if lines := c.Intro(ctx, inner); len(lines) > 0 {
			head = append(head, lines...)
			head = append(head, "")
		}
	}
	foot := []string{"", keys(th, th.G.UpDown, "Move", "Enter", cmp.Or(c.FooterSelect, "Select"), "Esc", cmp.Or(c.FooterCancel, "Cancel"))}

	// Each entry takes two rows; show a window of them around the cursor when
	// the terminal cannot hold the whole list.
	room := size.Rows - 2 - len(head) - len(foot)
	visible := max(room/2, 1)
	start := 0
	if len(items) > visible {
		start = min(max(c.Cursor-visible/2, 0), len(items)-visible)
	}
	end := min(start+visible, len(items))

	body := head
	for i := start; i < end; i++ {
		it := items[i]
		body = append(body, chart.ChoiceRows(th, i+1, it.Label, it.Badge, it.Detail, inner, i == c.Cursor)...)
	}
	body = append(body, foot...)

	badge := ""
	switch {
	case c.Badge != nil:
		badge = c.Badge(ctx)
	case len(items) > visible:
		badge = ctx.Count(len(items), "choices")
	}
	return popup(th, c.Title, badge, w, size.Rows, body)
}
