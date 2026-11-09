# AURORA TV — consignes pour les agents

Client Android TV (Xiaomi Box) du serveur AURORA (`~/Documents/Codex/2026-10-16-AuroraServer-Syno/aurora`, API `/api/v1`).
Lire d'abord [README.md](README.md) : décisions non évidentes et **interdits**.

## Commandes
- Build : `./gradlew :app:assembleDebug` ; tests : `./gradlew :app:testDebugUnitTest`
- Émulateur : AVD `GlassPlayerTV` (Google TV 1080p) ; le Mac y est joignable en `10.0.2.2`.
- Installation sur la box : adb en IPv6 link-local (README) — **demander à l'utilisateur avant**.

## Règles
- Design « Cinémathèque » partagé avec l'app macOS (`AURORA-GitHub/macos`) : toute retouche visuelle se fait des deux côtés.
- Écran de la box = 960 × 540 dp : vérifier qu'aucune largeur fixe ne dépasse.
- Toute évolution de l'API côté serveur se reflète ici (modèles `net/Models.kt`) et inversement.
- Identification TMDB/IMDb (25/10/2026) : contrat admin documenté dans `Models.kt` ; ces routes ne sont pas
  appelées par la TV. Les modèles catalogue/recherche/fiches restent inchangés (identifiants locaux).
- Fonds d’écran (25/10/2026, maquette approuvée) : Profil → Fond d’écran utilise `/backgrounds` et les
  images authentifiées du NAS (AURORA 0.1.11). Choix local par NAS, aperçu avant application ; Automatique
  restaure le film flouté. Ne pas effacer un choix lors d’une panne image ; garder le repli automatique.
- Navigation des dossiers (29/10/2026, maquette approuvée) : Films/Séries toujours ouverts à gauche, chacun
  dans un cadre visible façon macOS ; dossiers et compteurs du NAS, « Tous les dossiers » dans chaque groupe.
  Barre supérieure de dossiers supprimée, tri conservé. Grilles de 5 affiches par ligne, y compris « Tous ».
  Partager la sélection avec la grille ; préserver le retour de fiche et le fond personnel, sans repli de groupes.
  « Tous les dossiers » ne charge qu’une page par dossier ; au-delà, un bouton ouvre le dossier (toutes ses pages).
- Sélection au focus (29/10/2026) : pas de titre « Parcourir » ; catégories/dossiers/Tous affichent leur
  contenu au gain de focus sans OK, **différé de 200 ms** (le dernier survolé gagne ; entrer dans le contenu
  applique tout de suite, quitter la colonne sans y entrer annule).
  L’en-tête de catégorie **garde le dossier choisi** et n’est plus surligné en plus de lui ; sélection idempotente
  pour préserver la restauration de carte. Ne pas relancer un chargement par ligne traversée.
- Focus des vignettes (29/10/2026, maquette approuvée) : cadre crème-or épais et halo sur la carte réellement
  focalisée dans les grilles de droite ; garder cinq colonnes et la mémoire de focus. Épisodes toujours verticaux.
  Le « ⋯ » d’une affiche reste visible mais **n’est pas focusable** : ◀/▶ vont d’une affiche à la suivante,
  l’appui long sur OK ouvre Vu / Pas vu (comme Plex). Le « ⋯ » d’une ligne d’épisode reste atteignable par ▶.
- Position dans une série (29/10/2026, maquette approuvée) : la ligne d’épisode focalisée reprend le cadre or et
  le halo des affiches ; la barre de saison reste collée en haut pendant le défilement et affiche
  « Épisode 4 sur 10 · 3 vus ». Le « ⋯ » d’un épisode est **hors de la vignette**, à droite de la ligne :
  sur la photo il se confondait avec l’image et le résumé passait dessous. La ligne « X épisodes vus sur Y »
  reste (avancement de la série, pas une reprise). Poser le focus sur le chip
  de saison fait défiler la fiche d’environ 160 dp : revenir en haut de la liste après le focus, sinon la fiche
  s’ouvre sur une affiche coupée.
  La puce « Où j’en suis · Sxx Exx » de la barre amène sur la ligne du prochain épisode sans lancer la lecture ;
  la barre annonce cet épisode dès l’ouverture. Repli focusable si la fiche n’a ni fichier ni saison.
- Accueil à rangées (29/10/2026, validé) : panneau verre « Bienvenue » puis les rangées de `/home`
  (`Home.sections`) dans l’ordre du profil, une rangée vide n’est jamais affichée, et **jamais** `continue`
  ni `nextup` (retirés le 29/10 : la reprise passe par les dossiers « En cours » du NAS). Les cartes
  `collection` sont filtrées (identifiant de collection, pas d’œuvre). Aucun appel d’API en plus.
- Reprise : retirer « En cours » de la sidebar, conserver les dossiers NAS du même nom ; pas de rangée
  « Continuer à regarder » ni « Prochain épisode » à l’accueil. Progression horizontale
  directement sur l'épisode ; pas de bouton Reprendre global série ni Ma liste/Favori sur série/épisode.
- Vu : menu « ⋯ » toujours visible — sur l’affiche d’un film, à droite de la ligne pour un épisode, jamais sur
  une affiche de série (le « vu » se règle par épisode) ; appui long sur OK l’ouvre aussi. Chaque film/épisode
  atteint le seuil à 80 % de sa durée ; coordonner les écritures pour que le seuil différent du NAS ne remette
  pas l'élément en non vu ensuite.
- Erreurs : tout chargement en échec propose un bouton « Réessayer » focusable (`AppViewModel.retry`) ; ne jamais
  laisser la télécommande sans cible. Tri, recherche et défilement sont conservés par onglet.
- Lecteur (29/10/2026) : bouton « Suivant » dans le panneau, mode affiché (direct/remux/conversion), temps restant,
  OK sur la barre = lecture/pause, ◀ annule le compte à rebours. « Ouvrir dans VLC » ne ferme plus le lecteur.
- Fond continu (25/10/2026, maquette approuvée) : à l'accueil, une seule image sous sidebar et contenu ;
  sidebar avec voile sombre translucide. Préserver le repli automatique, sans seconde image recadrée.
- Disque USB (07/11/2026, maquette approuvée) : source locale dans la colonne (sous Séries), détectée à chaud
  (StorageManager). Lecture **directe** des volumes montés sous `/storage/…` (comme VLC), jamais de NAS ni de
  Storage Access Framework (le Formovie n'a pas de DocumentsUI : `ACTION_OPEN_DOCUMENT_TREE` plante). Accès par
  `MANAGE_EXTERNAL_STORAGE` (« Tous les fichiers »), accordé une fois dans les réglages. Navigateur de
  dossiers/vidéos, lecture locale ExoPlayer. Ne pas indexer le disque ni écrire dessus (aucune base). Couleurs
  uniquement via `Aurora`, grille de 5 tuiles, focus crème-or comme les affiches.
- Aucune suppression (`rm`) de données de l'utilisateur ; ce qui doit disparaître va à la Corbeille.
- Ne rien déclarer « testé sur la box » sans l'avoir fait.
- Toute modification de code met à jour README.md et ce fichier.
- Après modification : build + tests verts, puis revue par un agent distinct.
