# Dossiers Films et Séries dans la navigation TV

Maquette `auroratv-dossiers.html` approuvée explicitement le 25/10/2026 : « oui cest bon va y ».

Remplacer les simples entrées Films/Séries par deux groupes dépliables avec le total de chaque catégorie, puis leurs dossiers indentés et leur nombre d’œuvres. Chaque groupe propose « Tous les dossiers » ; choisir un dossier filtre la grille de droite. Les nombres viennent de `/libraries`, jamais de la capture.

Supprimer la barre horizontale des dossiers en haut du contenu, comme demandé explicitement. Conserver les boutons de tri et la vue existante regroupée par dossier pour « Tous les dossiers ». Les deux groupes peuvent être dépliés ; une liste verticale défilante permet d’atteindre les dossiers sur la TV 960 × 540 dp. Profil reste en pied. Conserver Rechercher, Accueil et En cours.

Une seule sélection par catégorie, partagée entre colonne et contenu ; choix et défilement conservés au retour d’une fiche. Au changement de NAS, aucun ancien identifiant de dossier ne doit filtrer le nouveau serveur. Un dossier disparu doit revenir à « Tous les dossiers ». Les noms longs sont tronqués, les compteurs restent visibles, et les lignes sont accessibles à la télécommande.

Approche retenue : déplacement du sélecteur existant dans la colonne de gauche. Le garder en haut créerait un doublon, explicitement refusé. Réutiliser la palette Aurora, Compose TV et l’API existants ; aucun changement Mac/serveur ni nouvelle dépendance. Les fonds personnels déjà réalisés sont conservés. Aucun émulateur ni outil visible non demandé.

Après build/tests et revue distincte, installer puis lancer sur la Xiaomi et le Formovie, autorisés dans cette conversation.
