package modal

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/text"
	"github.com/CeeSmag/dashygotui/theme"
)

// Input limits: a password or a path can be long, a confirmation word cannot.
const (
	defaultEditLimit = 200
	typedLimit       = 40
)

// newInput builds a focused one-line input styled from the theme.
func newInput(th *theme.Theme, value string, limit int, secret bool) textinput.Model {
	in := textinput.New()
	in.Prompt = ""
	in.CharLimit = limit
	if secret {
		in.EchoMode = textinput.EchoPassword
		in.EchoCharacter = []rune(th.G.EchoChar)[0]
	}
	in.SetStyles(inputStyles(th))
	in.SetValue(value)
	in.Focus()
	return in
}

func inputStyles(th *theme.Theme) textinput.Styles {
	styles := textinput.DefaultDarkStyles()
	styles.Cursor.Blink = false
	styles.Cursor.Color = lipgloss.Color(th.Palette.Cyan)
	styles.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.Color(th.Palette.White))
	styles.Focused.Placeholder = th.Muted
	return styles
}

// Edit is a one-line text editor, optionally masked for passwords.
//
// On Enter the trimmed text is checked with Validate, then handed to Submit
// (which runs on the update loop and closes the modal first) or to Save (which
// runs in the background while the modal stays open, showing any error inline
// and closing on success). Set at most one of Submit and Save; with neither,
// Enter just closes the modal.
type Edit[S any] struct {
	// Title is shown in the pop-up's top border.
	Title string
	// Help is explanatory text shown above the input.
	Help string
	// Current is the present value, shown for reference unless Secret.
	Current string
	// Initial prefills the input.
	Initial string
	// Secret masks the input and hides Current.
	Secret bool
	// Limit caps the input length. Default 200.
	Limit int
	// AllowEmpty lets an empty value through to Submit or Save. By default an
	// empty value closes the modal with "Nothing was changed."
	AllowEmpty bool
	// Pending is the footer message while Save runs.
	Pending string
	// Validate rejects a value with an error shown inline. Optional.
	Validate func(value string) error
	// Submit receives the accepted value. The modal is already closed.
	Submit func(ctx *dashygotui.Context[S], value string) tea.Cmd
	// Save stores the accepted value in the background and returns a message
	// for the footer, or an error to show in the editor.
	Save func(ctx context.Context, value string) (message string, err error)

	input   textinput.Model
	ready   bool
	saving  bool
	errText string
}

// HandleKey implements dashygotui.Modal.
func (e *Edit[S]) HandleKey(ctx *dashygotui.Context[S], msg tea.KeyPressMsg) tea.Cmd {
	e.ensure(ctx)
	switch strings.ToLower(msg.String()) {
	case "esc":
		ctx.CloseModal()
		return nil
	case "enter":
		return e.accept(ctx)
	}
	if e.saving {
		return nil
	}
	e.errText = ""
	in, cmd := e.input.Update(msg)
	e.input = in
	return cmd
}

func (e *Edit[S]) accept(ctx *dashygotui.Context[S]) tea.Cmd {
	if e.saving {
		return nil
	}
	value := strings.TrimSpace(e.input.Value())
	if value == "" && !e.AllowEmpty {
		ctx.CloseModal()
		ctx.Flash("Nothing was changed.")
		return nil
	}
	if e.Validate != nil {
		if err := e.Validate(value); err != nil {
			e.errText = text.OneLine(err.Error())
			return nil
		}
	}
	switch {
	case e.Save != nil:
		e.saving = true
		save := e.Save
		return ctx.RunTask(e.tag(), e.Pending, func(c context.Context) (string, error) { return save(c, value) })
	case e.Submit != nil:
		ctx.CloseModal()
		return e.Submit(ctx, value)
	}
	ctx.CloseModal()
	return nil
}

// OnResult implements dashygotui.ResultModal: it takes the result of Save.
func (e *Edit[S]) OnResult(ctx *dashygotui.Context[S], res dashygotui.TaskResult) (bool, tea.Cmd) {
	if res.Tag != e.tag() || !e.saving {
		return false, nil
	}
	e.saving = false
	if res.Err != nil {
		e.errText = text.OneLine(res.Err.Error())
		return true, nil
	}
	ctx.CloseModal()
	ctx.FlashResult(res.Text, nil)
	return true, ctx.Refresh()
}

