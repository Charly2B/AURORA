# AURORA pour macOS

Client natif du serveur multimédia AURORA, à parité avec AURORA TV et avec libmpv intégré.

## Développement

```bash
xcodegen generate
xcodebuild -project AURORA.xcodeproj -scheme AURORA -destination 'platform=macOS' test
```

## Build Release

```bash
Scripts/build-release.sh
Scripts/create-dmg.sh \
  Build/DerivedData/Build/Products/Release/AURORA.app \
  /Users/remyj/Documents/Codex/2026-10-20/tr/outputs/AURORA-0.1.0.dmg
Scripts/install-and-launch.sh Build/DerivedData/Build/Products/Release/AURORA.app
```

macOS 15 minimum. Le lecteur est libmpv 0.41 intégré via MPVKit 1.0.0 LGPL ; VLC n’est ni requis ni lancé. Les sessions restent dans le Trousseau macOS. Aucun mot de passe ni jeton n’est écrit dans les préférences ou les logs.

Les notices, l’inventaire exact des composants et les textes GNU LGPL/GPL sont livrés dans `AURORA/Resources` et copiés dans l’application.

Limites vérifiées : distribution locale signée ad hoc, non notarée ; HDR dépend du matériel/écran ; le NAS doit être joignable en HTTP(S). Le smoke test vidéo local H.264/AAC ne remplace pas un test avec chaque codec de votre médiathèque.

La livraison finale produit `AURORA.app` et un DMG autonome avec raccourci vers Applications. `Scripts/verify-release.sh` contrôle signature, frameworks intégrés, dépendances, version et contenu du DMG.

