// Package logs captures log output into a bounded, concurrency-safe buffer and
// shows it in a scrollable viewer, newest entry first.
//
// A full-screen dashboard cannot let the standard logger write to the terminal:
// every line would tear the frame. Divert the logger into a [Buffer] before the
// dashboard starts and the lines appear in the dashboard's log view instead.
// Both the standard [log] package and [log/slog] are supported.
package logs

import (
	"io"
	"log"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Level is the severity of an entry.
type Level int

// The levels, from least to most severe.
const (
	Debug Level = iota
	Info
	Warn
	Error
)

// String returns "DEBUG", "INFO", "WARN" or "ERROR".
func (l Level) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	default:
		return "INFO"
	}
}

// Entry is one captured log line.
type Entry struct {
	// Seq numbers entries from 1 in the order they were added. It never
	// repeats, so a viewer can keep its place while old entries are evicted.
	Seq uint64
	// Time is when the entry was captured.
	Time time.Time
	// Level is the severity, taken from the line when it names one and Info
	// otherwise.
	Level Level
	// Text is the message, without the timestamp and level the logger wrote.
	Text string
}

// DefaultCapacity is the number of entries a buffer keeps when none is given.
const DefaultCapacity = 1000

// Buffer is a bounded ring of log entries. When it is full the oldest entry is
// dropped. It is safe for concurrent use, so any goroutine may log into it
// while the dashboard reads it. It implements [io.Writer].
type Buffer struct {
	mu       sync.Mutex
	capacity int
	entries  []Entry // oldest first
	seq      uint64
	now      func() time.Time
}

var _ io.Writer = (*Buffer)(nil)

// NewBuffer returns a buffer that keeps the newest capacity entries. A
// non-positive capacity means [DefaultCapacity].
func NewBuffer(capacity int) *Buffer {
	if capacity <= 0 {
		capacity = DefaultCapacity
	}
	return &Buffer{capacity: capacity, now: time.Now}
}

// Add appends an entry.
func (b *Buffer) Add(level Level, text string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.seq++
	e := Entry{Seq: b.seq, Time: b.now(), Level: level, Text: text}
	if len(b.entries) >= b.capacity {
		copy(b.entries, b.entries[1:])
		b.entries[len(b.entries)-1] = e
		return
	}
	b.entries = append(b.entries, e)
}

// Write implements [io.Writer]: each line becomes one entry. The standard
// logger's timestamp prefix is dropped, and a leading level such as "[ERROR]",
// "ERROR:" or slog's "level=WARN" sets the entry's level. It always reports the
// whole input as written.
func (b *Buffer) Write(p []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimRight(string(p), "\r\n"), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		level, text := parseLine(line)
		b.Add(level, text)
	}
	return len(p), nil
}

var (
	stdTimestamp = regexp.MustCompile(`^(?:\d{4}/\d{2}/\d{2} )?\d{2}:\d{2}:\d{2}(?:\.\d+)? `)
	slogPrefix   = regexp.MustCompile(`^time=\S+ level=(\S+) (.*)$`)
	levelPrefix  = regexp.MustCompile(`(?i)^\s*\[?(debug|info|warning|warn|error|err|fatal|panic)\]?[:\s]+`)
	// A logger prefix ("myapp ") may come before a bracketed level.
	prefixedLevel = regexp.MustCompile(`(?i)^(\S+\s)\[(debug|info|warning|warn|error|err|fatal|panic)\]\s*`)
)

// parseLine splits a log line into its level and message.
func parseLine(line string) (Level, string) {
	line = stdTimestamp.ReplaceAllString(line, "")
	if m := slogPrefix.FindStringSubmatch(line); m != nil {
		return parseLevel(m[1]), m[2]
	}
	if m := levelPrefix.FindStringSubmatch(line); m != nil {
		return parseLevel(m[1]), line[len(m[0]):]
	}
	if m := prefixedLevel.FindStringSubmatch(line); m != nil {
		return parseLevel(m[2]), m[1] + line[len(m[0]):]
	}
	return Info, line
}

// parseLevel reads a level name; unknown names count as Info. slog's "INFO+2"
// style offsets are ignored.
func parseLevel(s string) Level {
	s = strings.ToUpper(s)
	if i := strings.IndexAny(s, "+-"); i > 0 {
		s = s[:i]
	}
	switch s {
	case "DEBUG":
		return Debug
	case "WARN", "WARNING":
		return Warn
	case "ERROR", "ERR", "FATAL", "PANIC":
		return Error
	default:
		return Info
	}
}

// Handler returns a [slog.Handler] that records into the buffer, as text. opts
// may be nil.
func (b *Buffer) Handler(opts *slog.HandlerOptions) slog.Handler {
	return slog.NewTextHandler(b, opts)
}

// Divert points l at the buffer, so everything it logs is captured instead of
// printed, and returns a function that puts l back as it was. Do this before
// the dashboard starts; pass the standard logger:
//
//	restore := buf.Divert(log.Default())
//	defer restore()
//
// A nil logger is ignored.
func (b *Buffer) Divert(l *log.Logger) (restore func()) {
	if l == nil {
		return func() {}
	}
	prevOut, prevFlags := l.Writer(), l.Flags()
	l.SetOutput(b)
	l.SetFlags(0) // the buffer records its own time
	return func() {
		l.SetOutput(prevOut)
		l.SetFlags(prevFlags)
	}
}

// Snapshot returns a copy of the entries, newest first.
func (b *Buffer) Snapshot() []Entry {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Entry, len(b.entries))
	for i, e := range b.entries {
		out[len(b.entries)-1-i] = e
	}
	return out
}

// Len reports how many entries are held.
func (b *Buffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}

// Count reports how many held entries are at or above the given level.
func (b *Buffer) Count(atLeast Level) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := 0
	for _, e := range b.entries {
		if e.Level >= atLeast {
			n++
		}
	}
	return n
}

// Clear drops every entry. Sequence numbers keep counting.
func (b *Buffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = b.entries[:0]
}
