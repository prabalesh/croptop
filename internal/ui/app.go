package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/prabalesh/croptop/internal/collector"
	"github.com/prabalesh/croptop/internal/models"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

type App struct {
	collector   *collector.StatsCollector
	stats       models.SystemStats
	processes   models.ProcessList
	activeTab   int
	tabs        []string
	width       int
	height      int
	selectedRow int
	// Tab scrolling state
	tabScrollOffset int
	// Vertical scrolling state
	verticalScrollOffset int
	contentHeight        int // Track content height for scrolling
	// Progress bars for different components
	cpuProgress     progress.Model
	memoryProgress  progress.Model
	diskProgress    progress.Model
	batteryProgress progress.Model
	coreProgresses  []progress.Model // For CPU cores
}

func NewApp() *App {
	// Initialize progress bars with consistent styling
	cpuProg := progress.New(progress.WithDefaultGradient())
	memoryProg := progress.New(progress.WithDefaultGradient())
	diskProg := progress.New(progress.WithDefaultGradient())
	batteryProg := progress.New(progress.WithDefaultGradient())

	return &App{
		collector:            collector.NewStatsCollector(),
		tabs:                 []string{"Overview", "CPU", "Memory", "Processes", "Network", "Disk", "Battery"},
		activeTab:            0,
		tabScrollOffset:      0,
		verticalScrollOffset: 0,
		cpuProgress:          cpuProg,
		memoryProgress:       memoryProg,
		diskProgress:         diskProg,
		batteryProgress:      batteryProg,
		coreProgresses:       make([]progress.Model, 0), // Will be initialized based on CPU cores
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.updateStats(),
		a.tick(),
	)
}

func (a *App) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (a *App) updateStats() tea.Cmd {
	return func() tea.Msg {
		stats := a.collector.GetSystemStats()
		processes := a.collector.GetProcessList()
		return struct {
			stats     models.SystemStats
			processes models.ProcessList
		}{stats, processes}
	}
}

// Initialize core progress bars based on the number of CPU cores
func (a *App) initializeCoreProgresses(coreCount int) {
	if len(a.coreProgresses) != coreCount {
		a.coreProgresses = make([]progress.Model, coreCount)
		for i := range a.coreProgresses {
			a.coreProgresses[i] = progress.New(progress.WithDefaultGradient())
			a.coreProgresses[i].Width = 30
		}
	}
}

// Calculate visible tabs based on screen width and scroll offset
func (a *App) getVisibleTabs() ([]string, []int, bool, bool) {
	if a.width <= 0 {
		return a.tabs, []int{}, false, false
	}

	// Estimate tab width (tab name + padding + borders)
	// This is an approximation - you might need to adjust based on your styling
	estimatedTabWidth := func(tabName string) int {
		return len(tabName) + 6 // 6 for padding and borders
	}

	visibleTabs := []string{}
	visibleIndices := []int{}
	currentWidth := 0
	availableWidth := a.width - 10 // Leave some margin

	// Ensure active tab is visible by adjusting scroll offset
	a.ensureActiveTabVisible()

	// Calculate visible tabs starting from scroll offset
	for i := a.tabScrollOffset; i < len(a.tabs); i++ {
		tabWidth := estimatedTabWidth(a.tabs[i])
		if currentWidth+tabWidth > availableWidth && len(visibleTabs) > 0 {
			break
		}
		visibleTabs = append(visibleTabs, a.tabs[i])
		visibleIndices = append(visibleIndices, i)
		currentWidth += tabWidth
	}

	// Check if we can scroll left or right
	canScrollLeft := a.tabScrollOffset > 0
	canScrollRight := a.tabScrollOffset+len(visibleTabs) < len(a.tabs)

	return visibleTabs, visibleIndices, canScrollLeft, canScrollRight
}

