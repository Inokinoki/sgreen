#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SGREEN_BIN="${SGREEN_BIN:-$ROOT_DIR/build/sgreen}"
SCREEN_BIN="${SCREEN_BIN:-/usr/bin/screen}"
REPORT_PATH="${REPORT_PATH:-$ROOT_DIR/test/behavior/gnu_screen_comparison_results.md}"
TIMEOUT_SEC="${TIMEOUT_SEC:-8}"

if [[ ! -x "$SGREEN_BIN" ]]; then
  echo "sgreen binary not found or not executable: $SGREEN_BIN" >&2
  exit 1
fi

if [[ ! -x "$SCREEN_BIN" ]]; then
  echo "screen binary not found or not executable: $SCREEN_BIN" >&2
  exit 1
fi

TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMP_ROOT"' EXIT

run_cmd() {
  local home_dir="$1"
  local screen_dir="$2"
  local sty_value="$3"
  local cmd="$4"
  local out_file="$5"
  local code_file="$6"
  local term_value="${TERM:-xterm}"

  (
    set +e
    env HOME="$home_dir" SCREENDIR="$screen_dir" STY="$sty_value" TERM="$term_value" \
      perl -e 'alarm shift @ARGV; exec @ARGV' "$TIMEOUT_SEC" /bin/zsh -lc "$cmd" \
      >"$out_file" 2>&1
    echo "$?" >"$code_file"
  )
}

escape_md() {
  local s="$1"
  s="${s//|/\\|}"
  s="${s//\`/\\\`}"
  printf "%s" "$s"
}

first_line() {
  local in_file="$1"
  local line
  line="$(head -n 1 "$in_file" || true)"
  if [[ -z "$line" ]]; then
    line="(no output)"
  fi
  printf "%s" "$line"
}

emit_case() {
  local id="$1"
  local desc="$2"
  local sty="$3"
  local sgreen_cmd="$4"
  local screen_cmd="$5"

  local case_dir
  case_dir="$(mktemp -d "$TMP_ROOT/case.XXXXXX")"
  local s_home="$case_dir/sgreen-home"
  local g_home="$case_dir/screen-home"
  local g_sock="$case_dir/screen-sock"
  mkdir -p "$s_home" "$g_home" "$g_sock"
  chmod 700 "$g_sock"

  local s_out="$case_dir/sgreen.out"
  local s_code="$case_dir/sgreen.code"
  local g_out="$case_dir/screen.out"
  local g_code="$case_dir/screen.code"

  run_cmd "$s_home" "$g_sock" "$sty" "$sgreen_cmd" "$s_out" "$s_code"
  run_cmd "$g_home" "$g_sock" "$sty" "$screen_cmd" "$g_out" "$g_code"

  local scode gcode exit_cmp behavior_cmp
  scode="$(cat "$s_code")"
  gcode="$(cat "$g_code")"

  if [[ "$scode" == "$gcode" ]]; then
    exit_cmp="same"
  else
    exit_cmp="different"
  fi

  if { [[ "$scode" == "0" ]] && [[ "$gcode" == "0" ]]; } || { [[ "$scode" != "0" ]] && [[ "$gcode" != "0" ]]; }; then
    behavior_cmp="same-class"
  else
    behavior_cmp="different-class"
  fi

  local s_head g_head
  s_head="$(escape_md "$(first_line "$s_out")")"
  g_head="$(escape_md "$(first_line "$g_out")")"

  printf '| %s | %s | `%s` | `%s` | %s | %s | %s | %s |\n' \
    "$id" \
    "$desc" \
    "$(escape_md "$sgreen_cmd")" \
    "$(escape_md "$screen_cmd")" \
    "$scode" \
    "$gcode" \
    "$exit_cmp / $behavior_cmp" \
    "sgreen: $s_head<br>screen: $g_head" \
    >>"$REPORT_PATH"
}

cat >"$REPORT_PATH" <<EOF
# GNU screen vs sgreen: CLI Use-Case Comparison

