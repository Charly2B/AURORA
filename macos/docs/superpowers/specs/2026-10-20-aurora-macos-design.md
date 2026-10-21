# AURORA pour macOS — conception

## Intention

Créer le lecteur macOS natif manquant pour le serveur multimédia AURORA du NAS. L’application vise l’iMac de Remy, s’utilise au clavier et à la souris, reprend à parité les fonctions et règles du client Android TV et est livrée dans un DMG installable.

## Livrables

- Projet Xcode autonome dans `/Users/remyj/Documents/Xcode/AuroraMac`.
- Produit `AURORA.app`, Swift 6 et SwiftUI, cible macOS 15 ou plus récent.
- DMG final dans le dossier `outputs` de la tâche.
- Installation automatique de la build validée dans `/Users/remyj/Applications/MesApps/AURORA.app`, après sauvegarde horodatée de l’ancienne version, vérification de la copie, puis lancement.
- README concis : build, tests, création du DMG, limites réellement vérifiées.

## Expérience

Même identité sombre et cinématographique que l’app Android TV : Accueil, Films, Séries, Recherche et Profil ; grand visuel de mise en avant ; rangées horizontales sur l’accueil ; grilles adaptatives dans les bibliothèques ; panneaux « Tous les dossiers » ; fiche immersive avec fond, affiche, synopsis, distribution, versions, saisons et épisodes. L’organisation et les libellés restent alignés sur Android ; seuls les contrôles deviennent natifs Mac : clavier/souris, survol, menu, redimensionnement, plein écran et accessibilité VoiceOver.

Le premier lancement propose les NAS enregistrés, puis l’ajout par IP ou nom réseau. Le port 8890 est ajouté par défaut et IPv6 est accepté. Connexion prioritaire par le code AURORA à valider dans le navigateur, mot de passe en secours. Plusieurs NAS et changement à chaud. Adresse enregistrée dans les préférences ; session par NAS dans le Trousseau macOS, jamais dans les journaux ni une URL.

## Fonctions de la première version

- Accueil identique aux règles serveur : mise en avant, Continuer, Prochain épisode, une rangée par dossier.
- Films/Séries : dossiers séparés, « Tous les dossiers », tri titre/ajout/année, pagination complète.
- Recherche serveur tolérante aux accents et fautes.
- Fiches film/série : lire/reprendre, recommencer, vu/non-vu, favori, Ma liste, saisons/épisodes.
- Retrait de Continuer à regarder avec confirmation, sans effacer progression ni fichier.
- Profil courant, déconnexion, changement/suppression d’un NAS enregistré.
- États de chargement, hors-ligne, session révoquée et erreurs serveur lisibles, sans plantage.
- Même conservation du dossier, du tri, du défilement et de l’élément ouvert au retour d’une fiche.

## Lecture

`libmpv` intégré dans l’application, comme Plex Desktop, sans lecteur externe. Les capacités macOS réellement disponibles sont envoyées au serveur. mpv/FFmpeg permettent la lecture directe de MKV, MP4, MOV, WebM, MPEG-TS et AVI, davantage de codecs et les sous-titres natifs ; le NAS conserve ses replis remux HLS et conversion. Aucun accès direct SMB et aucun changement du serveur.

Le lecteur gère lecture directe, remux et conversion décidés par AURORA ; reprise ; progression envoyée toutes les 10 secondes et à la fermeture ; pause ; sauts -10/+30 s ; barre de temps ; chapitres ; vitesse ; plein écran ; pistes audio ; sous-titres gérés par mpv, avec incrustation serveur si nécessaire ; préférences de langue ; épisode suivant avec compte à rebours annulable.

Pour HLS, la position absolue reste `offset + position locale`. Un saut hors de la zone produite, un retour dépassant les segments conservés ou une session expirée ouvre une nouvelle session. Les réponses `/play` obsolètes sont ignorées et leur session est arrêtée. À la fermeture, progression et session HLS sont finalisées même si la vue disparaît. Chaîne de repli interne : échec direct vers remux ; échec de décodage vidéo vers conversion ; erreur réseau HLS vers une reconnexion ; sinon erreur explicite. Aucun lecteur externe et aucune intégration VLC.

Raccourcis : espace lecture/pause, flèches gauche/droite -10/+30 s, `F` plein écran, `Esc` ferme les panneaux ou quitte le lecteur, `C` sous-titres, `A` audio.

## Architecture

- `App`: cycle de vie, dépendances, navigation et état global.
- `Networking`: client `URLSession`, cookies AURORA, modèles `Codable`, erreurs typées, annulation et génération des requêtes de lecture.
- `Security`: session par origine dans le Trousseau.
- `Features`: Setup, Home, Library, Search, Detail, Profile, Player.
- `Playback`: pont Swift/C vers libmpv, contrôleur `@MainActor`, capacités, HLS/reprise et sélection des pistes.
- `DesignSystem`: couleurs, typographie, cartes, rangées, images avec cache `URLCache`.

Observation moderne (`@Observable`) et concurrence structurée Swift. État UI sur le MainActor ; réseau et décodage hors interface. libmpv et ses bibliothèques sont embarqués dans l’app avec licences et provenance documentées ; aucun téléchargement de dépendance au lancement.

## Sécurité et contraintes

- Respect strict de l’API `/api/v1` actuelle et de l’en-tête anti-CSRF `X-Aurora`.
- Aucun mot de passe ou cookie dans URL, préférences ou logs.
- HTTP local autorisé uniquement pour joindre le NAS saisi par l’utilisateur ; pas d’exposition Internet ajoutée.
- Aucune écriture/suppression dans les vidéos et aucune modification du serveur ou du client Android sauf incompatibilité constatée.
- Icône dérivée des ressources AURORA existantes.
- libmpv construit/embarqué dans une configuration de licence redistribuable, licences tierces incluses dans le DMG et le bundle.
- Signature avec une identité Developer ID disponible localement ; sinon signature ad hoc et DMG local non notarisé, clairement documentés.

## Vérification et définition de terminé

- Tests unitaires : normalisation d’adresse/IPv6, décodage JSON, cookies et session, pagination/dédoublonnage, calculs de position HLS, commandes mpv et choix de repli.
- Tests réseau avec `URLProtocol` simulé : connexion par code, 401, erreurs non JSON, progression et arrêt de session.
- Build Debug et Release réussis sous Xcode 26.5 ; tests verts.
- Contrôle manuel : lancement, navigation clavier/souris, redimensionnement, écran d’ajout NAS et ouverture du lecteur avec réponses simulées ou serveur accessible.
- DMG montable contenant une seule `AURORA.app` et un raccourci Applications ; app copiée vérifiée par empreinte de l’exécutable et des ressources.
- Ne déclarer testée contre le NAS réel ou avec une vidéo réelle que si cela a effectivement été fait.

## Hors périmètre initial

Administration du serveur, téléchargement local, mode hors-ligne, édition des métadonnées, lecteur externe/VLC, AirPlay personnalisé, synchronisation iCloud, App Store et notarisation sans identifiants existants.

## Choix laissés au constructeur

Organisation exacte des fichiers et composants, détails visuels, stratégie de cache images, outil de génération du projet Xcode et script de création du DMG, tant que les contraintes et comportements ci-dessus restent vérifiables.
