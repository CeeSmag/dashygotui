package dashygotui

import "sort"

// Outline describes what a dashboard contains. Tools and tests use it to walk a
// dashboard without knowing its application; see dashtest.Audit.
type Outline struct {
	// Pages are the page IDs in tab order; a page with no ID is named by its
	// Title.
	Pages []string
	// Views are the IDs of the views and sub-pages, sorted.
	Views []string
	// Options are the rows of the tab pages, in page order.
	Options []OutlineOption
}

// OutlineOption is one row of an [Outline].
type OutlineOption struct {
	// Number is the row's number; Label and Hotkey are those of the [Option].
	Number int
	Label  string
	Hotkey string
	// Page is the ID of the page the row is on, or its Title when it has none.
	Page string
}

// Outline returns what the dashboard contains.
func (d *Dashboard[S]) Outline() Outline {
	var out Outline
	for _, p := range d.pages {
		name := p.page.ID
		if name == "" {
			name = p.page.Title
		}
		out.Pages = append(out.Pages, name)
		for _, panel := range p.panels {
			for _, o := range panel.opts {
				out.Options = append(out.Options, OutlineOption{
					Number: o.number, Label: o.Label, Hotkey: o.Hotkey, Page: name,
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
