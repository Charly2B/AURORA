# Galerie de fonds AURORA TV

Maquette approuvée par l’utilisateur le 25/10/2026 (« oui cest bon »).

Dans Profil → Fond d’écran, afficher les images du NAS connecté, une sélection à la télécommande, un aperçu de l’accueil et « Utiliser ce fond ». Appliquer revient à l’accueil. « Automatique » restaure le film flouté actuel. Retour depuis la galerie revient au profil sans appliquer la sélection provisoire.

Le choix est local à cette application et mémorisé par NAS, y compris après redémarrage. Les fonds personnels apparaissent nets, sous le voile et le panneau Bienvenue existants. L’image choisie indisponible laisse apparaître le fond automatique ; la préférence est conservée pour une prochaine tentative.

API existante : GET /api/v1/backgrounds → {images:[{id,name,url,width,height,size,createdAt}]}, puis GET /api/v1/backgrounds/{id}/image. Utiliser l’authentification AURORA et le chargeur Coil existants ; aucune saisie d’adresse ni nouvelle connexion. Galerie vide, ancienne API (404), erreur réseau et image manquante : état explicite, option Automatique accessible, aucun plantage. Une session expirée conserve le comportement de reconnexion existant.

Approches : galerie dans Profil retenue, car les réglages y sont déjà regroupés ; ajout d’un bouton sur l’accueil écarté, car cet écran conserve sa composition validée. Aucun import, suppression, changement serveur ou modification de l’application Mac dans ce travail demandé explicitement dans AuroraTV. Conserver la palette partagée « Cinémathèque » et les composants visuels existants.

Contraintes : écran de 960 × 540 dp, colonne de 212 dp, galerie paresseuse, aperçu et actions accessibles à la télécommande ; Android 11 compatible. Ne pas modifier les changements préexistants de README.md, CLAUDE.md ou Models.kt. Ne pas installer sur une box physique sans demande dédiée.
