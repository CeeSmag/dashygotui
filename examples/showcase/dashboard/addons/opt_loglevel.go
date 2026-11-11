package addons

import (
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [17] Log Level: a setting chosen from a list. stdaddons.Pick marks the current
// item, saves the chosen one's Label and confirms it.

func init() {
	dashboard.Registry.Add(stdaddons.Pick[dashboard.State]{
		Key: "logs.level", Section: "settings", Number: 17, Label: "Log Level", Default: "info",
		Help: "How much the service writes to its log.",
		Items: []modal.Item{
			{Label: "debug", Detail: "everything, including internals"},
			{Label: "info", Detail: "normal operation"},
			{Label: "warn", Detail: "problems only"},
		},
	})
}
