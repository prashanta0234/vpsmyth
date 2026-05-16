package cron

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/prashanta0234/vpsmyth/internal/db"
)

// Start aligns to the next minute boundary then fires every minute.
func Start() {
	go func() {
		now := time.Now()
		next := now.Truncate(time.Minute).Add(time.Minute)
		time.Sleep(time.Until(next))

		tick := time.NewTicker(time.Minute)
		defer tick.Stop()
		checkAndRun()
		for range tick.C {
			checkAndRun()
		}
	}()
}

func checkAndRun() {
	jobs, err := db.GetEnabledCronJobs()
	if err != nil {
		return
	}
	t := time.Now()
	for _, job := range jobs {
		if matchSchedule(job.Schedule, t) {
			go execute(job)
		}
	}
}

// RunNow triggers a job immediately in a goroutine.
func RunNow(jobID int) error {
	job, err := db.GetCronJob(jobID)
	if err != nil {
		return err
	}
	go execute(job)
	return nil
}

func execute(job db.CronJob) {
	start := time.Now()
	var output string
	var exitCode int

	switch job.Type {
	case "command":
		output, exitCode = runCommand(job.Command)
	case "script":
		output, exitCode = runScript(job.ScriptContent)
	case "curl":
		output, exitCode = runCurl(job)
	default:
		output = "unknown job type: " + job.Type
		exitCode = 1
	}

	if len(output) > 10240 {
		output = output[:10240] + "\n...[truncated]"
	}

	durationMS := time.Since(start).Milliseconds()
	_ = db.UpdateCronJobLastRun(job.ID, exitCode, output)
	_ = db.LogCronRun(job.ID, exitCode, output, durationMS)
}

func runCommand(cmd string) (string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sh", "-c", cmd).CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			return output, e.ExitCode()
		}
		if output == "" {
			output = err.Error()
		}
		return output, 1
	}
	return output, 0
}

func runScript(content string) (string, int) {
	f, err := os.CreateTemp("", "vpsmyth-cron-*.sh")
	if err != nil {
		return "failed to create temp script: " + err.Error(), 1
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return "failed to write script: " + err.Error(), 1
	}
	f.Close()
	if err := os.Chmod(f.Name(), 0700); err != nil {
		return "failed to chmod script: " + err.Error(), 1
	}
	return runCommand("bash " + f.Name())
}

func runCurl(job db.CronJob) (string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	method := job.CurlMethod
	if method == "" {
		method = "GET"
	}

	var bodyReader io.Reader
	if job.CurlBody != "" {
		bodyReader = strings.NewReader(job.CurlBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, job.CurlURL, bodyReader)
	if err != nil {
		return "failed to build request: " + err.Error(), 1
	}

	if job.CurlHeaders != "" && job.CurlHeaders != "{}" {
		var headers map[string]string
		if json.Unmarshal([]byte(job.CurlHeaders), &headers) == nil {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "request failed: " + err.Error(), 1
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10240))
	output := fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, strings.TrimSpace(string(body)))

	expected := job.CurlExpectedStatus
	if expected == 0 {
		expected = 200
	}
	if resp.StatusCode != expected {
		return output, 1
	}
	return output, 0
}

// ── Cron expression parser ────────────────────────────────────────────────────
// Supports: * */n n a-b a,b,c and combinations thereof. Standard 5-field format.

func matchSchedule(expr string, t time.Time) bool {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return false
	}
	return matchField(fields[0], t.Minute(), 0, 59) &&
		matchField(fields[1], t.Hour(), 0, 23) &&
		matchField(fields[2], t.Day(), 1, 31) &&
		matchField(fields[3], int(t.Month()), 1, 12) &&
		matchField(fields[4], int(t.Weekday()), 0, 6)
}

func matchField(field string, value, min, max int) bool {
	for _, part := range strings.Split(field, ",") {
		if matchPart(part, value, min, max) {
			return true
		}
	}
	return false
}

func matchPart(part string, value, min, max int) bool {
	if strings.Contains(part, "/") {
		p := strings.SplitN(part, "/", 2)
		step, err := strconv.Atoi(p[1])
		if err != nil || step <= 0 {
			return false
		}
		lo, hi := min, max
		if p[0] != "*" {
			if strings.Contains(p[0], "-") {
				r := strings.SplitN(p[0], "-", 2)
				lo, _ = strconv.Atoi(r[0])
				hi, _ = strconv.Atoi(r[1])
			} else {
				lo, _ = strconv.Atoi(p[0])
			}
		}
		return value >= lo && value <= hi && (value-lo)%step == 0
	}
	if strings.Contains(part, "-") {
		r := strings.SplitN(part, "-", 2)
		lo, _ := strconv.Atoi(r[0])
		hi, _ := strconv.Atoi(r[1])
		return value >= lo && value <= hi
	}
	if part == "*" {
		return true
	}
	n, err := strconv.Atoi(part)
	return err == nil && n == value
}

// ValidateSchedule returns an error message or "" if valid.
func ValidateSchedule(expr string) string {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return "must have exactly 5 fields: minute hour day month weekday"
	}
	limits := [][2]int{{0, 59}, {0, 23}, {1, 31}, {1, 12}, {0, 6}}
	names := []string{"minute", "hour", "day", "month", "weekday"}
	for i, f := range fields {
		if !isValidField(f, limits[i][0], limits[i][1]) {
			return "invalid " + names[i] + " field: " + f
		}
	}
	return ""
}

func isValidField(field string, min, max int) bool {
	for _, part := range strings.Split(field, ",") {
		if !isValidPart(part, min, max) {
			return false
		}
	}
	return true
}

func isValidPart(part string, min, max int) bool {
	if part == "*" {
		return true
	}
	if strings.Contains(part, "/") {
		p := strings.SplitN(part, "/", 2)
		step, err := strconv.Atoi(p[1])
		if err != nil || step <= 0 {
			return false
		}
		if p[0] == "*" {
			return true
		}
		return isValidPart(p[0], min, max)
	}
	if strings.Contains(part, "-") {
		r := strings.SplitN(part, "-", 2)
		lo, err1 := strconv.Atoi(r[0])
		hi, err2 := strconv.Atoi(r[1])
		return err1 == nil && err2 == nil && lo >= min && hi <= max && lo <= hi
	}
	n, err := strconv.Atoi(part)
	return err == nil && n >= min && n <= max
}
