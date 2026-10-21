-- AURORA — schéma initial.
-- Règle : une migration publiée n'est jamais modifiée ; toute évolution = nouveau fichier.

CREATE TABLE settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE users (
  id            INTEGER PRIMARY KEY,
  username      TEXT    NOT NULL UNIQUE COLLATE NOCASE,
  password_hash TEXT    NOT NULL,
  role          TEXT    NOT NULL CHECK (role IN ('admin', 'user', 'child')),
  max_age       INTEGER,                    -- âge maximal autorisé (ex. 12), NULL = aucune limite
  hide_unrated  INTEGER NOT NULL DEFAULT 0, -- masquer les contenus sans classification
  all_libraries INTEGER NOT NULL DEFAULT 1,
  prefs         TEXT    NOT NULL DEFAULT '{}',
  created_at    INTEGER NOT NULL
);

CREATE TABLE sessions (
  token_hash TEXT    PRIMARY KEY,           -- SHA-256 du jeton ; le jeton brut n'est jamais stocké
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  last_seen  INTEGER NOT NULL,
  user_agent TEXT    NOT NULL DEFAULT ''
);
CREATE INDEX sessions_user ON sessions(user_id);

CREATE TABLE libraries (
  id               INTEGER PRIMARY KEY,
  name             TEXT    NOT NULL,
  kind             TEXT    NOT NULL CHECK (kind IN ('movies', 'series')),
  path             TEXT    NOT NULL UNIQUE,
  created_at       INTEGER NOT NULL,
  last_scan_at     INTEGER,
  last_scan_status TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE user_libraries (
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, library_id)
);

-- Un « item » est une œuvre : film, série ou épisode. Les fichiers (versions) sont dans media_files.
CREATE TABLE items (
  id             INTEGER PRIMARY KEY,
  library_id     INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
  kind           TEXT    NOT NULL CHECK (kind IN ('movie', 'series', 'episode')),
  parent_id      INTEGER REFERENCES items(id) ON DELETE CASCADE, -- épisode -> série
  match_key      TEXT    NOT NULL,          -- clé de regroupement des versions
  title          TEXT    NOT NULL,
  sort_title     TEXT    NOT NULL,
  original_title TEXT    NOT NULL DEFAULT '',
  year           INTEGER,
  overview       TEXT    NOT NULL DEFAULT '',
  tagline        TEXT    NOT NULL DEFAULT '',
  season         INTEGER,
  episode        INTEGER,
  episode_end    INTEGER,                   -- épisodes doubles (S01E01E02)
  air_date       TEXT    NOT NULL DEFAULT '',
  runtime        INTEGER,                   -- minutes (métadonnées)
  genres         TEXT    NOT NULL DEFAULT '[]',
  rating         REAL,
  rating_source  TEXT    NOT NULL DEFAULT '',
  content_rating TEXT    NOT NULL DEFAULT '',
  min_age        INTEGER,                   -- âge minimal dérivé de content_rating, NULL = inconnu
  status         TEXT    NOT NULL DEFAULT '',
  tmdb_id        INTEGER,
  poster         TEXT    NOT NULL DEFAULT '',
  backdrop       TEXT    NOT NULL DEFAULT '',
  still          TEXT    NOT NULL DEFAULT '',
  trailer        TEXT    NOT NULL DEFAULT '',
  match_state    TEXT    NOT NULL DEFAULT 'local'
                 CHECK (match_state IN ('local', 'matched', 'ambiguous', 'locked', 'failed')),
  added_at       INTEGER NOT NULL,
  updated_at     INTEGER NOT NULL,
  metadata_at    INTEGER,
  UNIQUE (library_id, kind, match_key)
);
CREATE INDEX items_parent  ON items(parent_id, season, episode);
CREATE INDEX items_kind    ON items(kind, added_at);
CREATE INDEX items_sort    ON items(kind, sort_title);
CREATE INDEX items_match   ON items(match_state);

CREATE TABLE seasons (
  series_id INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  season    INTEGER NOT NULL,
  title     TEXT    NOT NULL DEFAULT '',
  overview  TEXT    NOT NULL DEFAULT '',
  poster    TEXT    NOT NULL DEFAULT '',
  air_date  TEXT    NOT NULL DEFAULT '',
  PRIMARY KEY (series_id, season)
);

CREATE TABLE credits (
  item_id   INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  name      TEXT    NOT NULL,
  role      TEXT    NOT NULL CHECK (role IN ('actor', 'director', 'creator')),
  character TEXT    NOT NULL DEFAULT '',
  ord       INTEGER NOT NULL DEFAULT 0,
  profile   TEXT    NOT NULL DEFAULT ''
);
CREATE INDEX credits_item ON credits(item_id, role, ord);
CREATE INDEX credits_name ON credits(name);

