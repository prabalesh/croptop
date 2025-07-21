package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/prabalesh/croptop/internal/models"
)

func (s *StatsCollector) getBatteryStats() models.BatteryStats {
	// Find battery directory
	batteryDirs, err := filepath.Glob("/sys/class/power_supply/BAT*")
	if err != nil || len(batteryDirs) == 0 {
		// No battery found (desktop system)
		return models.BatteryStats{
			Level:      100,
			Status:     "Not Available",
			TimeLeft:   "N/A",
			IsCharging: false,
			Health:     100,
		}
	}

	batteryDir := batteryDirs[0]

	// Read battery information
	level := s.readBatteryInt(batteryDir + "/capacity")
	status := s.readBatteryString(batteryDir + "/status")
	isCharging := status == "Charging"

	// Calculate time left (simplified estimation)
	timeLeft := "N/A"
	if !isCharging && level > 0 {
		hours := level / 10 // Rough estimation
		mins := (level % 10) * 6
		timeLeft = fmt.Sprintf("%dh %dm", hours, mins)
	}

	health := s.getBatteryHealth(batteryDir)

	return models.BatteryStats{
		Level:      level,
		Status:     status,
		TimeLeft:   timeLeft,
		IsCharging: isCharging,
		Health:     health,
	}
}

func (s *StatsCollector) readBatteryInt(path string) int {
	if content, err := os.ReadFile(path); err == nil {
		if val, err := strconv.Atoi(strings.TrimSpace(string(content))); err == nil {
			return val
		}
	}
	return 0
}

func (s *StatsCollector) readBatteryString(path string) string {
	if content, err := os.ReadFile(path); err == nil {
		return strings.TrimSpace(string(content))
	}
	return "Unknown"
}

func (s *StatsCollector) getBatteryHealth(batteryDir string) int {
	energyFull := s.readBatteryInt(batteryDir + "/energy_full")
	energyFullDesign := s.readBatteryInt(batteryDir + "/energy_full_design")

	if energyFullDesign > 0 && energyFull > 0 {
		health := (energyFull * 100) / energyFullDesign
		if health > 100 {
			health = 100
		}
		return health
	}
	return 100 // Default to 100% if can't determine
}
