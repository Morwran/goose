-- +goose Up
CREATE TABLE events (
    id int NOT NULL,
    name text,
    created_at timestamp
)
-- +goose WHEN ${WHEN_PROFILE}=='cluster'
ENGINE = ReplicatedMergeTree('/clickhouse/{cluster}/tables/{database}/events', '{replica}')
-- +goose ELSE
ENGINE = MergeTree()
-- +goose END WHEN
ORDER BY id;
-- +goose Down
DROP TABLE events;
