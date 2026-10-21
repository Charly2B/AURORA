# AURORA — consignes pour les agents (Codex et autres)

Mêmes consignes que [CLAUDE.md](CLAUDE.md), tenues identiques :

- Lire [ARCHITECTURE.md](ARCHITECTURE.md) (décisions, **interdits**) et [docs/LIMITATIONS.md](docs/LIMITATIONS.md).
- Tests : `go test -race ./...` ; `cd web && npx tsc --noEmit && npx vitest run`.
- Build : `scripts/build.sh host` ; paquet : `scripts/build-spk.sh <build>`.
- Jamais de Docker/VM ; jamais d'écriture dans les dossiers de vidéos ; aucune suppression de données
  (Corbeille) ; rien de « testé sur le NAS » sans l'avoir fait ; nouvelle migration (0005…) plutôt que modifier
  0001 à 0004, déjà publiées ; doc mise à jour à chaque modification ; revue par un agent distinct.
- Client Android TV séparé : `~/Documents/AndroidStudio/AuroraTV` — une évolution de l'API doit y être
  reportée (`net/Models.kt`), et inversement.
