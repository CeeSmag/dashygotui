package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/CeeSmag/dashygotui/store"
)

type pair struct {
	A int    `json:"a"`
	B string `json:"b"`
}

// impl lets the same tests run against both implementations.
type impl interface {
	Get(key string, dst any) (bool, error)
	Set(key string, v any) error
}

func implementations(t *testing.T) map[string]func() impl {
	t.Helper()
	return map[string]func() impl{
		"memory": func() impl { return store.NewMemory() },
		"file":   func() impl { return store.NewJSONFile(filepath.Join(t.TempDir(), "sub", "dashboard.json")) },
	}
}

func TestRoundTrip(t *testing.T) {
	for name, mk := range implementations(t) {
		t.Run(name, func(t *testing.T) {
			s := mk()
			if err := s.Set("n", 42); err != nil {
				t.Fatal(err)
			}
			if err := s.Set("p", pair{A: 7, B: "x"}); err != nil {
				t.Fatal(err)
			}
			var n int
			if ok, err := s.Get("n", &n); err != nil || !ok || n != 42 {
				t.Errorf("n = %d ok=%v err=%v", n, ok, err)
			}
			var p pair
			if ok, err := s.Get("p", &p); err != nil || !ok || p != (pair{7, "x"}) {
				t.Errorf("p = %+v ok=%v err=%v", p, ok, err)
			}
		})
	}
}

func TestMissingKeyLeavesDstUntouched(t *testing.T) {
	for name, mk := range implementations(t) {
		t.Run(name, func(t *testing.T) {
			n := 5
			ok, err := mk().Get("absent", &n)
			if err != nil || ok || n != 5 {
				t.Errorf("ok=%v err=%v n=%d", ok, err, n)
			}
		})
	}
}

func TestSetReplaces(t *testing.T) {
	for name, mk := range implementations(t) {
		t.Run(name, func(t *testing.T) {
			s := mk()
			_ = s.Set("k", 1)
			_ = s.Set("k", 2)
			var n int
			_, _ = s.Get("k", &n)
			if n != 2 {
				t.Errorf("n = %d, want the latest value", n)
			}
		})
	}
}

func TestTypeMismatchIsAnError(t *testing.T) {
	for name, mk := range implementations(t) {
		t.Run(name, func(t *testing.T) {
			s := mk()
			_ = s.Set("k", "text")
			var n int
			ok, err := s.Get("k", &n)
			if err == nil || !ok || !strings.Contains(err.Error(), `decoding "k"`) {
				t.Errorf("ok=%v err=%v, want a decoding error that still reports the key as present", ok, err)
			}
		})
	}
}

func TestUnencodableValueIsAnError(t *testing.T) {
	for name, mk := range implementations(t) {
		t.Run(name, func(t *testing.T) {
			s := mk()
			if err := s.Set("k", make(chan int)); err == nil || !strings.Contains(err.Error(), `encoding "k"`) {
				t.Errorf("err = %v", err)
			}
			var v any
			if ok, _ := s.Get("k", &v); ok {
				t.Error("a failed Set must not store anything")
			}
		})
	}
}

func TestConcurrentUse(t *testing.T) {
	for name, mk := range implementations(t) {
		t.Run(name, func(t *testing.T) {
			s := mk()
			var wg sync.WaitGroup
			for i := range 8 {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := range 20 {
						_ = s.Set("k", i*100+j)
						var n int
						_, _ = s.Get("k", &n)
					}
				}()
			}
			wg.Wait()
		})
	}
}

func TestJSONFilePersistsAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deep", "er", store.FileName)
	first := store.NewJSONFile(path)
	if first.Path() != path {
		t.Errorf("Path = %q", first.Path())
	}
	if err := first.Set("a", pair{1, "one"}); err != nil {
		t.Fatal(err)
	}
	if err := first.Set("b", 2); err != nil {
		t.Fatal(err)
	}

	second := store.NewJSONFile(path)
	var p pair
	if ok, err := second.Get("a", &p); err != nil || !ok || p != (pair{1, "one"}) {
		t.Errorf("a = %+v ok=%v err=%v", p, ok, err)
	}
	var n int
	if ok, err := second.Get("b", &n); err != nil || !ok || n != 2 {
		t.Errorf("b = %d ok=%v err=%v", n, ok, err)
	}
}

func TestJSONFileIsPrivateAndLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, store.FileName)
	s := store.NewJSONFile(path)
	for i := range 5 {
		if err := s.Set("k", i); err != nil {
			t.Fatal(err)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("directory holds %v, want only the settings file", names)
	}
}

func TestJSONFileEmptyOrMissingFileIsFine(t *testing.T) {
	dir := t.TempDir()
	var n int
	if ok, err := store.NewJSONFile(filepath.Join(dir, "missing.json")).Get("k", &n); err != nil || ok {
		t.Errorf("missing: ok=%v err=%v", ok, err)
	}
	empty := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.NewJSONFile(empty).Get("k", &n); err != nil || ok {
		t.Errorf("empty: ok=%v err=%v", ok, err)
	}
}

func TestJSONFileCorruptFileIsReportedThenReplacedSafely(t *testing.T) {
	path := filepath.Join(t.TempDir(), store.FileName)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := store.NewJSONFile(path)
	var n int
	if _, err := s.Get("k", &n); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Fatalf("Get should report the corrupt file, got %v", err)
	}
	if err := s.Set("k", 9); err != nil {
		t.Fatalf("Set should recover: %v", err)
	}
	if ok, err := s.Get("k", &n); err != nil || !ok || n != 9 {
		t.Errorf("after recovery: n=%d ok=%v err=%v", n, ok, err)
	}
	kept, err := os.ReadFile(path + ".corrupt")
	if err != nil || string(kept) != "{not json" {
		t.Errorf("the unreadable file should be kept aside: %q %v", kept, err)
	}
}

func TestJSONFileUnwritableLocationFailsSetAndKeepsNothing(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A path whose parent is a regular file cannot be created.
	s := store.NewJSONFile(filepath.Join(blocker, "sub", store.FileName))
	if err := s.Set("k", 1); err == nil {
		t.Fatal("Set into an impossible path must fail")
	}
	var n int
	if ok, _ := s.Get("k", &n); ok {
		t.Error("a failed Set must not leave the value in memory")
	}
}

func TestDefaultPathAndOpen(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	path, err := store.DefaultPath("myapp")
	if err != nil {
		t.Skipf("no user config directory on this platform: %v", err)
	}
	if filepath.Base(path) != store.FileName || filepath.Base(filepath.Dir(path)) != "myapp" {
		t.Errorf("DefaultPath = %q", path)
	}
	s, err := store.Open("myapp")
	if err != nil || s.Path() != path {
		t.Errorf("Open = %v %v", s, err)
	}
	if _, err := store.DefaultPath(""); err == nil {
		t.Error("an empty app name must be refused")
	}
	if _, err := store.Open(""); err == nil {
		t.Error("Open with an empty app name must be refused")
	}
}

func TestErrorsAreWrappable(t *testing.T) {
	s := store.NewMemory()
	_ = s.Set("k", "text")
	var n int
	_, err := s.Get("k", &n)
	var ute interface{ Error() string }
	if !errors.As(err, &ute) {
		t.Errorf("err %v is not an error value", err)
	}
	if errors.Unwrap(err) == nil {
		t.Error("the decoding error should wrap the underlying json error")
	}
}
