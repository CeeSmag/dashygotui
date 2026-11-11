#!/usr/bin/env bash
# API/changelog check (RULES.md section 12.10): when the exported API snapshot
# changed, CHANGELOG.md must have changed with it. Compares the working tree with
# $API_BASE (default HEAD; CI sets it to the pull request's base).
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

if ! git rev-parse --git-dir >/dev/null 2>&1; then
	echo "api-check: not a git checkout, skipped"
	exit 0
fi
base="${API_BASE:-HEAD}"
if ! git rev-parse --verify --quiet "$base" >/dev/null; then
	echo "api-check: base $base not found, skipped"
	exit 0
fi

if git diff --quiet "$base" -- api/; then
	echo "ok    the exported API is unchanged since $base"
	exit 0
fi
if git diff --quiet "$base" -- CHANGELOG.md; then
	echo "FAIL  api/ changed since $base but CHANGELOG.md did not: record the API change (and call out a breaking one)" >&2
	exit 1
fi
echo "ok    the exported API changed and CHANGELOG.md records it"
