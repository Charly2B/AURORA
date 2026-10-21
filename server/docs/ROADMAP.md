# Feuille de route

État des phases demandées. « Fait » = code, interface, données réelles, erreurs gérées, tests passés.

| Phase | État |
|---|---|
| 1 — Fondation native | **Fait** hors matériel : Go + React, SPK, icône DSM, service, configuration persistante. Installation réelle à valider. |
| 2 — Première version utilisable | **Fait** : assistant, admin, bibliothèques, indexation, catalogue, lecture, reprise, erreurs. |
| 3 — Médiathèque intelligente | **Fait** sauf collections manuelles : TMDB, affiches, séries/saisons/épisodes, recherche, filtres, correction manuelle, sagas. |
| 4 — Design premium | **Largement fait** : accueil cinéma, fiches immersives, profils, accueil personnalisable, responsive. Reste : thèmes, navigation télécommande. |
| 5 — Lecture avancée | **Partiel** : pistes, sous-titres texte/externes/incrustés, remux, conversion limitée, diagnostics. Reste : décalage des sous-titres, audio 5.1 en AAC multicanal, clients natifs. |
| 6 — Stabilisation | **En cours** : tests, revue, documentation, paquet. Reste : recette sur DS1825+. |

## Prochaines étapes, par priorité
1. **Recette sur le DS1825+** ([TESTS.md](TESTS.md#recette-sur-le-ds1825)) et corrections qui en découleront ;
   mesurer la vitesse de conversion réelle pour calibrer les limites.
2. Décalage manuel des sous-titres (±) et mémorisation par fichier.
3. AAC 5.1 quand le navigateur sait le restituer (au lieu de la stéréo).
4. Collections manuelles et « Ma liste » ordonnable.
5. Code PIN de profil (changement de profil rapide sur un appareil partagé).
6. Détection d'introduction/générique : analyse audio **à la demande** par saison, coût CPU mesuré avant activation.
7. Statistiques de visionnage, recommandations locales enrichies.
8. Client natif macOS/tvOS avec lecteur mpv/libmpv ou AVPlayer : Dolby Vision, Atmos, passthrough,
   changement de fréquence — impossibles dans un navigateur.
9. Thème clair éventuel. (La navigation télécommande est assurée par le client Android TV AuroraTV,
   projet séparé ; un client tvOS reste à faire, point 8.)
