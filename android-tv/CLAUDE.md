# AURORA TV — consignes pour les agents

Client Android TV (Xiaomi Box) du serveur AURORA (`~/Documents/Codex/2026-10-16/aurora`, API `/api/v1`).
Lire d'abord [README.md](README.md) : décisions non évidentes et **interdits**.

## Commandes
- Build : `./gradlew :app:assembleDebug` ; tests : `./gradlew :app:testDebugUnitTest`
- Émulateur : AVD `GlassPlayerTV` (Google TV 1080p) ; le Mac y est joignable en `10.0.2.2`.
- Installation sur la box : adb en IPv6 link-local (README) — **demander à l'utilisateur avant**.

## Règles
- Toute évolution de l'API côté serveur se reflète ici (modèles `net/Models.kt`) et inversement.
- Aucune suppression (`rm`) de données de l'utilisateur ; ce qui doit disparaître va à la Corbeille.
- Ne rien déclarer « testé sur la box » sans l'avoir fait.
- Toute modification de code met à jour README.md et ce fichier.
- Après modification : build + tests verts, puis revue par un agent distinct.
