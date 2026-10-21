-- 0.1.0-0002 : nouvelle tentative de correspondance TMDB (seconde recherche en anglais ajoutée).
-- Les œuvres restées « ambiguës » ou « introuvables » sont remises en file ; les corrections manuelles
-- (« locked ») et les correspondances établies ne sont jamais touchées.
UPDATE items SET match_state = 'local', metadata_at = NULL
 WHERE match_state IN ('ambiguous', 'failed') AND kind IN ('movie', 'series');

-- Une erreur réseau de la version 0001 pouvait contenir l'URL TMDB, donc la clé : messages masqués.
UPDATE tasks SET message = 'Métadonnées : erreur réseau (détail masqué)' WHERE message LIKE '%api_key=%';
