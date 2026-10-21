# AURORA — logiciels tiers

## MPVKit 1.0.0 (configuration LGPL)

AURORA utilise exclusivement le produit Swift Package `MPVKit`, jamais
`MPVKit-GPL`. Cette distribution embarque libmpv 0.41.0, FFmpeg et leurs
dépendances sous licences LGPL et licences compatibles.

- MPVKit : https://github.com/mpvkit/MPVKit/tree/1.0.0
- libmpv : https://github.com/mpv-player/mpv/tree/v0.41.0
- FFmpeg : https://github.com/FFmpeg/FFmpeg
- inventaire exact : `COMPONENTS.md` ;
- textes complets : `LGPL-3.0.txt` et `GPL-3.0.txt` ;
- sources correspondantes : dépôts et versions figés dans l’inventaire.

MPVKit et ses bibliothèques LGPL sont des bibliothèques séparées et
remplaçables dans le bundle de l’application. Vous pouvez relier une version
modifiée compatible des bibliothèques LGPL, pour votre usage, en remplaçant
les frameworks correspondants dans `AURORA.app/Contents/Frameworks`, puis en
signant de nouveau le bundle. AURORA n’ajoute aucune restriction à
l’ingénierie inverse nécessaire au débogage de ces modifications.

Les bibliothèques LGPL sont fournies sans garantie. Consultez la GNU Lesser
General Public License version 3 : https://www.gnu.org/licenses/lgpl-3.0.html

Les composants additionnels conservent leurs licences et notices propres.
Leur nom, version, licence, source correspondante et procédure de remplacement
sont consignés dans `COMPONENTS.md` livré avec l’application.
