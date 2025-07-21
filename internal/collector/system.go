package collector

import (
	"os"
	"strconv"
	"strings"
	"time"
)

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
