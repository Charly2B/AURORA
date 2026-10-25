# AURORA macOS — consignes agents

Client macOS du serveur AURORA. Lire `docs/superpowers/specs/2026-10-20-aurora-macos-design.md` et le plan actif avant toute modification.

- Projet généré par XcodeGen (`project.yml`) : relancer `xcodegen generate` après ajout/retrait de Swift.
- Swift 6, SwiftUI, macOS 15+ ; lecteur libmpv intégré via MPVKit LGPL, jamais VLC.
- API compatible avec le serveur et le client Android ; aucun secret dans URL, préférences, code ou logs.
- Tests : `xcodebuild -project AURORA.xcodeproj -scheme AURORA -destination 'platform=macOS' test`.
- Build final Release avec `Scripts/build-release.sh`, DMG avec `Scripts/create-dmg.sh`, puis sauvegarde/install/vérification/lancement avec `Scripts/install-and-launch.sh`.
- Aucun test réel NAS/vidéo ne peut être revendiqué sans exécution réelle correspondante.
- Interface « Cinémathèque » validée par l’utilisateur le 2026-10-24 : couleurs et styles uniquement via `AuroraTheme` (pas de couleur en dur dans les vues), titres en serif système. Pas d’animation continue (rejetée comme fatigante) ; seuls des fondus courts au changement de titre.
- Titre à la une de l’accueil : un seul tirage au hasard par session (`HomeModel.pickRandomFeature`, rang parmi les comptes de `/libraries` puis `list` à `limit: 1`) ; les rechargements (retour du lecteur, scan) le gardent. Le héros serveur sert de repli.
- Épisodes : aucun repère sur la vignette quand ils ne sont pas vus (demande utilisateur, 2026-10-25) ; « vu » = coche verte `AuroraTheme.seen` ; le choix Vu / Pas vu passe par le menu « ⋯ » (survol) ou le clic droit, jamais par un bouton toujours visible ; pas de bouton « Vu » global en tête d’une fiche série (gardé pour les films). Épisodes en liste verticale, résumé à droite. Dans les vues, écrire `SwiftUI.Section` : l’app a son propre type `Section` (rangées de l’accueil).
- Architecture : `AppSession` gère NAS/auth ; `MainShellView` garde l’ordre Accueil/Films/Séries/Recherche/Profil (Recherche affichée en champ ⌘F en tête de colonne, Profil en pied) ; `PlaybackCoordinator` est l’unique machine d’état vidéo.
- Toute destination fiche/lecteur doit conserver une identité dépendant de l’ID média afin d’éviter la réutilisation d’état entre épisodes.

