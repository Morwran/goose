CREATE TABLE events (
    id int NOT NULL,
    name text,
    created_at timestamp
)
ENGINE = MergeTree()
ORDER BY id;