# Limitations connues (réelles, version 0.1.8)

## Non vérifié sur le matériel
- Le paquet n'a **jamais été installé sur un vrai DS1825+ / DSM 7**. Structure, scripts et binaires sont
  vérifiés hors DSM ; voir la recette dans [TESTS.md](TESTS.md#recette-sur-le-ds1825).
- Le champ `arch="x86_64 v1000 v1000nk"` suit la table Synology (le DS1825+ est en `v1000nk`, absent de
  l'annexe du guide développeur) : son acceptation par le Centre de paquets reste à constater.
- Les clés `protocol`/`port` de `ui/config` (ouverture de l'icône sur le port 8890) suivent l'usage des
  paquets tiers (SynoCommunity), pas la documentation officielle, qui ne montre qu'une URL complète.
- Conservation de `@appdata/aurora` à la désinstallation : documentée par Synology, non constatée.
- Vitesse de conversion vidéo sur le Ryzen V1500B : non mesurée. La limite par défaut (1 conversion,
  ≤ 1080p) est prudente, pas calibrée.

## Lecture dans le navigateur
- Ce que le navigateur lit dépend du navigateur, du système et du matériel. AURORA le **mesure**
  (canPlayType / MediaSource) au lieu de le supposer.
- **HEVC** : Safari (Mac, iPhone, iPad) et Chrome/Edge sur matériel compatible ; Firefox en général non.
  Sans HEVC, la vidéo est convertie en H.264 si elle fait au plus 1080p ; **une source 4K est refusée**
  (le NAS n'a pas d'accélération matérielle) avec un message explicite.
- **Dolby Vision, HDR10+, Atmos, DTS:X, passthrough audio, changement de fréquence d'affichage** :
  non disponibles dans un navigateur. AURORA affiche ces informations mais ne promet pas leur rendu.
  En conversion, le HDR est ramené en SDR (zscale/tonemap) ; les couleurs sont donc approchées.
- **Audio** : AC3/E-AC3 lus tels quels si le navigateur les accepte (Safari), sinon convertis en **AAC stéréo**
  (le 5.1 est réduit). DTS et TrueHD sont toujours convertis.
- **Sous-titres** : SRT/WebVTT/ASS convertis en WebVTT ; **les styles ASS/SSA sont perdus** (texte seul).
  PGS et VobSub (images) ne s'affichent qu'**incrustés**, ce qui impose une conversion vidéo (refusée en 4K).
  Pas de réglage de décalage des sous-titres pour l'instant.
- **Recherche dans une vidéo en remux/conversion** : un saut hors de la partie déjà produite relance une
  session (1 à quelques secondes d'attente). En remux, la reprise se cale sur l'image clé précédente
  (jusqu'à quelques secondes plus tôt que demandé).
- MKV/AVI ne sont jamais lus « en direct » dans le navigateur (remux systématique, peu coûteux).
- Chapitres affichés et navigables ; pas de saut d'introduction/générique (non développé).

## Bibliothèque
- Sans clé TMDB : titres tirés des noms de fichiers, images extraites des vidéos, ni résumé, ni
  distribution, ni classification (le contrôle parental ne peut alors rien filtrer, sauf à masquer les
  contenus non classés).
- Correspondance TMDB volontairement prudente : en cas de doute, l'œuvre reste à identifier manuellement.
- Séries : le nom vient du premier dossier sous la racine de la bibliothèque (sinon du nom de fichier).
  Numérotation absolue (anime « Épisode 152 » sans saison) non gérée hors dossier « Saison N ».
- Détection immédiate des ajouts : testée sur macOS (kqueue) et en conditions réelles sur Mac (film détecté en
  22 s), **pas encore sous Linux/DSM** (inotify). Elle ne voit pas les changements d'un partage réseau monté
  sur le NAS (NFS/SMB distant) : là, seule l'analyse périodique fonctionne. Un watch par dossier : au-delà de
  `fs.inotify.max_user_watches` (valeur du DS1825+ à relever : `cat /proc/sys/fs/inotify/max_user_watches`),
  surveillance partielle, signalée dans Maintenance › Diagnostic.
- Bandes-annonces : lien YouTube (Internet requis), pas de lecture intégrée.
- Collections manuelles : prévues dans le schéma, pas encore d'interface (seules les sagas TMDB).
- Code PIN de profil : non développé (chaque profil a un mot de passe).
- Interface uniquement en français ; métadonnées en français ou anglais.

## Exploitation
- HTTP en local (port 8890). Pour HTTPS : proxy inverse DSM.
- Pas de mise à jour automatique : nouveau `.spk` en installation manuelle.
- FFmpeg embarqué : build statique 7.0.2 (août 2024), épinglé par SHA-256 mesuré (l'éditeur ne publie plus
  de somme de contrôle pour ce fichier).

## Client Android TV (AuroraTV)
- Testé sur l'émulateur Google TV ; sur la Xiaomi Box (18/10/2026) : installation, connexion au NAS et
  accueil vérifiés. Lecture sur la box et connexion par code (AURORA 0.1.6) : non vérifiées.
- Le NAS ne distingue pas HEVC 8 et 10 bits dans les capacités annoncées : une box qui décode le HEVC
  8 bits mais pas le 10 bits tente d'abord la lecture directe, échoue, puis l'app redemande une
  **conversion** (H.264, ≤ 1080p ; une source 4K est alors refusée par le NAS). Quelques secondes perdues.
- VLC (repli) lit via un lien signé valable 4 h ; la position n'est pas remontée à AURORA depuis VLC.

## Association par code
- Demandes de code acceptées depuis le réseau local seulement : adresses privées, lien-local, loopback, ou
  sous-réseaux des interfaces du NAS (IPv6 globale de la box). Une requête qui arrive de la machine elle-même
  **sans** `X-Forwarded-For` est considérée locale : un tunnel installé sur le NAS (cloudflared, Tailscale
  Funnel…) qui n'ajoute pas cet en-tête rendrait la demande de code possible depuis Internet. La validation
  en deux temps (nom et adresse affichés, confirmation explicite) reste alors la protection.
- Demandes en mémoire : un redémarrage d'AURORA annule les codes affichés (la TV en redemande un seule).
