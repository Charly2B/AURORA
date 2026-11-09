# Développement

Prérequis : Go ≥ 1.25, Node.js ≥ 20 (npm), ffmpeg/ffprobe dans le PATH pour les tests d'intégration.
Aucun Docker, aucun privilège root.

## Lancer en local
```bash
scripts/gen-testmedia.sh /tmp/aurora-media        # vidéos synthétiques (facultatif)
scripts/build.sh host                              # interface + binaire avec interface embarquée
build/aurora-darwin-amd64 -listen 127.0.0.1:18890 -data /tmp/aurora-data -browse-roots /tmp/aurora-media
```
Puis http://127.0.0.1:18890. Interface avec rechargement à chaud : `cd web && npm run dev` (proxy `/api`
vers 18890, voir `web/vite.config.ts`).

Options (ou variables `AURORA_*`) : `-listen`, `-data`, `-tmp`, `-log`, `-ffmpeg`, `-ffprobe`,
`-browse-roots` (séparées par `:`, défaut : `/volume*` sur DSM, sinon le dossier personnel), `-dev`, `-web`.

## Tests
Voir [TESTS.md](TESTS.md).

## Paquet Synology
```bash
scripts/build-spk.sh <numéro-de-build>
```
1. télécharge FFmpeg statique épinglé (`build/vendor`, SHA-256 vérifié, jamais exécuté sur le poste) ;
2. compile l'interface puis le binaire `linux/amd64` statique (`CGO_ENABLED=0`, `GOAMD64=v2`) ;
3. assemble le `.spk` (`cmd/mkspk`, reproductible) ;
4. vérifie la structure (`scripts/verify-spk.sh`).

Incrémenter le numéro de build à chaque nouvelle publication : DSM s'en sert pour proposer la mise à jour.

## Règles du dépôt
Voir la section « Interdits » d'[ARCHITECTURE.md](../ARCHITECTURE.md). Toute évolution du schéma :
nouveau fichier `internal/db/migrations/000N_….sql`.

Métadonnées : `Client.Search` interroge seulement TMDB ; `SearchCandidates` ajoute la recherche souple
et le secours IMDb pour l'administration. Les candidats IMDb sans équivalent TMDB ont `id: 0` et un
`imdbId` ; `/admin/items/{id}/match` reçoit alors `{ "imdbId": "tt…" }` au lieu de `{ "tmdbId": … }`.
Les contrats de catalogue/recherche/fiches consommés par Android TV restent identiques.
