-- +goose Up
CREATE TABLE users (id int NOT NULL);
-- +goose WHEN
ENGINE = SummingMergeTree()
-- +goose END WHEN