// Ensure the active tab is visible by adjusting scroll offset
func (a *App) ensureActiveTabVisible() {
	// If active tab is before the scroll offset, scroll left
	if a.activeTab < a.tabScrollOffset {
		a.tabScrollOffset = a.activeTab
		return
	}

	// If active tab is after visible range, scroll right
	_, visibleIndices, _, _ := a.getVisibleTabsRaw()
	if len(visibleIndices) > 0 {
		lastVisible := visibleIndices[len(visibleIndices)-1]
		if a.activeTab > lastVisible {
			// Calculate how much we need to scroll to make active tab visible
			a.tabScrollOffset = max(0, a.activeTab-2) // Show active tab with some context
		}
	}
}

// Raw calculation without ensuring active tab visibility (to avoid infinite recursion)
func (a *App) getVisibleTabsRaw() ([]string, []int, bool, bool) {
	if a.width <= 0 {
		return a.tabs, []int{}, false, false
	}

	estimatedTabWidth := func(tabName string) int {
		return len(tabName) + 6
	}

	visibleTabs := []string{}
	visibleIndices := []int{}
	currentWidth := 0
	availableWidth := a.width - 10

	for i := a.tabScrollOffset; i < len(a.tabs); i++ {
		tabWidth := estimatedTabWidth(a.tabs[i])
		if currentWidth+tabWidth > availableWidth && len(visibleTabs) > 0 {
			break
		}
		visibleTabs = append(visibleTabs, a.tabs[i])
		visibleIndices = append(visibleIndices, i)
		currentWidth += tabWidth
	}

	canScrollLeft := a.tabScrollOffset > 0
	canScrollRight := a.tabScrollOffset+len(visibleTabs) < len(a.tabs)

	return visibleTabs, visibleIndices, canScrollLeft, canScrollRight
}

// Get the height available for content (excluding sticky header elements)
func (a *App) getContentAreaHeight() int {
	// Reserve space for: title (2 lines), tabs (2 lines), help (2 lines), margins (2 lines)
	reservedHeight := 8
	return max(1, a.height-reservedHeight)
}

// Get the maximum scroll offset based on content height
func (a *App) getMaxScrollOffset() int {
	availableHeight := a.getContentAreaHeight()
	if a.contentHeight <= availableHeight {
		return 0
	}
	return a.contentHeight - availableHeight
}

// Clamp vertical scroll offset to valid range
func (a *App) clampVerticalScroll() {
	maxOffset := a.getMaxScrollOffset()
	a.verticalScrollOffset = max(0, min(a.verticalScrollOffset, maxOffset))
}

