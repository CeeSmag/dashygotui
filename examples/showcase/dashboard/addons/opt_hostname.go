package addons

import (
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// [15] Hostname: modal.Edit with Initial, Limit, AllowEmpty (an empty value
// means "use the default") and a synchronous Submit.

func init() {
	const hostKey, defaultHost = "hostname", "example.local"
	host := func(c *dashboard.Ctx) string {
		h := defaultHost
		_, _ = c.Store().Get(hostKey, &h)
		return h
	}

	dashboard.Registry.Add(dashboard.Option{
		Key: "hostname", Section: "settings", Number: 15, Label: "Hostname",
		Badge: func(c *dashboard.Ctx) string { return c.Badge(host(c)) },
		Modal: func(c *dashboard.Ctx) dashboard.Modal {
			return &modal.Edit[dashboard.State]{
				Title: "Hostname", Help: "Leave it empty to use the default.", Current: host(c),
				Initial: host(c), Limit: 24, AllowEmpty: true,
				Validate: func(v string) error {
					if strings.ContainsAny(v, " /\\") {
						return errors.New("a hostname cannot contain spaces or slashes")
					}
					return nil
				},
				Submit: func(c *dashboard.Ctx, value string) tea.Cmd {
					if value == "" {
						value = defaultHost
					}
					c.FlashResult("Hostname set to "+value+".", c.Store().Set(hostKey, value))
					return nil
				},
			}
		},
	})
}
