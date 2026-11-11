// Package store provides implementations of the dashygotui.Store interface: a
// JSON file under the user's config directory, and an in-memory store for
// tests and throwaway dashboards.
//
// Both keep values as JSON, so anything encoding/json can round-trip can be
// stored, and both are safe for concurrent use.
package store

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Memory keeps values in memory only. The zero value is not usable; create one
// with [NewMemory].
type Memory struct {
	mu   sync.Mutex
	data map[string]json.RawMessage
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory { return &Memory{data: map[string]json.RawMessage{}} }

// Get loads the value stored under key into dst, a pointer. It reports whether
// the key was present; a missing key leaves dst untouched.
func (s *Memory) Get(key string, dst any) (bool, error) {
	s.mu.Lock()
	raw, ok := s.data[key]
	s.mu.Unlock()
	return decode(key, raw, ok, dst)
}

// Set stores v under key, replacing any previous value.
func (s *Memory) Set(key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encoding %q: %w", key, err)
	}
	s.mu.Lock()
	s.data[key] = raw
	s.mu.Unlock()
	return nil
}

// decode unmarshals raw into dst when the key was present.
func decode(key string, raw json.RawMessage, ok bool, dst any) (bool, error) {
	if !ok {
		return false, nil
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return true, fmt.Errorf("decoding %q: %w", key, err)
	}
	return true, nil
}
