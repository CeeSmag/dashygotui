// Package templates embeds the addon templates so that the dashygotui command
// can write a new addon from one.
package templates

import "embed"

// FS holds the .example template files of this directory.
//
//go:embed *.example
var FS embed.FS