// Apply vertical scrolling to content by truncating lines
func (a *App) applyVerticalScroll(content string) string {
	lines := strings.Split(content, "\n")
	a.contentHeight = len(lines)

	a.clampVerticalScroll()

	availableHeight := a.getContentAreaHeight()

	// If content fits entirely, return as-is
	if len(lines) <= availableHeight {
		return content
	}

	// Calculate visible lines
	startLine := a.verticalScrollOffset
	endLine := min(startLine+availableHeight, len(lines))

	visibleLines := lines[startLine:endLine]

	// Add scroll indicators if needed
	result := strings.Join(visibleLines, "\n")

	// Add scroll indicators in the content area
	if a.verticalScrollOffset > 0 {
		scrollUpIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true).
			Render("▲ More content above")
		result = scrollUpIndicator + "\n" + result
	}

	if a.verticalScrollOffset < a.getMaxScrollOffset() {
		scrollDownIndicator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true).
			Render("▼ More content below")
		result = result + "\n" + scrollDownIndicator
	}

	return result
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

		// Update progress bar widths based on window size
		progressWidth := min(50, a.width-20)
		a.cpuProgress.Width = progressWidth
		a.memoryProgress.Width = progressWidth
		a.diskProgress.Width = min(40, a.width-25)
		a.batteryProgress.Width = min(40, a.width-25)

		return a, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "left", "h":
			if a.activeTab > 0 {
				a.activeTab--
				a.verticalScrollOffset = 0 // Reset scroll when changing tabs
			}
		case "right", "l":
			if a.activeTab < len(a.tabs)-1 {
				a.activeTab++
				a.verticalScrollOffset = 0 // Reset scroll when changing tabs
			}
		case "shift+left", "H":
			// Scroll tabs left
			if a.tabScrollOffset > 0 {
				a.tabScrollOffset--
			}
		case "shift+right", "L":
			// Scroll tabs right
			_, _, _, canScrollRight := a.getVisibleTabs()
			if canScrollRight {
				a.tabScrollOffset++
			}
		case "up", "k":
			// Handle different behaviors based on current tab
			if a.activeTab == 3 { // Processes tab
				if a.selectedRow > 0 {
					a.selectedRow--
				}
			} else {
				// Vertical scroll up for other tabs
				if a.verticalScrollOffset > 0 {
					a.verticalScrollOffset--
				}
			}
		case "down", "j":
			// Handle different behaviors based on current tab
			if a.activeTab == 3 { // Processes tab
				if a.selectedRow < len(a.processes.Processes)-1 {
					a.selectedRow++
				}
			} else {
				// Vertical scroll down for other tabs
				a.verticalScrollOffset++
				a.clampVerticalScroll() // Ensure we don't scroll past content
			}
		case "pgup", "ctrl+u":
			// Page up - scroll up by half the available height
			scrollAmount := max(1, a.getContentAreaHeight()/2)
			a.verticalScrollOffset = max(0, a.verticalScrollOffset-scrollAmount)
		case "pgdown", "ctrl+d":
			// Page down - scroll down by half the available height
			scrollAmount := max(1, a.getContentAreaHeight()/2)
			a.verticalScrollOffset += scrollAmount
			a.clampVerticalScroll()
		case "home", "ctrl+home":
			// Go to top
			a.verticalScrollOffset = 0
		case "end", "ctrl+end":
			// Go to bottom
			a.verticalScrollOffset = a.getMaxScrollOffset()
		}

	case tickMsg:
		return a, tea.Batch(a.updateStats(), a.tick())

	case struct {
		stats     models.SystemStats
		processes models.ProcessList
	}:
		a.stats = msg.stats
		a.processes = msg.processes

		// Initialize core progresses if needed
		a.initializeCoreProgresses(len(a.stats.CPU.Cores))
	}

	return a, nil
}

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	// Title (sticky)
	title := TitleStyle.Width(a.width).Render("CropTop")

	// Tabs (sticky)
	tabs := a.renderTabs()

	// Content (scrollable)
	var content string
	switch a.activeTab {
	case 0:
		content = a.renderOverview()
	case 1:
		content = a.renderCPU()
	case 2:
		content = a.renderMemory()
	case 3:
		content = a.renderProcesses()
	case 4:
		content = a.renderNetwork()
	case 5:
		content = a.renderDisk()
	case 6:
		content = a.renderBattery()
	}

	// Apply vertical scrolling to content
	scrollableContent := a.applyVerticalScroll(content)

	// Help text (sticky)
	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("←/→ h/l: tabs • Shift+←/→ H/L: scroll tabs • ↑/↓ k/j: scroll • PgUp/PgDn: page scroll • Home/End: top/bottom • q: quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		tabs,
		"",
		scrollableContent,
		"",
		help,
	)
}

func (a *App) renderTabs() string {
	visibleTabs, visibleIndices, canScrollLeft, canScrollRight := a.getVisibleTabs()

	var tabElements []string

	// Add left scroll indicator
	if canScrollLeft {
		scrollLeft := lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true).
			Render("‹")
		tabElements = append(tabElements, scrollLeft)
	}

	// Render visible tabs
	for i, tab := range visibleTabs {
		realIndex := visibleIndices[i]
		if realIndex == a.activeTab {
			tabElements = append(tabElements, ActiveTabStyle.Render(tab))
		} else {
			tabElements = append(tabElements, InactiveTabStyle.Render(tab))
		}
	}

	// Add right scroll indicator
	if canScrollRight {
		scrollRight := lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true).
			Render("›")
		tabElements = append(tabElements, scrollRight)
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, tabElements...)
}

