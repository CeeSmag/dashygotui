#!/usr/bin/env bash
# Consumer simulation: build a throw-away application on this working tree, the
# way a real application would use the library, and prove the guardrails work.
#
#   1. a copy of the demo app, depending on this tree through a replace directive
#   2. `dashygotui init`, then `dashygotui new` for every feature kind, each placed
#      in the map
#   3. the app must build and `dashygotui check` must pass
#   4. a deliberately bad feature per rule (A1..A9, A12, A13, G1) must make `check`
#      fail with exactly that rule
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT

cli="$scratch/dashygotui"
(cd "$root" && go build -o "$cli" ./cmd/dashygotui)

app="$scratch/app"
mkdir -p "$app"
cp -R "$root/examples/demo/." "$app/"
rm -f "$app"/*_test.go
find "$app" -name '*.go' -exec sed -i 's#github.com/CeeSmag/dashygotui/examples/demo#example.com/consumer#g' {} +
cat >"$app/go.mod" <<MOD
module example.com/consumer

go 1.27

require github.com/CeeSmag/dashygotui v0.4.0

tool github.com/CeeSmag/dashygotui/cmd/dashygotui

replace github.com/CeeSmag/dashygotui => $root
MOD
(cd "$app" && go mod tidy >/dev/null 2>"$scratch/err" || { cat "$scratch/err"; exit 1; })

# a git baseline, for `check -changed`
# (inside the scratch application: never in the real repository)
(cd "$app" && git init -q && git add -A && git -c user.name=t -c user.email=t@example.com commit -q -m base)

fail=0
step() { echo "ok    $1"; }
bad()  { echo "FAIL  $1"; fail=1; }

cd "$app"

# 2. init and new
"$cli" init >/dev/null
if grep -q 'dashygotui@v0.4.0 rules' AGENTS.md && [ -f CLAUDE.md ] && [ -f GAPS.md ]; then step "init writes AGENTS.md, CLAUDE.md, GAPS.md"; else bad "init"; fi

# the Claude Code skill and hook
"$cli" init -claude >/dev/null
if [ -f .claude/skills/dashygotui/SKILL.md ] && grep -q 'cmd/dashygotui@v0.4.0 check -q' .claude/settings.json && python3 -c 'import json;json.load(open(".claude/settings.json"))'; then step "init -claude writes a skill and a valid hook"; else bad "init -claude"; fi

for kind in opt widget view page modal shortcut hook map; do
	if "$cli" new "$kind" scaffold >"$scratch/out" 2>"$scratch/err" && grep -q 'Place it in .*addons.go' "$scratch/out"; then
		step "new $kind says where it goes"
	else
		bad "new $kind: $(cat "$scratch/err" "$scratch/out")"
	fi
done
for kind in opt widget view page modal shortcut hook map; do
	python3 "$root/scripts/place-in-map.py" dashboard/addons/addons.go "$kind" scaffold
done
gofmt -w dashboard/addons/addons.go

# 3. the app builds and passes the check
if go build ./... 2>"$scratch/err"; then step "the application builds"; else bad "build: $(cat "$scratch/err")"; fi
if "$cli" check . >"$scratch/out" 2>&1; then step "check passes on the scaffolded application"; else bad "check: $(cat "$scratch/out")"; fi

# one-file rule: new feature files and their lines in the map
if "$cli" check -changed -strict . >"$scratch/out" 2>&1; then step "check -changed passes when features and the map changed"; else bad "check -changed: $(cat "$scratch/out")"; fi
# editing the foundation file while adding an addon is flagged
echo '// edited' >>dashboard/dashboard.go
if "$cli" check -changed -strict . >"$scratch/out" 2>&1; then bad "C1: edit of dashboard.go while adding addons passed -strict"; elif grep -q ': C1:' "$scratch/out"; then step "C1 is caught (-strict)"; else bad "C1: failed without reporting it: $(cat "$scratch/out")"; fi
if "$cli" check -changed . >"$scratch/out" 2>&1 && grep -q 'advice' "$scratch/out"; then step "C1 is advice without -strict"; else bad "C1 advice: $(cat "$scratch/out")"; fi
git checkout -q -- dashboard/dashboard.go

# 4. each rule must trip on its own bad addon
dir=dashboard/addons
expect() { # rule, description, then files written by the caller
	local rule="$1"
	if "$cli" check . >"$scratch/out" 2>&1; then
		bad "$rule: check passed but should have failed"
	elif grep -q ": $rule:" "$scratch/out"; then
		step "$rule is caught"
	else
		bad "$rule: check failed without reporting it: $(cat "$scratch/out")"
	fi
}
reset() { rm -f "$dir"/bad*_opt.go "$dir"/helpers.go "$dir"/../bad_gap.go; }

printf 'package addons\n' >"$dir/helpers.go";                                                              expect A1; reset
printf 'package addons\nfunc helperX() int { return 1 }\n' >"$dir/bad_a_opt.go"
printf 'package addons\nfunc helperY() int { return helperX() }\n' >"$dir/bad_b_opt.go";                    expect A2; reset
# A2: from the dashboard package a feature uses only types
printf 'package addons\nimport "example.com/consumer/dashboard"\nfunc badLen() int { return dashboard.Logs.Len() }\n' >"$dir/bad_opt.go"; expect A2; reset
printf 'package addons\nvar counter int\n' >"$dir/bad_opt.go";                                             expect A3; reset
printf 'package addons\nimport "example.com/consumer/dashboard"\nvar Bad = dashboard.Option{}\n' >"$dir/bad_opt.go"; expect A3; reset
printf 'package addons\nimport "time"\nfunc init() { time.Sleep(1) }\n' >"$dir/bad_opt.go";                expect A4; reset
printf 'package addons\nvar _ = "#38BDF8"\n' >"$dir/bad_opt.go";                                           expect A5; reset
printf 'package addons\nvar _ = "done ✓"\n' >"$dir/bad_opt.go";                                            expect A6; reset
printf 'package addons\nimport tea "charm.land/bubbletea/v2"\nvar _ = tea.Quit\n' >"$dir/bad_opt.go";      expect A7; reset
printf 'package addons\nimport "github.com/CeeSmag/dashygotui/modal"\ntype m struct{ modal.Edit[int] }\n' >"$dir/bad_opt.go"; expect A8; reset
cat >"$dir/bad_opt.go" <<'GO'
package addons

import (
	tea "charm.land/bubbletea/v2"

	"example.com/consumer/dashboard"
)

var badOpt = dashboard.Option{Label: "Bad",
	Run: func(c *dashboard.Ctx) tea.Cmd { c.Flash("x"); return nil }}
GO
expect A9; reset
# A13: a feature that is not placed in the map
printf 'package addons\nimport "example.com/consumer/dashboard"\nvar lateOpt = dashboard.Option{Label: "Late", Do: func(*dashboard.Ctx) {}}\n' >"$dir/bad_opt.go"; expect A13; reset
printf 'package addons\nfunc init() {}\n' >"$dir/bad_opt.go";                                            expect A12; reset
printf 'package dashboard\n// dashygotui:gap\nfunc badGap() {}\n' >dashboard/bad_gap.go;                    expect G1; rm -f dashboard/bad_gap.go
# the original problem: a wrapper around a library modal in the dashboard package, not in an addon
printf 'package dashboard\nimport "github.com/CeeSmag/dashygotui/modal"\ntype SettingEdit struct{ *modal.Edit[State] }\n' >dashboard/bad_wrap.go; expect A8; rm -f dashboard/bad_wrap.go

# the escape hatch works and is reported
printf 'package addons\nimport "time"\nfunc pause() {\n\ttime.Sleep(1) //dashygotui:allow A4 a pause the demo needs\n}\n' >"$dir/bad_opt.go"
if "$cli" check . >"$scratch/out" 2>&1 && grep -q "a pause the demo needs" "$scratch/out"; then step "allow comment excuses and is listed"; else bad "allow: $(cat "$scratch/out")"; fi
reset

exit "$fail"
