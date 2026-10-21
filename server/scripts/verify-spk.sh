#!/bin/sh
# Vérifie la structure d'un paquet AURORA.spk selon le guide développeur Synology (DSM 7).
# Ne remplace pas un test d'installation sur un vrai NAS : il contrôle tout ce qui est vérifiable hors DSM.
# Usage : scripts/verify-spk.sh build/AURORA-x.y.z-NNNN.spk
set -u
SPK=${1:?fichier .spk requis}
W=$(mktemp -d "${TMPDIR:-/tmp}/aurora-verify.XXXXXX")
FAIL=0
ok()   { echo "  ok    $*"; }
ko()   { echo "  ÉCHEC $*"; FAIL=1; }
check() { if eval "$2" >/dev/null 2>&1; then ok "$1"; else ko "$1"; fi; }

echo "Paquet : $SPK"
tar -xf "$SPK" -C "$W" || { echo "archive illisible"; exit 1; }
LIST=$(tar -tvf "$SPK")

echo "Structure"
for f in INFO package.tgz conf/privilege conf/resource scripts/start-stop-status scripts/preinst scripts/postinst \
         scripts/preuninst scripts/postuninst scripts/preupgrade scripts/postupgrade PACKAGE_ICON.PNG PACKAGE_ICON_256.PNG LICENSE; do
  check "présent : $f" "[ -f '$W/$f' ]"
done
check "aucun fichier macOS parasite (._*, .DS_Store)" "! echo \"\$LIST\" | grep -E '(^|/)\\._|\\.DS_Store'"
check "scripts exécutables (rwxr-xr-x)" "! echo \"\$LIST\" | grep ' scripts/[a-z-][a-z-]*$' | grep -v '^-rwxr-xr-x'"
check "propriétaire root dans l'archive" "! echo \"\$LIST\" | grep -v 'root *root'"

echo "INFO"
INFO="$W/INFO"
for k in package version os_min_ver description arch maintainer displayname dsmuidir dsmappname adminport; do
  check "champ $k" "grep -q '^$k=\"[^\"]\\+\"' '$INFO'"
done
check "version au format X.Y.Z-NNNN" "grep -Eq '^version=\"[0-9]+(\\.[0-9]+)*-[0-9]+\"' '$INFO'"
check "os_min_ver >= 7.0-40000" "grep -Eq '^os_min_ver=\"7\\.[0-9]+-[0-9]{5}\"' '$INFO'"
check "arch contient x86_64 et v1000nk (DS1825+)" "grep -E '^arch=' '$INFO' | grep -q 'x86_64' && grep -E '^arch=' '$INFO' | grep -q 'v1000nk'"
SUM=$(sed -n 's/^checksum="\(.*\)"/\1/p' "$INFO")
if command -v md5sum >/dev/null; then GOT=$(md5sum "$W/package.tgz" | cut -d' ' -f1); else GOT=$(md5 -q "$W/package.tgz"); fi
check "checksum = MD5 de package.tgz" "[ '$SUM' = '$GOT' ]"

echo "Configuration DSM"
check "conf/privilege : JSON, run-as package (pas de root)" "python3 -I -c \"import json,sys; d=json.load(open('$W/conf/privilege')); sys.exit(d['defaults']['run-as']!='package')\""
check "conf/resource : JSON valide, port-config" "python3 -I -c \"import json; d=json.load(open('$W/conf/resource')); d['port-config']['protocol-file']\""
for s in "$W"/scripts/*; do check "syntaxe sh : $(basename "$s")" "sh -n '$s'"; done
check "icône 64x64" "file '$W/PACKAGE_ICON.PNG' | grep -q '64 x 64'"
check "icône 256x256" "file '$W/PACKAGE_ICON_256.PNG' | grep -q '256 x 256'"

echo "package.tgz"
mkdir -p "$W/pkg" && tar -xzf "$W/package.tgz" -C "$W/pkg"
for b in aurora ffmpeg ffprobe; do
  check "bin/$b : ELF x86-64 statique" "file '$W/pkg/bin/$b' | grep -q 'ELF 64-bit LSB executable, x86-64' && file '$W/pkg/bin/$b' | grep -q 'statically linked'"
done
check "ui/config : JSON, application url sur le port d'adminport" "python3 -I -c \"
import json,re,sys
d=json.load(open('$W/pkg/ui/config'))['.url']['com.aurora.mediacenter']
port=re.search(r'adminport=\\\"(\\d+)\\\"', open('$INFO').read()).group(1)
sys.exit(not (d['type']=='url' and d['port']==port and d['allUsers'] is True))\""
for s in 16 24 32 48 64 72 256; do check "ui/images/aurora_$s.png" "file '$W/pkg/ui/images/aurora_$s.png' | grep -q '$s x $s'"; done
check "port_conf/aurora.sc référencé par conf/resource" "[ -f '$W/pkg/port_conf/aurora.sc' ]"
check "port non publié vers Internet par défaut (port_forward=no)" "grep -q 'port_forward=\"no\"' '$W/pkg/port_conf/aurora.sc'"
check "licences tierces présentes" "[ -f '$W/pkg/licenses/FFMPEG-GPLv3.txt' ] && [ -f '$W/pkg/licenses/THIRD_PARTY_NOTICES.md' ]"
check "interface web embarquée dans le binaire" "grep -aq 'AURORA' '$W/pkg/bin/aurora' && grep -aq 'index.html' '$W/pkg/bin/aurora'"

echo
if [ $FAIL -eq 0 ]; then echo "Paquet conforme (vérifications hors DSM)."; else echo "Paquet NON conforme."; fi
echo "Contenu extrait pour inspection : $W"
exit $FAIL
