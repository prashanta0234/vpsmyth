package monitoring

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/prashanta0234/vpsmyth/internal/db"
)

const collectInterval = 30 * time.Second
const purgeInterval = 1 * time.Hour
const retentionDays = 30

func Start() {
	go run()
}

func run() {
	collectTick := time.NewTicker(collectInterval)
	purgeTick := time.NewTicker(purgeInterval)
	defer collectTick.Stop()
	defer purgeTick.Stop()

	collect() // collect immediately on startup

	for {
		select {
		case <-collectTick.C:
			collect()
		case <-purgeTick.C:
			if err := db.PurgeOldMetrics(retentionDays); err != nil {
				log.Printf("monitoring: purge error: %v", err)
			}
		}
	}
}

func collect() {
	now := time.Now().Unix()

	cpu, err := getCPUPercent()
	if err != nil {
		cpu = 0
	}
	ramUsed, ramTotal, err := getRAMMB()
	if err != nil {
		ramUsed, ramTotal = 0, 0
	}
	diskUsed, diskTotal, err := getDiskGB()
	if err != nil {
		diskUsed, diskTotal = 0, 0
	}

	if err := db.SaveSystemMetric(now, cpu, ramUsed, ramTotal, diskUsed, diskTotal); err != nil {
		log.Printf("monitoring: save system metric: %v", err)
	}

	collectContainers(now)
}

func collectContainers(ts int64) {
	out, err := exec.Command("docker", "stats", "--no-stream", "--format",
		"{{.ID}}|{{.Name}}|{{.CPUPerc}}|{{.MemUsage}}").Output()
	if err != nil {
		return // docker not running or no containers
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		id := parts[0]
		name := strings.TrimPrefix(parts[1], "/")
		cpu := parsePercent(parts[2])
		memUsed, memLimit := parseMemUsage(parts[3])

		if err := db.SaveContainerMetric(ts, id, name, cpu, memUsed, memLimit); err != nil {
			log.Printf("monitoring: save container %s: %v", name, err)
		}
	}
}

func getCPUPercent() (float64, error) {
	s1, err := readCPUStat()
	if err != nil {
		return 0, err
	}
	time.Sleep(200 * time.Millisecond)
	s2, err := readCPUStat()
	if err != nil {
		return 0, err
	}

	total1, total2 := cpuSum(s1), cpuSum(s2)
	idle1, idle2 := s1[3], s2[3]

	totalDelta := float64(total2 - total1)
	idleDelta := float64(idle2 - idle1)
	if totalDelta == 0 {
		return 0, nil
	}
	return (1 - idleDelta/totalDelta) * 100, nil
}

func readCPUStat() ([]int64, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)[1:]
		vals := make([]int64, len(fields))
		for i, field := range fields {
			vals[i], _ = strconv.ParseInt(field, 10, 64)
		}
		return vals, nil
	}
	return nil, fmt.Errorf("cpu line not found in /proc/stat")
}

func cpuSum(vals []int64) int64 {
	var total int64
	for _, v := range vals {
		total += v
	}
	return total
}

func getRAMMB() (int64, int64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	info := make(map[string]int64)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		val, _ := strconv.ParseInt(parts[1], 10, 64)
		info[key] = val
	}

	totalKB := info["MemTotal"]
	availKB := info["MemAvailable"]
	usedKB := totalKB - availKB
	return usedKB / 1024, totalKB / 1024, nil
}

func getDiskGB() (float64, float64, error) {
	out, err := exec.Command("df", "-k", "/").Output()
	if err != nil {
		return 0, 0, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return 0, 0, fmt.Errorf("unexpected df output")
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 3 {
		return 0, 0, fmt.Errorf("unexpected df fields")
	}
	totalKB, _ := strconv.ParseFloat(fields[1], 64)
	usedKB, _ := strconv.ParseFloat(fields[2], 64)
	return usedKB / (1024 * 1024), totalKB / (1024 * 1024), nil
}

func parsePercent(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "%")), 64)
	return v
}

func parseMemUsage(s string) (float64, float64) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0, 0
	}
	return parseMemValue(strings.TrimSpace(parts[0])), parseMemValue(strings.TrimSpace(parts[1]))
}

func parseMemValue(s string) float64 {
	units := []struct {
		suffix string
		factor float64
	}{
		{"GiB", 1024}, {"MiB", 1}, {"KiB", 1.0 / 1024},
		{"GB", 1000}, {"MB", 1}, {"kB", 1.0 / 1024}, {"B", 1.0 / (1024 * 1024)},
	}
	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			v, _ := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(s, u.suffix)), 64)
			return v * u.factor
		}
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
