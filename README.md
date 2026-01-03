# lazylogcat

A TUI for viewing Android logcat logs.

## IMPORTANT

This project is in early development. Expect bugs and missing features.

## Prerequisites

- Go 1.25.5 or higher
- `adb` (Android Debug Bridge) installed and in PATH

## Installation

```bash
go install github.com/parfenovvs/lazylogcat@latest
```

### Keyboard Shortcuts

**Device View:**
- Navigate devices with arrow keys or vim keys (`j`/`k`)
- `Enter` - Connect to selected device

**Logcat View:**
- `Alt+d` - Back to device selection
- `Alt+p` - Filter by package name
- `Alt+c` - Toggle color output
- `Alt+t` - Toggle tag format
- `Alt+b` - Jump to bottom
- Arrow keys/mouse - Scroll through logs
