# Fiche série : voir où l'on est

Maquette validée le 29/10/2026 : `docs/maquettes/2026-10-29-series-navigation.html`.

## Problème

Dans la fiche d'une série, la navigation à la télécommande ne se voyait pas :

- la ligne d'épisode focalisée n'avait qu'un liseré de 2 dp et un fond à 9 % de crème — illisible à trois mètres,
  alors que les affiches des grilles ont un cadre crème-or épais et un halo ;
- l'en-tête (affiche, titre) défilait avec les épisodes : dans une saison de dix ou vingt épisodes, on ne savait
  plus quelle saison on parcourait ni où l'on en était.

## Décision

1. **Focus d'épisode aligné sur les affiches** : cadre `FocusGold` de 3 dp, halo doré (`shadow` 16 dp), fond relevé,
   n° d'épisode et titre en or, vignette cerclée d'or. La vignette garde sa taille et la ligne ne bouge pas.
2. **Barre de saison collante** (`stickyHeader`) : les saisons restent en haut pendant le défilement, avec à droite
   « Épisode 4 sur 10 · 3 vus » quand une ligne a le focus, sinon « 10 épisodes · 3 vus ». Le nom de la série
   n'apparaît dans la barre qu'une fois l'en-tête défilé, pour ne pas doubler le grand titre.
3. **« ⋯ » ancré** : sa position est déduite de la taille de la vignette (`EpisodeThumbWidth/Height`,
   `MoreButtonSize`) au lieu de `offset(177, 89)` écrit à la main.

## Inchangé

- Sidebar sans les trois « En cours » (dossiers NAS du même nom conservés).
- Fiche série sans bouton « Reprendre » global, sans Ma liste / Favori ; reprise et progression sur l'épisode.
- « ⋯ » toujours visible ; sur un film en bas à droite de l’affiche.
- Vu automatique à 80 % de la durée du fichier, coordonné avec le seuil du NAS.
- La ligne « X épisodes vus sur Y » au-dessus des saisons reste : c’est l’avancement de la série, pas une reprise.

## Ajout du 29/10/2026 (après la première installation)

Aucun « ⋯ » sur une affiche de série, dans les grilles comme dans la recherche : « vu » se règle par épisode,
jamais pour toute la série d’un coup. Les films et les épisodes gardent le leur ; sur une série, ▶ passe
directement à l’affiche suivante.

Ensuite, sur une ligne d’épisode, le « ⋯ » est sorti de la vignette : il occupe une colonne à droite de la
ligne, séparée par un filet, centré verticalement (maquette `2026-10-29-episode-actions.html`, validée
« toujours visible »). Sur la photo, il se confondait avec l’image et le résumé passait dessous ; la barre de
progression reprend toute la largeur de la vignette. Navigation inchangée : ▶ vers le « ⋯ », OK ouvre
Vu / Pas vu, ◀ revient, OK sur la ligne lance la lecture.

## Vérification

Build debug et 25 tests unitaires ; revue distincte et contre-revue. Installée sur la Xiaomi (fiche, barre
collante, halo, « ⋯ », fenêtre Vu / Pas vu vérifiés par captures) et sur le Formovie (processus et activité
lancés, projecteur en veille : rendu non vérifié). Retour de focus depuis le lecteur non vérifié sur appareil.

Détail découvert sur la box : demander le focus sur le chip de saison fait défiler la fiche d’environ 160 dp
(la fiche s’ouvrait sur une affiche coupée). Le focus est donc posé puis la liste revient en haut après 250 ms ;
le chip reste visible, juste sous le titre.
