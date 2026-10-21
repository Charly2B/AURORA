# AURORA macOS — consignes agents

Client macOS du serveur AURORA. Lire `docs/superpowers/specs/2026-10-20-aurora-macos-design.md` et le plan actif avant toute modification.

- Projet généré par XcodeGen (`project.yml`) : relancer `xcodegen generate` après ajout/retrait de Swift.
- Swift 6, SwiftUI, macOS 15+ ; lecteur libmpv intégré via MPVKit LGPL, jamais VLC.
- API compatible avec le serveur et le client Android ; aucun secret dans URL, préférences, code ou logs.
- Tests : `xcodebuild -project AURORA.xcodeproj -scheme AURORA -destination 'platform=macOS' test`.
- Build final Release avec `Scripts/build-release.sh`, DMG avec `Scripts/create-dmg.sh`, puis sauvegarde/install/vérification/lancement avec `Scripts/install-and-launch.sh`.
- Aucun test réel NAS/vidéo ne peut être revendiqué sans exécution réelle correspondante.
- Architecture : `AppSession` gère NAS/auth ; `MainShellView` garde l’ordre Accueil/Films/Séries/Recherche/Profil ; `PlaybackCoordinator` est l’unique machine d’état vidéo.
- Toute destination fiche/lecteur doit conserver une identité dépendant de l’ID média afin d’éviter la réutilisation d’état entre épisodes.

