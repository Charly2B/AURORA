# Guide utilisateur

## L'essentiel
1. Ouvrez AURORA (icône dans DSM, ou `http://<nas>:8890`).
2. Trouvez un film ou une série (accueil, **Films**, **Séries**, ou la loupe — raccourci `/`).
3. **Lecture**. C'est tout : AURORA choisit seul la façon la plus légère de lire le fichier.

Sur la **box Android TV** : app *AURORA* (projet AuroraTV). Au premier lancement, saisissez l'IP du
Synology (le port 8890 est ajouté seul). La TV affiche alors un **code** (ex. `K7M-2PQ`) : sur un ordinateur
ou un téléphone déjà connecté à AURORA, ouvrez `http://<nas>:8890/link` (ou menu du profil › **Associer une TV**)
et saisissez-le ; AURORA affiche alors le nom et l'adresse de l'appareil : **confirmez** seulement si le code
est sur votre TV, devant vous. La TV se connecte seule sur votre profil. Le code expire après 10 minutes (un nouveau
s'affiche). « Utiliser un mot de passe » reste possible : c'est le compte **AURORA**, pas celui du Synology.
D'autres NAS s'ajoutent depuis
**Profil › Ajouter un Synology**. Vu/non vu, reprise et listes sont partagés avec le navigateur.

## Accueil
Rangées : *Continuer à regarder*, *Prochain épisode*, *Films / Séries récemment ajoutés*, *Favoris*,
*Ma liste*, *Collections*, *Films non visionnés*, *Suggestions de ma bibliothèque* (et *Séries en cours*,
masquée par défaut). Une rangée vide n'apparaît jamais. Ordre et visibilité : menu du profil ›
**Personnaliser l'accueil**. Le bandeau met en avant un film en cours ou récent, sans défilement automatique.

**Retirer de Continuer à regarder** : survolez la carte et cliquez sur **×** (sur la TV : appui long sur OK,
puis « Retirer »). Le film, ou la série entière pour un épisode, quitte les rangées *Continuer à regarder* et
*Prochain épisode*. Le fichier n'est pas supprimé, la progression est gardée, et l'œuvre revient si vous la
relancez. Chaque profil a ses propres retraits.

## Bibliothèques
Un film ou un épisode copié dans un dossier d'AURORA apparaît **tout seul, environ 20 secondes après la fin
de la copie** (Administration › Réglages › « Détecter immédiatement les ajouts », activé par défaut). Une
vérification complète a lieu en plus toutes les 6 heures, et le bouton **Analyser** (Administration ›
Bibliothèques) reste disponible.
- **Un panneau par dossier** dans « Tous les dossiers » : cadre arrondi façon fenêtre macOS, nom du dossier
  dans la barre de titre (site et TV).
- **Un espace par dossier** : chaque dossier ajouté (« Séries en cours », « Séries 2021 », « Séries complètes »…)
  a son onglet en haut des pages Films et Séries, et sa propre rangée à l'accueil (« Tout voir »).
  L'onglet « Tous les dossiers » les présente l'un sous l'autre : nom du dossier en titre, ses œuvres
  dessous (60 au plus, puis « Voir les N… ») ; jamais mélangés. Le dernier onglet choisi est mémorisé.
- **Tri** : titre, date d'ajout, année, durée, note, qualité vidéo — croissant ou décroissant.
- **Filtres** (seules les valeurs présentes sont proposées) : genre, année, résolution, langue audio,
  réalisateur/créateur, vu/non vu, HDR, favoris. Un clic sur un nom (fiche) filtre sur cette personne.
- **Vues** : grille, compacte, liste (mémorisée sur l'appareil).

## Fiche film
Affiche, fond, titre et titre original, année, durée, classification, note *avec sa provenance*
(ex. « 7,8 /10 TMDB »), genres, réalisation, distribution, résumé, informations techniques (résolution,
codec, HDR, pistes audio, sous-titres, taille), versions disponibles, progression et dernière lecture.
Boutons : **Lecture / Reprendre à …**, **Depuis le début**, **Ma liste**, **Favoris**, **Bande-annonce**
(ouvre YouTube, nécessite Internet), choix de **version** si le film existe en plusieurs fichiers.

## Séries
Série › saisons (onglets) › épisodes avec vignette, résumé, durée, progression, statut vu.
Le bouton principal lance le **prochain épisode** : l'épisode commencé s'il y en a un, sinon le premier
non vu après le dernier vu. Marquer vu/non vu : un épisode (menu « … »), une saison, ou toute la série.
Épisodes doubles (S01E01E02) et épisodes spéciaux (saison 0, listés en dernier) sont gérés.
AURORA ne lance jamais un autre épisode que celui que vous avez choisi ; l'enchaînement automatique
(compte à rebours de 8 s, annulable) se règle dans **Mon profil**.

## Lecteur
| Action | Souris / tactile | Clavier |
|---|---|---|
| Lecture / pause | clic sur l'image | Espace, K |
| ±10 s | boutons ↺ ↻ | ← →, J L |
| Volume / muet | survol de l'icône | ↑ ↓, M |
| Plein écran | bouton, double-clic | F |
| Sous-titres suivants | panneau « Audio et sous-titres » | C |
| Quitter | flèche en haut à gauche | Échap |

Panneaux : **Audio et sous-titres** (toutes les pistes du fichier, sous-titres externes compris),
**Vitesse** (0,75× à 2×), **Chapitres** (si le fichier en contient). La position est enregistrée toutes
les 10 s, à la pause et à la fermeture. Au-delà de 92 % (ou s'il reste moins de 3 min), l'œuvre est
marquée vue. Coupure réseau : AURORA tente trois reprises automatiques à la position courante.

### Préférences de pistes (Mon profil)
- **Langue audio préférée** : piste choisie automatiquement si elle existe.
- **Sous-titres** : *automatiques* (forcés seulement si l'audio est dans votre langue, complets sinon),
  *toujours*, ou *jamais*. Taille petite / moyenne / grande.
- Les sous-titres **image** (PGS des Blu-ray, VobSub des DVD) ne sont jamais activés d'office :
  les afficher oblige le NAS à convertir la vidéo (incrustation).

## Profils
Chaque profil a son historique, sa progression, ses favoris et sa liste. Les profils **enfant**
n'affichent que les contenus classés jusqu'à l'âge choisi (et, par défaut, masquent les contenus sans
classification). Le contrôle parental repose sur les classifications TMDB : **il n'est pas infaillible**
si les métadonnées manquent ou sont erronées.

## Affiches
Avec une clé TMDB, chaque film et chaque série reçoit son affiche et son fond (recherche en français, puis
en anglais si le fichier porte le titre original). Sans fiche trouvée, AURORA compose une affiche à partir
d'une image représentative de la vidéo et du titre. Une bibliothèque de séries déclarée comme « Films »
n'obtient pas d'affiches : AURORA le signale et la corrige en un clic (Admin › Bibliothèques).
Un épisode rangé parmi vos films (nommé « Série S01E06 ») est classé automatiquement dans Séries.

Mauvaise œuvre associée : fiche › **Corriger l'identification** (recherche TMDB avec aperçus, ou titre
manuel). Bonne œuvre mais affiche qui ne plaît pas : fiche › **Changer l'affiche** — toutes les affiches
et tous les fonds proposés par TMDB, filtrables par langue (français, anglais, sans texte) ; le choix est
conservé par les mises à jour automatiques (seule une nouvelle identification, donc une autre œuvre, le remplace).

## Administration (administrateurs)
Tableau de bord, bibliothèques (ajout, analyse, retrait du catalogue sans toucher aux fichiers),
**Identification** (corriger une correspondance incertaine, identifier un épisode non reconnu),
utilisateurs, réglages (TMDB, analyses automatiques, conversions), maintenance (historique, sauvegardes,
diagnostic, journal), à propos.
