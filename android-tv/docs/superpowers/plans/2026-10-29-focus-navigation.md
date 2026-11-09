# Focus Navigation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Changer le contenu des grilles au focus des dossiers et retirer « Parcourir ».
**Architecture:** Callback facultatif de focus des lignes de navigation ; sélection ViewModel idempotente ; producteurs et clés de grille existants conservés.
**Tech Stack:** Kotlin / Compose TV existants.
**Spec:** docs/superpowers/specs/2026-10-29-focus-navigation-design.md

## Global Constraints

- Focus des dossiers/catégories/Tous → contenu correspondant, sans OK, sans déplacer le focus dans la grille.
- Aucun déclenchement d’action de lecture ou de session au focus.
- Préserver cinq colonnes, cadres, fonds et restauration de carte ; pas de nouvelle dépendance ni émulateur.

## Review Focus

- Retour d’une fiche / dossier déjà actif : ne pas effacer la mémoire de focus.
- Défilement rapide : aucune réponse de dossier précédent affichée.
- Ligne supprimée : indices Accueil/Films/Séries décalés de un.

### Task 1: Relier le focus au choix du dossier

**Files:** MainScreen.kt, AppViewModel.kt ; tests existants ; documentation par le contrôleur.

- [ ] Retirer « Parcourir » et corriger les indices de restauration.
- [ ] Ajouter le callback de gain de focus aux catégories, dossiers et Tous ; rendre la sélection idempotente.
- [ ] Conserver OK et la navigation dans la colonne ; build et tests existants, git diff --check.
- [ ] Revue distincte puis installation/vérification sur Xiaomi et Formovie autorisées.
