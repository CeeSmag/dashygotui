package dashygotui_test

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
)

type state struct{ Count int }

type (
	reg    = dashygotui.Registry[state]
	ctx    = dashygotui.Context[state]
	option = dashygotui.Option[state]
	widget = dashygotui.Widget[state]
	cfg    = dashygotui.Config[state]
)

func noop(*ctx) tea.Cmd { return nil }

// baseConfig is a small valid dashboard: three pages, three sections.
func baseConfig(r *reg) cfg {
	return cfg{
		Title:    "TestApp",
		Registry: r,
		Pages: []dashygotui.Page[state]{
			{ID: "monitor", Title: "Monitor"},
			{ID: "setup", Title: "Setup"},
			{ID: "theme", Title: "Theme"},
		},
		Sections: []dashygotui.SectionSpec[state]{
			{ID: "actions", Page: "setup", Title: "Quick Actions", Order: 10},
			{ID: "settings", Page: "setup", Title: "Settings", Order: 20},
			{ID: "appearance", Page: "theme", Title: "Appearance"},
		},
	}
}

func validOption(key string, number int) option {
	return option{Key: key, Section: "actions", Number: number, Label: "Label " + key, Run: noop}
}

// errText builds the dashboard and returns the text of its error, or "".
func errText(t *testing.T, c cfg) string {
	t.Helper()
	_, err := dashygotui.New(c)
	if err == nil {
		return ""
	}
	if !errors.Is(err, dashygotui.ErrInvalidConfig) {
		t.Errorf("error %v does not wrap ErrInvalidConfig", err)
	}
	return err.Error()
}

func TestNewAcceptsAValidConfig(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	r.Option(validOption("a", 1))
	r.Widget(widget{Key: "w", Page: "monitor", Title: "W", Render: func(*ctx, dashygotui.Size) []string { return nil }})
	if msg := errText(t, baseConfig(r)); msg != "" {
		t.Fatalf("unexpected error: %s", msg)
	}
}

func TestRegistryRejectsMistakesAndNamesTheSite(t *testing.T) {
	tests := []struct {
		name     string
		register func(r *reg)
		want     []string
	}{
		{
			name: "duplicate number names both sites",
			register: func(r *reg) {
				r.Option(validOption("first", 7))
				r.Option(validOption("second", 7))
			},
			want: []string{"reuses number 7", "registry_test.go:"},
		},
		{
			name: "duplicate key",
			register: func(r *reg) {
				r.Option(validOption("same", 1))
				r.Option(validOption("same", 2))
			},
			want: []string{`duplicate option key "same"`},
		},
		{
			name:     "missing key",
			register: func(r *reg) { r.Option(option{Section: "actions", Number: 1, Label: "L", Run: noop}) },
			want:     []string{"option has no Key"},
		},
		{
			name:     "missing section",
			register: func(r *reg) { r.Option(option{Key: "k", Number: 1, Label: "L", Run: noop}) },
			want:     []string{`option "k" has no Section`},
		},
		{
			name:     "missing label",
			register: func(r *reg) { r.Option(option{Key: "k", Section: "actions", Number: 1, Run: noop}) },
			want:     []string{`option "k" has no Label`},
		},
		{
			name:     "missing run",
			register: func(r *reg) { r.Option(option{Key: "k", Section: "actions", Number: 1, Label: "L"}) },
			want:     []string{`option "k" needs exactly one of Run, Do, Opens and Modal, has 0`},
		},
		{
			name:     "non-positive number",
			register: func(r *reg) { r.Option(validOption("k", 0)) },
			want:     []string{"numbers must be positive"},
		},
		{
			name: "several mistakes are all reported",
			register: func(r *reg) {
				r.Option(validOption("a", 1))
				r.Option(validOption("a", 1))
			},
			want: []string{"duplicate option key", "reuses number"},
		},
		{
			name:     "unknown section",
			register: func(r *reg) { r.Option(option{Key: "k", Section: "nope", Number: 1, Label: "L", Run: noop}) },
			want:     []string{`option "k" targets unknown section "nope"`},
		},
		{
			name: "widget on unknown page",
			register: func(r *reg) {
				r.Widget(widget{Key: "w", Page: "nope", Title: "W", Render: func(*ctx, dashygotui.Size) []string { return nil }})
			},
			want: []string{`widget "w" targets unknown page "nope"`},
		},
		{
			name:     "widget without render",
			register: func(r *reg) { r.Widget(widget{Key: "w", Page: "monitor", Title: "W"}) },
			want:     []string{`widget "w" has no Render`},
		},
		{
			name: "duplicate widget key",
			register: func(r *reg) {
				render := func(*ctx, dashygotui.Size) []string { return nil }
				r.Widget(widget{Key: "w", Page: "monitor", Title: "W", Render: render})
				r.Widget(widget{Key: "w", Page: "monitor", Title: "W2", Render: render})
			},
			want: []string{`duplicate widget key "w"`},
		},
		{
			name:     "view that draws nothing",
			register: func(r *reg) { r.View(dashygotui.View[state]{ID: "v", Title: "V"}) },
			want:     []string{`view "v" draws nothing`},
		},
		{
			name: "duplicate view",
			register: func(r *reg) {
				v := dashygotui.View[state]{ID: "v", Title: "V", Body: func(*ctx, dashygotui.Size) []string { return nil }}
				r.View(v)
				r.View(v)
			},
			want: []string{`duplicate view ID "v"`},
		},
		{
			name:     "reserved shortcut key",
			register: func(r *reg) { r.Shortcut(dashygotui.Shortcut[state]{Key: "tab", Run: noop}) },
			want:     []string{`shortcut "tab" is reserved`},
		},
		{
			name:     "digit shortcut",
			register: func(r *reg) { r.Shortcut(dashygotui.Shortcut[state]{Key: "5", Run: noop}) },
			want:     []string{`shortcut "5" is reserved`},
		},
		{
			name: "duplicate shortcut",
			register: func(r *reg) {
				r.Shortcut(dashygotui.Shortcut[state]{Key: "d", Run: noop})
				r.Shortcut(dashygotui.Shortcut[state]{Key: "d", Run: noop})
			},
			want: []string{`duplicate shortcut key "d"`},
		},
		{
			name:     "nil hooks",
			register: func(r *reg) { r.OnTick(nil); r.OnResult(nil); r.OnMsg(nil) },
			want:     []string{"tick hook is nil", "result hook is nil", "message hook is nil"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := dashygotui.NewRegistry[state]()
			tt.register(r)
			got := errText(t, baseConfig(r))
			if got == "" {
				t.Fatal("want an error, got none")
			}
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("error does not mention %q:\n%s", w, got)
				}
			}
		})
	}
}

