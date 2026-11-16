#!/usr/bin/env bash
# Template test: every file in templates/ must compile and validate cleanly when
# dropped into the demo's addons directory with only its import path changed and
# the addon list regenerated.
# Keeps the starter files honest as the library evolves.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

mapfile -t templates < <(find templates -type f -name '*.go.example' | sort)
if [ "${#templates[@]}" -eq 0 ]; then
	echo "template-test: no templates found" >&2
	exit 1
fi

scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

cli="$scratch/dashygotui"
go build -o "$cli" ./cmd/dashygotui

fail=0
for t in "${templates[@]}"; do
	kind="$(basename "$t" .go.example)"        # opt, widget, view, modal
	rm -rf "${scratch:?}/tree"
	mkdir "$scratch/tree"
	tar --exclude=.git -cf - . | tar -xf - -C "$scratch/tree"
	dest="$scratch/tree/examples/demo/dashboard/addons/${kind}_from_template.go"
	sed 's#example.com/yourapp/dashboard#github.com/CeeSmag/dashygotui/examples/demo/dashboard#' "$t" >"$dest"
	# Build, vet, and run the demo's own tests: that proves the template registers
	# without colliding with the existing addons (numbers, keys, IDs).
	if (cd "$scratch/tree" && "$cli" gen -q examples/demo/dashboard/addons && go build ./... && go vet ./examples/... &&
		go test -count=1 -run 'TestTheAddonsAreValid|TestControlPageShowsEveryOption' ./examples/demo/ &&
		"$cli" check -q examples/demo) >"$scratch/out" 2>&1; then
		echo "ok    $t"
	else
		echo "FAIL  $t"
		sed 's/^/      /' "$scratch/out"
		fail=1
	fi
done
exit "$fail"