CREATE TABLE collections (
  id         INTEGER PRIMARY KEY,
  name       TEXT    NOT NULL,
  kind       TEXT    NOT NULL CHECK (kind IN ('tmdb', 'manual')),
  tmdb_id    INTEGER UNIQUE,
  overview   TEXT    NOT NULL DEFAULT '',
  poster     TEXT    NOT NULL DEFAULT '',
  backdrop   TEXT    NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL
);

CREATE TABLE collection_items (
  collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
  item_id       INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  ord           INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (collection_id, item_id)
);

CREATE TABLE media_files (
  id            INTEGER PRIMARY KEY,
  library_id    INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
  item_id       INTEGER REFERENCES items(id) ON DELETE SET NULL,
  path          TEXT    NOT NULL UNIQUE,
  size          INTEGER NOT NULL,
  mtime         INTEGER NOT NULL,
  present       INTEGER NOT NULL DEFAULT 1,
  missing_since INTEGER,
  version_label TEXT    NOT NULL DEFAULT '',
  edition       TEXT    NOT NULL DEFAULT '',
  container     TEXT    NOT NULL DEFAULT '',
  duration      REAL,
  bitrate       INTEGER,
  video_stream  INTEGER,                         -- index absolu du flux vidéo principal
  video_codec   TEXT    NOT NULL DEFAULT '',
  video_profile TEXT    NOT NULL DEFAULT '',
  pix_fmt       TEXT    NOT NULL DEFAULT '',
  width         INTEGER,
  height        INTEGER,
  hdr           TEXT    NOT NULL DEFAULT '',  -- '', 'HDR10', 'HLG', 'Dolby Vision', 'Dolby Vision + HDR10'
  audio         TEXT    NOT NULL DEFAULT '[]',
  subtitles     TEXT    NOT NULL DEFAULT '[]',
  external_subs TEXT    NOT NULL DEFAULT '[]',  -- sous-titres externes voisins (.srt, .vtt, .ass)
  chapters      TEXT    NOT NULL DEFAULT '[]',
  probe_state   TEXT    NOT NULL DEFAULT 'pending' CHECK (probe_state IN ('pending', 'ok', 'error')),
  probe_error   TEXT    NOT NULL DEFAULT '',
  probed_at     INTEGER,
  added_at      INTEGER NOT NULL
);
-- Index couvrant : les vignettes calculent résolution/durée/HDR des fichiers présents sans lire la table.
CREATE INDEX media_files_item    ON media_files(item_id, present, height, width, duration, hdr);
CREATE INDEX media_files_library ON media_files(library_id, present);
CREATE INDEX media_files_probe   ON media_files(probe_state);
CREATE INDEX media_files_size    ON media_files(size);

CREATE TABLE user_state (
  user_id        INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  item_id        INTEGER NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  position       REAL    NOT NULL DEFAULT 0,
  duration       REAL,
  watched        INTEGER NOT NULL DEFAULT 0,
  play_count     INTEGER NOT NULL DEFAULT 0,
  last_played_at INTEGER,
  last_file_id   INTEGER,
  favorite       INTEGER NOT NULL DEFAULT 0,
  watchlist      INTEGER NOT NULL DEFAULT 0,
  watchlist_at   INTEGER,
  PRIMARY KEY (user_id, item_id)
);
CREATE INDEX user_state_recent ON user_state(user_id, last_played_at);

CREATE TABLE tasks (
  id         INTEGER PRIMARY KEY,
  kind       TEXT    NOT NULL,
  library_id INTEGER,
  status     TEXT    NOT NULL CHECK (status IN ('running', 'done', 'failed', 'cancelled', 'interrupted')),
  started_at INTEGER NOT NULL,
  ended_at   INTEGER,
  total      INTEGER NOT NULL DEFAULT 0,
  done       INTEGER NOT NULL DEFAULT 0,
  errors     INTEGER NOT NULL DEFAULT 0,
  message    TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE task_errors (
  task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  path    TEXT    NOT NULL,
  message TEXT    NOT NULL,
  at      INTEGER NOT NULL
);

-- Recherche plein texte, insensible aux accents. rowid = items.id.
CREATE VIRTUAL TABLE items_fts USING fts5(
  title, original_title, people, extra,
  tokenize = 'unicode61 remove_diacritics 2'
);
