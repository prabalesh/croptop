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
	Temp      float32   `json:"temperature"`
	Model     string    `json:"model"`
}

type MemoryStats struct {
	Total        float64 `json:"total"`
	Used         float64 `json:"used"`
	Free         float64 `json:"free"`
	Available    float64 `json:"available"`
	UsagePercent float64 `json:"usage_percent"`
	SwapTotal    float64 `json:"swap_total"`
	SwapUsed     float64 `json:"swap_used"`
}

type BatteryStats struct {
	Level      int    `json:"level"`
	Status     string `json:"status"`
	TimeLeft   string `json:"time_left"`
	IsCharging bool   `json:"is_charging"`
	Health     int    `json:"health"`
}
