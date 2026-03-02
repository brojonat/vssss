-- Metadata table for signal information
CREATE TABLE IF NOT EXISTS signals (
    id INTEGER PRIMARY KEY,
    path TEXT UNIQUE NOT NULL,
    description TEXT NOT NULL,
    type TEXT NOT NULL,
    datatype TEXT NOT NULL,
    unit TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_signals_path ON signals(path);
CREATE INDEX IF NOT EXISTS idx_signals_type ON signals(type);

-- Vector table for embeddings (1536 dimensions for text-embedding-3-small)
CREATE VIRTUAL TABLE IF NOT EXISTS vec_signals USING vec0(
    id INTEGER PRIMARY KEY,
    embedding FLOAT[1536]
);
