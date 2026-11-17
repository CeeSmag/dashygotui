// Package browse is a tea-free directory navigator for dashboards: list a
// folder, jump by letter, filter by name, walk parent directories, and gather
// per-directory size and file counts. [stdaddons.DirBrowser] wires it into a
// full-screen view.
package browse

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

// Entry is one directory in the current listing.
type Entry struct {
	Name    string
	Path    string
	ModTime time.Time
	Files   int64 // recursive regular-file count; -1 when unknown
	Bytes   int64 // recursive size of those files; -1 when unknown
}

// Action is what [Navigator.HandleKey] asks the host to do beyond updating the
// navigator itself.
type Action int

// Actions a host may need to handle after [Navigator.HandleKey].
const (
	ActionNone   Action = iota
	ActionSelect        // confirm the highlighted directory
	ActionCreate        // open the create-directory UI
)

// Navigator holds the place in a directory tree: cwd, listing, cursor, search
// and letter-jump state. It is not safe for concurrent use; the dashboard
// drives it from its update loop.
type Navigator struct {
	cwd        string
	entries    []Entry
	cursor     int
	filter     string
	search     bool
	jump       rune // last letter-jump key; 0 when none
	err        string
	home       string
	cache      map[string]dirStats
	dirsOnly   bool
	showHidden bool // when false, names starting with '.' are omitted
}

// New returns a navigator rooted at start (or the user home when start is
// empty), listing directories only.
func New(start string) *Navigator {
	home, _ := os.UserHomeDir()
	n := &Navigator{
		home:     home,
		cache:    map[string]dirStats{},
		dirsOnly: true,
	}
	if start == "" {
		start = home
	}
	if start == "" {
		start = "/"
	}
	n.Reset(start)
	return n
}

// Reset opens dir as the current directory and clears search and jump state.
func (n *Navigator) Reset(dir string) {
	n.search = false
	n.filter = ""
	n.jump = 0
	n.open(dir, "")
}

// Dir is the absolute current directory.
func (n *Navigator) Dir() string { return n.cwd }

// Err is a short listing error to show, or empty.
func (n *Navigator) Err() string { return n.err }

// Searching reports whether the filter input is active.
func (n *Navigator) Searching() bool { return n.search }

// Filter is the current search text.
func (n *Navigator) Filter() string { return n.filter }

// ShowHidden reports whether directories whose names start with '.' are listed.
func (n *Navigator) ShowHidden() bool { return n.showHidden }

// ToggleHidden shows or hides dot-directories and keeps the cursor valid.
func (n *Navigator) ToggleHidden() {
	n.showHidden = !n.showHidden
	n.clampCursor()
}

// Cursor is the index into [Navigator.Visible].
func (n *Navigator) Cursor() int { return n.cursor }

// Home is the user home directory used for "~" display, or empty.
func (n *Navigator) Home() string { return n.home }

