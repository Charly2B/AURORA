# AURORA TV

App Android TV (Xiaomi Box) qui lit la médiathèque **AURORA** installée sur un ou plusieurs Synology.
Même contenu que l'interface web d'AURORA (`~/Documents/Codex/2026-10-16-AuroraServer-Syno/aurora`) :
on saisit l'IP du NAS, on se connecte avec son compte AURORA, et tout le reste vient du serveur.

| App de la box | Source | Rôle |
|---|---|---|
| **AURORA TV** (ici) | Serveur AURORA sur le Synology (port 8890) | Médiathèque perso : films/séries, reprise, vu/non vu |
| PlexSeries TV | Bibliothèques Plex | Parcourir et lire depuis Plex |
| glassplayer | Partages SMB/NAS | Lecture de fichiers |

## Ce que fait l'app

- **Interface « Cinémathèque »**, identique à l'app macOS (validée le 24/10/2026) : brun très sombre, crème et or,
  titres en serif. Colonne de gauche : Rechercher, Accueil, Films et Séries encadrés façon macOS, avec leurs dossiers toujours affichés
  et compteurs, Profil + NAS en pied. Liste défilante ; ◀ depuis le contenu y revient.
- **Accueil** (validé le 29/10/2026) : le panneau verre « Bienvenue sur AURORA » puis les **rangées du serveur**
  (ajouts récents par dossier, favoris, ma liste…) dans l'ordre des préférences du profil, sur l'image floutée
  d'un film tiré au hasard dans toute la médiathèque (une fois par NAS), ou sur le fond personnel choisi dans
  **Profil → Fond d’écran**. Le fond couvre toute la fenêtre, y compris la colonne de gauche, rendue translucide
  avec un voile sombre (maquette approuvée le 25/10). Une rangée vide n'apparaît jamais.
  **« Continuer à regarder » et « Prochain épisode » ne sont pas affichés** (retirés le 29/10 sur demande) :
  la reprise passe par les dossiers « En cours » du NAS. Une vignette ouvre la fiche (un épisode se lance) ;
  l'accueil ne démarre jamais une lecture tout seul.
- **Fond d’écran du NAS** (maquette validée le 25/10/2026) : galerie authentifiée AURORA, aperçu avant
  application, choix conservé localement pour chaque Synology. « Utiliser ce fond » ouvre l’accueil ;
  « Automatique » restaure le film flouté. Retour annule la sélection provisoire. Le fond personnel reste net.
- **Plusieurs Synology** : ajout par IP (port 8890 ajouté seul, IPv6 accepté), liste des NAS, changement à
  chaud. Une adresse déjà enregistrée est réutilisée (pas de doublon).
- **Connexion par code** (comme Plex) : la TV affiche un code et l'adresse `http://<nas>:8890/link` ; on le
  valide (puis confirme) depuis un navigateur déjà connecté et la TV se connecte seule. Suivi toutes les 2 s ;
  nouveau code seulement à l'expiration (une coupure réseau ne change pas le code affiché) ; demande gardée
  dans le ViewModel et réutilisée (sinon chaque aller-retour sur l'écran en créait une, et le NAS en refuse
  plus de 5 par adresse). Mot de passe AURORA en secours. Session chiffrée AES/GCM (clé AndroidKeyStore), une par NAS.
- **Films / Séries** : choix du dossier dans la colonne de gauche, plus « Tous les dossiers » qui affiche
  chaque dossier sous son titre. La barre de dossiers du haut est supprimée ; le tri Titre / Ajout récent /
  Année reste au-dessus du contenu, dans sa barre sombre au filet doré. Grilles de **5 vignettes plus grandes**
  espacées de 22 × 28 dp, chacune dans un **cadre verre** façon macOS (comme la TV Samsung). Un cadre crème-or épais,
  un halo et le titre mis en évidence suivent la vignette focalisée. Pastilles : épisodes non vus, vu, progression.
  Le tri, la recherche et la position de défilement sont conservés **par onglet** ; changer de dossier garde
  l'ancienne grille affichée pendant le chargement (plus de « Chargement… » à chaque survol de la colonne).
- **Fiche** : affiche, fond du film à droite, titre original, crédits, synopsis. Films : Lecture/Reprendre, Depuis le début, Vu/Non vu, Ma liste, Favori ;
  séries : « X épisodes vus sur Y », saisons en barre, sans Ma liste/Favori ni bouton Reprendre global redondant,
  **épisodes en liste verticale** (vignette à gauche, résumé à droite), reprise et progression horizontale sur l'épisode ;
  distribution en texte (« Acteur — Rôle »).
- **« Où j'en suis »** (29/10/2026) : dans la barre de saison, une puce « Où j'en suis · S02 E04 » amène sur la
  ligne du prochain épisode (défilement + focus) **sans lancer la lecture**. La barre annonce cet épisode dès
  l'ouverture de la fiche, plus seulement après avoir bougé le focus. Si la fiche n'a aucun bouton (film sans
  fichier, série sans saison indexée), un repli focusable évite une télécommande muette.
