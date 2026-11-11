// Command showcase is a dashygotui application that uses every feature of the
// library: six pages with different layouts, options with hotkeys, every kind
// of pop-up (standard and custom), full-screen views, background tasks and
// hooks, global shortcuts, six waveform styles, a scrollable log view, a TTY
// symbol set and themes that reach the terminal background.
//
// The addons package is imported only for its side effects: every file in it
// registers itself in init().
//
// Usage:
//
//	showcase [-glyphs auto|unicode|tty] [-background terminal|paint|off|transparent]
//	         [-palette id] [-altscroll] [-ephemeral] [-view id[,id...]]
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	_ "github.com/CeeSmag/dashygotui/examples/showcase/dashboard/addons" // registers every addon
	"github.com/CeeSmag/dashygotui/store"
)

func main() {
	err := run(os.Args[1:])
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp): // -h is not a failure
	default:
		fmt.Fprintln(os.Stderr, "showcase:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("showcase", flag.ContinueOnError)
	glyphs := fs.String("glyphs", "auto", "symbol set: auto, unicode or tty")
	background := fs.String("background", "terminal", "palette background: terminal, paint, or off (alias: transparent)")
	palette := fs.String("palette", "", "start with this palette id (default, catppuccin, dracula, nord, console, solarized)")
	altScroll := fs.Bool("altscroll", false, "turn the mouse wheel into arrow keys")
	startView := fs.String("view", "", "open these views at start, bottom first, e.g. diagnostics,stdaddons.logs")
	ephemeral := fs.Bool("ephemeral", false, "do not save settings between runs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	opts := dashboard.Options{Palette: *palette, AltScroll: *altScroll}
	if *startView != "" {
		opts.StartView = strings.Split(*startView, ",")
	}
	switch *glyphs {
	case "auto":
	case "unicode":
		opts.Glyphs = dashygotui.GlyphsUnicode
	case "tty":
		opts.Glyphs = dashygotui.GlyphsTTY
	default:
		return fmt.Errorf("unknown -glyphs %q (want auto, unicode or tty)", *glyphs)
	}
	switch *background {
	case "terminal":
	case "paint":
		opts.Background = dashygotui.BackgroundPaint
	case "off", "transparent":
		opts.Background = dashygotui.BackgroundTransparent
	default:
		return fmt.Errorf("unknown -background %q (want terminal, paint, off or transparent)", *background)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Settings live in ~/.config/dashygotui-showcase/dashboard.json. If that is
	// not possible, or -ephemeral is given, the showcase remembers nothing.
	opts.Store = store.NewMemory()
	var settings *store.JSONFile
	if !*ephemeral {
		if f, err := store.Open("dashygotui-showcase"); err == nil {
			settings, opts.Store = f, f
		} else {
			fmt.Fprintln(os.Stderr, "showcase: settings will not be saved:", err)
		}
	}

	// A full-screen UI cannot share the terminal with the standard logger: send
	// its output to the buffer the Logs view shows.
	restoreLog := dashboard.Logs.Divert(log.Default())
	defer restoreLog()
	log.Print("[INFO] showcase started")
	dashboard.Slog.Info("structured logging works too", "pid", os.Getpid())

	info, err := dashboard.Run(ctx, opts)
	if err != nil {
		return err
	}
	if info.Message != "" {
		fmt.Println(info.Message)
	}
	// The dashboard asked for post-exit work: here, forgetting every setting.
	if info.Reason == "reset" && settings != nil {
		if err := os.Remove(settings.Path()); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("removing settings: %w", err)
		}
	}
	return nil
}
