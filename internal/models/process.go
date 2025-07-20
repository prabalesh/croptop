package models

type Process struct {
	PID        int     `json:"pid"`
	Name       string  `json:"name"`
	Command    string  `json:"command"`
	CPUPercent float64 `json:"cpu_percent"`
	MemPercent float64 `json:"mem_percent"`
	MemRSS     uint64  `json:"mem_rss"`
	Status     string  `json:"status"`
	User       string  `json:"user"`
	Runtime    string  `json:"runtime"`
	Priority   int     `json:"priority"`
}

type ProcessList struct {
	Processes []Process `json:"processes"`
	Total     int       `json:"total"`
	Running   int       `json:"running"`
	Sleeping  int       `json:"sleeping"`
	Zombie    int       `json:"zombie"`
}