Generated on: $(date -u "+%Y-%m-%d %H:%M:%S UTC")

Environment:
- sgreen: \`$SGREEN_BIN\`
- screen: \`$SCREEN_BIN\`
- timeout per case: ${TIMEOUT_SEC}s
- isolation: each case runs in a fresh temp HOME and SCREENDIR

Legend:
- \`exit\`: exact exit code comparison
- \`same-class\`: both success (0) or both failure (non-zero)

| ID | Use Case | sgreen Command | screen Command | sgreen exit | screen exit | Comparison | First output line |
|---|---|---|---|---:|---:|---|---|
EOF

# Core non-interactive compatibility use cases
emit_case "UC01" "Version output" "" "$SGREEN_BIN -v" "$SCREEN_BIN -v"
emit_case "UC02" "Help short flag (-h)" "" "$SGREEN_BIN -h" "$SCREEN_BIN -h"
emit_case "UC03" "Help long flag (-help)" "" "$SGREEN_BIN -help" "$SCREEN_BIN -help"
emit_case "UC04" "List sessions (-ls) with no sessions" "" "$SGREEN_BIN -ls" "$SCREEN_BIN -ls"
emit_case "UC05" "List sessions (-list) with no sessions" "" "$SGREEN_BIN -list" "$SCREEN_BIN -list"
emit_case "UC06" "Reattach with no sessions (-r)" "" "$SGREEN_BIN -r" "$SCREEN_BIN -r"
emit_case "UC07" "Reattach missing named session (-r name)" "" "$SGREEN_BIN -r nosuchsession123" "$SCREEN_BIN -r nosuchsession123"
emit_case "UC08" "Wipe dead sessions (-wipe)" "" "$SGREEN_BIN -wipe" "$SCREEN_BIN -wipe"
emit_case "UC09" "Detach with no sessions (-d)" "" "$SGREEN_BIN -d" "$SCREEN_BIN -d"
emit_case "UC10" "Power detach with no sessions (-D)" "" "$SGREEN_BIN -D" "$SCREEN_BIN -D"
emit_case "UC11" "Send command with no sessions (-X stuff x)" "" "$SGREEN_BIN -X stuff x" "$SCREEN_BIN -X stuff x"
emit_case "UC12" "Unknown flag handling" "" "$SGREEN_BIN -unknown" "$SCREEN_BIN -unknown"
emit_case "UC13" "Quiet list (-q -ls)" "" "$SGREEN_BIN -q -ls" "$SCREEN_BIN -q -ls"
emit_case "UC14" "Ignore STY (-m -ls) with STY set" "12345.pts-0.host" "$SGREEN_BIN -m -ls" "$SCREEN_BIN -m -ls"
emit_case "UC15" "Session name with list (-S myname -ls)" "" "$SGREEN_BIN -S myname -ls" "$SCREEN_BIN -S myname -ls"
emit_case "UC16" "Missing config path with list (-c /nonexistent -ls)" "" "$SGREEN_BIN -c /nonexistent/screenrc -ls" "$SCREEN_BIN -c /nonexistent/screenrc -ls"
emit_case "UC17" "Multiuser attach with no sessions (-x)" "" "$SGREEN_BIN -x" "$SCREEN_BIN -x"
emit_case "UC18" "Detach+reattach with no sessions (-d -r)" "" "$SGREEN_BIN -d -r" "$SCREEN_BIN -d -r"
emit_case "UC19" "Reattach-or-create with no tty (-R)" "" "$SGREEN_BIN -R -S rtest" "$SCREEN_BIN -R -S rtest"
emit_case "UC20" "RR reattach-or-create with no tty (-RR)" "" "$SGREEN_BIN -RR -S rrtest" "$SCREEN_BIN -RR -S rrtest"

# Option parsing and screen-compatibility deltas
emit_case "UC21" "TERM override parsing (-T xterm -ls)" "" "$SGREEN_BIN -T xterm -ls" "$SCREEN_BIN -T xterm -ls"
emit_case "UC22" "UTF-8 mode parsing (-U -ls)" "" "$SGREEN_BIN -U -ls" "$SCREEN_BIN -U -ls"
emit_case "UC23" "All capabilities flag (-a -ls)" "" "$SGREEN_BIN -a -ls" "$SCREEN_BIN -a -ls"
emit_case "UC24" "Optimal output flag (-O -ls)" "" "$SGREEN_BIN -O -ls" "$SCREEN_BIN -O -ls"
emit_case "UC25" "Preselect window flag (-p 1 -ls)" "" "$SGREEN_BIN -p 1 -ls" "$SCREEN_BIN -p 1 -ls"
emit_case "UC26" "Flow control off flag (-fn -ls)" "" "$SGREEN_BIN -fn -ls" "$SCREEN_BIN -fn -ls"
emit_case "UC27" "Flow control auto flag (-fa -ls)" "" "$SGREEN_BIN -fa -ls" "$SCREEN_BIN -fa -ls"
emit_case "UC28" "Interrupt flag (-i -ls)" "" "$SGREEN_BIN -i -ls" "$SCREEN_BIN -i -ls"
emit_case "UC29" "Scrollback history flag semantics (-h 100 -ls)" "" "$SGREEN_BIN -h 100 -ls" "$SCREEN_BIN -h 100 -ls"
emit_case "UC30" "Sgreen scrollback flag vs screen (-H 100 -ls)" "" "$SGREEN_BIN -H 100 -ls" "$SCREEN_BIN -H 100 -ls"

# Detached start behavior (GNU screen feature)
emit_case "UC31" "Detached start via -dmS" "" "$SGREEN_BIN -dmS demo /bin/sh -c 'sleep 1'" "$SCREEN_BIN -dmS demo /bin/sh -c 'sleep 1'"
emit_case "UC32" "Detached start then list after command exits" "" "$SGREEN_BIN -dmS demo /bin/sh -c 'sleep 1'; sleep 2; $SGREEN_BIN -ls" "$SCREEN_BIN -dmS demo /bin/sh -c 'sleep 1'; sleep 2; $SCREEN_BIN -ls"

# PTY-backed cases (using `script` to allocate a terminal)
emit_case "UC33" "PTY list with no sessions (-ls)" "" "script -q /dev/null $SGREEN_BIN -ls" "script -q /dev/null $SCREEN_BIN -ls"
emit_case "UC34" "PTY reattach with no sessions (-r)" "" "script -q /dev/null $SGREEN_BIN -r" "script -q /dev/null $SCREEN_BIN -r"
emit_case "UC35" "PTY create-and-exit named session (-S name cmd)" "" "script -q /dev/null $SGREEN_BIN -S ptydemo /bin/sh -c 'exit 0'" "script -q /dev/null $SCREEN_BIN -S ptydemo /bin/sh -c 'exit 0'"
emit_case "UC36" "PTY reattach-or-create with auto-exit command (-R)" "" "script -q /dev/null $SGREEN_BIN -R -S ptyr /bin/sh -c 'exit 0'" "script -q /dev/null $SCREEN_BIN -R -S ptyr /bin/sh -c 'exit 0'"
emit_case "UC37" "PTY power detach with no sessions (-D)" "" "script -q /dev/null $SGREEN_BIN -D" "script -q /dev/null $SCREEN_BIN -D"
emit_case "UC38" "PTY send command with no sessions (-X)" "" "script -q /dev/null $SGREEN_BIN -X stuff x" "script -q /dev/null $SCREEN_BIN -X stuff x"
emit_case "UC39" "PTY power detach with missing named session (-D name)" "" "script -q /dev/null $SGREEN_BIN -D nosuch" "script -q /dev/null $SCREEN_BIN -D nosuch"

echo "Wrote report: $REPORT_PATH"
