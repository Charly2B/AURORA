#!/bin/sh
# Construit AURORA.spk pour Synology DSM 7 (x86-64, dont DS1825+ / plateforme v1000nk).
# Usage : scripts/build-spk.sh [numéro-de-build]
# Prérequis : Go, Node.js/npm, curl, shasum (ou sha256sum). Aucun Docker, aucun accès root.
set -eu
ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"
VERSION=$(cat VERSION)
BUILD=${1:-0001}

# FFmpeg statique épinglé. Le site ne publie plus de somme de contrôle pour ce fichier : l'empreinte
# ci-dessous a été mesurée lors de la première récupération (17/10/2026) et protège contre toute substitution.
FF_URL="https://johnvansickle.com/ffmpeg/releases/ffmpeg-7.0.2-amd64-static.tar.xz"
FF_SHA256="abda8d77ce8309141f83ab8edf0596834087c52467f6badf376a6a2a4c87cf67"
FF_TAR="build/vendor/ffmpeg-7.0.2-amd64-static.tar.xz"

sha256() { if command -v sha256sum >/dev/null; then sha256sum "$1" | cut -d' ' -f1; else shasum -a 256 "$1" | cut -d' ' -f1; fi; }

mkdir -p build/vendor
if [ ! -f "$FF_TAR" ]; then
  echo "==> Téléchargement de FFmpeg statique"
  curl -fL --retry 3 -o "$FF_TAR.part" "$FF_URL"
  mv "$FF_TAR.part" "$FF_TAR"
fi
GOT=$(sha256 "$FF_TAR")
[ "$GOT" = "$FF_SHA256" ] || { echo "ABANDON : empreinte FFmpeg inattendue ($GOT)"; exit 1; }

# Extraction dans un dossier neuf et dédié (contenu téléchargé = non fiable : jamais exécuté ici).
FFX="build/vendor/ffmpeg-$FF_SHA256"
if [ ! -x "$FFX/ffmpeg" ]; then
  mkdir -p "$FFX.tmp"
  tar -xJf "$FF_TAR" -C "$FFX.tmp" --strip-components=1
  mv "$FFX.tmp" "$FFX"
fi

echo "==> Compilation (linux/amd64)"
scripts/build.sh linux

echo "==> Assemblage du paquet"
OUT="build/AURORA-$VERSION-$BUILD.spk"
go run ./cmd/mkspk -version "$VERSION" -build "$BUILD" -bin build/aurora-linux-amd64 -ffmpeg "$FFX" -out "$OUT"

echo "==> Vérification"
scripts/verify-spk.sh "$OUT"
echo "Paquet prêt : $OUT"