// tag identifies this editor's save among other tasks.
func (e *Edit[S]) tag() string { return fmt.Sprintf("modal.edit.%p", e) }

func (e *Edit[S]) ensure(ctx *dashygotui.Context[S]) {
	if e.ready {
		return
	}
	e.ready = true
	limit := e.Limit
	if limit <= 0 {
		limit = defaultEditLimit
	}
	e.input = newInput(ctx.Theme(), e.Initial, limit, e.Secret)
}

// Render implements dashygotui.Modal.
func (e *Edit[S]) Render(ctx *dashygotui.Context[S], size dashygotui.Size) []string {
	e.ensure(ctx)
	th := ctx.Theme()
	w := width(size.Cols, 66)
	inner := w - 4
	var body []string
	if e.Help != "" {
		body = append(body, text.Wrap(e.Help, inner)...)
	}
	if e.Current != "" && !e.Secret {
		body = append(body, "", th.Muted.Render("Current: ")+th.Trunc(e.Current, max(inner-9, 1)))
	}
	e.input.SetWidth(max(inner-2, 1))
	body = append(body, "", th.Key.Render(th.G.Prompt+" ")+e.input.View())
	switch {
	case e.errText != "":
		body = append(body, "", th.Red.Render(th.Trunc(th.G.Cross+" "+e.errText, inner)))
	case e.saving:
		body = append(body, "", th.Muted.Render("Saving..."))
	}
	body = append(body, "", keys(th, "Enter", "Save", "Esc", "Cancel"))
	return popup(th, e.Title, "", w, size.Rows, body)
}

// Typed asks the user to type a word out before an irreversible action, so it
// cannot be confirmed by a stray Enter.
type Typed[S any] struct {
	// Title is shown in the pop-up's top border.
	Title string
	// Lines describe what is about to happen.
	Lines []string
	// Word is what must be typed, exactly.
	Word string
	// Prompt asks for it, e.g. "Type RESET to continue".
	Prompt string
	// OnConfirm runs, with the modal already closed, when Word was typed.
	OnConfirm func(ctx *dashygotui.Context[S]) tea.Cmd
	// OnEmpty runs, with the modal already closed, when Enter is pressed on an
	// empty input. Optional; by default an empty input is an error like any
	// other wrong word.
	OnEmpty func(ctx *dashygotui.Context[S]) tea.Cmd

	input   textinput.Model
	ready   bool
	errText string
}

// HandleKey implements dashygotui.Modal.
func (t *Typed[S]) HandleKey(ctx *dashygotui.Context[S], msg tea.KeyPressMsg) tea.Cmd {
	t.ensure(ctx)
	switch strings.ToLower(msg.String()) {
	case "esc":
		ctx.CloseModal()
		return nil
	case "enter":
		value := strings.TrimSpace(t.input.Value())
		switch {
		case value == "" && t.OnEmpty != nil:
			ctx.CloseModal()
			return t.OnEmpty(ctx)
		case value == t.Word && t.OnConfirm != nil:
			ctx.CloseModal()
			return t.OnConfirm(ctx)
		default:
			t.errText = "Type the word " + t.Word + " exactly, or press Esc to cancel."
			t.input.SetValue("")
			return nil
		}
	}
	t.errText = ""
	in, cmd := t.input.Update(msg)
	t.input = in
	return cmd
}

func (t *Typed[S]) ensure(ctx *dashygotui.Context[S]) {
	if !t.ready {
		t.ready = true
		t.input = newInput(ctx.Theme(), "", typedLimit, false)
	}
}

// Render implements dashygotui.Modal.
func (t *Typed[S]) Render(ctx *dashygotui.Context[S], size dashygotui.Size) []string {
	t.ensure(ctx)
	th := ctx.Theme()
	w := width(size.Cols, 72)
	inner := w - 4
	body := wrapAll(t.Lines, inner)
	t.input.SetWidth(max(inner-2, 1))
	body = append(body, "", th.Amber.Render(th.Trunc(t.Prompt, inner)))
	body = append(body, th.Key.Render(th.G.Prompt+" ")+t.input.View())
	if t.errText != "" {
		body = append(body, "", th.Red.Render(th.Trunc(th.G.Cross+" "+t.errText, inner)))
	}
	body = append(body, "", keys(th, "Esc", "Cancel"))
	return popup(th, t.Title, "", w, size.Rows, body)
}
