# Continuous Background Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Afficher le fond d’accueil aussi derrière la sidebar, conformément à la maquette approuvée.
**Architecture:** Couche de fond commune sous la Row principale ; sidebar translucide à l’accueil, panneau Bienvenue conservé dans le contenu.
**Tech Stack:** Compose TV et Coil existants.
**Spec:** docs/superpowers/specs/2026-10-25-continuous-background-design.md

## Global Constraints

- Préserver les modifications précédentes, le choix de fond et le comportement de navigation.
- Une seule image à l’échelle de toute la fenêtre ; voile sombre derrière le menu.
- Aucun autre onglet redessiné, aucune dépendance, aucun émulateur, aucun commit.

## Review Focus

- Pas de double image ni de raccord de recadrage à la limite de la sidebar.
- Repli automatique et 401 des images conservés.
- Textes et focus lisibles sur un fond clair.
- Aucun changement des dimensions et du focus de la navigation.

### Task 1: Déplacer le fond sous la navigation

**Files:** ui/MainScreen.kt ; README.md/CLAUDE.md par le contrôleur.

- [x] Extraire les couches du fond actuel dans un composable privé, posé avant la Row dans une Box plein écran à l’accueil.
- [x] Conserver le panneau Bienvenue à sa place ; sidebar avec voile translucide conforme à l’aperçu.
- [x] Exécuter les 17 tests existants et assembleDebug, puis git diff --check. Pas de tests qui reproduisent le style.
- [x] Revue distincte ; installation, vérification des empreintes, lancement et capture sur les deux appareils autorisés.
