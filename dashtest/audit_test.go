package dashtest_test

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/modal"
)

type auditState struct{}

type actx = dashygotui.Context[auditState]

// failureLog is a testing.TB that records failures instead of failing the test.
type failureLog struct {
	testing.TB
	mu   sync.Mutex
	errs []string
}

func (r *failureLog) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errs = append(r.errs, strings.TrimSpace(sprintf(format, args...)))
}

func (r *failureLog) Fatalf(format string, args ...any) {
	r.Errorf(format, args...)
	runtimeGoexit()
}

// audit runs dashtest.Audit on cfg against a failureLog and returns what failed.
func audit(t *testing.T, cfg dashygotui.Config[auditState], opts ...dashtest.AuditOption) []string {
	t.Helper()
	rec := &failureLog{TB: t}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		dashtest.Audit(rec, cfg, opts...)
	}()
	wg.Wait()
	return rec.errs
}

// goodConfig is a small dashboard that obeys every rule.
func goodConfig() dashygotui.Config[auditState] {
	r := dashygotui.NewRegistry[auditState]()
	r.Add(
		dashygotui.Option[auditState]{Key: "about", Section: "s", Number: 1, Label: "About",
			Modal: func(*actx) dashygotui.Modal[auditState] {
				return &modal.Notice[auditState]{Title: "About", Lines: []string{"plain ascii text"}}
			}},
		dashygotui.View[auditState]{ID: "v", Title: "A View",
			Body:  func(*actx, dashygotui.Size) []string { return []string{"hello"} },
			Entry: &dashygotui.Entry[auditState]{Section: "s", Number: 2}},
		dashygotui.Widget[auditState]{Key: "w", Page: "p", Title: "Widget",
			Render: func(*actx, dashygotui.Size) []string { return []string{"ok"} }},
	)
	return dashygotui.Config[auditState]{
		Title: "T", Registry: r,
		Pages:    []dashygotui.Page[auditState]{{ID: "p", Title: "Page"}},
		Sections: []dashygotui.SectionSpec[auditState]{{ID: "s", Page: "p", Title: "Section"}},
		Poll:     func(context.Context) (auditState, error) { return auditState{}, nil },
	}
}

func TestAuditPassesAWellBehavedDashboard(t *testing.T) {
	if errs := audit(t, goodConfig(), dashtest.WalkOptions()); len(errs) != 0 {
		t.Errorf("a good dashboard failed the audit:\n%s", strings.Join(errs, "\n"))
	}
}

func TestAuditFailsOnABrokenRegistration(t *testing.T) {
	cfg := goodConfig()
	cfg.Registry.Add(dashygotui.Option[auditState]{Key: "dup", Section: "s", Number: 1, Label: "Clash",
		Do: func(*actx) {}}) // [1] is About
	errs := audit(t, cfg)
	if len(errs) == 0 || !strings.Contains(errs[0], "building the dashboard") || !strings.Contains(errs[0], "reuses number 1") {
		t.Errorf("errors = %v", errs)
	}
}

func TestAuditCatchesAPanicWithItsLocation(t *testing.T) {
	cfg := goodConfig()
	cfg.Registry.Add(dashygotui.Widget[auditState]{Key: "boom", Page: "p", Title: "Boom",
		Render: func(*actx, dashygotui.Size) []string { panic("kaboom") }})
	errs := audit(t, cfg)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "panicked: kaboom") && strings.Contains(e, "page p") {
			found = true
		}
	}
	if !found {
		t.Errorf("want a panic report naming the page, got:\n%s", strings.Join(errs, "\n"))
	}
}

func TestAuditCatchesGlyphsTheConsoleCannotShow(t *testing.T) {
	cfg := goodConfig()
	cfg.Registry.Add(dashygotui.Widget[auditState]{Key: "glyph", Page: "p", Title: "Glyph",
		Render: func(*actx, dashygotui.Size) []string { return []string{"done ✓"} }})
	errs := audit(t, cfg)
	found := false
	for _, e := range errs {
		if strings.Contains(e, "U+2713") && strings.Contains(e, "tty") {
			found = true
		}
	}
	if !found {
		t.Errorf("want the check mark reported in TTY mode, got:\n%s", strings.Join(errs, "\n"))
	}
	for _, e := range errs {
		if strings.Contains(e, "(unicode)") {
			t.Errorf("a glyph is fine in Unicode mode: %s", e)
		}
	}
}

func TestAuditWalksOptionsOnlyWhenAsked(t *testing.T) {
	cfg := goodConfig()
	opened := 0
	cfg.Registry.Add(dashygotui.Option[auditState]{Key: "probe", Section: "s", Number: 9, Label: "Probe",
		Do: func(*actx) { opened++ }})
	if audit(t, cfg); opened != 0 {
		t.Fatalf("Audit ran an option without WalkOptions (%d times)", opened)
	}
	audit(t, cfg, dashtest.WalkOptions())
	if opened == 0 {
		t.Error("WalkOptions should have opened the option")
	}
}

func TestOutline(t *testing.T) {
	dash, err := dashygotui.New(goodConfig())
	if err != nil {
		t.Fatal(err)
	}
	out := dash.Outline()
	if len(out.Pages) != 1 || out.Pages[0] != "p" || len(out.Views) != 1 || out.Views[0] != "v" {
		t.Errorf("outline = %+v", out)
	}
	if len(out.Options) != 2 || out.Options[0].Number != 1 || out.Options[1].Key != "view.v" || out.Options[0].Page != "p" {
		t.Errorf("options = %+v", out.Options)
	}
}

func sprintf(format string, args ...any) string { return fmt.Sprintf(format, args...) }

func runtimeGoexit() { runtime.Goexit() }
