# Architecture et sécurité

## Flux principal

1. Le scanner du serveur lit les dossiers médias sans les modifier.
2. `ffprobe` extrait conteneur, codecs, pistes et chapitres.
3. SQLite conserve le catalogue, les profils et la progression.
4. Les clients appellent `/api/v1` avec un cookie de session.
5. `/play` choisit lecture directe, remux ou conversion HLS selon les capacités du client.
6. Les clients envoient la progression toutes les dix secondes et à la fermeture.

## Composants

### Serveur

- Go, SQLite, interface React/Vite embarquée ;
- FFmpeg/ffprobe statiques dans le paquet DSM ;
- service non-root, répertoires médias en lecture seule ;
- Argon2id pour les mots de passe ; sessions révocables ; limitation anti-bruteforce ;
- URLs de flux externes signées et temporaires uniquement quand nécessaires.

### Android TV

- Kotlin, Compose for TV, Media3/ExoPlayer ;
- capacités matérielles détectées puis envoyées au NAS ;
- cookie chiffré AES/GCM avec une clé AndroidKeyStore par appareil ;
- navigation et focus conçus pour télécommande.

### macOS

- Swift 6, SwiftUI, macOS 15+ ;
- binaire universel Intel/Apple Silicon ;
- MPVKit/libmpv et FFmpeg embarqués dans l’app ;
- session dans le Trousseau ;
- aucun VLC, lecteur externe ou dépendance Homebrew à l’exécution.

## Authentification

Deux méthodes : identifiant/mot de passe AURORA, ou code d’association. Le code est saisi sur `/link` depuis un navigateur déjà authentifié. Le secret de suivi reste uniquement sur le client qui a créé le code.

Interdits : mot de passe/cookie dans une URL, journal ou préférence ; média personnel dans le dépôt ; clé TMDB dans le code ; port 8890 exposé directement à Internet.

## Lecture et reprise

Le NAS sélectionne le mode initial. Un échec de conteneur/audio tente le remux ; un échec du décodeur vidéo tente la conversion H.264. Les sessions HLS obsolètes sont arrêtées. Les changements de piste et sauts lointains recréent la session en conservant le point absolu.

## Dépendances et licences

Le serveur assemble FFmpeg selon les conditions documentées dans son paquet. Android repose sur les bibliothèques AndroidX/Media3. macOS utilise MPVKit 1.0.0 et embarque `ThirdPartyNotices.md` dans `AURORA.app` et le DMG.
