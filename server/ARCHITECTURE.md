# Architecture d'AURORA

Un seul processus Go (binaire statique, ~14 Mo, interface web embarquée) + ffmpeg/ffprobe statiques,
le tout dans un paquet DSM 7 qui tourne sous l'utilisateur système `aurora`. Base SQLite unique.
Aucun Docker, aucun service auxiliaire.

```
Navigateur ──HTTP :8890──▶ aurora (Go)
   React + hls.js           ├─ API REST /api/v1 (auth, catalogue, lecture, admin) + SSE
                            ├─ SQLite (WAL, FTS5)              @appdata/aurora/aurora.db
                            ├─ indexation (stat → nom → ffprobe)  lecture seule des partages
                            ├─ sessions HLS (ffmpeg remux/conversion)  @apptemp/aurora/work/hls
                            └─ TMDB (facultatif) + cache images     @appdata/aurora/cache
```

## Fichiers
| Chemin | Rôle |
|---|---|
| `cmd/aurora/main.go` | démarrage, câblage, arrêt propre, maintenance périodique (purge sessions, analyses planifiées) |
| `cmd/mkspk/main.go` | assemble le `.spk` de façon reproductible (tri, dates fixes, root, sans attributs macOS) |
| `internal/config` (`volumesIn`) | options/variables `AURORA_*`, version injectée, journal tournant (5 Mio × 3) |
| `internal/db` | ouverture SQLite, migrations embarquées, sauvegarde `VACUUM INTO`, restauration différée |
| `internal/db/migrations/0004_continue_hidden.sql` | 0.1.8 : `continue_hidden` (œuvres retirées de Continuer / Prochain épisode par un profil) |
| `internal/db/migrations/0003_manual_assign.sql` | 0.1.2 : `media_files.manual` (rattachement manuel, jamais défait par l'indexation) |
| `internal/db/migrations/0002_retry_matching.sql` | 0.1.1 : remet en recherche TMDB les œuvres ambiguës/introuvables (jamais les corrections manuelles), masque d'anciens messages contenant la clé |
| `internal/db/migrations/0001_init.sql` | schéma : users, sessions, libraries, items (film/série/épisode), media_files (versions), user_state, credits, seasons, collections, tasks, FTS5 |
| `internal/auth` | Argon2id, sessions (jeton haché en base), limiteur de tentatives |
| `internal/parse` | analyse des noms de fichiers (films, épisodes, éditions, étiquettes), normalisation |
| `internal/probe` | appel et lecture de ffprobe (pistes, HDR attesté, chapitres) |
| `internal/scanner` | indexation incrémentale, détection des déplacements, identification, ffprobe en parallèle limité |
| `internal/watcher` | détection immédiate des ajouts (inotify via fsnotify) : un watch par dossier, analyse après 20 s de calme |
| `internal/metadata` | client TMDB, correspondance prudente, fusion des doublons, saisons/épisodes, images |
| `internal/catalog` | requêtes de lecture filtrées par profil (`Viewer.Where`), accueil, recherche FTS + approchante, état de lecture |
| `internal/playback` | décision direct/remux/conversion, sessions ffmpeg HLS (régulation SIGSTOP/SIGCONT, purge), sous-titres WebVTT, miniatures, calage sur image clé |
| `internal/api` | routes HTTP, middlewares (CSRF, en-têtes de sécurité, reprise sur panique), erreurs centralisées |
| `internal/api/link.go` | association d'un appareil par code (TV) : demandes en mémoire, validation par un profil connecté |
| `internal/events` | diffusion SSE (progression des analyses) |
| `internal/tools` | localisation de ffmpeg/ffprobe |
| `internal/webui` | interface compilée embarquée (`dist/` généré par Vite) |
| `internal/bench` | test de charge (`-tags bench`) |
| `web/src/lib` | client API (`X-Aurora`), types, détection des capacités, formats, session, hooks |
| `web/src/components` | système de design (`ui.tsx`), cartes/rangées/bandeau (`media.tsx`), fiches (`detail.tsx`), navigation, explorateur de dossiers |
| `web/src/pages` | Accueil, Bibliothèque, fiches Film/Série, Recherche, Collections, Ma liste, Profil, Lecteur, Assistant, Connexion, Associer une TV (`Link.tsx`, `/link`), `admin/*` |
| `web/src/test` | tests vitest |
| `packaging/synology/` | INFO, scripts de cycle de vie, `conf/privilege` (run-as package), `conf/resource` (port), `ui/config` (icône du menu), icônes, licences |
| `scripts/build.sh` | compile l'interface puis le binaire (`host` ou `linux`) |
| `scripts/build-spk.sh` | FFmpeg épinglé (SHA-256) + binaire linux/amd64 + `mkspk` + vérification |
| `scripts/verify-spk.sh` | contrôle de structure du paquet hors DSM |
| `scripts/gen-testmedia.sh` | médiathèque de test synthétique (ffmpeg) |
| `scripts/make-icons.py` | génère les icônes PNG (Pillow) ; les PNG sont versionnés |

## Décisions non évidentes (et pourquoi)
- **Go + SQLite pur Go (modernc)** : binaire statique sans cgo → aucune dépendance aux bibliothèques
  de DSM, compilation croisée depuis n'importe quel poste.
- **Un seul écrivain à la fois** : `_txlock=immediate` + `busy_timeout` ; une seule analyse de
  bibliothèque à la fois, ffprobe limité à 2 : les disques restent disponibles pour la lecture.
- **Jamais d'analyse au démarrage**, seulement périodique et incrémentale (taille + date de modification).
- **Fichiers disparus marqués absents, jamais supprimés de la base** ; un fichier de même nom et taille
  apparu ailleurs est un déplacement (même ligne → historique conservé).
- **Correspondance TMDB** : automatique seulement si titre exact (ou titre original, sans accents) **et**
  année qui lève l'ambiguïté (±1 an toléré si un seul candidat). Sinon `ambiguous` → choix humain.
  Une correction manuelle est `locked` et n'est plus jamais modifiée automatiquement.
- **Contrôle parental côté serveur** dans chaque requête ; un épisode sans classification hérite de
  celle de sa série (`COALESCE` dans `Viewer.Where`), sinon un épisode ajouté plus tard échapperait au filtre.
- **Décision de lecture** : direct si conteneur + vidéo + audio + piste par défaut sont lisibles
  (testés dans le navigateur, pas supposés) ; sinon remux HLS fMP4 (vidéo copiée, `hvc1` pour Safari,
  audio copié ou AAC) ; sinon conversion H.264 ≤ 1080p, refusée au-delà sauf réglage. H.264 10 bits →
  conversion (aucun navigateur ne le décode).
- **Remux et image clé** : ffmpeg en copie démarre sur l'image clé précédente ; `KeyframeBefore` lit les
  paquets (sans décodage) pour annoncer le vrai décalage, sinon horloge et sous-titres seraient faux de
  plusieurs secondes (mesuré : 30 s demandées → 20,8 s réelles).
- **Playlist HLS servie sans `Last-Modified`** : `http.ServeFile` date à la seconde, ffmpeg réécrit la
  playlist plusieurs fois par seconde → 304 et playlist figée (lecture bloquée, constaté en test).
- **HLS natif seulement sur navigateur Apple** : un Chromium répond « maybe » au type HLS puis échoue.
- **Régulation ffmpeg** : SIGSTOP au-delà de ~45 segments d'avance, SIGCONT sous 20 ; segments vus effacés
  au-delà de 40 en arrière (`purged`, réponse 410 immédiate) ; le lecteur rouvre une session pour un retour
  arrière de plus de 60 s.
- **Volumes explorables** : `/volumeN`, `/volumeUSBN` (disque USB externe, monté sur `/volumeUSB1/usbshare`) et
  `/volumeSATAN` (eSATA), recalculés à chaque exploration (un disque branché après le démarrage apparaît sans
  redémarrage). L'ancien motif `/volume[0-9]*` excluait les disques externes (signalé par l'utilisateur).
