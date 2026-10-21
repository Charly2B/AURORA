# Installation

## Serveur Synology

Prérequis : DSM 7, NAS x86-64, Go, Node.js/npm, `curl`, `shasum` ou `sha256sum` sur la machine de build.

```bash
cd server
scripts/build-spk.sh 0011
```

Le paquet apparaît dans `server/build/AURORA-0.1.8-0011.spk`. Dans DSM : **Centre de paquets → Installation manuelle**, sélectionner le `.spk`, puis ouvrir AURORA sur `http://<nas>:8890`.

L’assistant crée le compte administrateur AURORA, choisit les dossiers en lecture seule et configure éventuellement TMDB. Ne réutilisez pas forcément les identifiants DSM : ce sont deux comptes distincts.

Documentation détaillée : [`server/docs/INSTALLATION_DSM.md`](../server/docs/INSTALLATION_DSM.md).

## Android TV

Prérequis : Android Studio/JDK 17, Android SDK 37. Créer localement `android-tv/local.properties` :

```properties
sdk.dir=/chemin/vers/Android/sdk
```

```bash
cd android-tv
./gradlew :app:testDebugUnitTest :app:assembleDebug
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

Sur la TV : ajouter l’adresse du NAS, puis privilégier l’association par code. Depuis un navigateur déjà connecté à AURORA, ouvrir `http://<nas>:8890/link`, saisir le code et confirmer l’appareil.

## macOS

Prérequis : macOS 15+, Xcode et XcodeGen. Le projet récupère MPVKit 1.0.0 via Swift Package Manager ; VLC n’est ni installé ni lancé.

```bash
cd macos
Scripts/build-release.sh
Scripts/create-dmg.sh \
  Build/DerivedData/Build/Products/Release/AURORA.app \
  AURORA-0.1.0.dmg
```

Ouvrir le DMG, puis glisser `AURORA.app` vers Applications. La distribution locale est signée ad hoc et non notarée ; macOS peut demander une première ouverture explicite par clic droit → **Ouvrir**.

Pour l’installation automatisée du poste de développement :

```bash
Scripts/install-and-launch.sh Build/DerivedData/Build/Products/Release/AURORA.app
```

Le script sauvegarde l’ancienne app, vérifie tous les fichiers et frameworks, installe dans `/Users/remyj/Applications/MesApps/AURORA.app`, puis lance AURORA.

## Réseau

- réseau local : `http://<nas>:8890` ;
- accès distant : VPN recommandé ; ne pas exposer directement le port 8890 ;
- HTTPS : utiliser un proxy inverse DSM ;
- le serveur, Android TV et le Mac doivent pouvoir joindre la même adresse.

## Mise à jour

- serveur : sauvegarder depuis l’administration, installer le nouveau `.spk` ; les données restent dans le répertoire du paquet ;
- Android : `adb install -r ...` conserve la session ;
- macOS : remplacer l’app. Le Trousseau conserve les sessions par adresse de NAS.
