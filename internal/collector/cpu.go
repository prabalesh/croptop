package collector

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/prabalesh/croptop/internal/models"
)

// CPUError provides structured error information
type CPUError struct {
	Operation string
	Path      string
	Err       error
}

func (e *CPUError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("CPU %s failed for %s: %v", e.Operation, e.Path, e.Err)
	}
	return fmt.Sprintf("CPU %s failed: %v", e.Operation, e.Err)
}

func (s *StatsCollector) getCPUStats() models.CPUStats {
	// Use context with timeout for reliability
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Get cached or fresh data efficiently
	model, frequency := s.getCPUCachedInfo(ctx)
	temp := s.getCachedTemperature(ctx)
	usage, cores := s.getCachedCPUUsage(ctx)

	return models.CPUStats{
		Usage:     usage,
		Cores:     cores,
		Frequency: frequency,
		Temp:      temp,
		Model:     model,
	}
}

func (s *StatsCollector) getCPUCachedInfo(ctx context.Context) (string, float64) {
	// Check model cache first
	if s.cpuCache.IsModelCacheValid() {
		model, freq := s.cpuCache.GetCachedModel()

		// Model is cached, check frequency
		if s.cpuCache.IsFrequencyCacheValid() {
			return model, freq
		}

		// Model cached but frequency needs refresh
		select {
		case <-ctx.Done():
			return model, freq // Return cached frequency on timeout
		default:
			if freshModel, freshFreq, err := s.getCPUInfo(ctx); err == nil {
				s.cpuCache.SetCachedFrequency(freshFreq)
				return freshModel, freshFreq
			}
			return model, freq // Fallback to cached on error
		}
	}

	// Nothing cached, get fresh data
	select {
	case <-ctx.Done():
		return "Unknown CPU", 0
	default:
		model, freq, err := s.getCPUInfo(ctx)
		if err != nil {
			// Log error but don't fail completely
			return "Unknown CPU", 0
		}

		s.cpuCache.SetCachedModel(model)
		s.cpuCache.SetCachedFrequency(freq)
		return model, freq
	}
}

func (s *StatsCollector) getCachedTemperature(ctx context.Context) float32 {
	if s.cpuCache.IsTemperatureCacheValid() {
		return s.cpuCache.GetCachedTemperature()
	}

	select {
	case <-ctx.Done():
		return 0
	default:
		temp, err := s.getCPUTemperature(ctx)
		if err != nil {
			return 0 // Silent failure for temperature - not critical
		}

		s.cpuCache.SetCachedTemperature(temp)
		return temp
	}
}

func (s *StatsCollector) getCPUInfo(ctx context.Context) (string, float64, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "Unknown CPU", 0, &CPUError{"read_cpuinfo", "/proc/cpuinfo", err}
	}
	defer file.Close()

	// Use buffered scanner with reasonable buffer size
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 32*1024), 32*1024) // 32KB buffer

	modelName := "Unknown CPU"
	freq := 0.0
	foundModel := false
	foundFreq := false

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return modelName, freq, ctx.Err()
		default:
		}

		// Early termination when both values found
		if foundModel && foundFreq {
			break
		}

		line := scanner.Text()

		if !foundModel && strings.HasPrefix(line, "model name") {
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				modelName = strings.TrimSpace(parts[1])
				foundModel = true
			}
		} else if !foundFreq && strings.HasPrefix(line, "cpu MHz") {
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				if f, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					freq = f
					foundFreq = true
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return modelName, freq, &CPUError{"scan_cpuinfo", "/proc/cpuinfo", err}
	}

	return modelName, freq, nil
}

func (s *StatsCollector) getCPUTemperature(ctx context.Context) (float32, error) {
	// Common temperature sensor paths with priority order
	tempPaths := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/hwmon/hwmon0/temp1_input",
		"/sys/class/hwmon/hwmon1/temp1_input",
		"/sys/class/hwmon/hwmon2/temp1_input",
	}

	// Try glob patterns for more comprehensive detection
	globPatterns := []string{
		"/sys/class/hwmon/hwmon*/temp*_input",
		"/sys/devices/platform/coretemp.*/hwmon/hwmon*/temp*_input",
	}

	// Try direct paths first (faster)
	for _, path := range tempPaths {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		if temp, err := s.readTemperatureFromPath(path); err == nil {
			return temp, nil
		}
	}

	// Try glob patterns as fallback
	for _, pattern := range globPatterns {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		paths, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, path := range paths {
			if temp, err := s.readTemperatureFromPath(path); err == nil {
				return temp, nil
			}
		}
	}

	return 0, &CPUError{"read_temperature", "all_sensors", fmt.Errorf("no valid temperature sensors found")}
}

