# Focus visible et état de lecture

Spec: `docs/superpowers/specs/2026-10-29-visible-focus-watched-design.md`

## Global Constraints

- Android uniquement, aucun changement macOS ou serveur, aucun émulateur, aucun commit/push.
- Préserver toutes les modifications existantes. Cinq affiches par ligne, épisodes verticaux, données et sessions préservées.
- Le focus des grilles suit réellement la télécommande, sans sélection décorative persistante sur une autre carte.
- 80 % de durée par film/épisode, pas du nombre d'épisodes ; éviter les réécritures watched=false du serveur.
- Build, tests et revue distincte avant installation sur Xiaomi et Formovie, déjà autorisée.

## Task 1: Interface et menu Vu / Pas vu

Ownership: `ui/components/Components.kt`, `ui/theme/Theme.kt`, `ui/MainScreen.kt`, `ui/DetailScreen.kt`, et nouveau composant de menu si utile. Aucun autre fichier de production.

Appliquer exactement la spec UI : cadre épais/halo/titre au focus réel des affiches ; conserver cinq colonnes, focus mémoire et disposition. Retirer uniquement la section En cours sidebar et son code désormais inutilisé, pas les dossiers NAS portant ce nom. Conserver les épisodes verticaux, progression horizontale et reprise directement sur la ligne. Supprimer le bouton global Lecture/Reprendre série, Ma liste et Favori séries/épisodes, garder les actions films. Menu ⋯ toujours visible sur vignettes films/séries et épisodes, Vu/Pas vu fiable, erreurs visibles, pas de playback depuis ce menu, focus correctement restauré. Ne pas modifier la logique lecteur ni le réseau.

Valider avec build et tests existants. Pas de tests de style qui recopient l'implémentation. Revue indépendante sur un diff isolé du baseline.

## Task 2: Seuil de lecture et intégration

Ownership: `ui/PlayerScreen.kt`, nouveau composant de progression testable sous `player/`, `net/AuroraClient.kt` seulement si nécessaire, tests du composant.

Implémenter un coordinateur de progression par lecture : sérialiser les rapports, marquer vu à >=80% de durée valide, arrêter les progressions qui annuleraient watched après ce seuil ; conserver un essai/retry après erreurs et éviter de perdre l'intention de completion si un seek revient sous le seuil après échec. Toutes les émissions périodiques/finales doivent passer par le même coordinateur. Préserver les règles restart, HLS, durée/résolution inconnue, épisode suivant et nettoyage. Tests red-green pertinents : 79.99/80%, durée invalide/inconnue, appel périodique vs sortie concurrents, erreur/réessai de marquage, absence de réécriture après completion et isolation de deux lectures.

Le contrôleur documente, exécute le build et la suite finale, demande la revue indépendante, livre l'APK puis installe et vérifie les empreintes sur les deux appareils. Ne pas effectuer d'actions lecture/modification d'état sur les données réelles pendant la vérification.
