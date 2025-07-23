package collector

import (
	"sync"
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
	var (
		wg      sync.WaitGroup
		cpu     models.CPUStats
		mem     models.MemoryStats
		net     models.NetworkStats
		disk    []models.DiskStats
		battery models.BatteryStats
	)

	wg.Add(5)

	go func() {
		defer wg.Done()
		cpu = s.getCPUStats()
	}()

	go func() {
		defer wg.Done()
		mem = s.getMemoryStats()
	}()

	go func() {
		defer wg.Done()
		net = s.getNetworkStats()
	}()

	go func() {
		defer wg.Done()
		disk = s.getDiskStats()
	}()

	go func() {
		defer wg.Done()
		battery = s.getBatteryStats()
	}()

	wg.Wait()

	return models.SystemStats{
		CPU:     cpu,
		Memory:  mem,
		Network: net,
		Disk:    disk,
		Battery: battery,
		Uptime:  time.Since(s.bootTime),
	}
}

func (s *StatsCollector) ClearCPUCache() {
	s.cpuCache.Clear()
}
