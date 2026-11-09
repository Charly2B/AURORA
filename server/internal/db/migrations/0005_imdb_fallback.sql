-- IMDb complète les recherches TMDB infructueuses ; une identité IMDb reste distincte de TMDB.
ALTER TABLE items ADD COLUMN imdb_id TEXT;
CREATE INDEX items_imdb ON items(library_id, kind, imdb_id) WHERE imdb_id IS NOT NULL;
-- Les identifications établies et les choix manuels restent intacts.
UPDATE items SET match_state='local', metadata_at=NULL
 WHERE kind IN ('movie','series') AND match_state IN ('ambiguous','failed');
