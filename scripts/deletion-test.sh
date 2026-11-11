#!/usr/bin/env bash
# Clean-deletion test (RULES.md section 1): for every addon file, copy the
# module to a scratch directory, delete that one file, and require the module
# to still build. Proves addons are self-contained.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

mapfile -t addons < <(find . -type f -path '*/addons/*.go' ! -name '*_test.go' | sort)
if [ "${#addons[@]}" -eq 0 ]; then
	echo "deletion-test: no addon files found, nothing to check"
	exit 0
fi

scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

fail=0
for f in "${addons[@]}"; do
	rm -rf "${scratch:?}/tree"
	mkdir "$scratch/tree"
	tar --exclude=.git -cf - . | tar -xf - -C "$scratch/tree"
	rm -f "$scratch/tree/$f"
	# A test file for the deleted addon would not make sense without it.
	rm -f "$scratch/tree/${f%.go}_test.go"
	if (cd "$scratch/tree" && go build ./... >/dev/null 2>"$scratch/err"); then
		echo "ok    without $f"
	else
		echo "FAIL  without $f"
		sed 's/^/      /' "$scratch/err"
		fail=1
	fi
done
exit "$fail"
