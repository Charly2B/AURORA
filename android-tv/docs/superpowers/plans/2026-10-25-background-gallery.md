# Background Gallery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Choisir dans AURORA TV le fond d’accueil parmi les images du NAS connecté.

**Architecture:** Réutiliser AuroraClient et le client OkHttp/Coil authentifié. Stocker localement l’identifiant par NAS, le restaurer à la connexion et l’appliquer à l’accueil avec repli automatique. Un composant dédié affiche la galerie et l’aperçu depuis Profil.

**Tech Stack:** Kotlin, Compose for TV, kotlinx.serialization, SharedPreferences, Coil, OkHttp, JUnit/MockWebServer existants ; pas de dépendance ajoutée.

**Spec:** docs/superpowers/specs/2026-10-25-background-gallery-design.md

## Global Constraints

- Maquette approuvée le 25/10/2026 ; conserver son organisation galerie + aperçu + actions.
- Écran de 960 × 540 dp, colonne de 212 dp ; palette Aurora et composants TV existants.
- Choix local par NAS, restauré au redémarrage ; Automatique efface uniquement ce choix.
- Authentification existante pour liste et images ; URL sans jeton, NAS courant uniquement.
- Galerie vide/404/erreur réseau/image manquante : repli sans plantage ; ne pas effacer un choix pour une panne temporaire.
- Préserver toutes les modifications préexistantes ; pas de commit ni de changement Mac/serveur ni d’installation sur box physique.

## Review Focus

- NAS A → NAS B → NAS A et nouvelle instance de stockage : retrouver le choix de chaque NAS.
- Liste vide, route 404, JSON invalide et serveur indisponible : erreur affichable, Automatique utilisable.
- Session 401 : callback de reconnexion et absence de jeton dans les URL/images.
- Image choisie absente : fond automatique, choix conservé pour retenter plus tard.
- Longue galerie / noms longs / télécommande / 960 × 540 dp : aperçu et application accessibles, Retour annule la sélection provisoire.

### Task 1: Galerie authentifiée, mémorisation et intégration TV

**Files:**
- Create: app/src/main/kotlin/dev/remyj/auroratv/net/Backgrounds.kt (DTO galerie, URL image si nécessaire)
- Modify: app/src/main/kotlin/dev/remyj/auroratv/net/AuroraClient.kt
- Create/Modify: stockage des fonds dans data/ ; AppContainer si nécessaire
- Modify: app/src/main/kotlin/dev/remyj/auroratv/ui/AppViewModel.kt
- Create: app/src/main/kotlin/dev/remyj/auroratv/ui/BackgroundPicker.kt
- Modify: app/src/main/kotlin/dev/remyj/auroratv/ui/MainScreen.kt
- Test: app/src/test/kotlin/dev/remyj/auroratv/BackgroundsTest.kt et tests de stockage adaptés
- Modify: README.md, CLAUDE.md (contrôleur principal uniquement)

**Interfaces:**
- Consumes: AuroraClient.baseUrl, Sessions.interceptor(), client OkHttp partagé par Coil, AppViewModel.server/client/tab/featured.
- Produces: AuroraClient.backgrounds(): List<BackgroundImage>, choix d’identifiant par NAS lisible et modifiable dans le ViewModel, BackgroundPicker(...) appelable depuis Profil.

- [ ] Écrire les tests réseau et stockage avant l’implémentation : contrat JSON réel (id string SHA256, dimensions, size/createdAt Long), cookies liste et image, vide/404/401/JSON invalide, isolation et persistance par NAS, retour Automatique sans toucher un autre NAS.
- [ ] Exécuter les tests ciblés ; consigner l’échec attendu dû à l’absence de la fonctionnalité.
- [ ] Ajouter les DTO et l’appel /backgrounds ; conserver le client et les erreurs API existants. Utiliser les URL relatives du contrat, ou reconstruire l’URL canonique depuis l’identifiant sauvegardé sur le NAS courant.
- [ ] Ajouter le stockage local par NAS ; restaurer à la connexion et effacer ce choix si le NAS est supprimé. La déconnexion ne supprime pas l’apparence. Protéger les retours asynchrones d’un ancien NAS.
- [ ] Réaliser le sélecteur validé dans Profil : grille paresseuse 2 colonnes, aperçu à droite, nom tronqué si nécessaire, focus initial utile, appliquer/automatique, états chargement/vide/erreur avec réessai. La sélection provisoire n’altère pas l’accueil avant application.
- [ ] Appliquer le fond choisi sans flou à l’accueil, avec le voile et le panneau existants. Sur erreur image, afficher le film automatique ; maintenir la préférence. Préserver l’état/focus et les autres onglets.
- [ ] Exécuter ./gradlew :app:testDebugUnitTest :app:assembleDebug --console=plain ; résultats attendus : tous tests réussis et APK produit.
- [ ] Contrôler sur émulateur si disponible : accès Profil, galerie authentifiée, focus, application, Retour, Automatique, redémarrage. Aucun déploiement physique.
- [ ] Mettre à jour la documentation en préservant ses changements existants ; revue par un agent indépendant et correction des défauts avant livraison.
