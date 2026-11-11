package modal

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
)

// Confirm is a yes/no confirmation. Enter or y accepts; Esc, n or q cancels.
type Confirm[S any] struct {
	// Title is shown in the pop-up's top border.
	Title string
	// Lines describe what will happen; they are wrapped to fit.
	Lines []string
	// Accept labels the accepting key, e.g. "Delete". Default "Confirm". The
	// special label "Back" makes the modal an acknowledgement: Enter and Esc
	// both just close it.
	Accept string
	// OnAccept runs, with the modal already closed, when the user accepts.
	OnAccept func(ctx *dashygotui.Context[S]) tea.Cmd
	// OnCancel runs, with the modal already closed, when the user cancels.
	// Optional.
	OnCancel func(ctx *dashygotui.Context[S]) tea.Cmd
}

// HandleKey implements dashygotui.Modal.
func (c *Confirm[S]) HandleKey(ctx *dashygotui.Context[S], msg tea.KeyPressMsg) tea.Cmd {
	switch strings.ToLower(msg.String()) {
	case "enter", "y":
		action := c.OnAccept
		ctx.CloseModal()
		if action != nil {
			return action(ctx)
		}
	case "esc", "n", "q":
		cancel := c.OnCancel
		ctx.CloseModal()
		if cancel != nil {
			return cancel(ctx)
		}
	}
	return nil
}

// Render implements dashygotui.Modal.
func (c *Confirm[S]) Render(ctx *dashygotui.Context[S], size dashygotui.Size) []string {
	th := ctx.Theme()
	w := width(size.Cols, 66)
	body := wrapAll(c.Lines, w-4)
	accept := c.Accept
	if accept == "" {
		accept = "Confirm"
	}
	cancel := "Cancel"
	if c.OnCancel != nil {
		cancel = "Back"
	}
	if accept == "Back" {
		body = append(body, "", th.Muted.Render("Enter")+"/"+th.Muted.Render("Esc")+" Back")
	} else {
		body = append(body, "", keys(th, "Enter", accept, "Esc", cancel))
	}
	return popup(th, c.Title, "", w, size.Rows, body)
}
