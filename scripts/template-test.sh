#!/usr/bin/env bash
# Template test: every file in templates/ must compile and validate cleanly when
# written into the demo's addons directory by `dashygotui new` and placed in the
# demo's map. Keeps the starter files honest as the library evolves.
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
	kind="$(basename "$t" .go.example)" # opt, widget, view, page, modal, shortcut, hook
	rm -rf "${scratch:?}/tree"
	mkdir "$scratch/tree"
	tar --exclude=.git -cf - . | tar -xf - -C "$scratch/tree"
	# Build, vet, and run the demo's own tests: that proves the template places
	# cleanly beside the existing features (numbers, hotkeys, IDs) and the check
	# finds nothing in it.
	if (cd "$scratch/tree" &&
		"$cli" new -dir examples/demo/dashboard/addons "$kind" from_template >/dev/null &&
		python3 scripts/place-in-map.py examples/demo/dashboard/addons/addons.go "$kind" fromTemplate &&
		gofmt -w examples/demo/dashboard/addons/addons.go &&
		go build ./... && go vet ./examples/... &&
		go test -count=1 -run 'TestTheAddonsAreValid|TestControlPageShowsEveryOption|TestDemoPassesTheAudit' ./examples/demo/ &&
		"$cli" check -q examples/demo) >"$scratch/out" 2>&1; then
		echo "ok    $t"
	else
		echo "FAIL  $t"
		sed 's/^/      /' "$scratch/out"
		fail=1
	fi
done
exit "$fail"