func (a *App) renderOverview() string {
	cpu := fmt.Sprintf("CPU: %.1f%%", a.stats.CPU.Usage)
	memory := fmt.Sprintf("Memory: %.1f%%", a.stats.Memory.UsagePercent)
	processes := fmt.Sprintf("Processes: %d", a.processes.Total)
	uptime := fmt.Sprintf("Uptime: %v", a.stats.Uptime.Truncate(time.Second))

	// Create progress bars for overview
	cpuBar := a.cpuProgress.ViewAs(a.stats.CPU.Usage / 100.0)
	memBar := a.memoryProgress.ViewAs(a.stats.Memory.UsagePercent / 100.0)

	return BaseStyle.Width(a.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			HeaderStyle.Render("System Overview"),
			"",
			LabelStyle.Render(cpu),
			cpuBar,
			"",
			LabelStyle.Render(memory),
			memBar,
			"",
			LabelStyle.Render(processes),
			LabelStyle.Render(uptime),
			"",
			"",
			HeaderStyle.Render("Quick Stats"),
			fmt.Sprintf("CPU Temperature: %.1f°C", a.stats.CPU.Temp),
			fmt.Sprintf("CPU Cores: %d", len(a.stats.CPU.Cores)),
			fmt.Sprintf("Memory Total: %.1f GB", float64(a.stats.Memory.Total)/(1024*1024*1024)),
			"Disk Usage: Multiple drives",
			fmt.Sprintf("Network Interfaces: %d", len(a.stats.Network.Interfaces)),
		),
	)
}

func (a *App) renderCPU() string {
	content := []string{
		HeaderStyle.Render("CPU Information"),
		"",
		fmt.Sprintf("%s %s", LabelStyle.Render("Model:"), ValueStyle.Render(a.stats.CPU.Model)),
		fmt.Sprintf("%s %.1f MHz", LabelStyle.Render("Frequency:"), a.stats.CPU.Frequency),
		fmt.Sprintf("%s %.1f°C", LabelStyle.Render("Temperature:"), a.stats.CPU.Temp),
		"",
		fmt.Sprintf("%s %.1f%%", LabelStyle.Render("Overall Usage:"), a.stats.CPU.Usage),
		a.cpuProgress.ViewAs(a.stats.CPU.Usage / 100.0),
		"",
		HeaderStyle.Render("Per-Core Usage"),
	}

	for i, usage := range a.stats.CPU.Cores {
		if i < len(a.coreProgresses) {
			content = append(content,
				fmt.Sprintf("Core %d: %.1f%%", i, usage),
				a.coreProgresses[i].ViewAs(usage/100.0),
				"",
			)
		}
	}

	return BaseStyle.Width(a.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, content...),
	)
}

func (a *App) renderMemory() string {
	mem := a.stats.Memory

	content := []string{
		HeaderStyle.Render("Memory Information"),
		"",
		fmt.Sprintf("%s %.1f GB", LabelStyle.Render("Total:"), float64(mem.Total)/(1024*1024*1024)),
		fmt.Sprintf("%s %.1f GB", LabelStyle.Render("Used:"), float64(mem.Used)/(1024*1024*1024)),
		fmt.Sprintf("%s %.1f GB", LabelStyle.Render("Free:"), float64(mem.Free)/(1024*1024*1024)),
		fmt.Sprintf("%s %.1f GB", LabelStyle.Render("Available:"), float64(mem.Available)/(1024*1024*1024)),
		"",
		fmt.Sprintf("%s %.1f%% (%.1f GB/%.1f GB)", LabelStyle.Render("Usage:"), mem.UsagePercent, a.stats.Memory.Used/(1024*1024*1024), a.stats.Memory.Total/(1024*1024*1024)),
		a.memoryProgress.ViewAs(mem.UsagePercent / 100.0),
		"",
		HeaderStyle.Render("Swap"),
		fmt.Sprintf("%s %.1f GB", LabelStyle.Render("Total:"), float64(mem.SwapTotal)/(1024*1024*1024)),
		fmt.Sprintf("%s %.1f GB", LabelStyle.Render("Used:"), float64(mem.SwapUsed)/(1024*1024*1024)),
	}

	return BaseStyle.Width(a.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, content...),
	)
}

