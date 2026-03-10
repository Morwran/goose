CREATE TABLE users (
    id int NOT NULL
)
ENGINE = ReplicatedSummingMergeTree('/clickhouse/{cluster}/tables/{database}/users', '{replica}')
;