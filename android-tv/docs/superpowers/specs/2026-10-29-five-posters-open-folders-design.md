# Cinq affiches et dossiers toujours ouverts

Maquette `auroratv-grille-cinq.html` approuvée le 29/10/2026 ; mises à jour Xiaomi et Formovie explicitement demandées.

Afficher cinq affiches plus grandes par ligne, dans les grilles Films/Séries d’un dossier et dans les rangées de « Tous les dossiers ». Conserver les cadres verre, les titres, le tri, les badges, les actions et le focus des affiches. La dernière rangée garde cinq colonnes alignées, sans étirer les derniers éléments.

Dans la sidebar, les dossiers des deux catégories sont affichés en permanence : plus de dépliage ou de repliage au clic. La colonne reste défilante pour les longues listes, Profil en pied, avec les vrais noms et compteurs du NAS. Chaque groupe Films/Séries est entouré d’un cadre visible façon fenêtre macOS : bord arrondi clair, barre de titre et trois petites pastilles décoratives ; groupe actif souligné d’or. Les titres restent des accès à la catégorie, sans masquer ses dossiers.

Approche retenue : grille fixe de cinq colonnes et deux groupes ouverts encadrés ; ajouter seulement une ouverture automatique conserverait inutilement l’action de repli refusée. Préserver les sélections indépendantes par type/NAS, « Tous les dossiers », le retour de fiche, la navigation télécommande, le fond personnel et la sidebar translucide de l’accueil. Ne pas réintroduire la barre de dossiers en haut.

Portée : AuroraTV Android uniquement, 960 × 540 dp avec sidebar 212 dp, API et dépendances inchangées. Ne pas toucher aux modifications précédentes ni au Mac/serveur. Pas de nouvelles fonctions annexes, de tests miroirs de styles, d’émulateur ou de commit. Build/tests et revue distincte, puis installation -r, vérification et lancement sur les deux appareils.
