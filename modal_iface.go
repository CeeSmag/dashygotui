package dashygotui

import tea "charm.land/bubbletea/v2"

// Modal is a pop-up drawn centred over the current page and given every key
// press while it is open. State lives in the modal value itself.
//
// Open one with [Context.OpenModal] and close it with [Context.CloseModal].
// The modal package provides ready-made edit, confirm, typed-phrase, notice and
// choice modals.
type Modal[S any] interface {
	// HandleKey handles one key press and returns an optional command.
	HandleKey(ctx *Context[S], msg tea.KeyPressMsg) tea.Cmd
	// Render returns the rows of the pop-up box for a terminal of the given
	// size. The dashboard centres and clips it; it need not fill the screen.
	Render(ctx *Context[S], size Size) []string
}

// ScrollableModal is implemented by modals whose body scrolls with the arrow,
// page, home and end keys.
type ScrollableModal[S any] interface {
	Modal[S]
	// ScrollMax is the largest scroll offset for a terminal of the given size.
	ScrollMax(ctx *Context[S], size Size) int
}

// ResultModal is implemented by modals that want to see the result of a
// background task while they are open, such as an editor that shows a
// validation error.
type ResultModal[S any] interface {
	Modal[S]
	// OnResult may consume the result. When handled is true no other hook sees
	// it.
	OnResult(ctx *Context[S], res TaskResult) (handled bool, cmd tea.Cmd)
}
