// Package addons holds the showcase's features, one per file. Each top-level var in
// an addon file is an addon; All, in the generated addons_gen.go, lists them.
// After adding, renaming or deleting an addon file, run go generate ./...
package addons

//go:generate go run github.com/CeeSmag/dashygotui/cmd/dashygotui gen -state github.com/CeeSmag/dashygotui/examples/showcase/dashboard.State
