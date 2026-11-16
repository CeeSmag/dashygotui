package addons

import (
	"github.com/CeeSmag/dashygotui"
	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
)

// Two panels side by side.
func popupsPage() dashboard.Page {
	return dashboard.Page{ID: "popups", Title: "Pop-ups", ShortTitle: "Pop", TinyTitle: "P", Layout: dashygotui.LayoutColumns,
		Panels: []dashboard.Panel{
			{Title: "Standard Pop-ups", Subtitle: "from the modal package", Number: 30, Items: []dashboard.Addon{
				noticeOption, noticeLiveOption, noticeScrollOption, confirmOption, acknowledgeOption,
				typedOption, choiceOption, sliderOption, editOption, // [30] to [38]
			}},
			{Title: "Custom Pop-ups", Subtitle: "written for this app", Number: 40, Items: []dashboard.Addon{
				diceOption, pingOption, chainOption, // [40] to [42]
			}},
		}}
}
