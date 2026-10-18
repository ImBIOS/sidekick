#!/usr/bin/env bash
# Design lint: machine-checkable half of DESIGN.md. Fails CI on violations.
# Run from web/: bun scripts/design-lint.sh (or bash).
set -eu
cd "$(dirname "$0")/../.."
fail=0
ban() {
  matches=$(grep -rniE "$1" --include="*.tsx" --include="*.ts" --include="*.css" --include="*.md" web/apps/web/src web/packages/ui/src skill 2>/dev/null | grep -v node_modules | head -5 || true)
  if [ -n "$matches" ]; then
    echo "$matches" >&2
    echo "DESIGN VIOLATION: $2"
    fail=1
  fi
}
ban "from-purple|via-purple|to-purple|from-blue|to-blue|via-blue|from-indigo|to-indigo" "purple/blue gradients (rule 1)"
ban "lorem ipsum|placeholder\.com|TODO|FIXME" "placeholder copy or TODOs (rules 2,5)"
# __root.tsx is the single allowed home for the font link (re-check without it).
found=$(grep -rniE 'rel="stylesheet".*fonts\.googleapis' --include="*.tsx" web/apps/web/src skill 2>/dev/null | grep -v node_modules | grep -v "__root.tsx" | head -5 || true)
if [ -n "$found" ]; then
  echo "$found" >&2
  echo "DESIGN VIOLATION: font link outside __root.tsx (rule 4)"
  fail=1
fi
ban "from \"motion|from 'motion|framer-motion" "animation runtime; use Reveal/CSS (rule 6)"
if [ "$fail" -ne 0 ]; then exit 1; fi
echo "design lint ok"
