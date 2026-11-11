package dashygotui

import "sort"

// Outline describes what a dashboard contains. Tools and tests use it to walk a
// dashboard without knowing its application; see dashtest.Audit.
type Outline struct {
	// Pages are the page IDs in tab order.
	Pages []string
	// Views are the registered view IDs, sorted.
	Views []string
	// Options are the options of the pages' sections, in page order.
	Options []OutlineOption
}

// OutlineOption is one option of an [Outline].
type OutlineOption struct {
	// Key, Number, Label and Hotkey are those of the [Option].
	Key    string
	Number int
	Label  string
	Hotkey string
	// Page is the ID of the page the option's section is on.
	Page string
}

// Outline returns what the dashboard contains.
func (d *Dashboard[S]) Outline() Outline {
	var out Outline
	for _, p := range d.pages {
		out.Pages = append(out.Pages, p.page.ID)
		for _, panel := range p.panels {
			for _, o := range panel.opts {
				out.Options = append(out.Options, OutlineOption{
					Key: o.Key, Number: o.Number, Label: o.Label, Hotkey: o.Hotkey, Page: p.page.ID,
				})
			}
		}
	}
	for id := range d.views {
		out.Views = append(out.Views, id)
	}
	sort.Strings(out.Views)
	return out
}
