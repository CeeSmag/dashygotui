package addons

import (
	"github.com/CeeSmag/dashygotui/examples/buildwatch/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// Three settings, one declaration each: stdaddons gives the badge, the pop-up,
// validation, saving and the confirmation.

var notifyFailedSetting = stdaddons.Toggle[dashboard.State]{Key: "notify_failed", Label: "Notify on Failure", Default: true}

var keepBuildsSetting = stdaddons.Number[dashboard.State]{Key: "keep_builds", Label: "Builds to Keep",
	Unit: "builds", Min: 5, Max: 500, Default: 50}

var branchSetting = stdaddons.Pick[dashboard.State]{Key: "branch", Label: "Watched Branch", Default: "main",
	Items: []modal.Item{{Label: "main"}, {Label: "develop"}, {Label: "release"}}}