- **Savoir où l'on est dans une série** (maquette du 29/10/2026) : la ligne d'épisode focalisée reprend le repère
  des affiches (cadre crème-or de 3 dp, halo, n° et titre en or) ; la barre de saison **reste collée en haut**
  pendant le défilement et affiche « Épisode 4 sur 10 · 3 vus ». Le nom de la série n'y apparaît qu'une fois
  l'en-tête défilé.
- **Vu / Pas vu comme sur Plex** (et les apps Mac / Samsung) : rien sur la vignette d'un épisode non vu, coche
  verte s'il est vu. Sur un **film ou un épisode**, **appui long sur OK** ouvre Vu / Pas vu (état actuel coché et
  focalisé), Retour ferme la fenêtre. Sur un **épisode**, le « ⋯ » reste **hors de la vignette**, à droite de la
  ligne dans sa propre colonne : **▶** l'atteint, **◀** revient. Sur un **film**, le « ⋯ » reste visible en bas à
  droite de l'affiche mais **ne prend plus le focus** : ◀/▶ passent d'une affiche à la suivante (un appui par carte).
  Aucun « ⋯ » sur une affiche de série : « vu » se règle par épisode, jamais pour toute la série d'un coup.
- **Seuil automatique** : chaque film et chaque épisode est marqué vu à **80 % de sa durée**, seuil inclus.
  Les enregistrements de progression sont coordonnés pour préserver cet état malgré le seuil différent du NAS.
  La sidebar ne contient plus les trois reprises « En cours » ; les dossiers NAS Films en cours/Series en cours restent disponibles.
  L'accueil n'affiche plus non plus « Continuer à regarder » ni « Prochain épisode » (29/10/2026).
- **« Tous les dossiers »** : un panneau arrondi façon fenêtre macOS par dossier (barre de titre à pastilles),
  découpé en éléments de `LazyColumn` (barre, rangées de 5 à parts égales, bas arrondi) pour rester paresseux.
  Une seule page de 500 œuvres est demandée par dossier ; au-delà, un bouton « Voir les N films de « dossier » »
  ouvre le dossier (qui, lui, enchaîne toutes les pages). Avant, chaque survol de la colonne rapatriait toute
  la bibliothèque.
- **Erreurs réseau** : tout chargement en échec affiche un bouton **« Réessayer »** focusable (la télécommande
  ne reste plus sans cible) et relance accueil, dossiers, fiche et bibliothèques (`AppViewModel.retry`).
- **Recherche** du serveur (accents, pluriels et petites fautes tolérés, lettres inversées comprises) :
  résultats proches après les résultats exacts, sans doublons ; « résultats approchants » si aucun résultat exact.
  Entrer dans l'onglet place directement le focus dans le champ.
