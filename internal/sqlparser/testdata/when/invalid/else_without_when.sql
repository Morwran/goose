-- +goose Up
CREATE TABLE users (id int NOT NULL);
-- +goose ELSE
ENGINE = SummingMergeTree()
-- +goose END WHEN
