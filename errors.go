package dashygotui

import (
	"errors"
	"fmt"
)

// ErrInvalidConfig is wrapped by every error [New] returns for a configuration
// it cannot build a dashboard from.
var ErrInvalidConfig = errors.New("invalid dashboard configuration")

// AddonError describes a mistake in one addon, such as a number another addon
// already uses. [New] reports every one it finds, joined and wrapped in
// [ErrInvalidConfig], before the UI starts.
type AddonError struct {
	// Site is where the addon is declared, as [At] recorded it ("opt_trash.go:14"),
	// or its index in [Config.Addons] ("Config.Addons[3]").
	Site string
	// Msg says what is wrong.
	Msg string
}

// Error implements the error interface.
func (e *AddonError) Error() string {
	return fmt.Sprintf("%s: %s", e.Site, e.Msg)
}
