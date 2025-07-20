package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
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

func (s *StatsCollector) getCPUStats() models.CPUStats {
	// Read CPU info
	model, frequency := s.getCPUInfo()
	temp := s.getCPUTemperature()

	// Read CPU usage from /proc/stat
	usage, cores := s.getCPUUsage()

	return models.CPUStats{
		Usage:     usage,
		Cores:     cores,
		Frequency: frequency,
		Temp:      temp,
		Model:     model,
	}
}

func (s *StatsCollector) getCPUInfo() (string, float64) {
	content, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "Unknown CPU", 0
	}

	modelName := "Unknown CPU"
	freq := 0.0
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(line, "model name") {
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				modelName = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "cpu MHz") {
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				if f, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
					freq = f
				}
			}
		}
		if modelName != "Unknown CPU" && freq != 0.0 {
			break
		}
	}

	return modelName, freq
}

func (s *StatsCollector) getCPUTemperature() float64 {
	// Try different temperature sensor paths
	tempPaths := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/hwmon/hwmon0/temp1_input",
		"/sys/class/hwmon/hwmon1/temp1_input",
	}

	for _, path := range tempPaths {
		if content, err := os.ReadFile(path); err == nil {
			if temp, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64); err == nil {
				// Temperature is usually in millidegrees
				if temp > 1000 {
					return temp / 1000.0
				}
				return temp
			}
		}
	}
	return 0
}

func (s *StatsCollector) getCPUUsage() (float64, []float64) {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, nil
	}

	lines := strings.Split(string(content), "\n")
	var overallUsage float64
	var coreUsages []float64

	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			// Overall CPU usage
			overallUsage = s.parseCPULine(line)
		} else if strings.HasPrefix(line, "cpu") && len(line) > 3 {
			// Individual core usage
			usage := s.parseCPULine(line)
			coreUsages = append(coreUsages, usage)
		}
	}

	return overallUsage, coreUsages
}

func (s *StatsCollector) parseCPULine(line string) float64 {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return 0
	}

	// Parse CPU times: user, nice, system, idle, iowait, irq, softirq
	var times []uint64
	for i := 1; i < len(fields) && i < 8; i++ {
		if val, err := strconv.ParseUint(fields[i], 10, 64); err == nil {
			times = append(times, val)
		}
	}

	if len(times) < 4 {
		return 0
	}

	// Calculate total and idle time
	var total, idle uint64
	for i, time := range times {
		total += time
		if i == 3 { // idle time is at index 3
			idle = time
		}
	}

	if total == 0 {
		return 0
	}

	// Calculate usage percentage
	usage := float64(total-idle) / float64(total) * 100
	return usage
}

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

func (s *StatsCollector) getSystemBootTime() uint64 {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "btime ") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				if bootTime, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					return bootTime
				}
			}
		}
	}
	return 0
}

func getBootTime() time.Time {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Now()
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "btime ") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				if bootTime, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					return time.Unix(bootTime, 0)
				}
			}
		}
	}
	return time.Now()
}

// Fallback function for non-Linux systems
func init() {
	if runtime.GOOS != "linux" {
		// For non-Linux systems, you might want to implement
		// platform-specific data collection or use a cross-platform library
		// like gopsutil: github.com/shirou/gopsutil
	}
}
