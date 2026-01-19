#!/usr/bin/env bash
set -euo pipefail

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "not a git repository" >&2
  exit 1
fi

date=$(TZ=UTC git log -1 --date=format:%y.%m.%d --format=%cd)
if [ -z "$date" ]; then
  echo "unable to determine date" >&2
  exit 1
fi

rev=$(TZ=UTC git log --date=format:%y.%m.%d --format=%cd | awk -v date="$date" '$0==date {count++} END {print count+0}')
if [ "$rev" -le 0 ]; then
  rev=1
fi

printf "%s-%s" "$date" "$rev"
