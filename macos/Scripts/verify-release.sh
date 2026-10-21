#!/bin/bash
set -euo pipefail

usage() {
  echo "usage: $0 <AURORA.app> [AURORA.dmg]" >&2
  exit 2
}

fail() {
  echo "verify-release: $*" >&2
  exit 1
}

manifest() {
  local root=$1
  (
    cd "$root"
    find -s . -type f -print0 | while IFS= read -r -d '' file; do
      printf 'F %s ' "$file"
      shasum -a 256 "$file" | awk '{print $1}'
    done
    find -s . -type l -print0 | while IFS= read -r -d '' link; do
      printf 'L %s -> %s\n' "$link" "$(readlink "$link")"
    done
  )
}

[ $# -ge 1 ] && [ $# -le 2 ] || usage
app=$1
dmg=${2:-}

[ -d "$app" ] || fail "app manquante: $app"
[ -x "$app/Contents/MacOS/AURORA" ] || fail "exécutable AURORA manquant"
[ -f "$app/Contents/Resources/ThirdPartyNotices.md" ] || fail "notice LGPL manquante"
[ -f "$app/Contents/Resources/LGPL-3.0.txt" ] || fail "texte LGPL-3.0 manquant"
[ -f "$app/Contents/Resources/GPL-3.0.txt" ] || fail "texte GPL-3.0 manquant"
[ -f "$app/Contents/Resources/COMPONENTS.md" ] || fail "inventaire licences manquant"
grep -q 'GNU LESSER GENERAL PUBLIC LICENSE' "$app/Contents/Resources/LGPL-3.0.txt" || fail "texte LGPL invalide"
grep -q 'GNU GENERAL PUBLIC LICENSE' "$app/Contents/Resources/GPL-3.0.txt" || fail "texte GPL invalide"
grep -q 'MPVKit | 1.0.0' "$app/Contents/Resources/COMPONENTS.md" || fail "inventaire MPVKit invalide"
[ -d "$app/Contents/Frameworks/Libmpv.framework" ] || fail "Libmpv.framework manquant"
[ -d "$app/Contents/Frameworks/Libavcodec.framework" ] || fail "FFmpeg embarqué manquant"
[ ! -e "$app/Contents/Frameworks/Libmpv-GPL.framework" ] || fail "framework GPL interdit"

codesign --verify --deep --strict "$app" 2>/dev/null || fail "signature invalide"

while IFS= read -r -d '' binary; do
  if file "$binary" | grep -q 'Mach-O'; then
    links=$(otool -L "$binary")
    if grep -Eq '/usr/local|/opt/homebrew' <<<"$links"; then
      fail "dépendance externe détectée: $binary"
    fi
  fi
done < <(find "$app/Contents/MacOS" "$app/Contents/Frameworks" -type f -print0)

version=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$app/Contents/Info.plist")
[ "$version" = "0.1.0" ] || fail "version inattendue: $version"

if [ -n "$dmg" ]; then
  [ -f "$dmg" ] || fail "DMG manquant: $dmg"
  mount_point=$(mktemp -d "${TMPDIR:-/tmp}/aurora-dmg.XXXXXX")
  source_manifest=$(mktemp "${TMPDIR:-/tmp}/aurora-source.XXXXXX")
  mounted_manifest=$(mktemp "${TMPDIR:-/tmp}/aurora-mounted.XXXXXX")
  cleanup() {
    hdiutil detach "$mount_point" -quiet >/dev/null 2>&1 || true
    rmdir "$mount_point" >/dev/null 2>&1 || true
    rm -f "$source_manifest" "$mounted_manifest"
  }
  trap cleanup EXIT
  hdiutil attach -readonly -nobrowse -mountpoint "$mount_point" "$dmg" -quiet || fail "montage DMG impossible"
  [ -d "$mount_point/AURORA.app" ] || fail "AURORA.app absente du DMG"
  [ -L "$mount_point/Applications" ] || fail "lien Applications absent du DMG"
  [ "$(readlink "$mount_point/Applications")" = "/Applications" ] || fail "lien Applications invalide"
  manifest "$app" > "$source_manifest"
  manifest "$mount_point/AURORA.app" > "$mounted_manifest"
  cmp -s "$source_manifest" "$mounted_manifest" || fail "l’app du DMG diffère de la source"
  codesign --verify --deep --strict "$mount_point/AURORA.app" 2>/dev/null || fail "signature DMG invalide"
fi

echo "verify-release: OK — AURORA $version"
