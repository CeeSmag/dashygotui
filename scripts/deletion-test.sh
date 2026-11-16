#!/usr/bin/env bash
# Clean-deletion test (RULES.md section 1): for every feature file of an addons
# package, copy the module to a scratch directory, delete that one file, and
# require that the only compile errors are in the package's map (addons.go and
# its <name>_map.go files): the lines that placed the feature. Deleting a feature
# is deleting its file and its lines in the map, and nothing else may notice.
# Proves features are self-contained.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

mapfile -t features < <(find . -type f -path '*/addons/*.go' ! -name '*_test.go' ! -name addons.go ! -name '*_map.go' | sort)
if [ "${#features[@]}" -eq 0 ]; then
	echo "deletion-test: no feature files found, nothing to check"
	exit 0
fi

scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

fail=0
for f in "${features[@]}"; do
	rm -rf "${scratch:?}/tree"
	mkdir "$scratch/tree"
	tar --exclude=.git -cf - . | tar -xf - -C "$scratch/tree"
	rm -f "$scratch/tree/$f"
	# A test file for the deleted feature would not make sense without it.
	rm -f "$scratch/tree/${f%.go}_test.go"
	pkg="$(dirname "$f")"
	# -e reports every error, not the first ten.
	if (cd "$scratch/tree" && go build -gcflags=-e ./... >/dev/null 2>"$scratch/err"); then
		echo "ok    without $f (nothing placed it)"
		continue
	fi
	# Every error must point into the map of the feature's own package.
	if grep -E '^[^#].*\.go:[0-9]+' "$scratch/err" | grep -vqE "^${pkg#./}/(addons|[a-z0-9_]+_map)\.go:"; then
		echo "FAIL  without $f: something besides the map of ${pkg#./} used it"
		sed 's/^/      /' "$scratch/err"
		fail=1
	else
		echo "ok    without $f (only the map's lines broke)"
	fi
done
exit "$fail"
