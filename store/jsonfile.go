package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// FileName is the name of the settings file inside an application's config
// directory.
const FileName = "dashboard.json"

// JSONFile keeps values in one JSON object in a file. The file is read on
// first use and rewritten atomically (write to a temporary file, then rename)
// on every Set, so a crash never leaves a half-written file behind.
//
// A file that exists but cannot be parsed is reported as an error by Get. The
// first Set after that moves the unreadable file aside to path+".corrupt" and
// starts afresh, so settings can be saved again without silently destroying
// what was there.
type JSONFile struct {
	path string

	mu      sync.Mutex
	loaded  bool
	loadErr error
	data    map[string]json.RawMessage
}

// NewJSONFile returns a store backed by the file at path. Nothing is read or
// created until the store is used.
func NewJSONFile(path string) *JSONFile { return &JSONFile{path: path} }

// Open returns a store backed by app's settings file under the user's config
// directory (for example ~/.config/app/dashboard.json on Linux).
func Open(app string) (*JSONFile, error) {
	path, err := DefaultPath(app)
	if err != nil {
		return nil, err
	}
	return NewJSONFile(path), nil
}

// DefaultPath is where Open keeps app's settings: os.UserConfigDir()/app/dashboard.json.
func DefaultPath(app string) (string, error) {
	if app == "" {
		return "", errors.New("store: app name is empty")
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating the config directory: %w", err)
	}
	return filepath.Join(dir, app, FileName), nil
}

// Path returns the file the store reads and writes.
func (s *JSONFile) Path() string { return s.path }

// load reads the file once. The caller holds s.mu.
func (s *JSONFile) load() {
	if s.loaded {
		return
	}
	s.loaded = true
	s.data = map[string]json.RawMessage{}
	raw, err := os.ReadFile(s.path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return
	case err != nil:
		s.loadErr = fmt.Errorf("reading %s: %w", s.path, err)
		return
	}
	if len(raw) == 0 {
		return
	}
	if err := json.Unmarshal(raw, &s.data); err != nil {
		s.data = map[string]json.RawMessage{}
		s.loadErr = fmt.Errorf("parsing %s: %w", s.path, err)
	}
}

// Get loads the value stored under key into dst, a pointer. It reports whether
// the key was present; a missing key, and a missing file, leave dst untouched.
func (s *JSONFile) Get(key string, dst any) (bool, error) {
	s.mu.Lock()
	s.load()
	if s.loadErr != nil {
		err := s.loadErr
		s.mu.Unlock()
		return false, err
	}
	raw, ok := s.data[key]
	s.mu.Unlock()
	return decode(key, raw, ok, dst)
}

// Set stores v under key and writes the file.
func (s *JSONFile) Set(key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encoding %q: %w", key, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.load()
	if s.loadErr != nil {
		// Keep the unreadable file for inspection, then start afresh.
		_ = os.Rename(s.path, s.path+".corrupt") // best effort: failing to keep a copy must not block saving
		s.loadErr = nil
	}
	s.data[key] = raw
	if err := s.flush(); err != nil {
		delete(s.data, key)
		return err
	}
	return nil
}

// flush writes s.data atomically. The caller holds s.mu.
func (s *JSONFile) flush() error {
	out, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding settings: %w", err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(s.path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("creating a temporary settings file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) } // best effort: a stray temp file is harmless
	if _, err := tmp.Write(append(out, '\n')); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("writing settings: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("writing settings: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		cleanup()
		return fmt.Errorf("securing settings: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		cleanup()
		return fmt.Errorf("saving settings: %w", err)
	}
	return nil
}
