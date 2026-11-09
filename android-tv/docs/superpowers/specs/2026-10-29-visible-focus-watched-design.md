# Focus visible et état de lecture — approuvé le 29 octobre 2026

L'utilisateur approuve explicitement la maquette `auroratv-selection-series.html` et demande de coder aussi les autres changements de sa demande.

## Exigences

- Dans les grilles de droite (films et séries), le cadre lumineux suit le focus réel de la télécommande. Reprendre la maquette : bord crème-or épais, halo or discret, titre bien mis en évidence. Garder cinq vignettes par ligne et la restauration du focus.
- Retirer la section de reprises « En cours » de la sidebar. Conserver les dossiers NAS nommés « Films en cours » et « Series en cours ».
- Les épisodes restent en liste verticale, vignette à gauche, résumé à droite. L'utilisateur a explicitement corrigé la proposition de défilement horizontal des épisodes.
- La reprise et sa progression horizontale apparaissent directement sur l'épisode. Retirer le bouton global Lecture/Reprendre de l'en-tête série qui fait doublon.
- Retirer Ma liste et Favori des fiches séries/épisodes ; conserver ces actions pour les films.
- Menu « ⋯ » en bas à droite des vignettes pour Vu / Pas vu : toujours visible, accessible à la télécommande, sans lancement involontaire. Préserver la navigation entre vignette et menu et le focus au retour.
- Marquer automatiquement chaque film et chaque épisode vu à 80 % de sa durée, seuil inclus. Cela ne désigne jamais 80 % du nombre d'épisodes d'une série.
- La règle NAS actuelle peut remettre `watched=false` sous son seuil de 92 %. Ne pas envoyer un unique setState(true) suivi de progressions qui effaceraient cet état. Coordonner les appels Android sans changer le NAS.

## Contraintes

Android uniquement, aucun changement macOS ou serveur. Aucun émulateur. Préserver données, sessions, fonds NAS, dossiers, tris, restauration du focus, logique HLS et lecture suivante. Aucun commit/push demandé. Installer la build validée sur Xiaomi et Formovie selon autorisation déjà donnée. Tester et obtenir une revue indépendante avant installation.

## Décision de portée

L'expression « ascenseur horizontal » est traitée comme l'indicateur horizontal de progression intégré à l'épisode : l'utilisateur refuse explicitement de transformer les épisodes en défilement horizontal. Aucune nouvelle rangée de navigation ni nouvelle disposition d'épisodes n'est introduite.
