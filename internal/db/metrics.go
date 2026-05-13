package db

import (
	"database/sql"
	"time"
)

type SystemMetricPoint struct {
	Timestamp   int64   `json:"ts"`
	CPUPercent  float64 `json:"cpu"`
	RAMUsedMB   int64   `json:"ram_used_mb"`
	RAMTotalMB  int64   `json:"ram_total_mb"`
	DiskUsedGB  float64 `json:"disk_used_gb"`
	DiskTotalGB float64 `json:"disk_total_gb"`
}

type ContainerMetricPoint struct {
	Timestamp  int64   `json:"ts"`
	CPUPercent float64 `json:"cpu"`
	MemUsedMB  float64 `json:"mem_used_mb"`
	MemLimitMB float64 `json:"mem_limit_mb"`
}

type ContainerSummary struct {
	ContainerID   string  `json:"container_id"`
	ContainerName string  `json:"container_name"`
	CPUPercent    float64 `json:"cpu"`
	MemUsedMB     float64 `json:"mem_used_mb"`
	MemLimitMB    float64 `json:"mem_limit_mb"`
	LastSeen      int64   `json:"last_seen"`
}

type MonitoringApp struct {
	AppName   string `json:"app_name"`
	Enabled   bool   `json:"enabled"`
	EnabledAt *int64 `json:"enabled_at,omitempty"`
}

func InitMonitoringTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS system_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ts INTEGER NOT NULL,
		cpu_percent REAL NOT NULL,
		ram_used_mb INTEGER NOT NULL,
		ram_total_mb INTEGER NOT NULL,
		disk_used_gb REAL NOT NULL,
		disk_total_gb REAL NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_sm_ts ON system_metrics(ts);

	CREATE TABLE IF NOT EXISTS container_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		ts INTEGER NOT NULL,
		container_id TEXT NOT NULL,
		container_name TEXT NOT NULL,
		cpu_percent REAL NOT NULL,
		mem_used_mb REAL NOT NULL,
		mem_limit_mb REAL NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_cm_ts ON container_metrics(ts);
	CREATE INDEX IF NOT EXISTS idx_cm_cid ON container_metrics(container_id);

	CREATE TABLE IF NOT EXISTS monitoring_apps (
		app_name TEXT PRIMARY KEY,
		enabled INTEGER NOT NULL DEFAULT 0,
		enabled_at INTEGER
	);
	`
	_, err := DB.Exec(schema)
	return err
}

func SaveSystemMetric(ts int64, cpu float64, ramUsedMB, ramTotalMB int64, diskUsedGB, diskTotalGB float64) error {
	_, err := DB.Exec(
		`INSERT INTO system_metrics (ts, cpu_percent, ram_used_mb, ram_total_mb, disk_used_gb, disk_total_gb) VALUES (?, ?, ?, ?, ?, ?)`,
		ts, cpu, ramUsedMB, ramTotalMB, diskUsedGB, diskTotalGB,
	)
	return err
}

func GetSystemMetrics(since, until, bucketSecs int64) ([]SystemMetricPoint, error) {
	rows, err := DB.Query(`
		SELECT
			(ts / ?) * ? AS bucket,
			AVG(cpu_percent),
			CAST(AVG(ram_used_mb) AS INTEGER),
			CAST(MAX(ram_total_mb) AS INTEGER),
			AVG(disk_used_gb),
			MAX(disk_total_gb)
		FROM system_metrics
		WHERE ts >= ? AND ts <= ?
		GROUP BY bucket
		ORDER BY bucket ASC
	`, bucketSecs, bucketSecs, since, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []SystemMetricPoint
	for rows.Next() {
		var p SystemMetricPoint
		if err := rows.Scan(&p.Timestamp, &p.CPUPercent, &p.RAMUsedMB, &p.RAMTotalMB, &p.DiskUsedGB, &p.DiskTotalGB); err != nil {
			continue
		}
		points = append(points, p)
	}
	return points, nil
}

func SaveContainerMetric(ts int64, containerID, name string, cpu, memUsedMB, memLimitMB float64) error {
	_, err := DB.Exec(
		`INSERT INTO container_metrics (ts, container_id, container_name, cpu_percent, mem_used_mb, mem_limit_mb) VALUES (?, ?, ?, ?, ?, ?)`,
		ts, containerID, name, cpu, memUsedMB, memLimitMB,
	)
	return err
}

func GetContainerMetrics(containerID string, since, until, bucketSecs int64) ([]ContainerMetricPoint, error) {
	rows, err := DB.Query(`
		SELECT
			(ts / ?) * ? AS bucket,
			AVG(cpu_percent),
			AVG(mem_used_mb),
			MAX(mem_limit_mb)
		FROM container_metrics
		WHERE container_id = ? AND ts >= ? AND ts <= ?
		GROUP BY bucket
		ORDER BY bucket ASC
	`, bucketSecs, bucketSecs, containerID, since, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []ContainerMetricPoint
	for rows.Next() {
		var p ContainerMetricPoint
		if err := rows.Scan(&p.Timestamp, &p.CPUPercent, &p.MemUsedMB, &p.MemLimitMB); err != nil {
			continue
		}
		points = append(points, p)
	}
	return points, nil
}

func GetAllContainersLatest() ([]ContainerSummary, error) {
	rows, err := DB.Query(`
		SELECT cm.container_id, cm.container_name, cm.cpu_percent, cm.mem_used_mb, cm.mem_limit_mb, cm.ts
		FROM container_metrics cm
		INNER JOIN (
			SELECT container_id, MAX(ts) AS max_ts
			FROM container_metrics
			GROUP BY container_id
		) latest ON cm.container_id = latest.container_id AND cm.ts = latest.max_ts
		ORDER BY cm.container_name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []ContainerSummary
	for rows.Next() {
		var s ContainerSummary
		if err := rows.Scan(&s.ContainerID, &s.ContainerName, &s.CPUPercent, &s.MemUsedMB, &s.MemLimitMB, &s.LastSeen); err != nil {
			continue
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}

func PurgeOldMetrics(retentionDays int) error {
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).Unix()
	if _, err := DB.Exec(`DELETE FROM system_metrics WHERE ts < ?`, cutoff); err != nil {
		return err
	}
	_, err := DB.Exec(`DELETE FROM container_metrics WHERE ts < ?`, cutoff)
	return err
}

func GetMonitoringApps() ([]MonitoringApp, error) {
	rows, err := DB.Query(`SELECT app_name, enabled, enabled_at FROM monitoring_apps ORDER BY app_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []MonitoringApp
	for rows.Next() {
		var a MonitoringApp
		var enabledInt int
		var enabledAt sql.NullInt64
		if err := rows.Scan(&a.AppName, &enabledInt, &enabledAt); err != nil {
			continue
		}
		a.Enabled = enabledInt == 1
		if enabledAt.Valid {
			a.EnabledAt = &enabledAt.Int64
		}
		apps = append(apps, a)
	}
	return apps, nil
}

func SetAppMonitoring(appName string, enabled bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	var enabledAt interface{}
	if enabled {
		enabledAt = time.Now().Unix()
	}
	_, err := DB.Exec(
		`INSERT INTO monitoring_apps (app_name, enabled, enabled_at) VALUES (?, ?, ?)
		 ON CONFLICT(app_name) DO UPDATE SET enabled=excluded.enabled, enabled_at=excluded.enabled_at`,
		appName, enabledInt, enabledAt,
	)
	return err
}
