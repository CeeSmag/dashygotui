package stdaddons_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/dashtest"
	"github.com/CeeSmag/dashygotui/internal/collect"
	"github.com/CeeSmag/dashygotui/modal"
	"github.com/CeeSmag/dashygotui/stdaddons"
	"github.com/CeeSmag/dashygotui/store"
)

// settingsApp is a dashboard whose only addons are the given settings.
func settingsApp(t *testing.T, st dashygotui.Store, addons ...dashygotui.Addon[state]) *dashtest.Driver {
	t.Helper()
	r := collect.New[state]()
	r.Add(addons...)
	return dashtest.New(t, dashygotui.Config[state]{
		Title: "T", Store: st,
		Map:  dashygotui.Map[state]{Pages: []dashygotui.Page[state]{{ID: "p", Title: "Page", Panels: []dashygotui.Panel[state]{{Title: "Settings", Items: []dashygotui.Addon[state]{r.All("s")}}}}}},
		Poll: func(context.Context) (state, error) { return state{}, nil },
	}, 140, 40)
}

func TestToggle(t *testing.T) {
	st := store.NewMemory()
	var changed []bool
	d := settingsApp(t, st, stdaddons.Toggle[state]{
		Key: "notify", Label: "Notifications", Default: true,
		OnChange: func(_ *ctx, on bool) tea.Cmd { changed = append(changed, on); return nil },
	})
	d.RequireContains("[1] Notifications", "ON")
	d.Press("enter")
	d.RequireContains("Notifications off.", "OFF")
	var saved bool
	if ok, _ := st.Get("notify", &saved); !ok || saved {
		t.Errorf("stored = %v, want false", saved)
	}
	d.Press("enter")
	d.RequireContains("Notifications on.")
	if len(changed) != 2 || changed[0] || !changed[1] {
		t.Errorf("OnChange calls = %v, want [false true]", changed)
	}
}

func TestToggleRemembersAcrossRuns(t *testing.T) {
	st := store.NewMemory()
	mk := func() *dashtest.Driver {
		return settingsApp(t, st, stdaddons.Toggle[state]{Key: "k", Label: "Flag"})
	}
	d := mk()
	d.RequireContains("OFF") // the zero Default is off
	d.Press("enter")
	mk().RequireContains("ON")
}

func TestToggleReportsAStoreFailure(t *testing.T) {
	d := settingsApp(t, failingStore{}, stdaddons.Toggle[state]{Key: "k", Label: "Flag"})
	d.Press("enter")
	d.RequireContains("saving Flag", "disk is full")
}

// failingStore reads nothing and fails every write.
type failingStore struct{}

func (failingStore) Get(string, any) (bool, error) { return false, nil }
func (failingStore) Set(string, any) error         { return errors.New("disk is full") }

func TestNumber(t *testing.T) {
	st := store.NewMemory()
	var got []int
	d := settingsApp(t, st, stdaddons.Number[state]{
		Key: "limit", Label: "Storage Limit", Unit: "GB", Min: 1, Max: 1000, Default: 50,
		OnChange: func(_ *ctx, n int) tea.Cmd { got = append(got, n); return nil },
	})
	d.RequireContains("[1] Storage Limit", "[50 GB]")
	d.Press("enter")
	d.RequireContains("Storage Limit", "A whole number from 1 to 1000.", "Current: 50 GB")

	d.Type("abc")
	d.Press("enter")
	d.RequireContains(`"abc" is not a number from 1 to 1000`)
	d.Press("backspace", "backspace", "backspace")
	d.Type("5000")
	d.Press("enter")
	d.RequireContains(`"5000" is not a number from 1 to 1000`)
	if len(got) != 0 {
		t.Fatalf("OnChange ran for an invalid value: %v", got)
	}

	d.Press("backspace", "backspace", "backspace", "backspace")
	d.Type("75")
	d.Press("enter")
	d.RequireContains("Storage Limit set to 75 GB.", "[75 GB]")
	var n int
	if ok, _ := st.Get("limit", &n); !ok || n != 75 {
		t.Errorf("stored = %d", n)
	}
	if len(got) != 1 || got[0] != 75 {
		t.Errorf("OnChange calls = %v, want [75]", got)
	}
}

func TestNumberShowsAStoreFailureInThePopup(t *testing.T) {
	d := settingsApp(t, failingStore{}, stdaddons.Number[state]{Key: "k", Label: "Limit", Min: 1, Max: 9, Default: 5})
	d.Press("enter")
	d.Type("7")
	d.Press("enter")
	d.RequireContains("Limit", "disk is full") // the editor stays open with the error
}

func TestPick(t *testing.T) {
	st := store.NewMemory()
	var got []string
	d := settingsApp(t, st, stdaddons.Pick[state]{
		Key: "region", Label: "Region", Help: "Where it runs.",
		Items:    []modal.Item{{Label: "us-east", Detail: "Virginia"}, {Label: "eu-west", Detail: "Ireland"}},
		OnChange: func(_ *ctx, l string) tea.Cmd { got = append(got, l); return nil },
	})
	d.RequireContains("[1] Region", "[us-east]") // the first item is the default
	d.Press("enter")
	d.RequireContains("Where it runs.", "[current]", "Virginia", "Ireland", "Enter Save")
	d.Press("down", "enter")
	d.RequireContains("Region set to eu-west.", "[eu-west]")
	var name string
	if ok, _ := st.Get("region", &name); !ok || name != "eu-west" {
		t.Errorf("stored = %q", name)
	}
	if len(got) != 1 || got[0] != "eu-west" {
		t.Errorf("OnChange calls = %v", got)
	}
	d.Press("enter") // reopened: the saved item is the one marked and highlighted
	d.RequireContains("now eu-west")
}

