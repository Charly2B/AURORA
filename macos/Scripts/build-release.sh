#!/bin/bash
set -euo pipefail

script_dir=$(cd "$(dirname "$0")" && pwd)
project_root=$(cd "$script_dir/.." && pwd)
build_root="$project_root/Build"
derived_data="$build_root/DerivedData"
app="$derived_data/Build/Products/Release/AURORA.app"

mkdir -p "$build_root"
cd "$project_root"

echo "build-release: génération Xcode"
xcodegen generate

echo "build-release: compilation Release propre"
xcodebuild \
  -project AURORA.xcodeproj \
  -scheme AURORA \
  -configuration Release \
  -destination 'platform=macOS' \
  -derivedDataPath "$derived_data" \
  clean build

[ -d "$app" ] || {
  echo "build-release: AURORA.app introuvable" >&2
  exit 1
}

codesign --force --deep --sign - "$app"
"$script_dir/verify-release.sh" "$app"

echo "build-release: $app"
