# Sécurité

## Signaler un problème

Ne publiez jamais un mot de passe, cookie de session, clé TMDB, adresse publique ou extrait de médiathèque dans une issue. Utilisez un canal privé du propriétaire du dépôt.

Incluez : composant concerné, version, étapes minimales, impact et journaux expurgés. Remplacez les IP locales et noms de fichiers personnels.

## Modèle de sécurité

- AURORA est destiné à un réseau local ou un VPN.
- Le port HTTP 8890 ne doit pas être publié directement sur Internet.
- Les comptes AURORA sont distincts des comptes DSM.
- Les clients stockent uniquement des jetons de session protégés par le système.
- Les médias ne sont jamais téléversés vers ce dépôt.

## Secrets interdits dans Git

`.env`, `local.properties`, bases SQLite, sauvegardes, journaux, certificats, clés privées, tokens, builds et fichiers médias sont ignorés. Avant chaque publication, vérifier le diff et effectuer un scan de secrets.