// Visible returns the listing after hidden-name and search filters.
func (n *Navigator) Visible() []Entry {
	q := strings.ToLower(n.filter)
	var out []Entry
	for _, e := range n.entries {
		if !n.showHidden && strings.HasPrefix(e.Name, ".") {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(e.Name), q) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// Selected is the highlighted entry, or false when the list is empty.
func (n *Navigator) Selected() (Entry, bool) {
	vis := n.Visible()
	if len(vis) == 0 || n.cursor < 0 || n.cursor >= len(vis) {
		return Entry{}, false
	}
	return vis[n.cursor], true
}

// SelectedPath is the absolute path of the highlighted directory, or false
// when nothing is highlighted.
func (n *Navigator) SelectedPath() (string, bool) {
	e, ok := n.Selected()
	if !ok {
		return "", false
	}
	return e.Path, true
}

// Reload re-reads the current directory.
func (n *Navigator) Reload() {
	var focus string
	if e, ok := n.Selected(); ok {
		focus = e.Name
	}
	n.open(n.cwd, focus)
}

// Focus moves the cursor to the entry named name when it is visible.
func (n *Navigator) Focus(name string) {
	vis := n.Visible()
	for i, e := range vis {
		if e.Name == name {
			n.cursor = i
			return
		}
	}
}

// Parent moves to the parent directory. At the filesystem root it is a no-op.
func (n *Navigator) Parent() {
	parent := filepath.Dir(n.cwd)
	if parent == n.cwd {
		return
	}
	child := filepath.Base(n.cwd)
	n.search = false
	n.filter = ""
	n.jump = 0
	n.open(parent, child)
}

// Open enters the highlighted directory, or the named one when name is set.
func (n *Navigator) Open(name string) {
	if name == "" {
		e, ok := n.Selected()
		if !ok {
			return
		}
		name = e.Name
	}
	n.search = false
	n.filter = ""
	n.jump = 0
	n.open(filepath.Join(n.cwd, name), "")
}

// Exists reports whether name already exists under the current directory.
func (n *Navigator) Exists(name string) bool {
	_, err := os.Lstat(filepath.Join(n.cwd, name))
	return err == nil
}

// Mkdir creates a directory under the current directory. The name must already
// pass [ValidateName].
func (n *Navigator) Mkdir(name string) error {
	if err := ValidateName(n.cwd, name); err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(n.cwd, name), 0o755); err != nil {
		return err
	}
	return nil
}

// ValidateName checks a new directory name under dir for emptiness, path
// separators, reserved names and collisions.
func ValidateName(dir, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("enter a folder name")
	}
	if name == "." || name == ".." {
		return errors.New("that name is reserved")
	}
	if strings.ContainsAny(name, `/\:`) || strings.Contains(name, string(os.PathSeparator)) {
		return errors.New("the name must not contain path separators")
	}
	if strings.ContainsRune(name, 0) {
		return errors.New("the name is not valid")
	}
	_, err := os.Lstat(filepath.Join(dir, name))
	if err == nil {
		return errors.New("a folder with that name already exists")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("checking the name: %w", err)
	}
	return nil
}

// DisplayPath returns path with the home prefix rewritten as "~".
func (n *Navigator) DisplayPath(path string) string {
	if n.home != "" {
		if path == n.home {
			return "~"
		}
		prefix := n.home + string(os.PathSeparator)
		if strings.HasPrefix(path, prefix) {
			return "~/" + path[len(prefix):]
		}
	}
	return path
}

func (n *Navigator) open(dir, focus string) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		n.err = err.Error()
		return
	}
	abs = filepath.Clean(abs)
	ents, err := os.ReadDir(abs)
	if err != nil {
		n.err = err.Error()
		n.cwd = abs
		n.entries = nil
		n.cursor = 0
		return
	}
	n.err = ""
	n.cwd = abs
	n.entries = n.entries[:0]
	for _, e := range ents {
		if n.dirsOnly && !e.IsDir() {
			continue
		}
		info, err := e.Info()
		mod := time.Time{}
		if err == nil {
			mod = info.ModTime()
		}
		path := filepath.Join(abs, e.Name())
		st := n.statsFor(path, mod)
		n.entries = append(n.entries, Entry{
			Name:    e.Name(),
			Path:    path,
			ModTime: mod,
			Files:   st.files,
			Bytes:   st.bytes,
		})
	}
	sortEntries(n.entries)
	n.cursor = 0
	if focus != "" {
		n.Focus(focus)
	}
	n.clampCursor()
}

func (n *Navigator) clampCursor() {
	vis := n.Visible()
	if len(vis) == 0 {
		n.cursor = 0
		return
	}
	if n.cursor >= len(vis) {
		n.cursor = len(vis) - 1
	}
	if n.cursor < 0 {
		n.cursor = 0
	}
}

func sortEntries(ents []Entry) {
	for i := 1; i < len(ents); i++ {
		j := i
		for j > 0 && lessName(ents[j].Name, ents[j-1].Name) {
			ents[j], ents[j-1] = ents[j-1], ents[j]
			j--
		}
	}
}

func lessName(a, b string) bool {
	return strings.ToLower(a) < strings.ToLower(b)
}

