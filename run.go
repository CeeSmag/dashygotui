package dashygotui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
)

// Model returns a fresh Bubble Tea model for the dashboard, for embedding in a
// larger program or driving from tests (see the dashtest package). Most
// applications call [Dashboard.Run] instead.
//
// ctx cancels the dashboard's background work (polls and tasks).
func (d *Dashboard[S]) Model(ctx context.Context) tea.Model { return newModel(ctx, d) }

// Run shows the dashboard full-screen and blocks until the user quits or ctx is
// cancelled. It returns why the dashboard ended; the application decides what
// to do about it, for example printing ExitInfo.Message or starting an
// uninstaller once the terminal is back to normal.
//
// A quit the user asked for, and a cancelled ctx, are not errors.
func (d *Dashboard[S]) Run(ctx context.Context) (ExitInfo, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if d.cfg.AltScroll {
		restore := enableAltScroll(os.Stdout)
		defer restore()
	}

	m := newModel(ctx, d)
	prog := tea.NewProgram(m, tea.WithContext(ctx))
	detach := d.Attach(prog)
	defer detach()
	final, err := prog.Run()
	cancel()

	last := m
	if f, ok := final.(*model[S]); ok && f != nil {
		last = f
	}
	if err != nil && !errors.Is(err, tea.ErrInterrupted) && !errors.Is(err, tea.ErrProgramKilled) && !errors.Is(err, context.Canceled) {
		return last.exit, fmt.Errorf("running dashboard: %w", err)
	}
	return last.exit, nil
}

// enableAltScroll asks the terminal to translate wheel notches into arrow keys
// while the alternate screen is up (DEC private mode 1007), after saving
// whatever the terminal had set, and returns a function that puts the setting
// back. Terminals that do not implement it ignore both sequences and only the
// keyboard scrolls. This writes terminal control sequences only, never
// application output.
func enableAltScroll(w io.Writer) func() {
	_, _ = io.WriteString(w, "\x1b[?1007s\x1b[?1007h") // best effort: a terminal that rejects it just keeps its default
	return func() { _, _ = io.WriteString(w, "\x1b[?1007r") }
}