func (s *StatsCollector) readTemperatureFromPath(path string) (float32, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	tempStr := strings.TrimSpace(string(content))
	temp, err := strconv.ParseFloat(tempStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid temperature value: %s", tempStr)
	}

	temp32 := float32(temp)

	// Handle different temperature scales
	// Most sensors report in millidegrees Celsius
	if temp32 > 1000 {
		return temp32 / 1000.0, nil
	}

	// Validate reasonable temperature range (0-150°C)
	if temp32 < 0 || temp32 > 150 {
		return 0, fmt.Errorf("temperature out of range: %.1f", temp32)
	}

	return temp32, nil
}

func (s *StatsCollector) getCachedCPUUsage(ctx context.Context) (float64, []float64) {
	// Check if usage is cached and valid
	if s.cpuCache.IsUsageCacheValid() {
		return s.cpuCache.GetCachedUsage()
	}

	select {
	case <-ctx.Done():
		return 0, nil
	default:
	}

	// Get current CPU stats
	currentStats, err := s.getCurrentCPUStats()
	if err != nil {
		return 0, nil
	}

	// If we don't have previous stats, store current and return zero
	if !s.cpuCache.HasPreviousStats() {
		s.cpuCache.SetPreviousStats(currentStats)
		coreCount := max(len(currentStats)-1, 0) // -1 because "cpu" is overall
		return 0, make([]float64, coreCount)
	}

	// Get previous stats for comparison
	previousStats, _ := s.cpuCache.GetPreviousStats()

	// Check if enough time has passed for accurate calculation
	timeDiff := s.cpuCache.GetTimeSinceLastUsageUpdate().Seconds()
	if timeDiff < 0.1 { // Too small time difference
		return s.cpuCache.GetCachedUsage()
	}

	// Calculate overall usage
	overallUsage := s.calculateUsageWithValidation(previousStats["cpu"], currentStats["cpu"])

	// Calculate per-core usage efficiently
	coreUsages := make([]float64, 0, runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		cpuKey := fmt.Sprintf("cpu%d", i)
		if current, exists := currentStats[cpuKey]; exists {
			if previous, exists := previousStats[cpuKey]; exists {
				usage := s.calculateUsageWithValidation(previous, current)
				coreUsages = append(coreUsages, usage)
			}
		}
	}

	// Update cache with new data
	s.cpuCache.SetPreviousStats(currentStats)
	s.cpuCache.SetCachedUsage(overallUsage, coreUsages)

	return overallUsage, coreUsages
}

func (s *StatsCollector) getCurrentCPUStats() (map[string]CPUTimes, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, &CPUError{"read_proc_stat", "/proc/stat", err}
	}
	defer file.Close()

	// Pre-allocate map with expected capacity
	stats := make(map[string]CPUTimes, runtime.NumCPU()+1)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		times, err := s.parseCPUTimes(fields[1:])
		if err != nil {
			continue // Skip invalid lines
		}

		if times.Total > 0 {
			stats[fields[0]] = times
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, &CPUError{"scan_proc_stat", "/proc/stat", err}
	}

	return stats, nil
}

func (s *StatsCollector) parseCPUTimes(fields []string) (CPUTimes, error) {
	if len(fields) < 4 {
		return CPUTimes{}, fmt.Errorf("insufficient CPU time fields: %d", len(fields))
	}

	// Pre-allocate with known maximum size
	times := make([]uint64, 0, min(len(fields), 10))

	for i := 0; i < len(fields) && i < 10; i++ {
		val, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return CPUTimes{}, fmt.Errorf("invalid CPU time value at index %d: %s", i, fields[i])
		}
		times = append(times, val)
	}

	var total, idle uint64
	for i, time := range times {
		total += time
		// idle (index 3) and iowait (index 4) are considered idle time
		if i == 3 || i == 4 {
			idle += time
		}
	}

	return CPUTimes{
		Total: total,
		Idle:  idle,
	}, nil
}

func (s *StatsCollector) calculateUsageWithValidation(previous, current CPUTimes) float64 {
	// Validate input data
	if current.Total <= previous.Total {
		return 0 // Avoid negative or zero division
	}

	totalDiff := current.Total - previous.Total
	idleDiff := current.Idle - previous.Idle

	// Additional validation
	if totalDiff == 0 || idleDiff > totalDiff {
		return 0
	}

	usage := 100.0 * float64(totalDiff-idleDiff) / float64(totalDiff)

	// Clamp to valid range [0, 100]
	switch {
	case usage < 0:
		return 0
	case usage > 100:
		return 100
	default:
		return usage
	}
}

// Helper function for older Go versions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
