-- +goose WHEN ${WHEN_PROFILE}=='cluster'
ENGINE = SummingMergeTree()
-- +goose END WHEN