// HandleKey updates the navigator for key. bodyRows is the visible list height
// for page scrolling. The returned Action is ActionNone when the host need not
// do more; handled is true when the key was consumed.
func (n *Navigator) HandleKey(key string, bodyRows int) (action Action, handled bool) {
	if n.search {
		return n.searchKey(key)
	}
	key = strings.ToLower(key)
	switch key {
	case "up", "k", "ctrl+p":
		n.move(-1)
		return ActionNone, true
	case "down", "j", "ctrl+n":
		n.move(1)
		return ActionNone, true
	case "pgup":
		n.move(-max(bodyRows, 1))
		return ActionNone, true
	case "pgdown":
		n.move(max(bodyRows, 1))
		return ActionNone, true
	case "home":
		n.cursor = 0
		return ActionNone, true
	case "end":
		if vis := n.Visible(); len(vis) > 0 {
			n.cursor = len(vis) - 1
		}
		return ActionNone, true
	case "enter":
		n.Open("")
		return ActionNone, true
	case "ctrl+enter":
		if _, ok := n.Selected(); !ok {
			return ActionNone, true
		}
		return ActionSelect, true
	case "ctrl+b", "backspace":
		n.Parent()
		return ActionNone, true
	case "ctrl+s":
		n.search = true
		return ActionNone, true
	case "ctrl+h":
		n.ToggleHidden()
		return ActionNone, true
	case "ctrl+d":
		return ActionCreate, true
	case "esc":
		if n.filter != "" {
			n.filter = ""
			n.clampCursor()
			return ActionNone, true
		}
		return ActionNone, false
	}
	if r, ok := letterKey(key); ok {
		n.jumpTo(r)
		return ActionNone, true
	}
	return ActionNone, false
}

func (n *Navigator) searchKey(key string) (Action, bool) {
	low := strings.ToLower(key)
	switch low {
	case "esc", "ctrl+s":
		n.search = false
		if low == "esc" {
			n.filter = ""
			n.clampCursor()
		}
		return ActionNone, true
	case "enter":
		n.search = false
		n.Open("")
		return ActionNone, true
	case "ctrl+enter":
		n.search = false
		if _, ok := n.Selected(); !ok {
			return ActionNone, true
		}
		return ActionSelect, true
	case "ctrl+h":
		n.ToggleHidden()
		return ActionNone, true
	case "ctrl+d":
		return ActionCreate, true
	case "ctrl+b":
		n.search = false
		n.Parent()
		return ActionNone, true
	case "backspace":
		if n.filter != "" {
			r := []rune(n.filter)
			n.filter = string(r[:len(r)-1])
			n.clampCursor()
		} else {
			n.search = false
		}
		return ActionNone, true
	case "up", "k", "ctrl+p":
		n.move(-1)
		return ActionNone, true
	case "down", "j", "ctrl+n":
		n.move(1)
		return ActionNone, true
	case "space":
		n.filter += " "
		n.clampCursor()
		return ActionNone, true
	}
	if strings.HasPrefix(low, "ctrl+") || strings.HasPrefix(low, "alt+") ||
		low == "tab" || low == "shift+tab" || low == "pgup" || low == "pgdown" ||
		low == "home" || low == "end" || low == "left" || low == "right" {
		return ActionNone, false
	}
	if rs := []rune(key); len(rs) == 1 && !strings.Contains(key, "+") {
		n.filter += string(rs)
		n.clampCursor()
		return ActionNone, true
	}
	return ActionNone, false
}

func (n *Navigator) move(delta int) {
	vis := n.Visible()
	if len(vis) == 0 {
		n.cursor = 0
		return
	}
	n.cursor += delta
	if n.cursor < 0 {
		n.cursor = 0
	}
	if n.cursor >= len(vis) {
		n.cursor = len(vis) - 1
	}
}

func (n *Navigator) jumpTo(r rune) {
	vis := n.Visible()
	if len(vis) == 0 {
		return
	}
	r = unicode.ToLower(r)
	start := n.cursor
	if n.jump == r {
		start = n.cursor + 1
	}
	n.jump = r
	for i := 0; i < len(vis); i++ {
		idx := (start + i) % len(vis)
		name := []rune(strings.ToLower(vis[idx].Name))
		if len(name) > 0 && name[0] == r {
			n.cursor = idx
			return
		}
	}
}

func letterKey(key string) (rune, bool) {
	if strings.Contains(key, "+") {
		return 0, false
	}
	rs := []rune(key)
	if len(rs) != 1 {
		return 0, false
	}
	r := unicode.ToLower(rs[0])
	if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
		return r, true
	}
	return 0, false
}
