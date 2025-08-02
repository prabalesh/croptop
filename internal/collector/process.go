package collector

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/prabalesh/croptop/internal/models"
)

// SortBy represents different sorting options
type SortBy int

const (
	SortByPID SortBy = iota
	SortByCPU
	SortByMemory
	SortByName
)

// GetProcessList returns unsorted process list (maintains backward compatibility)
func (s *StatsCollector) GetProcessList() models.ProcessList {
	return s.GetProcessListSorted(SortByCPU, true)
}

// GetProcessListSorted returns process list sorted by specified criteria
func (s *StatsCollector) GetProcessListSorted(sortBy SortBy, descending bool) models.ProcessList {
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

	// Sort processes based on criteria
	s.sortProcesses(processes, sortBy, descending)

	total = len(processes)

	return models.ProcessList{
		Processes: processes,
		Total:     total,
		Running:   running,
		Sleeping:  sleeping,
		Zombie:    zombie,
	}
}

// sortProcesses sorts the process slice based on the specified criteria
func (s *StatsCollector) sortProcesses(processes []models.Process, sortBy SortBy, descending bool) {
	switch sortBy {
	case SortByCPU:
		sort.Slice(processes, func(i, j int) bool {
			if descending {
				return processes[i].CPUPercent > processes[j].CPUPercent
			}
			return processes[i].CPUPercent < processes[j].CPUPercent
		})
	case SortByMemory:
		sort.Slice(processes, func(i, j int) bool {
			if descending {
				return processes[i].MemPercent > processes[j].MemPercent
			}
			return processes[i].MemPercent < processes[j].MemPercent
		})
	case SortByName:
		sort.Slice(processes, func(i, j int) bool {
			if descending {
				return processes[i].Name > processes[j].Name
			}
			return processes[i].Name < processes[j].Name
		})
	case SortByPID:
		fallthrough
	default:
		sort.Slice(processes, func(i, j int) bool {
			if descending {
				return processes[i].PID > processes[j].PID
			}
			return processes[i].PID < processes[j].PID
		})
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
	// More accurate CPU calculation similar to htop
	if len(statFields) < 22 {
		return 0
	}

	utime, _ := strconv.ParseUint(statFields[13], 10, 64)
	stime, _ := strconv.ParseUint(statFields[14], 10, 64)
	starttime, _ := strconv.ParseUint(statFields[21], 10, 64)

	// Read system uptime and total CPU time
	uptimeContent, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}

	statContent, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}

	// Parse uptime
	uptimeFields := strings.Fields(string(uptimeContent))
	if len(uptimeFields) < 1 {
		return 0
	}
	uptime, _ := strconv.ParseFloat(uptimeFields[0], 64)

	// Parse total CPU time from first line of /proc/stat
	statLines := strings.Split(string(statContent), "\n")
	if len(statLines) < 1 {
		return 0
	}

	cpuLine := strings.Fields(statLines[0])
	if len(cpuLine) < 8 || cpuLine[0] != "cpu" {
		return 0
	}

	// Sum all CPU times to get total system CPU time
	var totalSystemCPU uint64
	for i := 1; i < len(cpuLine) && i < 8; i++ {
		val, _ := strconv.ParseUint(cpuLine[i], 10, 64)
		totalSystemCPU += val
	}

	// Calculate process CPU time in seconds
	processCPUTime := float64(utime+stime) / 100.0

	// Calculate process runtime in seconds
	processRuntime := uptime - (float64(starttime) / 100.0)
	if processRuntime <= 0 {
		return 0
	}

	// Calculate CPU usage as percentage of single core
	// This gives a more realistic percentage similar to htop
	cpuUsage := (processCPUTime / processRuntime) * 100.0

	// Cap at 100% per core (htop style)
	if cpuUsage > 100.0 {
		cpuUsage = 100.0
	}

	return cpuUsage
}

func (s *StatsCollector) getProcessMemory(statusContent []byte) (float64, uint64) {
	lines := strings.Split(string(statusContent), "\n")
	var rss uint64

	for _, line := range lines {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if val, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					rss = val
				}
			}
			break
		}
	}

	// Calculate memory percentage: (RSS / MemTotal) * 100
	memStats := s.getMemoryStats()
	// fmt.Println(memStats.Total)
	// fmt.Println(rss)
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