func TestRegistryValidateReportsRegistrationErrors(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	if err := r.Validate(); err != nil {
		t.Fatalf("an empty registry is valid: %v", err)
	}
	r.Option(validOption("a", 1))
	r.Option(validOption("b", 1))
	err := r.Validate()
	if err == nil {
		t.Fatal("duplicate number not reported")
	}
	var re *dashygotui.RegistrationError
	if !errors.As(err, &re) {
		t.Fatalf("error %T does not unwrap to *RegistrationError", err)
	}
	if !strings.HasPrefix(re.Site, "registry_test.go:") {
		t.Errorf("site = %q, want the test file", re.Site)
	}
}

func TestRegisteringAfterNewIsAnError(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	r.Option(validOption("a", 1))
	if _, err := dashygotui.New(baseConfig(r)); err != nil {
		t.Fatalf("New: %v", err)
	}
	r.Option(validOption("late", 2))
	err := r.Validate()
	if err == nil || !strings.Contains(err.Error(), "registered after the dashboard was created") {
		t.Fatalf("late registration not reported: %v", err)
	}
}

func TestNewRejectsBrokenConfigs(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	tests := []struct {
		name   string
		mutate func(c *cfg)
		want   string
	}{
		{"nil registry", func(c *cfg) { c.Registry = nil }, "Registry is nil"},
		{"no pages", func(c *cfg) { c.Pages = nil }, "no Pages declared"},
		{"duplicate page", func(c *cfg) { c.Pages[1].ID = "monitor" }, `duplicate page ID "monitor"`},
		{"page without title", func(c *cfg) { c.Pages[0].Title = "" }, `page "monitor" has no Title`},
		{"duplicate section", func(c *cfg) { c.Sections[1].ID = "actions" }, `duplicate section ID "actions"`},
		{"section on unknown page", func(c *cfg) { c.Sections[0].Page = "nope" }, `section "actions" is on unknown page "nope"`},
		{"section without title", func(c *cfg) { c.Sections[0].Title = "" }, `section "actions" has no Title`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := baseConfig(r)
			c.Pages = append([]dashygotui.Page[state](nil), c.Pages...)
			c.Sections = append([]dashygotui.SectionSpec[state](nil), c.Sections...)
			tt.mutate(&c)
			got := errText(t, c)
			if !strings.Contains(got, tt.want) {
				t.Errorf("error = %q, want it to mention %q", got, tt.want)
			}
		})
	}
}

func TestRegistrySiteSurvivesGenerics(t *testing.T) {
	r := dashygotui.NewRegistry[state]()
	r.Option(validOption("a", 1))
	r.Option(validOption("b", 1)) // the line the error must point at
	err := r.Validate()
	if err == nil || !strings.Contains(err.Error(), "registry_test.go:") {
		t.Fatalf("site missing from %v", err)
	}
}
