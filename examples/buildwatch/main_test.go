package main

import (
	"testing"

	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/examples/buildwatch/dashboard"
	"github.com/CeeSmag/dashygotui/store"
)

func newApp(t *testing.T) *dashtest.Driver {
	t.Helper()
	return dashtest.New(t, dashboard.Config(store.NewMemory()), 120, 36)
}

func TestFirstPageShowsThePipelines(t *testing.T) {
	d := newApp(t)
	d.RequireContains("Buildwatch", "Pipelines", "api", "web", "worker", "docs", "[l] Build log")
}

func TestOpenTheBuildLogFromItsHotkey(t *testing.T) {
	d := newApp(t)
	d.Press("l")
	d.RequireContains("Build Log", "BUILD LOG", "step 20 ok") // Follow starts at the newest line
	d.Press("esc")
	d.RequireContains("Pipelines")
}

func TestSettingsAreOneLineEach(t *testing.T) {
	st := store.NewMemory()
	d := dashtest.New(t, dashboard.Config(st), 120, 36)
	d.Press("tab")
	d.RequireContains("Notify on Failure", "Builds to Keep", "[50 builds]", "Watched Branch", "[main]")
	d.Type("11")
	d.Press("enter")
	d.Type("80")
	d.Press("enter")
	d.RequireContains("Builds to Keep set to 80 builds.", "[80 builds]")
}

func TestCancelAllAsksFirst(t *testing.T) {
	d := newApp(t)
	d.RequireContains("[Ctrl+X] Cancel all")
	d.Press("ctrl+x")
	d.RequireContains("Cancel every running build?")
	d.Press("enter")
	d.RequireContains("Cancelled all running builds")
}

func TestPassesTheAudit(t *testing.T) {
	dashtest.Audit(t, dashboard.Config(store.NewMemory()), dashtest.WalkOptions())
}
