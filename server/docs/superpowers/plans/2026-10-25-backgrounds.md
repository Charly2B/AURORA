# Galerie de fonds — plan d’implémentation

**Objectif :** importer et servir des fonds centralisés, librement choisis par les applications.
**Architecture :** table SQLite contenant JPEG/PNG et métadonnées ; liste et images authentifiées.
**Stack :** Go, SQLite, React/TypeScript (administration du serveur uniquement).
**Spec :** `docs/superpowers/specs/2026-10-25-backgrounds-design.md`.

## Contraintes
50 Mio/image, adaptation automatique jusqu’en 4K (proportions et alpha PNG conservés) ; migration 0006 ; pas de suppression ni d’écriture dans les vidéos. Préserver les modifications préexistantes. Aucun client TV/Android modifié, aucun choix global.

## Vérifications sensibles
Clients non administrateurs/anonymes ; faux formats et images tronquées ; images énormes ; imports identiques ; octets inclus dans la sauvegarde ; import multiple interrompu.

## Étapes
- [x] Écrire le test HTTP d’import/liste/lecture/validation/autorisations ; échec constaté sur la route absente (404).
- [x] Implémenter `internal/api/backgrounds.go`, migration 0006 et routes de `server.go`. Test ciblé vert.
- [x] Ajouter les tests Web de galerie/import multipart et erreur ; échec constaté sur l’absence du panneau ; implémenter `BackgroundsPanel.tsx`, API multipart et types ; tests ciblés et TypeScript verts.
- [x] Documenter le contrat client et les limites, exécuter les suites prescrites, revue indépendante ; construire serveur et paquet DSM 0.1.10 build 0013 ; conserver le livrable.

## Décisions de périmètre
La première hypothèse (fond sélectionné globalement) a été corrigée dès la précision de l’utilisateur : seul le serveur fournit une galerie. Aucun code de sélection globale conservé ; les clients ne sont pas modifiés.

Validation finale : suites Go avec race, TypeScript, 16 tests Web et go vet verts ; builds host et DSM 0013 vérifiés ; import réel Web et récupération Bearer/ETag vérifiés sur base locale isolée ; revue indépendante sans défaut actionnable.

Correction du 25/10/2026, 0.1.11 : remplacer le refus des grandes images par une adaptation automatique via FFmpeg embarqué ; aucun client TV/Android modifié.

Validation du correctif : reproductions grandes images, orientation EXIF et annulation observées rouges puis vertes ; arrondi alpha swscale borné à un niveau et vérification exacte des zones transparentes/opaques. Acquisition annulable avant lecture du corps, une image en mémoire à la fois.
