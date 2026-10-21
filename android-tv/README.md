# AURORA TV

App Android TV (Xiaomi Box) qui lit la médiathèque **AURORA** installée sur un ou plusieurs Synology.
Même contenu et mêmes règles que l'interface web d'AURORA (`~/Documents/Codex/2026-10-16/aurora`) :
on saisit l'IP du NAS, on se connecte avec son compte AURORA, et tout le reste vient du serveur.

| App de la box | Source | Rôle |
|---|---|---|
| **AURORA TV** (ici) | Serveur AURORA sur le Synology (port 8890) | Médiathèque perso : films/séries, reprise, vu/non vu |
| PlexSeries TV | Bibliothèques Plex | Parcourir et lire depuis Plex |
| glassplayer | Partages SMB/NAS | Lecture de fichiers |

## Ce que fait l'app

- **Plusieurs Synology** : ajout par IP (port 8890 ajouté seul, IPv6 accepté), liste des NAS, changement à
  chaud. Une adresse déjà enregistrée est réutilisée (pas de doublon).
- **Connexion par code** (comme Plex) : la TV affiche un code et l'adresse `http://<nas>:8890/link` ; on le
  valide (puis confirme) depuis un navigateur déjà connecté et la TV se connecte seule. Suivi toutes les 2 s ;
  nouveau code seulement à l'expiration (une coupure réseau ne change pas le code affiché) ; demande gardée
  dans le ViewModel et réutilisée (sinon chaque aller-retour sur l'écran en créait une, et le NAS en refuse
  plus de 5 par adresse). Mot de passe AURORA en secours. Session chiffrée AES/GCM (clé AndroidKeyStore), une par NAS.
- **Accueil** : en-tête « Ce soir à l'affiche », Continuer à regarder, Prochain épisode, puis **une rangée par
  dossier** (« Films en cours », « Séries 2021 »…) — les dossiers ne sont jamais mélangés.
- **Films / Séries** : un onglet par dossier, plus « Tous les dossiers » qui affiche chaque dossier sous son
  titre ; tri Titre / Ajout récent / Année. Pastilles : épisodes non vus, vu, progression.
- **Fiche** : affiche, fond, synopsis, Lecture/Reprendre, Depuis le début, Vu/Non vu, Ma liste, Favori ;
  séries : saisons et épisodes avec reprise.
- **Retirer de Continuer à regarder** : appui long (OK maintenu) sur une carte Continuer / Prochain épisode →
  fenêtre de confirmation (`Dialog` : le focus y reste piégé), focus initial sur « Annuler » (l'appui long
  peut se terminer dessus), erreur affichée dans la fenêtre. Fichier et progression intacts.
- **« Tous les dossiers »** : un panneau arrondi façon fenêtre macOS par dossier (barre de titre à pastilles),
  découpé en éléments de `LazyColumn` (barre, rangées de 7 à parts égales, bas arrondi) pour rester paresseux.
- **Recherche** du serveur (accents et fautes tolérés, « résultats approchants »).
- **Lecteur ExoPlayer** :
  - lecture directe si la box sait décoder, sinon remux ou conversion par le NAS (HLS) ;
  - reprise, position envoyée toutes les 10 s et à la sortie ;
  - enchaînement de l'épisode suivant avec compte à rebours ;
  - pistes audio (français par défaut) et sous-titres ;
  - repli automatique si la box échoue, puis « Ouvrir dans VLC » en dernier recours.

## Commandes à la télécommande

| Touche | Effet |
|---|---|
| OK pendant la lecture | Affiche le panneau (focus sur la barre de progression) |
| ◀ / ▶ | −10 s / +30 s (panneau masqué ou sur la barre) |
| ▼ dans le panneau | Boutons : −10 s, lecture/pause, +30 s, sous-titres, audio, arrêt |
| Retour | Ferme le panneau, sinon quitte le lecteur (position enregistrée) |
| OK sur un champ texte | Ouvre le clavier ; ▲/▼ sortent du champ |

Le panneau se masque après 6 s, **sauf en pause**.

## Décisions non évidentes (et pourquoi)

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

## Interdits (ne pas défaire)

- Jamais de champ texte dans une rangée horizontale (il capte ◀/▶).
- Ne pas retirer `key(screen)`, le mode imposé (`forcedMode`), la génération `/play` ni `setMediaItem(item, 0)`
  en HLS.
- Ne pas réintégrer les décodeurs logiciels dans les capacités.
- Ne pas remettre un `OutlinedTextField` focalisable en permanence : utiliser `Field`.
- Ne jamais mettre le mot de passe ou le cookie dans une URL ; seul le lien signé du NAS sert à VLC.

## Fichiers

| Fichier | Rôle |
|---|---|
| `AuroraApplication.kt` | Conteneur : OkHttp (cookie de session), magasin des NAS, chargeur d'images Coil |
| `MainActivity.kt` | Activité unique (leanback) |
| `data/ServerStore.kt` | NAS enregistrés ; `normalizeAddress()` (port 8890 par défaut, IPv6) |
| `security/TokenStore.kt` | Session chiffrée AES/GCM, clé AndroidKeyStore, une par NAS |
| `net/AuroraClient.kt` | API AURORA (`/api/v1`) : connexion, accueil, bibliothèques, fiches, lecture, progression |
| `net/Models.kt` | Modèles JSON (kotlinx.serialization) |
| `player/Capabilities.kt` | Capacités de décodage de la box envoyées au NAS |
| `ui/AppViewModel.kt` | Navigation (pile d'écrans), NAS courant, connexion |
| `ui/AuroraTvApp.kt` | Racine : écran courant, touche Retour |
| `ui/SetupScreens.kt` | Liste des NAS, ajout, connexion, composant `Field` |
| `ui/MainScreen.kt` | Onglets Accueil / Films / Séries / Recherche / Profil |
| `ui/DetailScreen.kt` | Fiche film/série, saisons, épisodes |
| `ui/PlayerScreen.kt` | Lecteur, panneau, pistes, épisode suivant, replis |
| `ui/components/Components.kt` | Affiches, cartes larges, rangées, boutons, pastilles |
| `ui/theme/Theme.kt` | Couleurs et polices AURORA |
| `app/src/test/.../AddressTest.kt` | Tests de `normalizeAddress` |

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
(fenêtre active, aucune erreur dans `logcat`) ; affichage non vérifiable par capture.

## Installer sur la Xiaomi Box

Même méthode que PlexSeries TV : **IPv6 link-local**, depuis le Mac connecté au **Wi-Fi** du même réseau que la
box (TP-Link, 192.168.68.x). Si le Mac est seulement en Ethernet (192.168.1.x), la box est invisible : ni
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

Sa capture d'écran (`screencap`) renvoie une image noire précédée de lignes de débogage (« open mma dev
failed ») : vérifier l'app par `dumpsys window` (fenêtre active) et `logcat`, pas par capture.

**Réseau, non vérifié** : la box est sur `192.168.68.0/24` derrière le TP-Link. Le Mac ne la joint pas en IPv4
(d'où l'IPv6), mais le sens box → Synology passe par la sortie du TP-Link et devrait fonctionner. Si
« ne répond pas » s'affiche avec la bonne IP, tester d'abord ce chemin (navigateur de la box sur
`http://<ip-du-nas>:8890`) avant de conclure à un bug de l'app.
