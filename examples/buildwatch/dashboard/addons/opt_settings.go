package addons

import (
	"github.com/CeeSmag/dashygotui/examples/buildwatch/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// Three settings, one declaration each: stdaddons gives the badge, the pop-up,
// validation, saving and the confirmation.

func init() {
	dashboard.Registry.Add(
		stdaddons.Toggle[dashboard.State]{Key: "notify_failed", Section: "prefs", Number: 10, Label: "Notify on Failure", Default: true},
		stdaddons.Number[dashboard.State]{Key: "keep_builds", Section: "prefs", Number: 11, Label: "Builds to Keep",
			Unit: "builds", Min: 5, Max: 500, Default: 50},
		stdaddons.Pick[dashboard.State]{Key: "branch", Section: "prefs", Number: 12, Label: "Watched Branch", Default: "main",
			Items: []modal.Item{{Label: "main"}, {Label: "develop"}, {Label: "release"}}},
	)
}
