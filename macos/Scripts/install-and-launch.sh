#!/bin/bash
set -euo pipefail

usage() {
  echo "usage: $0 <AURORA.app>" >&2
  exit 2
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

[ $# -eq 1 ] || usage
source_app=$1
[ -d "$source_app" ] || {
  echo "install-and-launch: app manquante: $source_app" >&2
  exit 1
}

script_dir=$(cd "$(dirname "$0")" && pwd)
install_root=/Users/remyj/Applications/MesApps
backup_root="$install_root/.backups"
installed_app="$install_root/AURORA.app"
timestamp=$(date '+%Y%m%d-%H%M%S')

mkdir -p "$backup_root"
pkill -x AURORA >/dev/null 2>&1 || true

backup_app=
if [ -e "$installed_app" ]; then
  backup_app="$backup_root/AURORA-$timestamp.app"
  [ ! -e "$backup_app" ] || backup_app="$backup_root/AURORA-$timestamp-$$.app"
  mv "$installed_app" "$backup_app"
  echo "install-and-launch: sauvegarde: $backup_app"
fi

if ! ditto "$source_app" "$installed_app"; then
  if [ -e "$installed_app" ]; then
    mv "$installed_app" "$backup_root/AURORA-failed-$timestamp-$$.app"
  fi
  if [ -n "$backup_app" ] && [ ! -e "$installed_app" ]; then
    mv "$backup_app" "$installed_app"
  fi
  echo "install-and-launch: copie impossible" >&2
  exit 1
fi

temp_base=${TMPDIR:-/tmp}
temp_base=${temp_base%/}
source_manifest=$(mktemp "$temp_base/aurora-install-source.XXXXXX")
installed_manifest=$(mktemp "$temp_base/aurora-install-copy.XXXXXX")
cleanup() {
  rm -f "$source_manifest" "$installed_manifest"
}
trap cleanup EXIT

rollback() {
  if [ -e "$installed_app" ]; then
    failed_app="$backup_root/AURORA-failed-$timestamp-$$.app"
    mv "$installed_app" "$failed_app"
    echo "install-and-launch: copie invalide conservée: $failed_app" >&2
  fi
  if [ -n "$backup_app" ] && [ -e "$backup_app" ]; then
    mv "$backup_app" "$installed_app"
    echo "install-and-launch: ancienne version restaurée" >&2
  fi
}

manifest "$source_app" > "$source_manifest"
manifest "$installed_app" > "$installed_manifest"
cmp -s "$source_manifest" "$installed_manifest" || {
  echo "install-and-launch: copie différente de la build" >&2
  rollback
  exit 1
}

if ! "$script_dir/verify-release.sh" "$installed_app"; then
  rollback
  exit 1
fi
open "$installed_app"

for _ in $(seq 1 50); do
  if pgrep -x AURORA >/dev/null; then
    echo "install-and-launch: lancée: $installed_app"
    exit 0
  fi
  sleep 0.1
done

echo "install-and-launch: processus AURORA non détecté" >&2
exit 1
