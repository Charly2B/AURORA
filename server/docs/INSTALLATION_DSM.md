# Installer AURORA sur un Synology (DSM 7)

Cible : **DS1825+** (AMD Ryzen V1500B, plateforme Synology `v1000nk`), et plus largement tout NAS
Synology **x86-64** sous **DSM 7.0 ou plus récent**. Aucun Docker, aucun Container Manager, aucune machine virtuelle.

> Statut : le paquet a été construit et vérifié **hors DSM** (structure, scripts, binaires statiques,
> simulation du cycle de vie). Il **n'a pas encore été installé sur un vrai DS1825+**. Suivez la
> « recette sur NAS » de [TESTS.md](TESTS.md#recette-sur-le-ds1825) lors de la première installation.

## 1. Installer le paquet

1. Récupérez `AURORA-<version>-<build>.spk` (dans `build/` après `scripts/build-spk.sh`).
2. DSM › **Centre de paquets** › **Installation manuelle** › choisissez le fichier `.spk`.
3. DSM affiche un avertissement : *paquet d'un éditeur inconnu*. C'est normal pour tout paquet tiers
   depuis DSM 7 (Synology a supprimé la signature des paquets). Acceptez.
4. Choisissez le volume d'installation, puis **Terminé** en laissant « Exécuter après l'installation » coché.

DSM crée alors l'utilisateur système **`aurora`** (sans droit de connexion) sous lequel tourne le service.
AURORA ne tourne jamais en root.

## 2. Donner à AURORA le droit de lire vos vidéos (une seule fois)

AURORA ne voit que les dossiers partagés auxquels l'utilisateur `aurora` a accès. Pour chaque dossier
partagé contenant des vidéos (ex. `Films`, `Series`, `Medias`) :

1. **Panneau de configuration** › **Dossier partagé** › sélectionnez le dossier › **Modifier**.
2. Onglet **Permissions** › menu déroulant en haut : **Utilisateurs internes du système**.
3. Ligne **aurora** › cochez **Lecture seule** › **Enregistrer**.

La lecture seule suffit : AURORA ne modifie, ne déplace et ne supprime jamais un fichier vidéo.

**Disque externe (USB ou eSATA)** : DSM le présente comme un dossier partagé `usbshare1` (ou `esatashare1`),
monté sur `/volumeUSB1/usbshare`. Donnez-lui la même permission de lecture à `aurora` ; il apparaît ensuite
dans l'explorateur d'AURORA comme `volumeUSB1`. Si le disque est débranché, ses vidéos sont masquées
(jamais effacées du catalogue) et réapparaissent avec leur historique à son retour.
L'assistant d'AURORA signale tout dossier encore illisible (icône orange, « accès refusé »).

## 3. Ouvrir AURORA

- Depuis DSM : **menu principal** › icône **AURORA** (ou bouton **Ouvrir** du Centre de paquets).
- Ou directement : `http://<adresse-du-nas>:8890/`

Au premier lancement, l'assistant guide pas à pas : compte administrateur → langue → dossiers de films
et de séries → clé TMDB (facultative) → première indexation. Il reprend où il s'était arrêté en cas
d'interruption.

## 4. Réseau et sécurité

- AURORA écoute sur le port **8890/tcp** (HTTP). Le paquet déclare ce port à DSM (règle « AURORA »
  sélectionnable dans le **pare-feu**) avec `port_forward="no"` : **aucun port n'est ouvert vers Internet**.
- Si le pare-feu DSM est actif, autorisez « AURORA » pour votre réseau local uniquement.
- **Accès à distance** : n'ouvrez pas le port 8890 sur votre box. Préférez :
  - le **VPN** (VPN Server de Synology, ou WireGuard sur la box), puis `http://<nas>:8890` ;
  - ou un **proxy inverse HTTPS** DSM : Panneau de configuration › Portail de connexion › Avancé ›
    Proxy inversé : source `https://aurora.votre-domaine:443` → destination `http://localhost:8890`,
    avec un certificat valide. AURORA détecte alors HTTPS (cookies `Secure`) via l'en-tête
    `X-Forwarded-Proto`, qu'il n'accepte que depuis la machine locale.

## 5. Où sont les données

| Contenu | Emplacement | Conservé à la mise à jour | Conservé à la désinstallation |
|---|---|---|---|
| Programme (aurora, ffmpeg, ffprobe) | `/var/packages/aurora/target` | remplacé | supprimé par DSM |
| Base, sauvegardes, affiches, journaux | `/var/packages/aurora/var` → `/volumeX/@appdata/aurora` | **oui** | **oui** (selon la doc Synology, à confirmer sur votre DSM) |
| Segments vidéo temporaires | `/var/packages/aurora/tmp` → `/volumeX/@apptemp/aurora` | vidé | vidé |
| Vos vidéos | vos dossiers partagés | jamais touchées | jamais touchées |

## 6. Mise à jour

Centre de paquets › Installation manuelle › nouveau `.spk` (numéro de build supérieur). DSM arrête
AURORA ; le script `preupgrade` copie la base dans `backups/avant-maj-…db` ; au redémarrage, AURORA
sauvegarde encore la base avant toute migration de schéma et refuse de démarrer sur une base plus
récente que lui (aucune corruption silencieuse). Voir [SAUVEGARDE.md](SAUVEGARDE.md).

## 7. Désinstallation

Centre de paquets › AURORA › Désinstaller. Les scripts d'AURORA **ne suppriment aucune donnée** :
la base et les sauvegardes restent dans `@appdata/aurora`. Pour tout effacer, supprimez vous-même ce
dossier (File Station, en tant qu'administrateur) après la désinstallation.

## 8. Dépannage rapide

| Symptôme | Cause probable | Solution |
|---|---|---|
| Le paquet ne démarre pas | port 8890 occupé, base illisible | Centre de paquets › AURORA › Journal ; `@appdata/aurora/logs/console.log` |
| Dossier grisé dans l'assistant | permission manquante | étape 2 ci-dessus |
| « ffprobe introuvable » | paquet incomplet | réinstaller le `.spk` |
| Lecture impossible d'un fichier 4K HEVC dans Chrome/Firefox | navigateur sans HEVC, conversion 4K refusée | Safari sur Mac, ou une version 1080p ; voir [LIMITATIONS.md](LIMITATIONS.md) |
