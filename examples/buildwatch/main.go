// Command buildwatch is a second, unrelated dashygotui application: a monitor
// for continuous-integration builds. It exists to prove the library is not
// shaped around any one application, and to show the short forms an addon is
// written in. Compare its addons with the showcase's: everything an addon needs
// is a few lines.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/CeeSmag/dashygotui/examples/buildwatch/dashboard"
	_ "github.com/CeeSmag/dashygotui/examples/buildwatch/dashboard/addons" // registers every addon
	"github.com/CeeSmag/dashygotui/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "buildwatch:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	st, err := store.Open("dashygotui-buildwatch")
	if err != nil {
		fmt.Fprintln(os.Stderr, "buildwatch: settings will not be saved:", err)
		st = nil
	}
	info, err := dashboard.Run(ctx, st)
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	if info.Message != "" {
		fmt.Println(info.Message)
	}
	return nil
}
