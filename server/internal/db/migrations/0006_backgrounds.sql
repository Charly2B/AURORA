-- Images personnalisées : les octets suivent les sauvegardes/restaurations SQLite.
CREATE TABLE backgrounds (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    mime TEXT NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    size INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    data BLOB NOT NULL
);
