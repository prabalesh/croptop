package collector

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/prabalesh/croptop/internal/models"
)

const (
	ProcMemInfoPath = "/proc/meminfo"
)

func (s *StatsCollector) getMemoryStats() models.MemoryStats {
	// handle the error here
	file, _ := os.Open(ProcMemInfoPath)
	defer file.Close()

	// fields we need to collect
	var memTotal, memFree, memAvailable, swapTotal, swapFree float64
	var foundFields uint8
	var requiredFields uint8 = 3

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		valueStr := fields[1]

		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue // Skip malformed lines
		}

		// Convert from KB to bytes
		//
		// fmt.Printf("%s -> %f\n", key, value)

		switch key {
		case "MemTotal":
			memTotal = value
			foundFields++
		case "MemFree":
			memFree = value
			foundFields++
		case "MemAvailable":
			memAvailable = value
			foundFields++
		case "SwapTotal":
			swapTotal = value
		case "SwapFree":
			swapFree = value
		}

		// Early exit if we have all required fields
		if (foundFields >= requiredFields) && swapTotal > 0 && swapFree > 0 {
			break
		}

		// TODO: add proper error handling
		if err := scanner.Err(); err != nil {
			return models.MemoryStats{}
		}

	}

	// TODO: add proper error handling
	if memTotal == 0 {
		return models.MemoryStats{}
	}
	if foundFields < 2 {
		return models.MemoryStats{}
	}

	// Handle case where MemAvailable doesn't exist (older kernels)
	// Fallback: approximate as MemFree + Buffers + Cached
	if memAvailable == 0 {
		memAvailable = memFree
		// Note: This is a simplified fallback. For older kernels, you might want to
		// re-scan for Buffers and Cached values for a more accurate calculation
	}

	// Calculate derived values
	memUsed := memTotal - memAvailable
	var usagePercent float64
	if memTotal > 0 {
		usagePercent = (memUsed / memTotal) * 100
	}

	swapUsed := swapTotal - swapFree

	// fmt.Println(models.MemoryStats{
	// 	Total:        memTotal,
	// 	Used:         memUsed,
	// 	Free:         memFree,
	// 	Available:    memAvailable,
	// 	UsagePercent: usagePercent,
	// 	SwapTotal:    swapTotal,
	// 	SwapUsed:     swapUsed,
	// })

	return models.MemoryStats{
		Total:        memTotal,
		Used:         memUsed,
		Free:         memFree,
		Available:    memAvailable,
		UsagePercent: usagePercent,
		SwapTotal:    swapTotal,
		SwapUsed:     swapUsed,
	}
}
