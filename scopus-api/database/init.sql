CREATE TABLE IF NOT EXISTS research (
    id SERIAL PRIMARY KEY,
    title TEXT,
    journal TEXT,
    year INT,
    doi TEXT UNIQUE,
    cited INT,
    university TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);