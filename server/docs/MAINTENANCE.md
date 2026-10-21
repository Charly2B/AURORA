# Maintenance

## Où regarder en cas de problème
1. **Administration › Tableau de bord** : état, erreurs importantes, analyse en cours.
2. **Administration › Maintenance** : historique des analyses (cliquer sur le nombre d'erreurs pour la
   liste des fichiers en cause), diagnostic (versions, intégrité de la base, outils, limites), journal.
3. Fichiers sur le NAS (`/volumeX/@appdata/aurora/logs/`) :
   - `aurora.log` : journal JSON du serveur (5 Mio max, 3 archives `.1` `.2` `.3`), sans mot de passe ni clé ;
   - `console.log` : sortie brute du processus (erreurs fatales de démarrage).
4. Centre de paquets › AURORA › **Journal** : messages des scripts DSM.

## Opérations courantes
| Besoin | Où |
|---|---|
| Bibliothèque déclarée du mauvais type | Admin › Bibliothèques › bouton ⇄ (ou retirer puis rajouter le dossier) ; les épisodes nommés S01E01 rangés parmi des films sont classés en séries automatiquement |
| Nouveaux fichiers non visibles | Admin › Bibliothèques › **Analyser** (sinon automatique, 6 h par défaut) |
| Mauvais film / mauvaise série associés | fiche › **Corriger l'identification** (choix TMDB ou titre manuel ; verrouillé ensuite) |
| Bonne œuvre, affiche ou fond à changer | fiche › **Changer l'affiche** (choix parmi les images TMDB) |
| Épisode non reconnu | Admin › **Identification** › Fichiers non identifiés |
| Fichier introuvable (disque démonté, fichier déplacé hors bibliothèque) | il est masqué, son historique est conservé ; il réapparaît s'il revient |
| Réduire la charge du NAS | Réglages : analyses simultanées = 1, conversions = 0 ou 1 |
| Arrêter une analyse | Tableau de bord › Interrompre (elle reprendra là où elle s'est arrêtée) |

## Comportements de fond (et pourquoi)
- **Aucune analyse au démarrage** du NAS. Vérification incrémentale périodique (réglable, 0 = jamais) :
  seuls les fichiers nouveaux ou modifiés (taille/date) sont relus par ffprobe.
- **Une seule analyse à la fois**, ffprobe limité (2 par défaut) : les disques et le processeur restent
  disponibles pour la lecture.
- **Conversions vidéo** limitées (1 par défaut) ; refus explicite au-delà, et refus des sources > 1080p
  sauf réglage contraire. ffmpeg est **suspendu** quand il a plus de ~3 min d'avance sur le lecteur et
  les segments déjà vus sont effacés au fil de l'eau : un remux de 50 Go ne remplit pas le disque.
- **Sessions inactives** (> 90 s sans requête) arrêtées automatiquement ; segments temporaires effacés
  (dossier `@apptemp/aurora`, propre au paquet).

## Mise à jour d'AURORA
Voir [INSTALLATION_DSM.md](INSTALLATION_DSM.md#6-mise-à-jour). Construire une nouvelle version :
incrémenter `VERSION` (ou le numéro de build) puis `scripts/build-spk.sh <build>`.

## Mettre à jour FFmpeg embarqué
`scripts/build-spk.sh` télécharge une archive **épinglée** (URL + SHA-256). Pour changer de version :
modifier `FF_URL`/`FF_SHA256`, vérifier que les binaires sont statiques (`file`), relancer la
construction et `scripts/verify-spk.sh`. Documenter la version dans `THIRD_PARTY_NOTICES.md`.