func (a *App) renderProcesses() string {
	// Calculate visible rows (leave space for header and stats)
	visibleRows := a.height - 8
	if visibleRows < 1 {
		visibleRows = 1
	}

	// Calculate scroll window
	startIdx := 0
	if a.selectedRow >= visibleRows {
		startIdx = a.selectedRow - visibleRows + 1
	}
	endIdx := startIdx + visibleRows
	if endIdx > len(a.processes.Processes) {
		endIdx = len(a.processes.Processes)
	}

	var content strings.Builder

	// Header section
	content.WriteString(HeaderStyle.Render("Process List"))
	content.WriteString("\n\n")

	// Stats
	stats := fmt.Sprintf("Total: %d | Running: %d | Sleeping: %d | Zombie: %d",
		a.processes.Total, a.processes.Running, a.processes.Sleeping, a.processes.Zombie)
	content.WriteString(stats)
	content.WriteString("\n\n")

	// Table header with proper styling
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")). // Bright blue
		PaddingLeft(1).
		PaddingRight(1)

	header := fmt.Sprintf("%-8s %-20s %8s %8s %-12s %-s",
		"PID", "NAME", "CPU%", "MEM%", "STATUS", "COMMAND")
	content.WriteString(headerStyle.Render(header))
	content.WriteString("\n")

	// Process rows with proper alignment
	for i := startIdx; i < endIdx; i++ {
		proc := a.processes.Processes[i]

		// Truncate strings to fit columns
		name := truncateString(proc.Name, 20)
		status := truncateString(proc.Status, 12)

		// Calculate remaining width for command
		usedWidth := 8 + 1 + 20 + 1 + 8 + 1 + 8 + 1 + 12 + 1 // PID + spaces + NAME + spaces + CPU% + spaces + MEM% + spaces + STATUS + spaces
		remainingWidth := a.width - usedWidth - 4            // -4 for padding
		if remainingWidth < 10 {
			remainingWidth = 10
		}
		command := truncateString(proc.Command, remainingWidth)

		row := fmt.Sprintf("%-8d %-20s %7.1f%% %7.1f%% %-12s %s",
			proc.PID, name, proc.CPUPercent, proc.MemPercent, status, command)

		// Style the row
		rowStyle := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1)

		// Highlight selected row
		if i == a.selectedRow {
			rowStyle = rowStyle.
				Background(lipgloss.Color("240")). // Light gray background
				Foreground(lipgloss.Color("15")).  // White text
				Bold(true)
		} else {
			// Alternate row colors for better readability
			if (i-startIdx)%2 == 0 {
				rowStyle = rowStyle.Foreground(lipgloss.Color("252")) // Light gray text
			} else {
				rowStyle = rowStyle.Foreground(lipgloss.Color("245")) // Slightly darker gray text
			}
		}

		content.WriteString(rowStyle.Render(row))
		content.WriteString("\n")
	}

	// Add some spacing and scroll indicator
	if len(a.processes.Processes) > visibleRows {
		content.WriteString("\n")
		scrollInfo := fmt.Sprintf("Showing %d-%d of %d processes • Use ↑↓ arrows or j/k to navigate",
			startIdx+1, endIdx, len(a.processes.Processes))
		scrollStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true).
			PaddingLeft(1)
		content.WriteString(scrollStyle.Render(scrollInfo))
	}

	return BaseStyle.Render(content.String())
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func (a *App) renderNetwork() string {
	content := []string{
		HeaderStyle.Render("Network Interfaces"),
		"",
		fmt.Sprintf("%s %.1f MB", LabelStyle.Render("Total RX:"), float64(a.stats.Network.TotalRx)/(1024*1024)),
		fmt.Sprintf("%s %.1f MB", LabelStyle.Render("Total TX:"), float64(a.stats.Network.TotalTx)/(1024*1024)),
		"",
	}

	for _, iface := range a.stats.Network.Interfaces {
		content = append(content,
			HeaderStyle.Render("Interface: "+iface.Name),
			fmt.Sprintf("%s %s", LabelStyle.Render("Status:"), ValueStyle.Render(iface.Status)),
			fmt.Sprintf("%s %s", LabelStyle.Render("Speed:"), ValueStyle.Render(iface.Speed)),
			fmt.Sprintf("%s %.1f MB", LabelStyle.Render("RX:"), float64(iface.RxBytes)/(1024*1024)),
			fmt.Sprintf("%s %.1f MB", LabelStyle.Render("TX:"), float64(iface.TxBytes)/(1024*1024)),
			fmt.Sprintf("%s %d", LabelStyle.Render("RX Packets:"), iface.RxPackets),
			fmt.Sprintf("%s %d", LabelStyle.Render("TX Packets:"), iface.TxPackets),
		)
	}

	return BaseStyle.Width(a.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, content...),
	)
}

