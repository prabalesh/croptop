package collector

import (
	"time"

	"github.com/prabalesh/croptop/internal/models"
)

type StatsCollector struct {
	lastUpdate   time.Time
	lastCPUTimes []uint64
	bootTime     time.Time
}

func NewStatsCollector() *StatsCollector {
	bootTime := getBootTime()
	return &StatsCollector{
		lastUpdate: time.Now(),
		bootTime:   bootTime,
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
