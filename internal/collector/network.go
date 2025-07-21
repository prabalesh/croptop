package collector

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/prabalesh/croptop/internal/models"
)

func (s *StatsCollector) getNetworkStats() models.NetworkStats {
	content, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return models.NetworkStats{}
	}

	lines := strings.Split(string(content), "\n")
	var interfaces []models.NetworkInterface
	var totalRx, totalTx uint64

	for i, line := range lines {
		if i < 2 { // Skip header lines
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse interface line
		parts := strings.Fields(line)
		if len(parts) < 17 {
			continue
		}

		name := strings.TrimSuffix(parts[0], ":")
		if name == "lo" { // Skip loopback
			continue
		}

		rxBytes, _ := strconv.ParseUint(parts[1], 10, 64)
		rxPackets, _ := strconv.ParseUint(parts[2], 10, 64)
		txBytes, _ := strconv.ParseUint(parts[9], 10, 64)
		txPackets, _ := strconv.ParseUint(parts[10], 10, 64)

		status := s.getInterfaceStatus(name)
		speed := s.getInterfaceSpeed(name)

		interfaces = append(interfaces, models.NetworkInterface{
			Name:      name,
			RxBytes:   rxBytes,
			TxBytes:   txBytes,
			RxPackets: rxPackets,
			TxPackets: txPackets,
			Status:    status,
			Speed:     speed,
		})

		totalRx += rxBytes
		totalTx += txBytes
	}

	return models.NetworkStats{
		Interfaces: interfaces,
		TotalRx:    totalRx,
		TotalTx:    totalTx,
	}
}

func (s *StatsCollector) getInterfaceStatus(name string) string {
	operstatePath := fmt.Sprintf("/sys/class/net/%s/operstate", name)
	if content, err := os.ReadFile(operstatePath); err == nil {
		return strings.TrimSpace(string(content))
	}
	return "unknown"
}

func (s *StatsCollector) getInterfaceSpeed(name string) string {
	speedPath := fmt.Sprintf("/sys/class/net/%s/speed", name)
	if content, err := os.ReadFile(speedPath); err == nil {
		if speed, err := strconv.Atoi(strings.TrimSpace(string(content))); err == nil {
			return fmt.Sprintf("%d Mb/s", speed)
		}
	}
	return "unknown"
}
