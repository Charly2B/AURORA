#!/bin/sh
# Génère une petite médiathèque de test (vidéos synthétiques) pour les tests manuels et automatisés.
# Usage : scripts/gen-testmedia.sh <dossier-cible>
set -eu
OUT=${1:?dossier cible requis}
FF=${FFMPEG:-ffmpeg}
D=${DURATION:-90}
mkdir -p "$OUT/Films/Big Test Movie (2020)" "$OUT/Films/Collection" "$OUT/Series/Ma Série (2019)/Saison 1" "$OUT/Series/Ma Série (2019)/Saison 2" "$OUT/Series/Inconnus"
gen() { # gen <sortie> <taille> <codecs vidéo...> -- <options audio...>
  out=$1; shift; size=$1; shift
  [ -f "$out" ] && return 0
  "$FF" -hide_banner -loglevel error -y -f lavfi -i "testsrc2=size=$size:rate=24:duration=$D" \
    -f lavfi -i "sine=frequency=440:sample_rate=48000:duration=$D" "$@" "$out"
}
# 1. MP4 H.264/AAC : Direct Play partout.
gen "$OUT/Films/Big Test Movie (2020)/Big.Test.Movie.2020.1080p.WEB-DL.mp4" 1920x1080 \
  -c:v libx264 -preset ultrafast -pix_fmt yuv420p -c:a aac -b:a 128k -ac 2 -movflags +faststart
# 2. MKV H.264 + AC3 5.1 + sous-titres SRT internes : remux (conteneur + audio).
printf '1\n00:00:01,000 --> 00:00:04,000\nBonjour, ceci est un sous-titre interne.\n\n2\n00:00:30,000 --> 00:00:35,000\nTrente secondes.\n' > "$OUT/sub_fr.srt"
"$FF" -hide_banner -loglevel error -y -f lavfi -i "testsrc2=size=1280x720:rate=24:duration=$D" \
  -f lavfi -i "sine=frequency=330:sample_rate=48000:duration=$D" -i "$OUT/sub_fr.srt" \
  -map 0:v -map 1:a -map 1:a -map 2:s -c:v libx264 -preset ultrafast -pix_fmt yuv420p \
  -c:a:0 ac3 -ac:a:0 6 -c:a:1 aac -ac:a:1 2 -metadata:s:a:0 language=fre -metadata:s:a:1 language=eng \
  -c:s srt -metadata:s:s:0 language=fre "$OUT/Films/Sintel.Like.2010.720p.BluRay.MULTI.mkv"
# Sous-titre externe voisin.
printf '1\n00:00:02,000 --> 00:00:06,000\nExternal English subtitle.\n' > "$OUT/Films/Sintel.Like.2010.720p.BluRay.MULTI.en.srt"
# 3. HEVC 10 bits en MKV : remux pour Safari, conversion pour les navigateurs sans HEVC.
gen "$OUT/Films/Hevc.Sample.2021.720p.x265.mkv" 1280x720 \
  -c:v libx265 -preset ultrafast -pix_fmt yuv420p10le -x265-params log-level=error -tag:v hvc1 -c:a aac -ac 2
# 4. Deux versions du même film.
gen "$OUT/Films/Collection/Double.Feature.2018.720p.mp4" 1280x720 -c:v libx264 -preset ultrafast -pix_fmt yuv420p -c:a aac -ac 2
gen "$OUT/Films/Collection/Double.Feature.2018.1080p.Extended.mkv" 1920x1080 -c:v libx264 -preset ultrafast -pix_fmt yuv420p -c:a aac -ac 2
# 5. Série : saisons, épisode double, fichier non identifiable.
for e in 01 02 03; do
  gen "$OUT/Series/Ma Série (2019)/Saison 1/Ma.Serie.S01E$e.1080p.mkv" 1280x720 -c:v libx264 -preset ultrafast -pix_fmt yuv420p -c:a aac -ac 2
done
gen "$OUT/Series/Ma Série (2019)/Saison 2/Ma.Serie.S02E01E02.1080p.mkv" 1280x720 -c:v libx264 -preset ultrafast -pix_fmt yuv420p -c:a aac -ac 2
gen "$OUT/Series/Inconnus/vacances_plage.mp4" 640x360 -c:v libx264 -preset ultrafast -pix_fmt yuv420p -c:a aac -ac 2
echo "Médiathèque de test prête dans $OUT"