- **Lecteur ExoPlayer** :
  - lecture directe si la box sait décoder, sinon remux ou conversion par le NAS (HLS) ;
  - reprise, position envoyée toutes les 10 s et à la sortie ;
  - enchaînement de l'épisode suivant avec compte à rebours (◀ l'annule) ;
  - bouton **« Suivant »** dans le panneau, sans attendre la fin de l'épisode ;
  - mode de lecture affiché (**lecture directe / remux / conversion**) et **temps restant** à côté de la durée ;
  - pistes audio (français par défaut) et sous-titres ;
  - repli automatique si la box échoue, puis « Ouvrir dans VLC » en dernier recours (lancer VLC ne ferme plus
  le lecteur : au retour, le panneau d'erreur est toujours là).
- **Disque USB** (07/11/2026, maquette approuvée) : un disque branché sur le port USB du Formovie est détecté
  à chaud et apparaît dans la colonne de gauche, sous Séries, comme **source locale** (une ligne par partition).
  La lecture est **directe** : le volume monté sous `/storage/…` est parcouru par le système de fichiers (comme
  VLC), avec l'autorisation « Tous les fichiers » (`MANAGE_EXTERNAL_STORAGE`). Le navigateur liste **dossiers
  puis vidéos** (vignette et durée extraites du fichier, repli façon affiche), fil d'Ariane, Retour remonte d'un
  dossier, grille de 5 tuiles au focus crème-or. La lecture est locale (ExoPlayer, sans NAS) : lecture/pause,
  ±10/30 s, temps restant et arrêt.

## Commandes à la télécommande

| Touche | Effet |
|---|---|
| OK pendant la lecture | Affiche le panneau (focus sur la barre de progression) |
| OK sur la barre de progression | Lecture / pause |
| ◀ / ▶ | −10 s / +30 s (panneau masqué ou sur la barre) |
| ▼ dans le panneau | Boutons : −10 s, lecture/pause, +30 s, sous-titres, audio, suivant, arrêt |
| ◀ pendant le compte à rebours | Annule l'enchaînement de l'épisode suivant |
| Retour | Ferme le panneau, sinon quitte le lecteur (position enregistrée) |
| Appui long sur OK (affiche / ligne d'épisode) | Ouvre Vu / Pas vu |
| OK sur un champ texte | Ouvre le clavier ; ▲/▼ sortent du champ |
| ▲ / ▼ dans les dossiers de gauche | Affiche les vignettes du dossier focalisé sans appuyer sur OK ; ▶ entre dans le contenu |

Le panneau se masque après 6 s, **sauf en pause**.

## Décisions non évidentes (et pourquoi)

- **Fond continu à l’accueil** : une seule couche d’image sous la rangée sidebar/contenu, avec le repli
  automatique existant. La sidebar superpose seulement son voile sombre translucide ; une seconde image
  recadrée séparément produirait un raccord. Le panneau Bienvenue et les autres onglets gardent leur place.
- **Dossiers dans la colonne de gauche** (mise à jour approuvée le 29/10/2026) : le choix de dossier est partagé
  avec la grille, séparément pour Films et Séries ; les noms et nombres viennent des bibliothèques du NAS.
  Les deux groupes restent ouverts en permanence, entourés d’un cadre visible façon fenêtre macOS, avec
  « Tous les dossiers » dans chacun. Le défilement vertical permet d’atteindre les longues listes sur la TV ;
  Profil reste en pied. Ne pas réintroduire le repli des groupes ni le sélecteur de dossiers supérieur.
  Les dossiers, les titres de catégories et « Tous les dossiers » affichent leur contenu dès le gain de focus,
  sans OK. Une sélection déjà active ne réinitialise pas la mémoire de la carte. Le titre « Parcourir » est retiré.
  Le gain de focus est **différé de 200 ms** (dernier dossier survolé gagnant ; entrer dans le contenu applique
  tout de suite le dossier survolé, le quitter sans y entrer l'annule) : traverser la liste pour atteindre
  Profil ne déclenche plus un chargement par ligne. L'en-tête « Films » / « Séries » **garde le dossier choisi**
  et n'est plus surligné en plus de lui : un seul repère.
- **Accueil à rangées** (validé le 29/10/2026) : les rangées viennent telles quelles de `/home` (`Home.sections`,
  ordre et visibilité réglés dans le profil AURORA), filtrées sur celles qui ont des œuvres **et sur les types
  que la TV sait ouvrir** (film, série, épisode). La rangée « Collections » est donc ignorée : le serveur y met
  des cartes `kind = "collection"` dont l'identifiant est celui de la collection, pas d'une œuvre — les ouvrir
  affichait la fiche d'un film au hasard (et « Vu » écrivait sur une œuvre sans rapport). Les sections
  `continue` et `nextup` sont **écartées** (`HiddenHomeRails`) : la reprise reste dans les dossiers du NAS, et
  l'accueil ne lance pas de lecture directement. Aucun nouvel appel d'API ; une panne de `/home` n'empêche pas
  la colonne et les onglets de marcher.
- **État par onglet** : `SaveableStateHolder` clé = nom de l'onglet, à l'intérieur de l'état de l'écran `Main`.
  Passer par Recherche puis revenir sur Films garde le tri, la requête et le défilement ; un retour de fiche
  continue d'être protégé par l'état de l'écran (clé = NAS + écran).
- **« ⋯ » des vignettes** (29/10/2026) : sur une affiche, le « ⋯ » reste visible mais n'est **plus focusable** ;
  l'appui long sur OK ouvre Vu / Pas vu, ce qui rend ◀/▶ aux affiches (une carte = un appui). Sur une ligne
  d'épisode, le « ⋯ » reste dans sa colonne et s'atteint par ▶, comme validé le 29/10. Conserver le focus au
  retour et afficher l'erreur si la modification échoue.
- **Galerie de fonds** : `GET /api/v1/backgrounds` renvoie `{images:[...]}` ; les images sont servies par
  `/api/v1/backgrounds/{id}/image`, avec le même cookie AURORA que les autres images. La liste requiert
  AURORA 0.1.11 ; aucun changement de `/home`, de réglage global du NAS ou d’import depuis la TV.
  L’identifiant est mémorisé par NAS dans les préférences de cette app. Une image indisponible laisse
  apparaître le fond automatique sans effacer le choix ; une galerie vide, une ancienne API (404) ou
  une panne réseau affiche un état explicite et laisse le retour à Automatique accessible.
- **Identification TMDB/IMDb côté NAS** (25/10/2026) : les routes d'administration peuvent proposer un
  candidat avec `imdbId` et `id: 0`, puis recevoir `tmdbId` **ou** `imdbId` dans `/admin/items/{id}/match`.
  La TV n'appelle pas ces routes : ses modèles `Card`, `Detail` et `SearchResults` restent compatibles,
  avec les mêmes identifiants locaux. Aucun DTO d'administration ni écran supplémentaire nécessaire.
- **Écran = 960 × 540 dp** sur la Xiaomi (1920×1080, densité 320, mesuré le 24/10/2026) : colonne de gauche de
  212 dp, en-tête et fiche dimensionnés pour tenir dans ~750 dp (une colonne de 860 dp débordait).
- **Tirage unique par NAS** (`AppViewModel.loadFeatured`) : rang au hasard parmi les comptes de `/libraries`, puis
  `list(limit = 1)` ; les rechargements (retour du lecteur) gardent le titre. Héros du serveur en repli.
- **Flou du fond d'accueil** (`ui/components/Blur.kt`) : `Modifier.blur` exige l'API 31 (box en Android 11) et
  agrandir seule une image 64 × 36 donnait des pavés ; Coil réduit à 160 × 90 puis `BoxBlur` fait 3 passes de
  moyenne glissante. Pas d'ombre sous le panneau verre : elle se voit à travers (rectangle sombre).
- **Tons sable clairs** (`Mist`, `Ash` plus clairs que sur Mac) : même raison que les gris clairs d'avant (texte à
  contraste élevé des téléviseurs).
- **Capacités mesurées, pas supposées** (`player/Capabilities.kt`) : décodeurs **matériels** présents
  (`MediaCodecList`, `isHardwareAccelerated`), passthrough audio, hauteur max de ces décodeurs. Les décodeurs
  logiciels (`c2.android.*`, `OMX.google.*`) sont exclus : ils annoncent AV1/HEVC jusqu'en 4K mais saccadent
  sur une box **sans lever d'erreur**, donc sans repli. H.264 reste annoncé (format des conversions du NAS).
  ExoPlayer lit MKV/MP4/TS/AVI : aucun remux pour le conteneur, contrairement au navigateur.
- **Repli du lecteur selon la cause** :
  - si c'est le **décodeur vidéo** qui échoue (ex. HEVC 10 bits sur une puce 8 bits), on demande une
    **conversion**, car un remux recopierait la même vidéo ;
  - sinon (conteneur, audio), on demande un **remux** ;
  - puis message d'erreur avec VLC.
  - Le mode imposé est **conservé pour la suite de la lecture** (saut lointain, changement de piste) :
    sinon le NAS rechoisirait la lecture directe qui vient d'échouer.
- **VLC** : le lecteur externe ne transmet pas le cookie de session ; l'app demande au NAS un **lien signé**
  (`external: true` → `signedUrl`, valable 4 h, révoqué si le mot de passe change).
- **HLS = sessions du NAS** :
  - saut dans la partie déjà produite = saut local ;
  - au-delà, ou retour de plus de 60 s (segments purgés) = nouvelle session, après 0,5 s sans autre appui :
    les appuis rapprochés s'additionnent (position visée en attente) et une seule session est ouverte ;
  - chaque demande `/play` porte un numéro de génération : une réponse dépassée est ignorée et sa session
    arrêtée ;
  - la piste audio choisie est reportée sur les sessions suivantes ;
  - session fermée par le NAS (pause de plus de 90 s) ou coupure réseau : une nouvelle session, une fois ;
  - la playlist EVENT sans fin est un « direct » pour ExoPlayer, qui démarrerait au bord produit (minutes
    plus loin en remux) : on force `setMediaItem(item, 0)` et une vitesse fixe de 1,0
    (`LiveConfiguration`).
- **Fermeture du lecteur** : progression et arrêt de session partent en `GlobalScope` (le scope de l'écran
  est déjà annulé) ; un `/play` en vol est non annulable puis arrête lui-même sa session.
- **VLC ne ferme plus le lecteur** (29/10/2026) : l'intent VLC met l'activité en `ON_STOP` ; sans le drapeau
  `leavingForVlc`, l'app appelait `stop()` et l'on revenait de VLC sur la fiche, panneau d'erreur disparu.
  Le drapeau retombe au `ON_START` ; une vraie sortie (touche Accueil) coupe toujours la lecture.
- **« Suivant » dans le panneau** : demandé à la volée (`/items/{id}/next`), puis `vm.replace(Player(...))`.
  Aucune requête quand on ne s'en sert pas ; une fin d'épisode garde le compte à rebours existant.
- **Piste française** appliquée une seule fois par média : un choix manuel n'est plus écrasé.
- **Bibliothèques complètes** : le NAS plafonne à 500 œuvres par page ; l'app enchaîne les pages.
- **Retour d'une fiche** :
  - dossier, tri et défilement conservés (`rememberSaveableStateHolder`, clé = NAS + écran ; les états des
    écrans sortis de la pile sont purgés) ;
  - focus rendu à la carte ouverte (`CardFocusMemory`, clé préfixée par la rangée car une même œuvre peut
    figurer dans plusieurs rangées), sinon à l'onglet après 1,5 s.
- **Pages dédoublonnées** : l'ordre peut glisser entre deux pages ; une clé en double ferait planter la grille.
- **Pas d'arrêt anticipé de l'ancienne session HLS** : le NAS l'arrête lui-même au `/play` suivant.
- **Retour à la racine quitte l'app** (le `BackHandler` n'est actif que s'il reste un écran ou un onglet).
- **Session révoquée** (401 sur une requête, ex. mot de passe changé) : retour à l'écran de connexion de ce
  NAS. Corps de réponse coupé ou non-JSON (proxy, autre service sur le port) : message, jamais de plantage.
- **`key(screen)` à la racine** (`AuroraTvApp.kt`) : sans elle, passer d'un épisode au suivant (même type
  d'écran au même endroit) réutilisait l'état — et le lecteur — de l'épisode précédent : le compte à
  rebours restait bloqué à 0.
- **Champ texte = bouton tant qu'on n'édite pas** (`Field`, `SetupScreens.kt`) : l'IME Android TV s'ouvre
  dès qu'un éditeur prend le focus (même avec `showKeyboardOnFocus = false`) et capte alors la
  télécommande. OK passe en édition ; quand le focus part, le champ redevient bouton.
- **▲/▼ explicites dans un champ texte** : un `TextField` les garde pour lui, la télécommande ne pouvait plus
  atteindre « Continuer » / « Entrer ».
- **Code plutôt que mot de passe** : sur la box, la connexion par mot de passe a échoué en réel (compte DSM
  confondu avec le compte AURORA, saisie au clavier TV). Le code évite toute saisie ; le nom de l'appareil
  envoyé (`Settings.Global` « device_name », sinon le modèle) s'affiche sur /link lors de la validation.
- **Retour depuis Connexion → liste des NAS**, pas le formulaire d'ajout (qui recréait un doublon).
- **Disque USB local** (07/11/2026) : la lecture USB ne passe **jamais par le NAS** — c'est un fichier local lu
  tel quel par ExoPlayer (pas de session HLS, de remux, de conversion ni de reprise/progression). L'accès au
  contenu est **direct** par le point de montage (`StorageVolume.getDirectory()` → `/storage/…`), comme VLC :
  le Formovie n'embarque pas de sélecteur de documents (`DocumentsUI`), donc le Storage Access Framework
  (`ACTION_OPEN_DOCUMENT_TREE`) y **plante** — il est écarté au profit de `MANAGE_EXTERNAL_STORAGE`
  (« Tous les fichiers », accordé une fois dans les réglages ; aucun intent direct pour y accéder sur ce modèle,
  on ouvre donc `android.settings.SETTINGS`). La détection à chaud repose sur
  `StorageManager.registerStorageVolumeCallback` (Android 11+). Le disque n'est ni scanné ni indexé : on parcourt
  les dossiers à la demande (aucune base, aucune métadonnée TMDB). Chaque partition NTFS apparaît comme un volume
  distinct dans la colonne. Revenir du lecteur rouvre le navigateur à la racine du volume.

## Interdits (ne pas défaire)

- Jamais de champ texte dans une rangée horizontale (il capte ◀/▶).
- Ne pas retirer `key(screen)`, le mode imposé (`forcedMode`), la génération `/play` ni `setMediaItem(item, 0)`
  en HLS.
- Ne pas réintégrer les décodeurs logiciels dans les capacités.
- Ne pas remettre un `OutlinedTextField` focalisable en permanence : utiliser `Field`.
- Ne jamais mettre le mot de passe ou le cookie dans une URL ; seul le lien signé du NAS sert à VLC.
- Ne pas remettre de repère sur un épisode non vu, ni de bouton « Vu » global sur une série.
- Ne pas rendre le « ⋯ » d'une **affiche** focusable : ◀/▶ doivent passer d'une affiche à la suivante, et
  l'appui long sur OK ouvre Vu / Pas vu. Sur une **ligne d'épisode**, il reste atteignable par ▶.
- Ne pas recharger la bibliothèque à chaque ligne de la colonne survolée (anti-rebond 200 ms) ni faire de
  l'en-tête de catégorie un second « Tous les dossiers ».
- Ne pas réintroduire de `remember { vm.servers }` figé dans l'écran des NAS : la liste est un état du ViewModel.
- Ne pas afficher la rangée « Collections » de `/home` : ses cartes portent un identifiant de collection, pas
  d'œuvre (fiche au hasard, « Vu » écrit sur un film sans rapport). `homeRails` filtre film/série/épisode.
- Ne pas figer la barre de saison hors du défilement (elle doit rester collante) ni remplacer le halo de la
  ligne d'épisode par un simple liseré : c'est ce qui rend la navigation lisible à trois mètres.
- Couleurs uniquement via `Aurora` (`ui/theme/Theme.kt`), comme `AuroraTheme` côté Mac : les deux apps restent identiques.

## Fichiers

| Fichier | Rôle |
|---|---|
| `AuroraApplication.kt` | Conteneur : OkHttp (cookie de session), magasin des NAS, chargeur d'images Coil |
| `MainActivity.kt` | Activité unique (leanback) |
| `data/ServerStore.kt` | NAS enregistrés ; `normalizeAddress()` (port 8890 par défaut, IPv6) |
| `security/TokenStore.kt` | Session chiffrée AES/GCM, clé AndroidKeyStore, une par NAS |
| `net/AuroraClient.kt` | API AURORA (`/api/v1`) : connexion, accueil, bibliothèques, fiches, lecture, progression |
| `net/Models.kt` | Modèles JSON (kotlinx.serialization) |
| `net/Backgrounds.kt` | Métadonnées JSON de la galerie de fonds du NAS |
| `data/BackgroundStore.kt` | Fond personnel mémorisé localement par NAS |
| `player/Capabilities.kt` | Capacités de décodage de la box envoyées au NAS |
| `usb/UsbStore.kt` | Volumes USB amovibles, autorisation SAF mémorisée par volume |
| `usb/UsbSupport.kt` | Aides pures : détection vidéo, tailles/durées lisibles |
| `ui/AppViewModel.kt` | Navigation (pile d'écrans), NAS courant, connexion |
| `ui/AuroraTvApp.kt` | Racine : écran courant, touche Retour |
| `ui/SetupScreens.kt` | Liste des NAS, ajout, connexion, composant `Field` |
| `ui/MainScreen.kt` | Colonne de gauche, Accueil (rangées du serveur, verre), Films / Séries, Recherche, Profil |
| `ui/HomeRails.kt` | Rangées d'accueil retenues (œuvres ouvrables, dans l'ordre du serveur) |
| `ui/LibraryNavigation.kt` | Sélections par catégorie/NAS, repli des dossiers absents, pagination |
| `ui/BackgroundPicker.kt` | Galerie du NAS, aperçu et choix du fond d’accueil |
| `ui/Featured.kt` | Tirage du film du fond d'accueil (`featuredSlot`), ligne de crédits de la fiche (`creditLine`) |
| `ui/DetailScreen.kt` | Fiche film/série, barre de saison collante, épisodes |
| `ui/PlayerScreen.kt` | Lecteur, panneau, pistes, épisode suivant, replis |
| `ui/UsbScreen.kt` | Source « Disque USB » : états, navigateur de fichiers, tuiles |
| `ui/UsbPlayer.kt` | Lecture locale d'une vidéo du disque (ExoPlayer, sans NAS) |
| `ui/components/Components.kt` | Affiches (cadre verre), cartes larges, rangées, boutons, pastilles, `SegmentBar`, `Eyebrow` |
| `ui/components/Blur.kt` | `BoxBlur` : flou Coil pour le fond d'accueil (Android 11) |
| `ui/theme/Theme.kt` | Palette « Cinémathèque » et polices (mêmes valeurs que l'app Mac) |
| `app/src/test/.../AddressTest.kt` | Tests de `normalizeAddress` |
| `app/src/test/.../FeaturedTest.kt` | Tests du tirage à la une et de la ligne de crédits |
| `app/src/test/.../HomeRailsTest.kt` | Tests des rangées d'accueil (ordre conservé, rangées vides retirées) |
| `app/src/test/.../UsbSupportTest.kt` | Tests des aides USB (détection vidéo, tailles, durées) |

## Stack

Kotlin 2.3 · Compose for TV (`tv-material` 1.1) · Media3/ExoPlayer 1.10 (+ HLS) · OkHttp 5 ·
kotlinx.serialization · Coil 3. minSdk 24, compileSdk 37, JDK 17. Mêmes versions que PlexSeries TV.

## Build

```bash
./gradlew :app:assembleDebug        # APK : app/build/outputs/apk/debug/app-debug.apk
./gradlew :app:testDebugUnitTest    # tests unitaires
```

`local.properties` doit pointer vers le SDK : `sdk.dir=/Users/<toi>/Library/Android/sdk`

## Tests effectués

Disque USB, le 07/11/2026 : build debug et **34 tests unitaires** réussis (dont 3 nouveaux sur les aides USB :
détection vidéo, tailles et durées lisibles). Revue par un agent distinct, puis corrections (focus du navigateur
et du lecteur, panneau de commandes du lecteur USB inatteignable). Installée sur le Formovie et **vérifiée en
réel** : deux partitions NTFS détectées (« Disque1-Externe », « Disque2-Externe »), le navigateur liste leur
contenu (`System Volume Information`, `The.Odyssey.2026.2160p.mkv` avec « 2 h 52 · 83,3 Go » — vignette et durée
extraites du fichier). Deux crashs corrigés en route : l'ordre d'initialisation du callback USB (`by lazy`
déclaré après le bloc `init` qui l'utilisait), et `ACTION_OPEN_DOCUMENT_TREE` qui plantait faute de DocumentsUI
sur le Formovie — remplacé par la lecture directe des volumes montés via `MANAGE_EXTERNAL_STORAGE` (accordée par
`adb appops` pour le test ; sinon le panneau « Autoriser l'accès au stockage » ouvre les réglages). Détection,
parcours des partitions NTFS et lecture locale confirmés sur le Formovie.

Navigation facilitée, le 29/10/2026 : 31 tests unitaires et build debug réussis (dont 6 nouveaux : 5 sur les
rangées d'accueil, les collections et les rangées de reprise retirées, 1 sur la pagination). Revue indépendante
distincte, puis
contre-revue des correctifs (focus « à usage unique » après « Réessayer »/« Voir les N », drapeau VLC réarmé
au retour, « Dernier » seulement sur une vraie fin de série, « Annuler » focalisé dans le retrait d'un NAS).
Corrigés aussi : collections, focus du champ de recherche au retour d'une fiche, saison de l'épisode annoncé,
focus après retrait d'un NAS. Rangées « Continuer à regarder » et « Prochain épisode » **retirées de l'accueil**
à la demande de l'utilisateur (reprise par les dossiers « En cours » du NAS). Installée sur la **Xiaomi** et le
**Formovie** ; SHA-256 de l'APK installé identique à la build sur les deux : `eb390197…`.
Sur la **Xiaomi** (captures) : accueil avec le panneau Bienvenue et les rangées du serveur (dont « Films en cours »
et « Films-4K », sans rangée de reprise) ; colonne sans double surlignage (en-tête non sélectionné, dossier seul) ;
grille « Tous les
dossiers » à 5 colonnes et une page par dossier ; appui long sur OK → fenêtre Vu / Pas vu, puis Retour → focus
rendu à l'affiche ; ▶ passe d'une affiche à la suivante sans passer par « ⋯ » ; onglet Recherche → focus dans le
champ ; fiche série → puce « Où j'en suis · S01 E01 » → focus sur la ligne de l'épisode, barre de saison collante
« Épisode 1 sur 10 · 0 vu ». Sur le **Formovie** : installation, lancement, accueil à rangées et focus sur
« Accueil » vérifiés (le NAS y met ~20 s à répondre, contre ~8 s sur la Xiaomi).
Non vérifiés sur appareil : le repli de focus après « Réessayer » (aucune panne réseau provoquée), le bouton
« Voir les N … » (aucun dossier de plus de 500 œuvres ouvert), la fenêtre de retrait d'un NAS, et le retour du
focus après lecture (aucune lecture lancée pour ne pas modifier la progression réelle).

Position dans une série, le 29/10/2026 : build debug et 25 tests unitaires réussis (dont 8 sur la progression
et le seuil de 80 %), revue distincte en deux passes, puis contre-revue des correctifs. Installée sur la
**Xiaomi** et le **Formovie** ; SHA-256 de l’APK installé identique à la build sur les deux.
Sur la **Xiaomi** (captures) : fiche « Agent Kim Reactivated » ouverte sur son titre avec le chip de saison
focalisé, barre de saison collante en haut affichant « Épisode 2 sur 10 · 0 vu », ligne d’épisode sous cadre
crème-or et halo, « ⋯ » atteint par ▶, fenêtre Vu / Pas vu ouverte (état coché) puis refermée par Retour, ◀
ramène sur la ligne. Les « ⋯ » ont ensuite été retirés des affiches de séries (films et épisodes conservés),
vérifié par capture : grille Séries sans pastille, grille Films avec, ▶ passe d’une affiche de série à la
suivante, épisodes toujours avec leur « ⋯ » ; APK réinstallé sur les deux appareils, SHA-256 `ec4c5a4…`
identique à la build.
Puis le « ⋯ » d’un épisode est passé hors de la vignette, à droite de la ligne (maquette
`2026-10-29-episode-actions.html`, « toujours visible ») : vérifié par capture sur la Xiaomi — bouton visible
sur chaque ligne, ▶ depuis la ligne l’atteint (cercle or), OK ouvre Vu / Pas vu, Retour puis ◀ ramènent sur la
ligne, ▼ passe à l’épisode suivant. Réinstallé sur les deux appareils, SHA-256 `70d9b4e…` identique.
Le focus au retour du lecteur n’a pas été vérifié sur appareil : aucune lecture n’a été
lancée pour ne pas modifier la progression réelle. Formovie en veille (`mWakefulness=Asleep`) : processus et
activité visibles, capture noire, rendu non vérifié.

Cinq affiches et groupes toujours ouverts, le 29/10/2026 : 17 tests unitaires et build debug réussis,
revue distincte validée. Installée et lancée sur Xiaomi et Formovie ; empreintes des APK identiques à la build.
Capture Xiaomi : dossiers affichés et cadres macOS vérifiés. Formovie en veille ; rendu non vérifié.
Les cinq colonnes, la dernière rangée et les indices de focus sont contrôlés par lecture/revue du code ;
parcours complet de grille à la télécommande non exécuté sur appareil.

Fond continu derrière la sidebar, le 25/10/2026 : build et 17 tests réussis, revue distincte validée.
Installée et lancée sur Xiaomi et Formovie, empreintes des APK installés identiques à la build. Capture
Xiaomi : continuité du fond et voile de la sidebar vérifiés. Formovie en veille (`Display Power: OFF`) :
processus et activité vérifiés, capture noire ; rendu non vérifié sur le projecteur.

Navigation des dossiers, le 25/10/2026 : 17 tests unitaires réussis, dont 6 nouveaux sur les sélections
Films/Séries, les compteurs par type, les dossiers absents et le changement de NAS ; build debug réussi.
Installée et lancée sur Xiaomi et Formovie après demande explicite ; empreinte SHA-256 de l’APK installé
identique à la build sur les deux appareils. Parcours complet à la télécommande non vérifié.

Galerie de fonds, le 25/10/2026 : 11 tests unitaires réussis et build debug réussi. Contrat JSON,
authentification liste/images, erreurs réseau/404/401/JSON, choix local et isolation entre NAS vérifiés.
Installation de cette modification demandée explicitement sur la Xiaomi et le Formovie le même jour ;
mise à jour et lancement effectués sur les deux. Rendu du sélecteur et télécommande non vérifiés.

Sur l'émulateur Google TV (API 36, 1920×1080), contre un serveur AURORA local (`10.0.2.2`) :

- ajout du NAS, connexion, déconnexion et reconnexion ;
- accueil, onglets par dossier, vue « Tous les dossiers », recherche ;
- fiche série, lecture, reprise, ◀/▶, pause, Retour ;
- épisode suivant automatique ;
- HEVC 10 bits non décodable par l'émulateur → conversion par le NAS, reprise HLS au bon point (40 s) ;
- Retour : onglet → Accueil → sortie de l'app ; retour d'une fiche avec dossier et focus conservés.

Sur la **Xiaomi Box** (MiTV-AFKR0, Android 11) le 18/10/2026 : installation, connexion au Synology
(192.168.1.37, par mot de passe) et accueil avec le contenu réel vérifiés ; la session survit à la mise à
jour de l'app. Connexion par code sur la box et lecture : **non vérifiées** (le code exige AURORA 0.1.6 sur
le NAS).
Sur le **Formovie Theater** (L206FGN_X2, Android 11) le 18/10/2026 : installation et lancement vérifiés
(fenêtre active, aucune erreur dans `logcat`).

Interface « Cinémathèque », le 24/10/2026 : installée sur la **Xiaomi** et le **Formovie** ; accueil (colonne de
gauche, À la une), Films et Séries vérifiés par capture sur la Xiaomi, accueil vérifié par capture sur le
Formovie (la capture `screencap` vers un fichier puis `adb pull` fonctionne). Fiche et lecteur : **non vérifiés**
visuellement.
Le 25/10/2026 sur la **Xiaomi** (captures) : accueil « Bienvenue sur AURORA » en verre, grille Films à cadre verre,
fiche série (Kill Jackie) sans bouton Vu, épisodes en liste avec résumé et coches vertes, « ⋯ » atteint par ▶,
fenêtre Vu / Pas vu ouverte puis refermée par Retour. Installée aussi sur le Formovie (lancement vérifié).

## Installer sur la Xiaomi Box

Même méthode que PlexSeries TV : **IPv6 link-local**, depuis le Mac connecté au **Wi-Fi** du même réseau que la
box (TP-Link, 192.168.68.x). Le 24/10/2026 la box était aussi joignable en IPv4 (`192.168.68.126:5555`, trouvée par `dns-sd -G v4v6`). Si le Mac est seulement en Ethernet (192.168.1.x), la box est invisible : ni
mDNS ni ping6 (constaté le 18/10/2026) — rallumer le Wi-Fi du Mac.

```bash
dns-sd -B _googlecast._tcp
adb connect "[fe80::…%en1]:5555"
adb -s "[fe80::…%en1]:5555" install -r app/build/outputs/apk/debug/app-debug.apk
```

Détails (résolution de l'adresse, serveur adb figé) : README de PlexSeries TV.

### Formovie Theater (projecteur, Android TV 11)

Sur le réseau principal (192.168.1.x, Ethernet du Mac), joignable directement en IPv4 ; il annonce son adb en
mDNS (`dns-sd -B _adb._tcp` → `Android.local.:5555`, constaté le 18/10/2026 en `192.168.1.128`) :

```bash
adb connect 192.168.1.128:5555
adb -s 192.168.1.128:5555 install -r app/build/outputs/apk/debug/app-debug.apk
```

Sa capture `adb exec-out screencap` est polluée de lignes de débogage (« open mma dev failed ») ; en revanche
`adb shell screencap -p /sdcard/Download/x.png` puis `adb pull` donne une image correcte (24/10/2026).

**Réseau, non vérifié** : la box est sur `192.168.68.0/24` derrière le TP-Link. Le Mac ne la joint pas en IPv4
(d'où l'IPv6), mais le sens box → Synology passe par la sortie du TP-Link et devrait fonctionner. Si
« ne répond pas » s'affiche avec la bonne IP, tester d'abord ce chemin (navigateur de la box sur
`http://<ip-du-nas>:8890`) avant de conclure à un bug de l'app.
