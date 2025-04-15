package dialectquery

import "fmt"

type ClickhouseCluster struct {
	ClusterName string
}

var _ Querier = (*ClickhouseCluster)(nil)

func (c *ClickhouseCluster) CreateTable(tableName string) string {
	q := `CREATE TABLE IF NOT EXISTS %s ON CLUSTER %s (
		version_id Int64,
		is_applied UInt8,
		date Date default now(),
		tstamp DateTime default now()
	  )
	  ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/%s', '{replica}')
		ORDER BY (date) SETTINGS index_granularity = 8192`
	return fmt.Sprintf(q, tableName, c.ClusterName, tableName)
}

func (c *ClickhouseCluster) InsertVersion(tableName string) string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)`
	return fmt.Sprintf(q, tableName)
}

func (c *ClickhouseCluster) DeleteVersion(tableName string) string {
	q := `ALTER TABLE %s ON CLUSTER %s DELETE WHERE version_id = $1 SETTINGS mutations_sync = 2`
	return fmt.Sprintf(q, tableName, c.ClusterName)
}

func (c *ClickhouseCluster) GetMigrationByVersion(tableName string) string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id = $1 ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, tableName)
}

func (c *ClickhouseCluster) ListMigrations(tableName string) string {
	q := `SELECT version_id, is_applied FROM %s ORDER BY version_id DESC`
	return fmt.Sprintf(q, tableName)
}

func (c *ClickhouseCluster) GetLatestVersion(tableName string) string {
	q := `SELECT max(version_id) FROM %s`
	return fmt.Sprintf(q, tableName)
}
