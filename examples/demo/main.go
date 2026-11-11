// Command demo shows a dashygotui dashboard: three pages, options, widgets, a
// full-screen view and modals, with settings kept between runs.
//
// The addons package is imported only for its side effects: every file in it
// registers itself in init().
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	_ "github.com/CeeSmag/dashygotui/examples/demo/dashboard/addons" // registers every addon
	"github.com/CeeSmag/dashygotui/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "demo:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Settings live in ~/.config/dashygotui-demo/dashboard.json. If that is not
	// possible the demo still runs, remembering nothing.
	var st dashygotui.Store = store.NewMemory()
	var settings *store.JSONFile
	if f, err := store.Open("dashygotui-demo"); err == nil {
		settings, st = f, f
	} else {
		fmt.Fprintln(os.Stderr, "demo: settings will not be saved:", err)
	}

	// A full-screen UI cannot share the terminal with the standard logger: send
	// its output to the buffer the Logs view shows.
	restoreLog := dashboard.Logs.Divert(log.Default())
	defer restoreLog()
	log.Print("[INFO] demo started")

	info, err := dashboard.Run(ctx, st)
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
