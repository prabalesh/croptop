# 🌱 CropTop

> *A beautiful, interactive terminal-based system monitor built with Go*

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/prabalesh/croptop)


## ✨ Overview

CropTop is a modern, feature-rich system monitoring tool that brings beautiful terminal UI to system administration. Built with Go and the powerful [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework, it provides real-time insights into your system's performance with an intuitive, keyboard-driven interface.

## 🚀 Features

### 📊 **Multi-Tab Interface**
- **Overview** - Quick system summary with key metrics
- **CPU** - Detailed CPU usage, temperature, and per-core statistics  
- **Memory** - RAM and swap usage with visual progress bars
- **Processes** - Interactive process list with sorting and navigation
- **Network** - Network interface statistics and traffic monitoring
- **Disk** - Disk usage for all mounted filesystems
- **Battery** - Battery status, health, and charging information

### 🎨 **Beautiful Terminal UI**
- Responsive design that adapts to terminal size
- Smooth progress bars and visual indicators
- Color-coded status information
- Scrollable content with navigation indicators
- Tab scrolling for smaller terminals

### ⚡ **Performance & Usability**
- Real-time updates (1-second refresh rate)
- Efficient resource usage
- Keyboard shortcuts for quick navigation
- Cross-platform compatibility (Linux, macOS, Windows)
- No external dependencies required

## 📦 Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/prabalesh/croptop.git

# Navigate to project directory
cd croptop

# Build and install
go mod tidy
go build -o croptop ./cmd/croptop

# Run CropTop
./croptop
```

### Using Go Install

```bash
go install github.com/prabalesh/croptop/cmd/croptop@latest
```

## 🎮 Usage

### Basic Commands

```bash
# Start CropTop
croptop
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `←/→` or `h/l` | Switch between tabs |
| `Shift+←/→` or `H/L` | Scroll tabs (when they don't fit) |
| `↑/↓` or `k/j` | Navigate processes / Scroll content |
| `PgUp/PgDn` | Page up/down scrolling |
| `Home/End` | Jump to top/bottom of content |
| `Ctrl+C` or `q` | Quit application |

### Screenshots

#### Overview Tab
- System summary with CPU and memory usage
- Quick stats including uptime and process count
- Visual progress bars for key metrics

#### CPU Tab
- CPU model and frequency information
- Real-time temperature monitoring
- Per-core usage with individual progress bars

#### Processes Tab
- Interactive process list with PID, name, CPU%, memory%
- Process status and command information
- Scrollable with selection highlighting

## 🏗️ Architecture

CropTop follows a clean, modular architecture:

```
croptop/
├── cmd/croptop/        # Application entry point
├── internal/
│   ├── collector/      # System data collection
│   ├── models/         # Data structures
│   └── ui/            # Terminal UI components
└── README.md
```

### Key Components

- **Collector**: Gathers system statistics (CPU, memory, processes, etc.)
- **Models**: Defines data structures for system information
- **UI**: Implements the terminal interface using Bubble Tea
- **Styles**: Manages consistent visual styling

## 🛠️ Development

### Prerequisites

- Go 1.21 or higher
- Terminal with color support

### Building from Source

```bash
# Clone and enter directory
git clone https://github.com/prabalesh/croptop.git
cd croptop

# Install dependencies
go mod download

# Run in development mode
go run ./cmd/croptop

# Build for production
go build -ldflags="-s -w" -o croptop ./cmd/croptop
```

### Dependencies

CropTop leverages these excellent Go libraries:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling and layout
- [Bubbles](https://github.com/charmbracelet/bubbles) - UI components (progress bars)

## 🤝 Contributing

Contributions are welcome! Here's how you can help:

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing-feature`)
5. **Open** a Pull Request

### Development Guidelines

- Follow Go best practices and formatting (`gofmt`, `golint`)
- Add tests for new functionality
- Update documentation as needed
- Ensure compatibility across platforms

## 📊 System Requirements

- **Operating System**: Linux, macOS, Windows
- **Go Version**: 1.21+
- **Terminal**: Any terminal with color support
- **Permissions**: Standard user permissions (no root required)

## 🔍 Troubleshooting

### Common Issues

**Permission denied errors:**
- CropTop doesn't require root permissions for basic functionality
- Some system stats may be limited without elevated privileges

**Terminal display issues:**
- Ensure your terminal supports color and Unicode characters
- Try resizing the terminal if content appears cut off

**Performance issues:**
- CropTop uses minimal resources, but you can adjust refresh rates if needed

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Charm](https://charm.sh/) libraries for beautiful terminal UI
- Inspired by classic system monitors like `htop` and `btop`
- Thanks to the Go community for excellent tooling and libraries

## 📞 Support & Community

- 🐛 **Issues**: [GitHub Issues](https://github.com/prabalesh/croptop/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/prabalesh/croptop/discussions)
- 📧 **Contact**: [@prabalesh](https://github.com/prabalesh)

---

<div align="center">
  <strong>Made with ❤️ and Go</strong>
  <br>
  <sub>⭐ Star us on GitHub if CropTop helps you monitor your system!</sub>
</div>
