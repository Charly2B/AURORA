# Sauvegarde et restauration

## Ce qui est sauvegardé
La base SQLite `aurora.db` contient tout ce qui a de la valeur : bibliothèques, catalogue et corrections,
profils et mots de passe (hachés), historique, progression, favoris, listes, réglages (dont la clé TMDB).
Les images en cache (`cache/`) et les segments temporaires ne sont pas sauvegardés : ils se régénèrent.

## Sauvegardes automatiques
| Moment | Fichier dans `@appdata/aurora/backups/` |
|---|---|
| Avant une mise à jour du paquet (script `preupgrade`) | `avant-maj-<ancienne-version>-<date>.db` |
| Avant toute migration du schéma (au démarrage) | `pre-migration-v<N>-<date>.db` |
| Avant une restauration | `avant-restauration-<date>.db` |
| Avant un changement de type de bibliothèque (Films ↔ Séries) | `avant-changement-type-<date>.db` |

Aucune sauvegarde n'est supprimée automatiquement. Faites le ménage vous-même si l'espace manque.

## Sauvegarde manuelle
Admin › Maintenance › **Sauvegarder maintenant** : copie cohérente (`VACUUM INTO`), même pendant
l'utilisation. **Télécharger** la récupère sur votre ordinateur. Pour une protection hors NAS,
incluez `@appdata/aurora/backups` dans Hyper Backup (ou copiez les fichiers téléchargés ailleurs).

## Restauration
1. Admin › Maintenance › sauvegarde › **Restaurer** › confirmer. L'état actuel est d'abord sauvegardé.
2. Centre de paquets › AURORA › **Arrêter** puis **Exécuter**.
3. Au démarrage, la base courante est renommée `aurora.db.before-restore-<date>` (jamais effacée) et la
   sauvegarde prend sa place, puis les migrations éventuelles s'appliquent.

Restaurer un fichier venu d'ailleurs : déposez-le dans `@appdata/aurora/backups/` (nom en `.db`), il
apparaît dans la liste.

## Récupérer une installation après erreur
- **Migration échouée** : AURORA refuse de démarrer, la base reste dans l'état d'avant (transaction
  annulée) et `pre-migration-…db` existe. Journal : `logs/console.log`.
- **Base plus récente que le programme** (retour à une ancienne version) : refus explicite au démarrage.
  Réinstallez la version la plus récente, ou restaurez une sauvegarde faite par l'ancienne version.
- **Base corrompue** : Admin › Maintenance › Diagnostic › Intégrité (`PRAGMA quick_check`). Si le
  serveur ne démarre plus : arrêter le paquet, renommer `aurora.db` (File Station), copier une sauvegarde
  sous le nom `aurora.db`, redémarrer.
