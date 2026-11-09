# Galerie de fonds hébergés par AURORA

Demande confirmée : déposer des images sur le serveur ; chaque application TV, Android ou autre doit choisir son propre fond. Le 25/10/2026, l’utilisateur précise : modifier uniquement le serveur, il adaptera les autres applications lui-même.

Administration › Réglages : importer plusieurs JPEG/PNG et consulter la galerie. Aucun choix global ni modification de `/home`. Les fichiers et choix des clients restent hors périmètre.

Approches : un dossier manuel impose une configuration des droits NAS ; fichiers et métadonnées séparés compliquent les sauvegardes ; une table SQLite contenant images et métadonnées garde imports atomiques et sauvegardes complètes. Cette dernière approche est retenue. Limites : 50 Mio/image, adaptation automatique jusqu’en 4K (proportions et alpha PNG conservés) ; conservation des images sans suppression.

API authentifiée : galerie `GET /api/v1/backgrounds`, images `GET /api/v1/backgrounds/{id}/image`, import multipart administrateur `POST /api/v1/admin/backgrounds`. Les clients consultent la liste puis chargent l’URL relative avec leur session et mémorisent eux-mêmes leur choix. Contrat complet : `docs/API_FONDS.md`.

Contraintes : migration 0006, aucune écriture dans les vidéos, aucun Docker, conservation des modifications préexistantes, cookie/Bearer et CSRF existants. Tests HTTP : contenu/type réels, permissions, taille/dimensions limitées, déduplication et sauvegarde. Tests Web, build serveur/paquet DSM, revue indépendante. Aucun déploiement NAS présumé.

Correction du 25/10/2026, 0.1.11 : remplacer le refus des grandes images par une adaptation automatique via FFmpeg embarqué ; aucun client TV/Android modifié.
