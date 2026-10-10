#!/usr/bin/env bash
# Prints the CHANGELOG.md section for a version, without its heading, for use
# as release notes. Fails when the section is missing or empty, so a version
# cannot be released without its changelog entry.
#
#   scripts/release-notes.sh 1.2.0 > notes.md
set -euo pipefail

version="${1#v}"
changelog="${2:-CHANGELOG.md}"

if [[ -z "$version" ]]; then
  echo "usage: $0 <version> [changelog]" >&2
  exit 2
fi

notes="$(awk -v version="$version" '
  /^## \[/ {
    if (found) exit
    if (index($0, "## [" version "]") == 1) { found = 1; next }
  }
  /^\[[^]]+\]: / { if (found) exit }
  found { print }
' "$changelog")"

# Drop leading and trailing blank lines.
notes="$(printf '%s\n' "$notes" | sed -e '/./,$!d' | sed -e ':a' -e '/^\n*$/{$d;N;ba' -e '}')"

if [[ -z "$notes" ]]; then
  echo "error: $changelog has no section for $version" >&2
  exit 1
fi

# GitHub renders every line break in a release body, so join the lines of
# each paragraph and list item. Headings, tables, code blocks and blank lines
# keep their own lines.
printf '%s\n' "$notes" | awk '
  function flush() { if (buf != "") { print buf; buf = "" } }
  /^```/ { flush(); print; fence = !fence; next }
  fence { print; next }
  /^[[:space:]]*$/ { flush(); print; next }
  /^(#|\||[-*+] |[0-9]+\. )/ { flush(); buf = $0; next }
  {
    line = $0
    sub(/^[[:space:]]+/, "", line)
    buf = (buf == "") ? line : buf " " line
  }
  END { flush() }
'
