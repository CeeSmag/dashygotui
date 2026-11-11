package dashygotui

import (
	"errors"
	"fmt"
)

// ErrInvalidConfig is wrapped by every error [New] returns for a configuration
// it cannot build a dashboard from.
var ErrInvalidConfig = errors.New("invalid dashboard configuration")

// RegistrationError describes a mistake made while registering an addon. The
// registry records it instead of panicking, because registration happens in
// init() where an error cannot be returned; [New] and [Registry.Validate]
// report it before the UI starts.
type RegistrationError struct {
	// Site is the "file.go:line" of the offending registration call.
	Site string
	// Msg says what is wrong.
	Msg string
}

// Error implements the error interface.
func (e *RegistrationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Site, e.Msg)
}
