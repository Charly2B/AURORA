-- 0.1.8 : « Retirer de Continuer à regarder » (comme Plex). Un film ou une série (pour ses épisodes) retiré
-- des rangées Continuer / Prochain épisode, sans toucher au fichier ni à la progression. L'œuvre revient
-- d'elle-même dès qu'elle est relue (last_played_at > hidden_at).
CREATE TABLE continue_hidden (
  user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  item_id   INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  hidden_at INTEGER NOT NULL,
  PRIMARY KEY (user_id, item_id)
);
