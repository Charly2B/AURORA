# AURORA

Médiathèque personnelle haut de gamme, installée **nativement** sur un Synology (DSM 7, x86-64 —
cible DS1825+). Films et séries organisés, lus dans le navigateur. Aucun Docker, aucun abonnement,
aucun compte en ligne.

## Ce que fait l'application (fonctionnel et testé)
- Paquet DSM `.spk` : installation manuelle, icône dans le menu principal, service démarré avec le NAS,
  exécution sous un utilisateur système non root, données conservées aux mises à jour.
- Assistant de premier démarrage (reprenable) : administrateur, langue, dossiers, clé TMDB facultative, indexation en direct.
- **Détection immédiate des ajouts** (comme Plex) : un film ou un épisode copié est indexé ~20 s après la fin
  de la copie ; vérification complète périodique en plus (6 h par défaut).
- Indexation incrémentale en lecture seule : nouveaux / modifiés / disparus / déplacés, ffprobe en arrière-plan
  limité, reprise après interruption, journal des erreurs.
- Identification par les noms (films, `S01E03`, `1x03`, épisodes doubles, spéciaux, dossiers « Saison N »),
  versions regroupées (1080p/4K/éditions), correction manuelle.
- Métadonnées TMDB (français, repli anglais), affiches/fonds/portraits en cache local, sagas, classifications.
- « Retirer de Continuer à regarder » (×  sur la carte, appui long sur la TV) : sans toucher au fichier ni à la
  progression ; l'œuvre revient si on la relance.
- Accueil personnalisable (reprise, prochain épisode, récents, favoris, ma liste, collections, non vus, suggestions).
- Type de bibliothèque vérifié à l'ajout et modifiable en un clic ; les épisodes rangés parmi des films
  sont automatiquement classés dans Séries.
- Dossiers jamais mélangés : une rangée d'accueil et un onglet par dossier (« Séries en cours », « Séries 2021 »…).
- Bibliothèques : tri, filtres (genre, année, résolution, HDR, langue, réalisateur, vu, favoris), 3 vues.
- Fiches film/série immersives, informations techniques complètes, distribution.
- Recherche locale instantanée, insensible aux accents, tolérante aux fautes.
- Lecteur : Direct Play prioritaire, remux sans réencodage, conversion H.264 limitée ; pistes audio,
  sous-titres (internes, externes, incrustés), chapitres, vitesse, plein écran, raccourcis clavier,
  reprise, enchaînement des épisodes, reprise automatique après coupure réseau.
- Profils (admin, utilisateur, enfant) : historique, progression, favoris, liste, préférences de pistes,
  bibliothèques autorisées, contrôle parental.
- Administration : tableau de bord, bibliothèques, identification, utilisateurs, réglages, historique,
  sauvegardes/restauration, diagnostic, journal.
- **Association d'une TV par code** (comme Plex) : la TV affiche un code, on le valide sur `http://<nas>:8890/link`
  depuis un navigateur déjà connecté — aucun mot de passe à taper à la télécommande.
- **Client Android TV** (Xiaomi Box), projet séparé `~/Documents/AndroidStudio/AuroraTV` : même contenu,
  saisie de l'IP du ou des Synology, lecture ExoPlayer (directe/remux/conversion choisie par le NAS selon
  les décodeurs de la box), reprise, épisode suivant, repli VLC par lien signé. Testé sur émulateur
  Google TV ; sur la Xiaomi Box, connexion au NAS et accueil vérifiés, **lecture pas encore testée** (voir son README).

## Documentation
| Document | Contenu |
|---|---|
| [docs/INSTALLATION_DSM.md](docs/INSTALLATION_DSM.md) | installer, permissions, réseau, mise à jour, désinstallation |
| [docs/GUIDE_UTILISATEUR.md](docs/GUIDE_UTILISATEUR.md) | utilisation au quotidien |
| [docs/MAINTENANCE.md](docs/MAINTENANCE.md) | journaux, opérations courantes, FFmpeg |
| [docs/SAUVEGARDE.md](docs/SAUVEGARDE.md) | sauvegarde, restauration, récupération |
| [docs/TESTS.md](docs/TESTS.md) | tests exécutés, résultats, recette sur NAS |
| [docs/LIMITATIONS.md](docs/LIMITATIONS.md) | limites réelles |
| [docs/ROADMAP.md](docs/ROADMAP.md) | état des phases et suite |
| [docs/DEVELOPPEMENT.md](docs/DEVELOPPEMENT.md) | compiler, lancer en local, construire le paquet |
| [ARCHITECTURE.md](ARCHITECTURE.md) | structure, décisions techniques, interdits |

## Construire le paquet
```bash
scripts/build-spk.sh 0011   # numéro de build supérieur au précédent (dernier publié : 0010)
```
Résultat : `build/AURORA-<version>-<build>.spk` (≈ 63 Mo, ffmpeg/ffprobe statiques inclus), vérifié par
`scripts/verify-spk.sh`. **Pas encore installé sur un vrai DS1825+** : voir la recette dans docs/TESTS.md.
