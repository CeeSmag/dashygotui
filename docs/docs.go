// Package docs embeds the design documents of dashygotui so that the
// dashygotui command can print the ones that match the library version an
// application uses.
package docs

import "embed"

// FS holds the markdown documents of this directory.
//
//go:embed *.md
var FS embed.FS
