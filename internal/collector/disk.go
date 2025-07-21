package collector

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/prabalesh/croptop/internal/models"
)

func (s *StatsCollector) getDiskStats() []models.DiskStats {
	content, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	var diskStats []models.DiskStats

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		device := fields[0]
		mountpoint := fields[1]
		filesystem := fields[2]

		// Skip special filesystems
		if strings.HasPrefix(device, "/dev") &&
			!strings.Contains(device, "loop") &&
			filesystem != "tmpfs" {

			var stat syscall.Statfs_t
			if err := syscall.Statfs(mountpoint, &stat); err == nil {
				total := uint64(stat.Blocks) * uint64(stat.Bsize)
				free := uint64(stat.Bavail) * uint64(stat.Bsize)
				used := total - free

				var usagePercent float64
				if total > 0 {
					usagePercent = float64(used) / float64(total) * 100
				}

				// Get disk I/O stats
				readBytes, writeBytes, readOps, writeOps := s.getDiskIO(device)

				diskStats = append(diskStats, models.DiskStats{
					Device:       device,
					Mountpoint:   mountpoint,
					Total:        total,
					Used:         used,
					Free:         free,
					UsagePercent: usagePercent,
					Filesystem:   filesystem,
					ReadBytes:    readBytes,
					WriteBytes:   writeBytes,
					ReadOps:      readOps,
					WriteOps:     writeOps,
				})
			}
		}
	}

	return diskStats
}

func (s *StatsCollector) getDiskIO(device string) (uint64, uint64, uint64, uint64) {
	// Extract device name (e.g., sda1 -> sda)
	deviceName := filepath.Base(device)
	if len(deviceName) > 3 {
		deviceName = deviceName[:3] // Get base device name
	}

	content, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return 0, 0, 0, 0
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}

		if fields[2] == deviceName {
			readOps, _ := strconv.ParseUint(fields[3], 10, 64)
			readBytes, _ := strconv.ParseUint(fields[5], 10, 64)
			writeOps, _ := strconv.ParseUint(fields[7], 10, 64)
			writeBytes, _ := strconv.ParseUint(fields[9], 10, 64)

			// Convert sectors to bytes (assuming 512 bytes per sector)
			readBytes *= 512
			writeBytes *= 512

			return readBytes, writeBytes, readOps, writeOps
		}
	}

	return 0, 0, 0, 0
}
