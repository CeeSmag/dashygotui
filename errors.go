package dashygotui

import (
	"errors"
	"fmt"
)

// ErrInvalidConfig is wrapped by every error [New] returns for a configuration
// it cannot build a dashboard from.
var ErrInvalidConfig = errors.New("invalid dashboard configuration")

// AddonError describes a mistake in one place of the map, such as a row with
// nothing to do. [New] reports every one it finds, joined and wrapped in
// [ErrInvalidConfig], before the UI starts.
type AddonError struct {
	// Site is the place in the map, such as "Control > Quick Actions > row 3",
	// or "Map.Shortcuts[1]".
	Site string
	// Msg says what is wrong.
	Msg string
}

// Error implements the error interface.
func (e *AddonError) Error() string {
	return fmt.Sprintf("%s: %s", e.Site, e.Msg)
}
