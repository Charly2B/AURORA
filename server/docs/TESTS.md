# Tests et résultats

Règle : un test n'est déclaré réussi que s'il a été **exécuté**. Tout ce qui exige un vrai DS1825+ est
listé à part et **n'a pas encore été fait**.

## Lancer les tests
```bash
go test -race ./...                                   # backend (intégration incluse, nécessite ffmpeg/ffprobe)
cd web && npx tsc --noEmit && npx vitest run          # frontend
go test -tags bench -run TestLoad -v -timeout 30m ./internal/bench   # charge
scripts/build-spk.sh 0001                              # paquet + vérification de structure
```

## Résultats exécutés (17/10/2026, macOS 26.6 x86-64, Go 1.27, Node 24)

### Automatisés — tous verts
| Domaine | Tests | Couvre |
|---|---|---|
| Analyse des noms | `internal/parse` | films (année, éditions, sigles, titres-années « 1917 », « Blade Runner 2049 »), épisodes (SxxEyy, NxNN, doubles, spéciaux, dossiers « Saison N »), normalisation sans accents |
| ffprobe | `internal/probe` | pistes, pochette ignorée, Dolby Vision/HDR10 seulement si attestés, Atmos, sous-titres image/texte, chapitres |
| Décision de lecture | `internal/playback` | 12 cas direct / remux / conversion (Chrome, Safari, H.264 10 bits, DTS, sous-titre image…), refus 4K, décalage WebVTT, noms de segments (traversée), course démarrage/arrêt ffmpeg |
| Métadonnées | `internal/metadata` | correspondance prudente TMDB (ambiguïté jamais tranchée au hasard), classifications FR/US, clé jamais présente dans les erreurs |
| Indexation | `internal/scanner` | sous-titres externes (langue, forcé, SDH), dossiers techniques ignorés (@eaDir, #recycle…) |
| Base | `internal/db` | migrations, FTS5 sans accents, sauvegarde/restauration (ancienne base conservée), refus d'une base plus récente |
| Authentification | `internal/auth` | Argon2id, limitation des tentatives (attente exponentielle) |
| **Bout en bout** | `internal/api` (vrais fichiers vidéo, vrai serveur HTTP) | assistant, CSRF, bibliothèques hors racine refusées, indexation, recherche accentuée et approchante, lecture directe avec requête partielle, reprise et marquage vu, remux HLS réel (offset calé sur l'image clé, playlist sans cache), profil enfant (films non classés masqués, épisodes d'une série −18 masqués, admin/flux/fichiers interdits), limitation de connexion, prochain épisode, déplacement de fichier sans perte d'historique, sous-titre en lien symbolique ignoré, sauvegarde |
| Frontend | `web/src/test` (12) | formats, détection des capacités (Chromium vs Safari), choix automatique des pistes |

### Manuels dans le navigateur intégré (Chromium) — réussis
Assistant complet (compte, langue, 2 bibliothèques via l'explorateur, indexation), accueil, bibliothèque,
fiche film, page série, **lecture réelle** : MKV H.264 + AC3 5.1 (remux HLS + audio AAC), sous-titres
français synchronisés, changement de piste audio (nouvelle session à 52,08 s = image clé réelle),
reprise après rechargement, fin d'épisode → compte à rebours → épisode suivant, HEVC 10 bits (remux,
ce Chromium décodant le HEVC), affichage mobile 375 px, administration.

Défauts trouvés et corrigés pendant ces essais : HLS natif annoncé à tort par Chromium, playlist servie
en 304 (lecture figée), clic masquant/pausant, sous-titres sous la barre, navigation mobile, titre
commençant par « HEVC ».

### Revue de code indépendante (agent relecteur, modèle Opus)
Première passe : 7 défauts (2 graves, 3 moyens, 2 mineurs) — fuite possible de la clé TMDB, contrôle
parental contournable sur les épisodes ajoutés après coup, sous-titre en lien symbolique, retour arrière
bloqué en remux, ffmpeg orphelin en cas de relances rapprochées, accès concurrents aux réglages,
restauration sans journal WAL. Tous corrigés avec tests ; **seconde passe du relecteur : 7/7 confirmés
corrigés**. Elle a signalé un nouveau défaut moyen (cache des sous-titres indexé par position), corrigé
avec test (`TestSubtitleCacheKey`).

### Paquet
- `scripts/verify-spk.sh` : **conforme** (structure, INFO, checksum, privilèges non root, scripts
  exécutables et syntaxe, icônes 64/256, binaires ELF x86-64 **statiques**, ui/config, port non publié,
  licences, interface embarquée).
- Reproductibilité : deux assemblages successifs → empreintes SHA-256 identiques (paquet et binaire).
- Cycle de vie DSM **simulé** sur macOS (vrais scripts, variables `SYNOPKG_*`, binaire macOS) :
  preinst/postinst, status=3 → start=0 → status=0 → page web 200 → start idempotent → stop=0 (arrêt
  propre journalisé) → status=3 ; preupgrade copie la base ; données conservées après redémarrage.

### Charge (Mac de développement, 16 cœurs, SSD — **pas** le NAS)
Fichiers vides : l'analyse ffprobe n'est pas incluse (elle dépend des disques). Meilleur de 5 essais.

| Mesure | 1 000 films | 5 000 films + 10 000 épisodes | 50 000 fichiers |
|---|---|---|---|
| Indexation initiale (parcours + identification) | 0,35 s | 5,3 s | 17,7 s |
| Nouvelle analyse sans changement | 0,05 s | 0,36 s | 1,4 s |
| Tas mémoire après indexation | 2,6 Mo | 4,2 Mo | 8,4 Mo |
| Liste films, 1re page | 4 ms | 13 ms | 53 ms |
| Liste films, page 30, tri qualité + filtre 4K | 5 ms | 23 ms | 97 ms |
| Filtres disponibles | 9 ms | 60 ms | 288 ms |
| Accueil complet | 8 ms | 52 ms | 164 ms |
| Recherche plein texte | 0,8 ms | 1,6 ms | 3,5 ms |
| Recherche avec fautes (repli approchant) | 16 ms | 73 ms | 261 ms |
| Fiche film | 2 ms | 8 ms | 29 ms |
| Taille de la base | < 1 Mo | 12 Mo | 42 Mo |

Ce ne sont pas des performances garanties sur le DS1825+ (processeur plus lent, disques mécaniques).

## Non testé — exige le vrai matériel
Rien de ce qui suit n'a été exécuté. À faire lors de la première installation :

### Recette sur le DS1825+
1. Version DSM exacte (Panneau de configuration › Infos) : ______
2. Installation manuelle du `.spk` acceptée (plateforme `v1000nk` reconnue par `arch=`) ; icône AURORA
   dans le menu principal ; « Ouvrir » mène à `http://nas:8890/`.
3. Utilisateur système `aurora` créé ; permissions de lecture accordées ; dossiers lisibles dans l'assistant.
4. Indexation d'une vraie bibliothèque : durée, charge CPU (Moniteur de ressources), aucune erreur ffprobe inattendue.
5. Lecture : MP4 H.264 (direct), MKV H.264 + DTS (remux), HEVC 4K HDR dans Safari Mac (remux), même
   fichier dans Chrome Windows (refus explicite attendu), HEVC 1080p dans Chrome sans HEVC (conversion :
   **mesurer la vitesse réelle** et la charge CPU), sous-titres SRT, PGS (incrustation), reprise.
6. Arrêt/démarrage depuis le Centre de paquets ; redémarrage du NAS (démarrage automatique) ; données conservées.
7. Mise à jour vers un build supérieur : sauvegarde `avant-maj-…` créée, données conservées.
8. Désinstallation : `@appdata/aurora` conservé (à vérifier : comportement documenté par Synology).
9. Accès via proxy inverse HTTPS DSM (cookies `Secure`, flux SSE non mis en tampon).

## Client Android TV (émulateur, 17/10/2026)
Émulateur Google TV (API 36, 1920×1080) contre une instance locale (`10.0.2.2:18893`), à la télécommande
simulée (`adb shell input keyevent`) :
- ajout du NAS, connexion, déconnexion et reconnexion ; onglets par dossier, « Tous les dossiers » ;
  recherche approchante ; fiche série ;
- lecture : reprise exacte, ◀/▶, pause (le panneau reste affiché), Retour ;
  épisode suivant automatique (E02 → E03) ;
- HEVC 10 bits refusé par le décodeur de l'émulateur → session `transcode` côté NAS → lecture.

Bugs trouvés et corrigés pendant ce test :
- focus bloqué dans les champs texte ;
- épisode suivant figé à « 0 s » ;
- clavier ouvert au passage du focus ;
- doublon de NAS via Retour ;
- repli remux inutile pour un décodeur vidéo en échec.

Puis la revue indépendante a demandé des changements, tous corrigés :
- plantage sur un corps de réponse coupé ou non-JSON ;
- départ HLS au bord produit (ExoPlayer voyait un direct) ;
- sessions concurrentes sur sauts rapprochés ;
- erreur après une pause de plus de 90 s ;
- piste audio française réimposée sur un choix manuel ;
- piste choisie perdue après un saut ;
- décodeurs logiciels comptés dans les capacités ;
- grilles plafonnées à 500 ;
- Retour qui ne quittait jamais l'app ;
- session révoquée sans retour à la connexion ;
- dossier et focus perdus au retour d'une fiche.

Xiaomi Box (18/10/2026) : APK installé par adb ; connexion au Synology réel et accueil avec son contenu
constatés. Lecture sur la box **non testée**.

## Association par code (18/10/2026)
- `TestDeviceLink` :
  - demande, suivi en attente, validation anonyme refusée (401), mauvais code (404) ;
  - code normalisé (casse, tiret, espace) ; premier temps = description seule (la TV reste en attente) ;
  - confirmation, double validation refusée, session obtenue une seule fois (410 ensuite) ;
  - expiration à 10 min ; demande venue d'Internet refusée (403), y compris avec un premier élément
    `X-Forwarded-For` falsifié ; plafond de 5 demandes par adresse (429).
- `TestNormalizeCode` : formats acceptés/refusés.
- `TestSignedStream` rendu déterministe : la signature « falsifiée » restait identique une fois sur 16.
- Bout en bout : émulateur Google TV affiche un code → `/link?code=…` décrit l'appareil sans l'associer (TV
  toujours en attente) → « Confirmer » → la TV passe seule à l'accueil. Aller-retour mot de passe ↔ code :
  même code réutilisé.

## Détection immédiate des ajouts (18/10/2026)
- `TestWatcher` (watch réel, kqueue sur macOS) : bruit `@eaDir` et `.txt` ignoré ; nouveau dossier + vidéo
  écrite en 5 fois → **une** analyse après le calme ; sous-titre ajouté dans le nouveau dossier (watch posé
  à la volée) → analyse ; dossier supprimé → analyse et watch oublié ; réglage désactivé → plus aucun watch.
- Réel sur Mac : copie de `Test.Detection.2024.1080p.mp4` dans « Films 2021 » → film trouvé par la recherche
  **22 s** après la copie, sans action. Non testé sous Linux/DSM (inotify).

## Retirer de Continuer à regarder, panneaux de dossiers (18/10/2026)
- `TestHideContinue` : film en cours et épisode suivant présents ; retrait du film et de la série (via l'épisode
  proposé) → absents des deux rangées, progression gardée ; autre profil non concerné ; relecture (seconde
  suivante) → le film revient ; œuvre inconnue → 404. Migration 0004 appliquée par `TestOpen` (schéma à jour).
- Site : × au survol sur la carte, message « retiré », rangée rechargée. Panneaux macOS de « Tous les dossiers »
  vérifiés à l'écran.
- TV (émulateur) : appui long sur une carte Continuer → fenêtre « Retirer » (focus piégé) → carte retirée ;
  panneaux arrondis avec barre de titre dans « Tous les dossiers », 7 colonnes alignées.
