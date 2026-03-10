-- +goose Up
CREATE TABLE users (
    id int NOT NULL
)
-- +goose WHEN ${WHEN_PROFILE}=='cluster'
ENGINE = ReplicatedSummingMergeTree('/clickhouse/{cluster}/tables/{database}/users', '{replica}')
-- +goose END WHEN
;

SELECT 1;
-- +goose Down
DROP TABLE users;