- **Association par code** (`/api/v1/auth/link/start|poll|approve`, comme plex.tv/link) : la TV obtient un code
  de 6 caractères (alphabet sans 0/O/1/I/L, ~8,9·10⁸ combinaisons) et un secret de suivi de 128 bits ; au suivi
  suivant la validation, la TV reçoit une session normale (cookie), une seule fois.
  - Validation **en deux temps** par un profil **connecté** : `{code}` décrit la demande (nom, adresse, âge),
    `{code, confirm: true}` associe. Un lien `/link?code=…` envoyé par un tiers n'associe donc jamais en un clic.
  - `start` refusé hors réseau local (adresse ni privée, ni loopback, ni lien-local, ni dans un sous-réseau
    d'une interface du NAS — ce dernier cas pour l'IPv6 globale des box, `2a01:…`) : pas de demande lancée
    depuis Internet à travers un proxy inverse exposé.
  - Expiration 10 min (prolongée d'1 min à la validation si elle tombait juste avant) ; 5 demandes en cours
    par adresse, 100 au total ; validations ratées limitées par profil **et** par adresse, compteur jamais
    remis à zéro par une réussite.
  - En mémoire seulement : un redémarrage annule les demandes, la TV en redemande une.
  - Pourquoi : saisir un mot de passe à la télécommande échouait en réel (clavier TV, compte DSM confondu).
- **Adresse du client derrière un proxy** (`clientIP`) : **dernier** élément de `X-Forwarded-For` (celui
  ajouté par le nginx de DSM), seulement si la connexion vient de la machine locale. Le premier élément est
  fourni par le client : le prendre permettait de contourner les limites par adresse.
- **Détection immédiate des ajouts** (`internal/watcher`, comme Plex) : inotify surveille chaque dossier des
  bibliothèques (non récursif : un watch par dossier, ajouté aussi pour chaque nouveau dossier).
  - Seuls comptent vidéos, sous-titres et dossiers ; les dossiers techniques (`@eaDir` que DSM crée après chaque
    ajout, `#recycle`, `.…`) sont ignorés, sinon chaque miniature de DSM relancerait une analyse.
  - Analyse incrémentale de la bibliothèque après **20 s sans changement** (une copie écrit en continu : on
    attend sa fin), au plus tard après 10 min ; débordement de la file du noyau → toutes les bibliothèques.
  - `Scanner.Enqueue` reprogramme une bibliothèque **en cours** d'analyse (avant : ignorée), car un ajout peut
    tomber dans un dossier déjà parcouru.
  - Limite `fs.inotify.max_user_watches` atteinte : surveillance partielle, signalée (Diagnostic), retentée au plus
    une fois par heure (le nouvel essai reparcourt tout et réveillerait les disques) ; l'analyse périodique (6 h par défaut) reste le filet de sécurité.
  - `Sync` (sérialisé, relit lui-même le réglage `watch_enabled`, activé par défaut) toutes les 10 min, et
    aussitôt après un changement de bibliothèque ou de réglage. Piège : au démontage d'un disque
    (`IN_UNMOUNT`), fsnotify abandonne ses watches **sans évènement** ; `Sync` recale donc ses dossiers sur
    `WatchList()` et resurveille une racine dont le périphérique (`st_dev`) a changé (disque rebranché).
  - Parcours des dossiers **hors verrou** (seuls les `Add` le prennent) : sinon les évènements s'accumulent
    pendant le parcours d'une grosse bibliothèque, jusqu'au débordement de la file du noyau.
  - La maintenance ne remet pas en file la bibliothèque en cours d'analyse (sinon réanalyse complète juste après).
- **« Retirer de Continuer à regarder »** (`POST /api/v1/items/{id}/continue/hide`, comme Plex) : ligne
  `continue_hidden(user, œuvre, hidden_at)` ; un épisode retire sa **série**. Les rangées Continuer et Prochain
  épisode, le bandeau et « Séries en cours » excluent l'œuvre tant que `hidden_at >= dernière lecture` (filtre
  `notHidden` de home.go) : elle revient d'elle-même à la lecture suivante. Fichier, progression et « vu »
  intacts ; propre à chaque profil ; reporté lors d'une fusion d'œuvres (`MergeItems`), sinon perdu en cascade.
