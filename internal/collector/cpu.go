package collector

import (
	"os"
	"strconv"
	"strings"

	"github.com/prabalesh/croptop/internal/models"
)

func (s *StatsCollector) getCPUStats() models.CPUStats {
	// Read CPU info
	model, frequency := s.getCPUCachedInfo()
	temp := s.getCachedTemperature()

	// Read CPU usage from /proc/stat
	usage, cores := s.getCachedCPUUsage()

	return models.CPUStats{
		Usage:     usage,
		Cores:     cores,
		Frequency: frequency,
		Temp:      temp,
		Model:     model,
	}
}

func (s *StatsCollector) getCPUCachedInfo() (string, float64) {
	if s.cpuCache.IsModelCacheValid() {
		model, freq := s.cpuCache.GetCachedModel()

		if s.cpuCache.IsFrequencyCacheValid() {
			return model, freq
		}

		_, freshFreq := s.getCPUInfo()
		s.cpuCache.SetCachedFrequency(freshFreq)
		return model, freshFreq
	}

	model, freq := s.getCPUInfo()
	s.cpuCache.SetCachedModel(model)
	s.cpuCache.SetCachedFrequency(freq)

	return model, freq
}

func (s *StatsCollector) getCachedTemperature() float32 {
	if s.cpuCache.IsTemperatureCacheValid() {
		return s.cpuCache.GetCachedTemperature()
	}
	temperature := s.getCPUTemperature()
	s.cpuCache.SetCachedTemperature(temperature)
	return temperature
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

func (s *StatsCollector) getCachedCPUUsage() (float64, []float64) {
	// Check if usage is cached and valid
	if s.cpuCache.IsUsageCacheValid() {
		return s.cpuCache.GetCachedUsage()
	}

	// Get current CPU stats
	currentStats := s.getCurrentCPUStats()

	// If we don't have previous stats, store current and return zero
	if !s.cpuCache.HasPreviousStats() {
		s.cpuCache.SetPreviousStats(currentStats)
		coreCount := len(currentStats) - 1 // -1 because "cpu" is overall

		// if the core count comes to be negative it returns 0
		coreCount = max(coreCount, 0)

		return 0, make([]float64, coreCount)
	}

	// Get previous stats for comparison
	previousStats, _ := s.cpuCache.GetPreviousStats()

	// Check if enough time has passed for accurate calculation
	timeDiff := s.cpuCache.GetTimeSinceLastUsageUpdate().Seconds()
	if timeDiff < 0.1 { // Too small time difference
		return s.cpuCache.GetCachedUsage()
	}

	// Calculate usage based on difference
	overallUsage := s.calculateUsage(previousStats["cpu"], currentStats["cpu"])

	var coreUsages []float64
	for key, current := range currentStats {
		if strings.HasPrefix(key, "cpu") && key != "cpu" {
			previous, exists := previousStats[key]
			if exists {
				usage := s.calculateUsage(previous, current)
				coreUsages = append(coreUsages, usage)
			}
		}
	}

	// Update cache with new data
	s.cpuCache.SetPreviousStats(currentStats)
	s.cpuCache.SetCachedUsage(overallUsage, coreUsages)

	return overallUsage, coreUsages
}

func (s *StatsCollector) getCurrentCPUStats() map[string]CPUTimes {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return nil
	}

	stats := make(map[string]CPUTimes)
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "cpu") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}

			times := s.parseCPUTimes(fields[1:])
			if times.Total > 0 {
				// field[0] => cpu name
				stats[fields[0]] = times
			}
		}
	}

	return stats
}

func (s *StatsCollector) parseCPUTimes(fields []string) CPUTimes {
	var times []uint64
	for i := 0; i < len(fields) && i < 7; i++ {
		if val, err := strconv.ParseUint(fields[i], 10, 64); err == nil {
			times = append(times, val)
		}
	}

	if len(times) < 4 {
		return CPUTimes{}
	}

	var total, idle uint64
	for i, time := range times {
		total += time
		if i == 3 || i == 4 { // idle time is at index 3
			idle += time
		}
	}

	return CPUTimes{
		Total: total,
		Idle:  idle,
	}
}

func (s *StatsCollector) calculateUsage(previous, current CPUTimes) float64 {
	totalDiff := current.Total - previous.Total
	idleDiff := current.Idle - previous.Idle

	if totalDiff == 0 {
		return 0
	}

	usage := 100 * float64(totalDiff-idleDiff) / float64(totalDiff)
	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}

	return usage
}
