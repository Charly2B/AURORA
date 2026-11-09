# Sidebar Folders Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Déplacer le choix des dossiers Films/Séries dans la colonne de gauche, avec compteurs et groupes dépliables.
**Architecture:** Partager les bibliothèques chargées entre navigation et contenu. Remonter la sélection par catégorie dans l’état de l’application afin que les clics de la colonne pilotent les requêtes des grilles existantes. Garder le tri et la restauration du focus des cartes.
**Tech Stack:** Kotlin, Compose for TV, API et tests JUnit existants ; pas de dépendance ajoutée.
**Spec:** docs/superpowers/specs/2026-10-25-sidebar-folders-design.md

## Global Constraints

- Maquette approuvée, écran 960 × 540 dp, colonne 212 dp, palette Aurora.
- Dossiers et nombres dynamiques issus du NAS courant ; Films et Séries dépliables, « Tous les dossiers » pour chacun.
- Barre supérieure de dossiers supprimée ; tri existant conservé.
- Conserver fond personnalisé, recherche, reprises, profil, retour de fiche et état des grilles.
- Pas de modification Mac/serveur, de dépendance, de commit, d’émulateur ou d’installation par l’implémenteur.

## Review Focus

- Deux types dans un même dossier : chaque section utilise son compteur et son filtre propre.
- Dossier absent / aucune bibliothèque : repli cohérent vers Tous, sans ancien identifiant.
- Changement de dossier pendant une requête ou changement de NAS : ne pas afficher une réponse dépassée.
- Longue liste / noms longs : défilement, compteurs et Profil accessibles.
- Retour de fiche et passage colonne/contenu : focus et état conservés.

### Task 1: Navigation et filtre partagés

**Files:**
- Modify: ui/MainScreen.kt, ui/AppViewModel.kt ; ui/AuroraTvApp.kt seulement si indispensable à la restauration.
- Create if useful: ui/LibraryNavigation.kt et ui/Sidebar.kt, responsabilités sélection pure et navigation composée.
- Test: app/src/test/kotlin/dev/remyj/auroratv/LibraryNavigationTest.kt pour la sélection, le filtrage par type et le repli d’un dossier absent.
- README.md et CLAUDE.md : contrôleur uniquement.

**Interfaces:** consommer LibraryRef(id,name,movies,series), AuroraClient.libraries() et les grilles existantes ; produire une sélection de dossier par type partagée entre sidebar et LibraryTab.

- [x] Ajouter les tests comportementaux utiles avant leur code et constater l’échec attendu. Pas de tests miroirs de mise en page.
- [x] Charger/partager les bibliothèques indépendamment du succès de /home, gérer vide/erreur et éviter une sélection du NAS précédent.
- [x] Réaliser les groupes dépliables, lignes de dossiers, compteurs, Tous les dossiers, liste défilante et focus télécommande.
- [x] Piloter les grilles depuis ces sélections ; retirer seulement les onglets horizontaux de dossiers et conserver les tris.
- [x] Vérifier ./gradlew :app:testDebugUnitTest :app:assembleDebug --console=plain et git diff --check.
- [x] Revue indépendante, corrections éventuelles, documentation ; installation et lancement sur les deux appareils par le contrôleur.
