#!/usr/bin/env bash
# Regenerate .pyc and .expected.txt for every .py fixture in this directory.
# Requires python3.14 on PATH.
set -euo pipefail
cd "$(dirname "$0")"
PY=${PYTHON:-python3.14}
for src in *.py; do
  base=${src%.py}
  $PY -c "import py_compile; py_compile.compile('$src', cfile='$base.pyc', doraise=True)"
  $PY "$src" > "$base.expected.txt"
done
echo "generated $(ls *.pyc | wc -l) fixtures"
