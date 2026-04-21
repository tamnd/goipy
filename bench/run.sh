#!/usr/bin/env bash
# Run every bench/cases/*.py under both python3.14 and goipy, three times
# each, and produce a Markdown comparison table. CHECKSUM from each run is
# verified byte-for-byte between the two interpreters; TIME_MS is the
# in-process wall time around the benchmark's main() so we compare
# workloads without interpreter startup noise.
set -euo pipefail

cd "$(dirname "$0")/.."

PY=${PYTHON:-python3.14}
RUNS=${RUNS:-3}

BIN="bench/.bin/goipy"
go build -o "$BIN" ./cmd/goipy

cases=$(ls bench/cases/*.py | sort)

printf '| Case | CPython 3.14 (ms) | goipy (ms) | ratio |\n'
printf '|---|---:|---:|---:|\n'

median_of_three() {
  # Sort three numbers, echo the middle one. No bc, so we leave them as
  # strings and sort lexically with -g (general numeric).
  printf '%s\n%s\n%s\n' "$1" "$2" "$3" | sort -g | sed -n '2p'
}

run_one() {
  local cmd=$1 pyc=$2
  # Extract TIME_MS line from stdout; CHECKSUM line goes to stdout too
  # so callers can diff.
  local out
  out=$("$cmd" "$pyc")
  echo "$out"
}

total_cases=0
matched=0

tmp_results=$(mktemp)
trap 'rm -f "$tmp_results"' EXIT

for src in $cases; do
  base=$(basename "$src" .py)
  pyc="bench/cases/${base}.pyc"
  "$PY" -c "import py_compile; py_compile.compile('$src', cfile='$pyc', doraise=True)"

  # Capture CPython output once for checksum comparison.
  py_out=$("$PY" "$pyc")
  py_checksum=$(grep '^CHECKSUM:' <<<"$py_out" | head -1)

  gp_out=$("$BIN" "$pyc" 2>/dev/null || true)
  gp_checksum=$(grep '^CHECKSUM:' <<<"$gp_out" | head -1)

  total_cases=$((total_cases + 1))
  if [[ "$py_checksum" == "$gp_checksum" && -n "$py_checksum" ]]; then
    matched=$((matched + 1))
    match_mark=""
  else
    match_mark=" ⚠ mismatch"
  fi

  # Collect RUNS samples per interpreter.
  py_times=()
  gp_times=()
  for _ in $(seq 1 "$RUNS"); do
    t=$("$PY" "$pyc" | grep '^TIME_MS:' | sed 's/^TIME_MS://')
    py_times+=("$t")
  done
  for _ in $(seq 1 "$RUNS"); do
    t=$("$BIN" "$pyc" 2>/dev/null | grep '^TIME_MS:' | sed 's/^TIME_MS://')
    # If goipy failed or the case mismatched, fall back to an empty row.
    gp_times+=("${t:-NaN}")
  done

  py_med=$(median_of_three "${py_times[@]}")
  gp_med=$(median_of_three "${gp_times[@]}")

  if [[ "$gp_med" == "NaN" || -z "$gp_med" ]]; then
    ratio="—"
  else
    ratio=$(awk -v g="$gp_med" -v p="$py_med" 'BEGIN{if(p>0){printf "%.1fx", g/p}else{print "—"}}')
  fi

  printf '| %s | %s | %s | %s%s |\n' "$base" "$py_med" "$gp_med" "$ratio" "$match_mark" | tee -a "$tmp_results"
done

printf '\n%d/%d cases matched CHECKSUM.\n' "$matched" "$total_cases" >&2
