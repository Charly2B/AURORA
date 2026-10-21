# Contribuer

Le dépôt est organisé en trois projets autonomes. Lire le README et les consignes du dossier modifié.

Avant une proposition :

```bash
cd server && go test ./...
cd ../android-tv && ./gradlew :app:testDebugUnitTest
cd ../macos && xcodegen generate && xcodebuild -project AURORA.xcodeproj -scheme AURORA -destination 'platform=macOS' test
```

Règles : aucun secret ou média personnel ; aucune dépendance d’exécution Homebrew dans l’app macOS ; jamais VLC dans le client macOS ; conserver les générations et nettoyages de sessions HLS ; documenter tout changement d’API dans les trois clients concernés.
