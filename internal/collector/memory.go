package collector

import (
	"os"
	"strconv"
	"strings"

	"github.com/prabalesh/croptop/internal/models"
)

func (s *StatsCollector) getMemoryStats() models.MemoryStats {
	content, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return models.MemoryStats{}
	}

	memInfo := make(map[string]float64)
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			key := strings.TrimSuffix(fields[0], ":")
			value, err := strconv.ParseFloat(fields[1], 64)
			if err == nil {
				memInfo[key] = value * 1024 // Convert from KB to bytes
			}
		}
	}

	total := memInfo["MemTotal"]
	free := memInfo["MemFree"]
	available := memInfo["MemAvailable"]
	// buffers := memInfo["Buffers"]
	// cached := memInfo["Cached"]
	used := total - available

	var usagePercent float64
	if total > 0 {
		usagePercent = float64(used) / float64(total) * 100
	}

	return models.MemoryStats{
		Total:        total,
		Used:         used,
		Free:         free,
		Available:    available,
		UsagePercent: usagePercent,
		SwapTotal:    memInfo["SwapTotal"],
		SwapUsed:     memInfo["SwapTotal"] - memInfo["SwapFree"],
	}
}