- **Lien de lecture signé** (`/api/v1/files/{id}/signed`) : HMAC (clé régénérée à chaque démarrage) sur
  fichier|profil|expiration|empreinte du mot de passe — un changement de mot de passe révoque tous les liens ;
  valable 4 h ; droits revérifiés à chaque requête ; fourni seulement si `/play` reçoit `external: true`
  (lecteur externe comme VLC sur la box, qui ne transmet pas la session).
- **Client Android TV** (`~/Documents/AndroidStudio/AuroraTV`, projet séparé) : utilise la même API que le
  web, cookie de session compris. Il envoie ses capacités mesurées (décodeurs MediaCodec, passthrough,
  conteneurs MKV/MP4/TS/AVI lus nativement) ; `forceMode` (`remux`/`transcode`), prévu pour le diagnostic,
  lui sert aussi de repli quand le décodeur de la box échoue. Toute évolution de `/play`, des cartes ou des
  fiches doit rester compatible avec ce client (modèles dans `net/Models.kt`).
- **Type de bibliothèque** : à l'ajout, les noms sont échantillonnés (400 vidéos) ; si le contenu contredit
  le type choisi (épisodes dans « Films »), réponse 409 avec le type suggéré, sauf confirmation (`force`).
  Changer le type efface seulement l'identification de cette bibliothèque (œuvres, progression associée)
  et relance l'analyse ; l'analyse technique des fichiers est conservée.
