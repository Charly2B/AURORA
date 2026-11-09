# Five Posters and Open Folders Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Affiches plus grandes par rangées de cinq et groupes de dossiers Films/Séries ouverts en permanence, avec cadres macOS.
**Architecture:** Passer les deux grilles existantes à cinq colonnes, partager la constante si utile. Retirer l’état et les actions de repli obsolètes ; encadrer les groupes de la sidebar sans perdre défilement et restauration du focus.
**Tech Stack:** Kotlin/Compose TV et tests existants.
**Spec:** docs/superpowers/specs/2026-10-29-five-posters-open-folders-design.md

## Global Constraints

- Cinq affiches par ligne dans un dossier et dans Tous les dossiers, y compris une dernière rangée partielle.
- Les deux groupes restent ouverts ; noms/compteurs issus du NAS, défilement et Profil fixe.
- Cadres macOS visibles autour des groupes, sélection et navigation préservées.
- Aucun changement de fond, API, dépendance, Mac/serveur ou modification préexistante.

## Review Focus

- Dernière rangée : largeur et alignement des affiches conservés.
- Groupe Séries plus haut que le viewport : ses derniers dossiers restent accessibles à la télécommande.
- Clic sur Films/Séries : navigation sans repli ; sélections cohérentes entre colonne et contenu.
- Retour de fiche et changement de NAS : focus et filtre corrects.

### Task 1: Grilles et navigation approuvées

**Files:** ui/MainScreen.kt, ui/LibraryNavigation.kt, ui/AppViewModel.kt si nécessaire ; LibraryNavigationTest.kt pour adapter les tests d’ouverture/repli devenus obsolètes. README/CLAUDE : contrôleur.

- [x] Mettre cinq colonnes dans PosterGrid et les rangées GroupedGrid, avec emplacements vides de fin cohérents.
- [x] Supprimer le repli des dossiers et son état obsolète ; adapter les tests comportementaux concernés sans tester les styles.
- [x] Réaliser les deux cadres macOS permanents, conserver le focus et le défilement sur 960 × 540 dp.
- [x] Exécuter testDebugUnitTest, assembleDebug et git diff --check ; revue par un agent distinct.
- [x] Installer -r puis lancer sur Xiaomi/Formovie ; vérifier l’APK installé et l’affichage accessible sans émulateur.
