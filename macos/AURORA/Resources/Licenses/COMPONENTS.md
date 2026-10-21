# Composants embarqués par MPVKit 1.0.0

Inventaire déterministe issu de `MPVKit` 1.0.0, fichier
`Sources/BuildScripts/XCFrameworkBuild/main.swift`. Les sources et notices de
chaque binaire sont accessibles depuis la release ou le dépôt indiqué.

| Composant | Version MPVKit | Licence amont | Sources correspondantes |
|---|---:|---|---|
| MPVKit | 1.0.0 | LGPL-3.0 | https://github.com/mpvkit/MPVKit/tree/1.0.0 |
| mpv/libmpv | v0.41.0 | LGPL-2.1-or-later, build sans GPL | https://github.com/mpv-player/mpv/tree/v0.41.0 |
| FFmpeg | n8.1.2 | LGPL-2.1-or-later, build sans GPL | https://github.com/FFmpeg/FFmpeg/tree/n8.1.2 |
| OpenSSL | 3.3.5 | Apache-2.0 | https://github.com/mpvkit/openssl-build/tree/3.3.5 |
| GnuTLS, Nettle, GMP | 3.8.11 | LGPL/GPL selon composant | https://github.com/mpvkit/gnutls-build/tree/3.8.11 |
| libass, libunibreak, FreeType, FriBidi, HarfBuzz | 0.17.5 | ISC/Zlib/FTL/LGPL/MIT selon composant | https://github.com/mpvkit/libass-build/tree/0.17.5 |
| dav1d | 1.5.3 | BSD-2-Clause | https://github.com/mpvkit/libdav1d-build/tree/1.5.3 |
| Little CMS | 2.17.0 | MIT | https://github.com/mpvkit/lcms2-build/tree/2.17.0 |
| libplacebo | 7.360.1 | LGPL-2.1-or-later | https://github.com/mpvkit/libplacebo-build/tree/7.360.1 |
| libdovi | 3.3.2 | MIT | https://github.com/mpvkit/libdovi-build/tree/3.3.2 |
| MoltenVK | 1.4.2 | Apache-2.0 | https://github.com/mpvkit/moltenvk-build/tree/1.4.2 |
| shaderc | 2025.5.0 | Apache-2.0 et licences tierces | https://github.com/mpvkit/libshaderc-build/tree/2025.5.0 |
| uchardet | 0.0.8 | MPL-1.1 | https://github.com/mpvkit/libuchardet-build/tree/0.0.8 |
| libbluray | 1.4.0 | LGPL-2.1-or-later | https://github.com/mpvkit/libbluray-build/tree/1.4.0 |
| LuaJIT | 2.1.0-fix | MIT | https://github.com/mpvkit/libluajit-build/tree/2.1.0-fix |
| uavs3d | 1.2.1-fix | BSD-3-Clause | https://github.com/mpvkit/libuavs3d-build/tree/1.2.1-fix |

Les textes complets GNU LGPL-3.0 et GPL-3.0 accompagnent ce fichier. Les
notices particulières et autres textes complets sont conservés dans les
sources correspondantes ci-dessus. La release `MPVKit` contient les archives
binaires exactes et leurs sommes SwiftPM.

Pour reconstruire/remplacer les bibliothèques : cloner `MPVKit` 1.0.0,
utiliser le produit `MPVKit` (jamais `MPVKit-GPL`), puis suivre `make build`.
Les frameworks restent séparés dans `AURORA.app/Contents/Frameworks`.
