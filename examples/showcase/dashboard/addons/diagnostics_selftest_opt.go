package addons

import (
	"context"
	"log"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// Run Self-Test, a row of the Diagnostics sub-page. Its hotkey (t) works only
// there, and HotkeyHint lists it in the sub-page's footer.

var diagnosticsSelfTestOpt = dashboard.Option{
	Label: "Run Self-Test", Hotkey: "t", HotkeyHint: "Self-test", HotkeyTone: dashygotui.ToneGood,
	Run: func(c *dashboard.Ctx) tea.Cmd {
		return c.RunTask("selftest", "Testing...", func(ctx context.Context) (string, error) {
			select {
			case <-time.After(600 * time.Millisecond):
			case <-ctx.Done():
				return "", ctx.Err()
			}
			log.Print("[INFO] self-test passed")
			return "Self-test passed: 12 of 12 checks.", nil
		})
	},
}
