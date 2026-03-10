-- +goose Up
CREATE TABLE users (id int NOT NULL);
-- +goose WHEN ${WHEN_PROFILE}=='cluster'
ENGINE = ReplicatedSummingMergeTree()
-- +goose ELSE
ENGINE = SummingMergeTree()
-- +goose ELSE
ENGINE = MergeTree()
-- +goose END WHEN
