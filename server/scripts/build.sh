#!/bin/sh
# Compile AURORA : interface web (Vite) puis serveur Go avec l'interface embarquée.
# Usage : scripts/build.sh [linux|host]   (défaut : host)
# Sortie : build/aurora-<os>-<arch>
#
# Règle du projet : aucune suppression. L'ancienne interface compilée est déplacée (Corbeille sur macOS,
# build/.old ailleurs), jamais effacée.
set -eu
ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"
TARGET=${1:-host}
VERSION=$(cat VERSION)
STAMP=$(date +%Y%m%d-%H%M%S)

park() { # park <chemin> : met de côté un dossier existant, sans rien effacer
  [ -e "$1" ] || return 0
  if [ -d "$HOME/.Trash" ]; then dest="$HOME/.Trash/aurora-$(basename "$1")-$STAMP"
  else mkdir -p "$ROOT/build/.old"; dest="$ROOT/build/.old/$(basename "$1")-$STAMP"; fi
  mv "$1" "$dest"
}

echo "==> Interface web"
( cd web && { [ -d node_modules ] || npm ci --no-audit --no-fund; } && npx tsc --noEmit )
if [ -n "$(ls -A internal/webui/dist 2>/dev/null | grep -v '^.keep$' || true)" ]; then
  park internal/webui/dist
  mkdir -p internal/webui/dist && : > internal/webui/dist/.keep
fi
( cd web && npx vite build --logLevel warn )
[ -f internal/webui/dist/index.html ] || { echo "ABANDON : interface non générée"; exit 1; }

echo "==> Serveur Go ($TARGET)"
mkdir -p build
LDFLAGS="-s -w -X aurora/internal/config.Version=$VERSION"
if [ "$TARGET" = linux ]; then
  # Binaire statique (CGO désactivé, SQLite en Go pur) : aucune dépendance aux bibliothèques de DSM.
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v2 go build -trimpath -ldflags "$LDFLAGS" -o build/aurora-linux-amd64 ./cmd/aurora
  echo "build/aurora-linux-amd64"
else
  CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" -o "build/aurora-$(go env GOOS)-$(go env GOARCH)" ./cmd/aurora
  echo "build/aurora-$(go env GOOS)-$(go env GOARCH)"
fi
