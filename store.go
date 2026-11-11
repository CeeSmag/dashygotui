package dashygotui

// Store persists dashboard and addon settings between runs.
//
// Keys are namespaced strings. The library owns the "dashygotui." prefix;
// addons use their own Key, e.g. "storage.limit". Values are anything
// encoding/json can round-trip.
//
// The store package provides a JSON file implementation and an in-memory one.
type Store interface {
	// Get loads the value stored under key into dst, which must be a pointer.
	// It reports whether the key was present; a missing key is not an error and
	// leaves dst untouched.
	Get(key string, dst any) (found bool, err error)
	// Set stores v under key, replacing any previous value.
	Set(key string, v any) error
}

// Keys the library itself persists in the [Store].
const (
	storeKeySpacing  = "dashygotui.spacing"
	storeKeyPalette  = "dashygotui.palette"
	storeKeyInterval = "dashygotui.interval_ms"
)
