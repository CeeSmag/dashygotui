package addons

import (
	"errors"
	"strings"

	"github.com/CeeSmag/dashygotui/examples/showcase/dashboard"
	"github.com/CeeSmag/dashygotui/stdaddons"
)

// [18] Display Name: a text setting. stdaddons.Text gives the badge, the one-line
// pop-up, validation, saving, and "not set" until a value exists.

func init() {
	dashboard.Registry.Add(stdaddons.Text[dashboard.State]{
		Key: "display_name", Section: "settings", Number: 18, Label: "Display Name",
		Help: "Shown in the header of reports.", Limit: 40, AllowEmpty: true,
		Validate: func(v string) error {
			if strings.ContainsAny(v, "<>") {
				return errors.New("a name cannot contain < or >")
			}
			return nil
		},
	})
}
