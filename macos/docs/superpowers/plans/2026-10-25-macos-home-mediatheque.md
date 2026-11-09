# Accueil « MEDIATHEQUE AURORA » Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reproduire dans AURORA macOS l’accueil validé : colonne gauche conservée, fond Aurora, aucune vignette, titre centré à droite.

**Architecture:** Un visuel local remplit le `MainShellView` quand l’onglet Accueil est actif. La colonne reste interactive sur un voile sombre ; `HomeView` devient un écran de titre tout en conservant le chargement du modèle nécessaire aux compteurs.

**Tech Stack:** Swift 6, SwiftUI, Asset Catalog, macOS 15+.

**Spec:** `docs/superpowers/specs/2026-10-25-macos-home-mediatheque-design.md`

## Global Constraints

- macOS seulement ; ne toucher à aucun autre client ni au serveur.
- Texte exact : `MEDIATHEQUE AURORA`.
- Conserver toutes les destinations et actions de navigation.
- Build final, DMG, sauvegarde/install/vérification/lancement obligatoires.

## Review Focus

- Petite fenêtre : le titre reste centré et ne déborde pas.
- Accueil actif : le fond couvre colonne et contenu sans bloquer les clics.
- Autres onglets : leur fond et leur comportement restent inchangés.
- Colonne redimensionnable : le titre reste centré dans la zone droite restante.
- Données vides ou NAS lent : l’accueil reste visible et la navigation utilisable.

---

### Task 1: Fond local et accueil minimal

**Files:**
- Create: `AURORA/Assets.xcassets/MediathequeAuroraBackdrop.imageset/*`
- Modify: `AURORA/Features/Catalog/HomeView.swift`
- Modify: `AURORA/App/MainShellView.swift`
- Test: `AURORATests/HomePresentationTests.swift`

**Interfaces:**
- Produces: `HomePresentation.title == "MEDIATHEQUE AURORA"` et écran d’accueil sans collection visuelle.

- [x] Écrire le test du titre et de la présentation sans rangées.
- [x] Vérifier le test rouge.
- [x] Ajouter le fond approuvé à l’Asset Catalog.
- [x] Implémenter le fond commun, le voile de colonne et le titre centré.
- [x] Retirer le héros/rangées et les mini-vignettes « En cours » sans modifier les autres onglets.
- [x] Vérifier le test vert et toute la suite.

### Task 2: Livraison macOS

**Files:**
- Modify: `outputs/AURORA-0.1.0.dmg` via les scripts existants.

- [x] Construire la Release avec `Scripts/build-release.sh`.
- [x] Créer et vérifier le DMG avec `Scripts/create-dmg.sh`.
- [x] Sauvegarder l’ancienne app, installer/vérifier dans `/Users/remyj/Applications/MesApps/AURORA.app`, puis lancer avec `Scripts/install-and-launch.sh`.
- [x] Vérifier que le processus AURORA est actif et que la copie installée correspond à la build.

