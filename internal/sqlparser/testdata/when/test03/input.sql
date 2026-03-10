-- +goose Up
CREATE TABLE users (
    id int NOT NULL
)
-- +goose WHEN ${WHEN_PROFILE}=='cluster'
ENGINE = ReplicatedSummingMergeTree('/clickhouse/{cluster}/tables/{database}/users', '{replica}')
-- +goose WHEN ${WHEN_PROFILE}=='standalone'
ENGINE = SummingMergeTree()
-- +goose ELSE
ENGINE = MergeTree()
-- +goose END WHEN
;
-- +goose Down
DROP TABLE users;
