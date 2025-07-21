package collector

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prabalesh/croptop/internal/models"
)

func (s *StatsCollector) GetProcessList() models.ProcessList {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return models.ProcessList{}
	}

	var processes []models.Process
	var total, running, sleeping, zombie int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if directory name is a PID (numeric)
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		proc := s.getProcessInfo(pid)
		if proc.PID != 0 {
			processes = append(processes, proc)

			// Count process states
			switch proc.Status {
			case "R":
				running++
			case "S", "D":
				sleeping++
			case "Z":
				zombie++
			}
		}
	}

	total = len(processes)

	return models.ProcessList{
		Processes: processes,
		Total:     total,
		Running:   running,
		Sleeping:  sleeping,
		Zombie:    zombie,
	}
}

func (s *StatsCollector) getProcessInfo(pid int) models.Process {
	// Read /proc/[pid]/stat for basic info
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	statContent, err := os.ReadFile(statPath)
	if err != nil {
		return models.Process{}
	}

	statFields := strings.Fields(string(statContent))
	if len(statFields) < 24 {
		return models.Process{}
	}

	// Read /proc/[pid]/status for additional info
	statusPath := fmt.Sprintf("/proc/%d/status", pid)
	statusContent, err := os.ReadFile(statusPath)
	if err != nil {
		return models.Process{}
	}

	// Parse process information
	name := s.getProcessName(statusContent)
	status := statFields[2]
	user := s.getProcessUser(pid)
	command := s.getProcessCommand(pid)
	cpuPercent := s.getProcessCPUPercent(statFields)
	memPercent, memRSS := s.getProcessMemory(statusContent)
	runtime := s.getProcessRuntime(statFields)
	priority := s.getProcessPriority(statFields)

	return models.Process{
		PID:        pid,
		Name:       name,
		Command:    command,
		CPUPercent: cpuPercent,
		MemPercent: memPercent,
		MemRSS:     memRSS,
		Status:     status,
		User:       user,
		Runtime:    runtime,
		Priority:   priority,
	}
}

func (s *StatsCollector) getProcessName(statusContent []byte) string {
	lines := strings.Split(string(statusContent), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Name:") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				return fields[1]
			}
		}
	}
	return "unknown"
}

func (s *StatsCollector) getProcessUser(pid int) string {
	statusPath := fmt.Sprintf("/proc/%d/status", pid)
	content, err := os.ReadFile(statusPath)
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				// This would need proper UID to username conversion
				// For simplicity, just return the UID
				return fields[1]
			}
		}
	}
	return "unknown"
}

func (s *StatsCollector) getProcessCommand(pid int) string {
	cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", pid)
	content, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return "unknown"
	}

	// Replace null bytes with spaces
	cmdline := strings.ReplaceAll(string(content), "\x00", " ")
	cmdline = strings.TrimSpace(cmdline)

	if cmdline == "" {
		return "unknown"
	}

	// Limit length for display
	if len(cmdline) > 50 {
		return cmdline[:47] + "..."
	}

	return cmdline
}

func (s *StatsCollector) getProcessCPUPercent(statFields []string) float64 {
	// This is a simplified CPU calculation
	// For accurate CPU usage, you'd need to track changes over time
	if len(statFields) > 15 {
		utime, _ := strconv.ParseUint(statFields[13], 10, 64)
		stime, _ := strconv.ParseUint(statFields[14], 10, 64)

		// Simple approximation - in a real implementation you'd track
		// the difference over time and divide by elapsed time
		totalTime := utime + stime
		return float64(totalTime) / 100000.0 // Rough approximation
	}
	return 0
}

func (s *StatsCollector) getProcessMemory(statusContent []byte) (float64, uint64) {
	lines := strings.Split(string(statusContent), "\n")
	var rss uint64

	for _, line := range lines {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if val, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					rss = val * 1024 // Convert from KB to bytes
				}
			}
			break
		}
	}

	// Calculate memory percentage: (RSS / MemTotal) * 100
	memStats := s.getMemoryStats()
	var memPercent float64
	if memStats.Total > 0 {
		memPercent = float64(rss) / float64(memStats.Total) * 100
	}

	return memPercent, rss
}

func (s *StatsCollector) getProcessRuntime(statFields []string) string {
	if len(statFields) > 21 {
		startTime, _ := strconv.ParseUint(statFields[21], 10, 64)

		// Get system boot time and calculate runtime
		bootTime := s.getSystemBootTime()
		currentTime := uint64(time.Now().Unix())
		processStart := bootTime + (startTime / 100) // startTime is in clock ticks

		runtime := time.Duration(currentTime-processStart) * time.Second

		hours := int(runtime.Hours())
		minutes := int(runtime.Minutes()) % 60
		seconds := int(runtime.Seconds()) % 60

		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return "00:00:00"
}

func (s *StatsCollector) getProcessPriority(statFields []string) int {
	if len(statFields) > 17 {
		if priority, err := strconv.Atoi(statFields[17]); err == nil {
			return priority
		}
	}
	return 20 // Default priority
}
