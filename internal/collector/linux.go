//go:build linux

package collector

import "runtime"

// Fallback function for non-Linux systems
func init() {
	if runtime.GOOS != "linux" {
		// For non-Linux systems, you might want to implement
		// platform-specific data collection or use a cross-platform library
		// like gopsutil: github.com/shirou/gopsutil
	}
}
