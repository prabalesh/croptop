package collector

import (
	"os"
	"strconv"
	"strings"

	"github.com/prabalesh/croptop/internal/models"
)

func (s *StatsCollector) getCPUStats() models.CPUStats {
	// Read CPU info
	model, frequency := s.getCPUInfo()
	temp := s.getCPUTemperature()

	// Read CPU usage from /proc/stat
	usage, cores := s.getCPUUsage()

	return models.CPUStats{
		Usage:     usage,
		Cores:     cores,
		Frequency: frequency,
		Temp:      temp,
		Model:     model,
	}
}

func (s *StatsCollector) getCPUInfo() (string, float64) {
	content, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "Unknown CPU", 0
	}

	modelName := "Unknown CPU"
	freq := 0.0
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(line, "model name") {
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				modelName = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "cpu MHz") {
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				if f, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					freq = f
				}
			}
		}
		if modelName != "Unknown CPU" && freq != 0.0 {
			break
		}
	}

	return modelName, freq
}

func (s *StatsCollector) getCPUTemperature() float32 {
	// Try different temperature sensor paths
	tempPaths := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/hwmon/hwmon0/temp1_input",
		"/sys/class/hwmon/hwmon1/temp1_input",
	}

	for _, path := range tempPaths {
		if content, err := os.ReadFile(path); err == nil {
			if temp, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64); err == nil {
				temp32 := float32(temp)
				// Temperature is usually in millidegrees
				if temp32 > 1000 {
					return temp32 / 1000.0
				}
				return temp32
			}
		}
	}
	return 0
}

func (s *StatsCollector) getCPUUsage() (float64, []float64) {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, nil
	}

	lines := strings.Split(string(content), "\n")
	var overallUsage float64
	var coreUsages []float64

	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			// Overall CPU usage
			overallUsage = s.parseCPULine(line)
		} else if strings.HasPrefix(line, "cpu") && len(line) > 3 {
			// Individual core usage
			usage := s.parseCPULine(line)
			coreUsages = append(coreUsages, usage)
		}
	}

	return overallUsage, coreUsages
}

func (s *StatsCollector) parseCPULine(line string) float64 {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return 0
	}

	// Parse CPU times: user, nice, system, idle, iowait, irq, softirq
	var times []uint64
	for i := 1; i < len(fields) && i < 8; i++ {
		if val, err := strconv.ParseUint(fields[i], 10, 64); err == nil {
			times = append(times, val)
		}
	}

	if len(times) < 4 {
		return 0
	}

	// Calculate total and idle time
	var total, idle uint64
	for i, time := range times {
		total += time
		if i == 3 { // idle time is at index 3
			idle = time
		}
	}

	if total == 0 {
		return 0
	}

	// Calculate usage percentage
	usage := float64(total-idle) / float64(total) * 100
	return usage
}