func TestPickWithADefault(t *testing.T) {
	d := settingsApp(t, store.NewMemory(), stdaddons.Pick[state]{
		Key: "level", Label: "Level", Default: "warn",
		Items: []modal.Item{{Label: "debug"}, {Label: "warn"}},
	})
	d.RequireContains("[warn]")
}

// A setting that makes no sense is a startup error naming the call site, not a
// surprise at run time.
func TestSettingsWithBadSpecsAreStartupErrors(t *testing.T) {
	items := []modal.Item{{Label: "a"}, {Label: "b"}}
	cases := map[string]struct {
		addon dashygotui.Addon[state]
		want  string
	}{
		"Number min not below max": {stdaddons.Number[state]{Key: "n", Label: "N", Min: 5, Max: 5}, "Min (5) must be less than Max (5)"},
		"Number default outside":   {stdaddons.Number[state]{Key: "n", Label: "N", Min: 1, Max: 9, Default: 20}, "Default (20) is outside"},
		"Pick without items":       {stdaddons.Pick[state]{Key: "p", Label: "P"}, "no Items to pick from"},
		"Pick duplicate labels":    {stdaddons.Pick[state]{Key: "p", Label: "P", Items: []modal.Item{{Label: "a"}, {Label: "a"}}}, `two items labelled "a"`},
		"Pick empty label":         {stdaddons.Pick[state]{Key: "p", Label: "P", Items: []modal.Item{{Label: ""}}}, "an item with no Label"},
		"Pick unknown default":     {stdaddons.Pick[state]{Key: "p", Label: "P", Items: items, Default: "z"}, `Default "z" is not one of its items`},
		"reserved store key":       {stdaddons.Toggle[state]{Key: "dashygotui.palette", Label: "T"}, "reserved for the library"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := collect.New[state]()
			r.Add(tc.addon)
			err := collect.Errors(r)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate = %v, want it to mention %q", err, tc.want)
			}
			if !strings.Contains(err.Error(), "Collect > Rows > row 1") {
				t.Errorf("the error should name the place in the map: %v", err)
			}
		})
	}
}

func TestText(t *testing.T) {
	st := store.NewMemory()
	var got []string
	d := settingsApp(t, st, stdaddons.Text[state]{
		Key: "host", Label: "Hostname", Default: "example.local", AllowEmpty: true,
		Validate: func(v string) error {
			if strings.ContainsAny(v, " /") {
				return errors.New("no spaces or slashes")
			}
			return nil
		},
		OnChange: func(_ *ctx, v string) tea.Cmd { got = append(got, v); return nil },
	})
	d.RequireContains("[example.local]")
	d.Press("enter")
	d.Press("ctrl+u") // clear the prefilled value
	d.Type("bad name")
	d.Press("enter")
	d.RequireContains("no spaces or slashes")
	d.Press("ctrl+u")
	d.Type("box.lan")
	d.Press("enter")
	d.RequireContains("Hostname saved.", "[box.lan]")

	d.Press("enter") // an empty value clears back to the default
	d.Press("ctrl+u")
	d.Press("enter")
	d.RequireContains("[example.local]")
	if len(got) != 2 || got[0] != "box.lan" || got[1] != "example.local" {
		t.Errorf("OnChange calls = %v", got)
	}
}

func TestSecretTextIsNeverShown(t *testing.T) {
	st := store.NewMemory()
	d := settingsApp(t, st, stdaddons.Text[state]{Key: "token", Label: "API Token", Secret: true})
	d.RequireContains("[not set]")
	d.Press("enter")
	d.Type("hunter2")
	d.RequireAbsent("hunter2")
	d.Press("enter")
	d.RequireContains("[set]")
	d.RequireAbsent("hunter2")
}

// A setting is saved under its Key. In a map nothing else asks for one, so a
// setting without it is an error, not four settings sharing one Store entry.
func TestASettingWithoutAKeyIsAnError(t *testing.T) {
	items := []modal.Item{{Label: "a"}, {Label: "b"}}
	for name, a := range map[string]dashygotui.Addon[state]{
		"Toggle": stdaddons.Toggle[state]{Label: "On"},
		"Number": stdaddons.Number[state]{Label: "Count", Min: 1, Max: 9},
		"Pick":   stdaddons.Pick[state]{Label: "Choice", Items: items},
		"Text":   stdaddons.Text[state]{Label: "Name"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := dashygotui.New(dashygotui.Config[state]{Map: dashygotui.Map[state]{Pages: []dashygotui.Page[state]{
				{Title: "Page", Panels: []dashygotui.Panel[state]{{Title: "Settings", Items: []dashygotui.Addon[state]{a}}}},
			}}})
			if err == nil || !strings.Contains(err.Error(), "stdaddons."+name) || !strings.Contains(err.Error(), "has no Key") {
				t.Errorf("New = %v, want an error naming stdaddons.%s and its missing Key", err, name)
			}
		})
	}
}
