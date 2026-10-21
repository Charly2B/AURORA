#!/bin/bash
set -euo pipefail

usage() {
  echo "usage: $0 <AURORA.app> <AURORA.dmg>" >&2
  exit 2
}

[ $# -eq 2 ] || usage
app=$1
dmg=$2

[ -d "$app" ] || {
  echo "create-dmg: app manquante: $app" >&2
  exit 1
}

script_dir=$(cd "$(dirname "$0")" && pwd)
output_dir=$(dirname "$dmg")
temp_base=${TMPDIR:-/tmp}
temp_base=${temp_base%/}

mkdir -p "$output_dir"
stage=$(mktemp -d "$temp_base/aurora-dmg-stage.XXXXXX")
cleanup() {
  case "$stage" in
    "$temp_base"/aurora-dmg-stage.*) rm -rf -- "$stage" ;;
  esac
}
trap cleanup EXIT

ditto "$app" "$stage/AURORA.app"
ln -s /Applications "$stage/Applications"

if [ -e "$dmg" ]; then
  timestamp=$(date '+%Y%m%d-%H%M%S')
  previous="${dmg%.dmg}-previous-$timestamp.dmg"
  [ ! -e "$previous" ] || previous="${dmg%.dmg}-previous-$timestamp-$$.dmg"
  mv "$dmg" "$previous"
  echo "create-dmg: ancien DMG conservé: $previous"
fi

hdiutil create \
  -volname AURORA \
  -srcfolder "$stage" \
  -format UDZO \
  -imagekey zlib-level=9 \
  "$dmg"

"$script_dir/verify-release.sh" "$app" "$dmg"
echo "create-dmg: $dmg"
