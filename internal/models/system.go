package models

import "time"

type SystemStats struct {
	CPU     CPUStats      `json:"cpu"`
	Memory  MemoryStats   `json:"memory"`
	Network NetworkStats  `json:"network"`
	Disk    []DiskStats   `json:"disk"`
	Battery BatteryStats  `json:"battery"`
	Uptime  time.Duration `json:"uptime"`
}

type CPUStats struct {
	Usage     float64   `json:"usage"`
	Cores     []float64 `json:"cores"`
	Frequency float64   `json:"frequency"`
	Temp      float64   `json:"temperature"`
	Model     string    `json:"model"`
}

type MemoryStats struct {
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Free         uint64  `json:"free"`
	Available    uint64  `json:"available"`
	UsagePercent float64 `json:"usage_percent"`
	SwapTotal    uint64  `json:"swap_total"`
	SwapUsed     uint64  `json:"swap_used"`
}

type BatteryStats struct {
	Level      int    `json:"level"`
	Status     string `json:"status"`
	TimeLeft   string `json:"time_left"`
	IsCharging bool   `json:"is_charging"`
	Health     int    `json:"health"`
}
