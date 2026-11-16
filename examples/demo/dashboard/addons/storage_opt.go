package addons

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/CeeSmag/dashygotui/examples/demo/dashboard"
	"github.com/CeeSmag/dashygotui/modal"
)

// Storage Limit: a text editor that validates, then saves in the background.

const storageKey = "storage.limit_gb"

const storageDefaultGB = 50

func storageLimit(c *dashboard.Ctx) int {
	var gb int
	if ok, _ := c.Store().Get(storageKey, &gb); !ok {
		return storageDefaultGB
	}
	return gb
}

var storageOpt = dashboard.Option{
	Label: "Storage Limit",
	Badge: func(c *dashboard.Ctx) string { return c.Badge(fmt.Sprintf("%d GB", storageLimit(c))) },
	Modal: func(c *dashboard.Ctx) dashboard.Modal {
		st := c.Store()
		return &modal.Edit[dashboard.State]{
			Title:   "Storage Limit",
			Help:    "How much disk space the service may use, in gigabytes (1 to 1000).",
			Current: fmt.Sprintf("%d GB", storageLimit(c)),
			Pending: "Saving the limit...",
			Validate: func(v string) error {
				n, err := strconv.Atoi(v)
				if err != nil || n < 1 || n > 1000 {
					return fmt.Errorf("%q is not a number from 1 to 1000", v)
				}
				return nil
			},
			Save: func(_ context.Context, v string) (string, error) {
				n, _ := strconv.Atoi(v) // already validated
				if err := st.Set(storageKey, n); err != nil {
					return "", err
				}
				log.Printf("[INFO] storage limit set to %d GB", n)
				return fmt.Sprintf("Storage limit set to %d GB.", n), nil
			},
		}
	},
}
