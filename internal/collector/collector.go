package collector

import (
	"time"

	"github.com/prabalesh/croptop/internal/models"
)

type StatsCollector struct {
	lastUpdate   time.Time
	lastCPUTimes []uint64
	bootTime     time.Time
	cpuCache     *CPUCache
}

func NewStatsCollector() *StatsCollector {
	bootTime := getBootTime()
	return &StatsCollector{
		lastUpdate: time.Now(),
		bootTime:   bootTime,
		cpuCache:   NewCPUCache(),
	}
}

func (s *StatsCollector) GetSystemStats() models.SystemStats {
	return models.SystemStats{
		CPU:     s.getCPUStats(),
		Memory:  s.getMemoryStats(),
		Network: s.getNetworkStats(),
		Disk:    s.getDiskStats(),
		Battery: s.getBatteryStats(),
		Uptime:  time.Since(s.bootTime),
	}
}

func (s *StatsCollector) ClearCPUCache() {
	s.cpuCache.Clear()
}
