package db

import (
	"database/sql"
	"time"
)

type CronJob struct {
	ID                 int        `json:"id"`
	Name               string     `json:"name"`
	Schedule           string     `json:"schedule"`
	Type               string     `json:"type"`
	Command            string     `json:"command"`
	ScriptContent      string     `json:"script_content"`
	CurlURL            string     `json:"curl_url"`
	CurlMethod         string     `json:"curl_method"`
	CurlHeaders        string     `json:"curl_headers"`
	CurlBody           string     `json:"curl_body"`
	CurlExpectedStatus int        `json:"curl_expected_status"`
	Enabled            bool       `json:"enabled"`
	CreatedAt          time.Time  `json:"created_at"`
	LastRunAt          *time.Time `json:"last_run_at"`
	LastExitCode       *int       `json:"last_exit_code"`
	LastOutput         string     `json:"last_output"`
}

type CronRun struct {
	ID         int       `json:"id"`
	JobID      int       `json:"job_id"`
	RanAt      time.Time `json:"ran_at"`
	ExitCode   int       `json:"exit_code"`
	Output     string    `json:"output"`
	DurationMS int64     `json:"duration_ms"`
}

func InitCronTables() error {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS cron_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			schedule TEXT NOT NULL,
			type TEXT NOT NULL,
			command TEXT NOT NULL DEFAULT '',
			script_content TEXT NOT NULL DEFAULT '',
			curl_url TEXT NOT NULL DEFAULT '',
			curl_method TEXT NOT NULL DEFAULT 'GET',
			curl_headers TEXT NOT NULL DEFAULT '{}',
			curl_body TEXT NOT NULL DEFAULT '',
			curl_expected_status INTEGER NOT NULL DEFAULT 200,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_run_at DATETIME,
			last_exit_code INTEGER,
			last_output TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS cron_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id INTEGER NOT NULL,
			ran_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			exit_code INTEGER NOT NULL DEFAULT 0,
			output TEXT NOT NULL DEFAULT '',
			duration_ms INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY(job_id) REFERENCES cron_jobs(id) ON DELETE CASCADE
		);
	`)
	return err
}

func scanCronJob(row interface{ Scan(...any) error }) (CronJob, error) {
	var j CronJob
	var enabled int
	var lastRunAt sql.NullTime
	var lastExitCode sql.NullInt64
	err := row.Scan(
		&j.ID, &j.Name, &j.Schedule, &j.Type,
		&j.Command, &j.ScriptContent,
		&j.CurlURL, &j.CurlMethod, &j.CurlHeaders, &j.CurlBody, &j.CurlExpectedStatus,
		&enabled, &j.CreatedAt, &lastRunAt, &lastExitCode, &j.LastOutput,
	)
	if err != nil {
		return j, err
	}
	j.Enabled = enabled == 1
	if lastRunAt.Valid {
		t := lastRunAt.Time
		j.LastRunAt = &t
	}
	if lastExitCode.Valid {
		code := int(lastExitCode.Int64)
		j.LastExitCode = &code
	}
	return j, nil
}

const cronJobSelect = `
	SELECT id, name, schedule, type, command, script_content,
	       curl_url, curl_method, curl_headers, curl_body, curl_expected_status,
	       enabled, created_at, last_run_at, last_exit_code, last_output
	FROM cron_jobs`

func GetCronJobs() ([]CronJob, error) {
	rows, err := DB.Query(cronJobSelect + " ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []CronJob
	for rows.Next() {
		j, err := scanCronJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func GetEnabledCronJobs() ([]CronJob, error) {
	rows, err := DB.Query(cronJobSelect + " WHERE enabled=1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []CronJob
	for rows.Next() {
		j, err := scanCronJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func GetCronJob(id int) (CronJob, error) {
	row := DB.QueryRow(cronJobSelect+" WHERE id=?", id)
	return scanCronJob(row)
}

func CreateCronJob(j CronJob) (int64, error) {
	enabled := 0
	if j.Enabled {
		enabled = 1
	}
	if j.CurlExpectedStatus == 0 {
		j.CurlExpectedStatus = 200
	}
	if j.CurlMethod == "" {
		j.CurlMethod = "GET"
	}
	if j.CurlHeaders == "" {
		j.CurlHeaders = "{}"
	}
	res, err := DB.Exec(`
		INSERT INTO cron_jobs (name, schedule, type, command, script_content,
			curl_url, curl_method, curl_headers, curl_body, curl_expected_status, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		j.Name, j.Schedule, j.Type, j.Command, j.ScriptContent,
		j.CurlURL, j.CurlMethod, j.CurlHeaders, j.CurlBody, j.CurlExpectedStatus, enabled,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateCronJob(j CronJob) error {
	if j.CurlExpectedStatus == 0 {
		j.CurlExpectedStatus = 200
	}
	if j.CurlMethod == "" {
		j.CurlMethod = "GET"
	}
	if j.CurlHeaders == "" {
		j.CurlHeaders = "{}"
	}
	_, err := DB.Exec(`
		UPDATE cron_jobs SET name=?, schedule=?, type=?, command=?, script_content=?,
			curl_url=?, curl_method=?, curl_headers=?, curl_body=?, curl_expected_status=?
		WHERE id=?`,
		j.Name, j.Schedule, j.Type, j.Command, j.ScriptContent,
		j.CurlURL, j.CurlMethod, j.CurlHeaders, j.CurlBody, j.CurlExpectedStatus, j.ID,
	)
	return err
}

func DeleteCronJob(id int) error {
	_, err := DB.Exec("DELETE FROM cron_jobs WHERE id=?", id)
	return err
}

func ToggleCronJob(id int, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	_, err := DB.Exec("UPDATE cron_jobs SET enabled=? WHERE id=?", v, id)
	return err
}

func UpdateCronJobLastRun(id, exitCode int, output string) error {
	_, err := DB.Exec(`
		UPDATE cron_jobs SET last_run_at=CURRENT_TIMESTAMP, last_exit_code=?, last_output=?
		WHERE id=?`, exitCode, output, id)
	return err
}

func LogCronRun(jobID, exitCode int, output string, durationMS int64) error {
	_, _ = DB.Exec(`
		DELETE FROM cron_runs WHERE job_id=? AND id NOT IN (
			SELECT id FROM cron_runs WHERE job_id=? ORDER BY id DESC LIMIT 49
		)`, jobID, jobID)
	_, err := DB.Exec(`
		INSERT INTO cron_runs (job_id, exit_code, output, duration_ms) VALUES (?, ?, ?, ?)`,
		jobID, exitCode, output, durationMS)
	return err
}

func GetCronRuns(jobID int) ([]CronRun, error) {
	rows, err := DB.Query(`
		SELECT id, job_id, ran_at, exit_code, output, duration_ms
		FROM cron_runs WHERE job_id=? ORDER BY ran_at DESC LIMIT 20`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []CronRun
	for rows.Next() {
		var r CronRun
		if err := rows.Scan(&r.ID, &r.JobID, &r.RanAt, &r.ExitCode, &r.Output, &r.DurationMS); err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, nil
}
