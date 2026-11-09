# Fond continu derrière la sidebar

Maquette `auroratv-fond-continu.html` approuvée le 25/10/2026 ; installation explicitement demandée.

À l’accueil, une seule image couvre toute la largeur de la fenêtre, sidebar comprise. La sidebar laisse voir cette image à travers un voile brun sombre translucide (environ 74 % à gauche, 55 % à droite), conforme à l’aperçu. Garder les textes, les focus, les dossiers dépliables et le panneau Bienvenue lisibles et à leur place.

Déplacer la composition du fond d’accueil au parent de la rangée sidebar/contenu ; ne pas charger ou recadrer une deuxième image pour la sidebar. Préserver exactement le choix personnel par NAS, l’authentification Coil, le fond automatique flouté de secours et le traitement des erreurs. Les autres onglets gardent leur composition actuelle. Approche retenue : couche de fond commune, qui assure la continuité ; deux images séparées risqueraient des raccords incorrects.

Portée : AuroraTV Android uniquement, aucun changement serveur/Mac ni dépendance. Modification visuelle réversible : build et 17 tests existants, contrôle par revue distincte, capture de l’accueil après installation sur Xiaomi et Formovie. Aucun émulateur.
