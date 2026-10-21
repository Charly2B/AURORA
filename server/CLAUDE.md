# AURORA — consignes pour les agents

Médiathèque native Synology DSM 7 (Go + SQLite + React). Lire d'abord [ARCHITECTURE.md](ARCHITECTURE.md)
(décisions et **interdits**) et [docs/LIMITATIONS.md](docs/LIMITATIONS.md).

## Commandes
- Tests : `go test -race ./...` ; `cd web && npx tsc --noEmit && npx vitest run`
- Charge : `go test -tags bench -run TestLoad -v ./internal/bench`
- Build local : `scripts/build.sh host` ; paquet : `scripts/build-spk.sh <build>`

## Règles
- Jamais de Docker, VM ou Container Manager. Jamais d'écriture dans les dossiers de vidéos.
- Aucune suppression (`rm`) de données de l'utilisateur ; ce qui doit disparaître va à la Corbeille.
- Ne rien déclarer « testé sur le NAS » sans l'avoir fait : la recette est dans docs/TESTS.md.
- Schéma : 0001 à 0004 sont publiées ; toute évolution = nouvelle migration (0005…).
- Toute modification de code met à jour la doc (README, ARCHITECTURE, docs/*, ce fichier, AGENTS.md).
- Après modification : tests verts, puis revue par un agent distinct (pas d'auto-validation).
- Client Android TV séparé : `~/Documents/AndroidStudio/AuroraTV` — une évolution de l'API doit y être
  reportée (`net/Models.kt`), et inversement.