func (a *App) renderDisk() string {
	content := []string{
		HeaderStyle.Render("Disk Usage"),
		"",
	}

	for _, disk := range a.stats.Disk {
		// Create a temporary progress bar for this disk
		diskBar := a.diskProgress.ViewAs(disk.UsagePercent / 100.0)

		content = append(content,
			HeaderStyle.Render(disk.Device+" ("+disk.Mountpoint+")"),
			fmt.Sprintf("%s %s", LabelStyle.Render("Filesystem:"), ValueStyle.Render(disk.Filesystem)),
			fmt.Sprintf("%s %.1f GB", LabelStyle.Render("Total:"), float64(disk.Total)/(1024*1024*1024)),
			fmt.Sprintf("%s %.1f GB", LabelStyle.Render("Used:"), float64(disk.Used)/(1024*1024*1024)),
			fmt.Sprintf("%s %.1f GB", LabelStyle.Render("Free:"), float64(disk.Free)/(1024*1024*1024)),
			fmt.Sprintf("%s %.1f%%", LabelStyle.Render("Usage:"), disk.UsagePercent),
			diskBar,
			"",
		)
	}

	return BaseStyle.Width(a.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, content...),
	)
}

func (a *App) renderBattery() string {
	battery := a.stats.Battery

	statusStyle := SuccessStyle
	if battery.Level < 20 {
		statusStyle = ErrorStyle
	} else if battery.Level < 50 {
		statusStyle = WarningStyle
	}

	// Create battery progress bar
	batteryBar := a.batteryProgress.ViewAs(float64(battery.Level) / 100.0)

	content := []string{
		HeaderStyle.Render("Battery Information"),
		"",
		fmt.Sprintf("%s %s", LabelStyle.Render("Status:"), statusStyle.Render(battery.Status)),
		fmt.Sprintf("%s %d%%", LabelStyle.Render("Level:"), battery.Level),
		batteryBar,
		"",
		fmt.Sprintf("%s %s", LabelStyle.Render("Time Left:"), ValueStyle.Render(battery.TimeLeft)),
		fmt.Sprintf("%s %d%%", LabelStyle.Render("Health:"), battery.Health),
		fmt.Sprintf("%s %v", LabelStyle.Render("Charging:"), battery.IsCharging),
	}

	return BaseStyle.Width(a.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Left, content...),
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
