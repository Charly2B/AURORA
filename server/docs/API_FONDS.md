# API — galerie de fonds d’écran (AURORA 0.1.11)

Le serveur héberge une galerie commune. **Chaque application choisit et mémorise son propre fond**,
par exemple l’identifiant d’une image dans ses préférences locales par serveur. Aucun réglage global
ne sélectionne de fond, et `/api/v1/home` ne change pas.

## Authentification

Toutes les routes nécessitent une session AURORA : cookie `aurora_session` ou en-tête
`Authorization: Bearer <jeton>`. Les URL restent relatives au serveur connecté et ne contiennent
jamais de jeton. Les requêtes d’images doivent porter la même authentification que les appels API.
L’import exige en plus un profil administrateur et `X-Aurora: 1` (CSRF).

## Lister les images

`GET /api/v1/backgrounds` → HTTP 200, sans mise en cache de la liste :

```json
{
  "images": [
    {
      "id": "<sha256 de 64 caractères hexadécimaux>",
      "name": "Montagnes.jpg",
      "url": "/api/v1/backgrounds/<id>/image",
      "width": 3840,
      "height": 2160,
      "size": 2456789,
      "createdAt": 1792922400
    }
  ]
}
```

`size` = octets ; `createdAt` = secondes Unix ; dimensions = pixels. Ordre : import récent d’abord,
puis identifiant pour départager. Sans images : `{"images":[]}`. `id` est le SHA-256 des octets stockés et reste stable pour un même résultat d’adaptation.

## Récupérer une image

`GET /api/v1/backgrounds/{id}/image` → octets stockés (après adaptation éventuelle), `Content-Type: image/jpeg` ou `image/png`.
Cache privé 30 jours, image immuable, ETag et requêtes conditionnelles pris en charge.
Image inconnue : 404 ; session manquante/expirée : 401. `HEAD` est également accepté.

Pour un client : charger la liste, laisser choisir une image, mémoriser son `id` localement,
charger `url` en la résolvant contre l’adresse du NAS. Garder un fond de secours si la galerie est
vide, si le serveur n’a pas encore cette API (404), ou si l’image choisie est indisponible.

## Importer (administrateur)

`POST /api/v1/admin/backgrounds` avec `Content-Type: multipart/form-data; boundary=…` et exactement
un fichier dans le champ **`image`**. Laisser la bibliothèque HTTP définir la boundary ; ne pas envoyer
de JSON/base64. Pour plusieurs fichiers, effectuer un appel par image. L’interface d’administration
permet la sélection multiple et effectue ces envois successivement.

Réponse : métadonnées d’une image (mêmes champs que ci-dessus), HTTP **201** si ajoutée, **200** si
les mêmes octets stockés existent déjà (nom et date d’origine conservés). L’import ne change aucun fond client.

JPEG/PNG seulement, 50 Mio maximum en entrée. Depuis 0.1.11, les grandes images sont adaptées
automatiquement à un rectangle de 3840 × 2160 maximum, sans agrandissement ni recadrage ;
proportions, format et transparence PNG conservés. Les images plus petites restent inchangées.
Les métadonnées `width`, `height` et `size` décrivent l’image stockée et servie après adaptation.
Protection du serveur : sources jusqu’à 128 mégapixels et 32768 pixels/côté, conversion sérialisée
avec délai de 60 s. FFmpeg est fourni dans le paquet DSM ; s’il manque, un import exigeant une
adaptation répond 503.
400 pour image invalide, tronquée, démesurée ou multipart incorrect ; 413 pour corps
trop volumineux ; 401 sans session ; 403 sans droits administrateur ou sans `X-Aurora: 1`.
Erreurs JSON : `{"error":"message"}`. Aucune route de suppression : les images restent conservées.

Les images sont stockées dans la base SQLite (migration 0006) et incluses dans les sauvegardes
AURORA, y compris leurs octets. Les dossiers de vidéos ne sont jamais modifiés.