- **Choix d'affiche** : `GET/POST /api/v1/admin/items/{id}/artwork` ; seuls des chemins TMDB stricts
  (`/[A-Za-z0-9_-]+.jpg|png`) sont acceptés, l'image est téléchargée par le NAS depuis image.tmdb.org puis
  servie depuis le cache. La CSP autorise `https://image.tmdb.org` pour les seuls aperçus d'administration.
- **Changement de type** : `Scanner.Exclusive` *réserve* la bibliothèque (aucune analyse ne peut la prendre)
  sans tenir le verrou du planificateur pendant la transaction — le tenir provoquait un interblocage de 10 s
  avec une analyse d'une autre bibliothèque en cours de transaction. Sauvegarde complète faite dans la
  réservation (donc seulement si le changement a lieu).
- **Épisodes dans une bibliothèque de films** : tout fichier au numéro explicite (S01E06, 1x06) devient un
  épisode de sa série (`ParseEpisodeInMovies` : nom de série tiré du fichier ou de son dossier, jamais d'un
  dossier de collection). Les fichiers indexés comme films par une version antérieure sont reclassés à
  l'analyse suivante, dans la même transaction, **avec report de l'historique de lecture** sur l'épisode ;
  les « films » vidés sont retirés du catalogue. Jamais reclassés : fichiers rattachés à la main
  (`manual=1`), films verrouillés ou déjà associés à TMDB. L'alerte de type ne porte donc plus que
  sur des films rangés dans une bibliothèque Séries.
- **Dossiers séparés** : `/api/v1/libraries` renvoie films et séries présents par dossier (filtrés par
  profil) ; l'accueil produit une rangée « récents » par dossier dès qu'il y en a plusieurs du même type
  (titre = nom du dossier) ; les pages Films/Séries affichent un onglet par dossier (`?library=`) et,
  pour « Tous les dossiers », un bloc par dossier (titre puis grille), jamais une liste mélangée.
- **TMDB** : recherche dans la langue configurée, puis en anglais si aucune correspondance exacte (fichiers
  nommés avec le titre original), puis avec l'« année » réintégrée au titre (« Wonder Woman 1984 ») ; fiches
  toujours récupérées dans la langue configurée. Films et séries sont enrichis quel que soit le type de bibliothèque.
- **Vignettes** : filtre ffmpeg `thumbnail=30` (image la plus représentative), pas une capture au hasard
  (souvent noire). Réduction à 640 px **avant** le filtre : il garde 30 images en mémoire (≈ 750 Mo en 4K sinon).
- **Cache des sous-titres** : clé = source réelle (chemin + taille + date du fichier externe, ou n° de flux
  interne), jamais la position dans la liste, qui change quand un `.srt` est ajouté (sinon mauvaise piste servie).
- **Sessions ffmpeg** : contexte et canaux créés avant publication de la session → un arrêt concurrent
  ne peut pas laisser de processus orphelin.
- **Sécurité** : cookie HttpOnly SameSite=Lax (Secure derrière proxy HTTPS local) ; en-tête `X-Aurora`
  obligatoire sur toute requête modifiante (CSRF) ; CSP stricte ; noms d'images/segments/sauvegardes par
  motif ; dossiers explorables limités aux volumes (liens résolus) ; sous-titres externes : fichiers
  réguliers dans la bibliothèque uniquement ; erreurs TMDB sans URL (la clé v3 y figure) ;
  flux SSE réservé aux administrateurs ; X-Forwarded-For/Proto crus seulement depuis la boucle locale.
- **Sauvegardes** : avant chaque migration, avant restauration, avant mise à jour (script) ; une base
  plus récente que le binaire est refusée.

## Interdits (ne pas défaire)
- Aucune écriture, aucun déplacement, aucune suppression dans les dossiers de vidéos.
- Aucune suppression de données utilisateur dans les scripts DSM (pas de `rm`), ni de sauvegarde
  automatique. Le seul effacement fait par AURORA concerne ses propres segments HLS temporaires.
- Pas de `run-as root` dans `conf/privilege`, pas de port publié vers Internet (`port_forward="no"`).
- Ne pas modifier une migration publiée (0001 à 0004) : ajouter `0005_…sql`.
- Ne pas surveiller les dossiers techniques (`scanner.SkipDir`) ni réagir aux évènements `Chmod` : boucle
  d'analyses déclenchée par DSM lui-même.
- Ne pas réactiver le HLS natif hors navigateurs Apple, ni servir la playlist avec `ServeFile`.
- Ne pas associer automatiquement une fiche TMDB en cas de doute.
- Ne pas journaliser mots de passe, jetons, clé TMDB (ni URL TMDB complète).
- Build : ne jamais vider `internal/webui/dist` par suppression ; `scripts/build.sh` le met à la Corbeille.
