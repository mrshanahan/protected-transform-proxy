CREATE TABLE IF NOT EXISTS
    routes
    ( id TEXT PRIMARY KEY
    , key_hash BLOB NOT NULL
    , key_salt BLOB NOT NULL
    , route TEXT NOT NULL
    );
