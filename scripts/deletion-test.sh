#!/usr/bin/env bash
# Clean-deletion test (RULES.md section 1): for every addon file, copy the
# module to a scratch directory, delete that one file, regenerate the addon list
# when its package has one (addons_gen.go), and require the module to still
# build. Proves addons are self-contained.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

mapfile -t addons < <(find . -type f -path '*/addons/*.go' ! -name '*_test.go' ! -name doc.go ! -name addons_gen.go | sort)
if [ "${#addons[@]}" -eq 0 ]; then
	echo "deletion-test: no addon files found, nothing to check"
	exit 0
fi

scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

cli="$scratch/dashygotui"
go build -o "$cli" ./cmd/dashygotui

fail=0
for f in "${addons[@]}"; do
	rm -rf "${scratch:?}/tree"
	mkdir "$scratch/tree"
	tar --exclude=.git -cf - . | tar -xf - -C "$scratch/tree"
	rm -f "$scratch/tree/$f"
	# A test file for the deleted addon would not make sense without it.
	rm -f "$scratch/tree/${f%.go}_test.go"
	dir="$(dirname "$scratch/tree/$f")"
	if [ -f "$dir/addons_gen.go" ] && ! "$cli" gen -q "$dir" >/dev/null 2>"$scratch/err"; then
		echo "FAIL  without $f: regenerating the list"
		sed 's/^/      /' "$scratch/err"
		fail=1
		continue
	fi
	if (cd "$scratch/tree" && go build ./... >/dev/null 2>"$scratch/err"); then
		echo "ok    without $f"
	else
		echo "FAIL  without $f"
		sed 's/^/      /' "$scratch/err"
		fail=1
	fi
done
exit "$fail"
