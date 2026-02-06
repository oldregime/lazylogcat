package logcatui

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/parfenovvs/lazylogcat/internal/model"
	"github.com/parfenovvs/lazylogcat/internal/tui"
	"github.com/parfenovvs/lazylogcat/internal/tui/commandui"
	"github.com/parfenovvs/lazylogcat/internal/tui/theme"
	"github.com/parfenovvs/lazylogcat/internal/util"
)

const maxLogLines = 10000
const batchTimeout = 50 * time.Millisecond

var (
	titleStyle = func() lipgloss.Style {
		return theme.Panel().
			Padding(0, 1)
	}()
	dialogStyle = func() lipgloss.Style {
		return theme.ActivePanel().
			Padding(1, 2)
	}

	helpTextNormal = "ctrl+f filters • ctrl+r reconnect • ctrl+d devices • W toggle wrap • L toggle level • G jump to recent • C clear • v visual"
	helpTextVisual = "j/↓ down • k/↑ up • V select multiple • y copy • esc exit visual"
)

type LogcatViewModel struct {
	parentSize        model.Size
	viewport          viewport.Model
	device            model.Device
	filter            model.Filter
	format            model.Format
	log               *util.RingBuffer
	pendingLogs       []string
	visualMode        bool
	currentLine       int
	startSelected     int
	softWrap          bool
	err               error
	showCommandDialog bool
	commandTable      table.Model
	commandSkipRows   map[int]bool
}

type logcatMsg struct {
	Line string
}

type logcatEmptyMsg struct{}

type logcatErrorMsg struct {
	Err error
}

type logcatConnectedMsg struct{}

type batchTickMsg struct{}

// updateResult is returned by key handlers to indicate what action to take
type updateResult struct {
	cmd         tea.Cmd
	needsRender bool
}

func readNext(m LogcatViewModel) tea.Msg {
	line, err := util.ReadNextLogLine()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil // End of stream
		}
		slog.Warn("Error reading logcat line", "error", err)
		return nil
	}

	// Filter empty lines (allowed in long format)
	if !m.format.Long && strings.Trim(line, "\n\r ") == "" {
		return logcatEmptyMsg{}
	}

	// Filter by text search (case-insensitive)
	if m.filter.Text != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(m.filter.Text)) {
		return logcatEmptyMsg{}
	}

	return logcatMsg{Line: line}
}

func tickForBatch() tea.Cmd {
	return tea.Tick(batchTimeout, func(t time.Time) tea.Msg {
		return batchTickMsg{}
	})
}

func New(parentSize model.Size, device model.Device, filter model.Filter, format model.Format) LogcatViewModel {
	m := LogcatViewModel{
		parentSize:    parentSize,
		device:        device,
		log:           util.NewRingBuffer(maxLogLines),
		softWrap:      true,
		startSelected: -1,
		filter:        filter,
		format:        format,
	}

	headerHeight := lipgloss.Height(m.headerView())
	footerHeight := lipgloss.Height(m.footerView())
	vp := viewport.New(parentSize.Width, parentSize.Height-footerHeight-headerHeight-1)
	m.viewport = vp

	return m
}

func (m LogcatViewModel) Update(msg tea.Msg) (LogcatViewModel, tea.Cmd) {
	var (
		cmd         tea.Cmd
		cmds        []tea.Cmd
		needsRender bool
		gotoBottom  bool
	)

	switch msg := msg.(type) {
	case model.Size:
		m.parentSize = msg
		return m, func() tea.Msg {
			return tui.MeasureCmd{}
		}

	case tui.MeasureCmd:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		m.viewport.Width = m.parentSize.Width
		m.viewport.Height = m.parentSize.Height - footerHeight - headerHeight - 1
		needsRender = true

	case tea.KeyMsg:
		result := m.handleKeyMsg(msg)
		cmd = result.cmd
		needsRender = result.needsRender
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tui.ReconnectLogcatCmd:
		util.CloseLogcat()
		m.log = util.NewRingBuffer(maxLogLines)
		m.pendingLogs = nil
		return m, tea.Batch(
			func() tea.Msg {
				err := util.ConnectLogcat(m.device.Id, m.filter, m.format)
				if err != nil {
					return logcatErrorMsg{Err: err}
				}
				return logcatConnectedMsg{}
			},
			func() tea.Msg {
				return tui.MeasureCmd{}
			},
		)

	case logcatConnectedMsg:
		return m, tea.Batch(
			func() tea.Msg { return readNext(m) },
			tickForBatch(),
		)

	case logcatMsg:
		if m.visualMode {
			return m, nil
		}
		m.pendingLogs = append(m.pendingLogs, msg.Line)
		return m, func() tea.Msg {
			return readNext(m)
		}

	case logcatEmptyMsg:
		if m.visualMode {
			return m, nil
		}
		return m, func() tea.Msg {
			return readNext(m)
		}

	case batchTickMsg:
		if !m.visualMode && len(m.pendingLogs) > 0 {
			wasAtBottom := m.viewport.AtBottom()
			for _, line := range m.pendingLogs {
				m.log.Append(line + "\n")
			}
			m.pendingLogs = nil
			needsRender = true
			if wasAtBottom {
				gotoBottom = true
			}
		}
		cmds = append(cmds, tickForBatch())

	case logcatErrorMsg:
		m.err = msg.Err
		util.CloseLogcat()
		return m, nil
	}

	if needsRender {
		m.Render()
	}
	if gotoBottom {
		m.viewport.GotoBottom()
	}

	// Viewport update for non-visual mode scrolling (skip internal tick messages)
	if !m.visualMode && !m.showCommandDialog {
		if _, isBatchTick := msg.(batchTickMsg); !isBatchTick {
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *LogcatViewModel) Render() {
	var b strings.Builder
	logs := m.log.All()
	for i, msg := range logs {
		if m.visualMode {
			selected := false
			if m.startSelected >= 0 {
				min := min(m.currentLine, m.startSelected)
				max := max(m.currentLine, m.startSelected)
				selected = i >= min && i <= max
			} else if i == m.currentLine {
				selected = true
			}
			line := strings.TrimSuffix(msg, "\n")
			if selected {
				styled := lipgloss.NewStyle().
					Bold(true).
					Background(theme.BGCursor).
					Foreground(theme.FGSelected).
					Width(m.viewport.Width).
					Render(line)
				b.WriteString(styled)
				b.WriteString("\n")
				continue
			}
		}
		if m.format.Color {
			line := strings.TrimSuffix(msg, "\n")
			style := lipgloss.NewStyle().
				Foreground(theme.GetLogColor(util.GetLogLevel(line, m.format)))
			if m.softWrap {
				style = style.Width(m.viewport.Width)
			}
			styled := style.Render(line)
			b.WriteString(styled)
			b.WriteString("\n")
		} else {
			b.WriteString(msg)
		}
	}
	wrapped := b.String()
	if m.softWrap {
		wrapped = lipgloss.NewStyle().Width(m.viewport.Width).Render(wrapped)
	}
	m.viewport.SetContent(wrapped)
}

// handleKeyMsg routes key messages to appropriate handlers based on mode
func (m *LogcatViewModel) handleKeyMsg(msg tea.KeyMsg) updateResult {
	key := msg.String()

	// When command dialog is open, capture all keys
	if m.showCommandDialog {
		if key == "ctrl+p" || key == "esc" {
			m.showCommandDialog = false
			return updateResult{needsRender: true}
		}
		prevCursor := m.commandTable.Cursor()
		m.commandTable, _ = m.commandTable.Update(msg)
		newCursor := m.commandTable.Cursor()

		if m.commandSkipRows[newCursor] && newCursor != prevCursor {
			dir := 1
			if newCursor < prevCursor {
				dir = -1
			}
			rowCount := len(m.commandTable.Rows())
			target := newCursor + dir
			for target >= 0 && target < rowCount && m.commandSkipRows[target] {
				target += dir
			}
			if target >= 0 && target < rowCount {
				m.commandTable.SetCursor(target)
			} else {
				m.commandTable.SetCursor(prevCursor)
			}
		}

		return updateResult{}
	}

	// Try global keys first (work in both modes)
	if result, handled := m.handleGlobalKey(key); handled {
		return result
	}

	// Mode-specific handling
	if m.visualMode {
		return m.handleVisualModeKey(key)
	}
	return m.handleNormalModeKey(key)
}

// handleGlobalKey handles keys that work in both normal and visual modes
func (m *LogcatViewModel) handleGlobalKey(key string) (updateResult, bool) {
	switch key {
	case "ctrl+r":
		m.visualMode = false
		return updateResult{
			cmd: func() tea.Msg { return tui.ReconnectLogcatCmd{} },
		}, true

	case "ctrl+d":
		return updateResult{
			cmd: func() tea.Msg { return tui.NavigateToDevicesCmd{} },
		}, true

	case "ctrl+f":
		return updateResult{
			cmd: func() tea.Msg { return tui.NavigateToFilterCmd{} },
		}, true

	case "G":
		m.viewport.GotoBottom()
		return updateResult{}, true

	case "v":
		m.visualMode = !m.visualMode
		if m.visualMode {
			m.viewport.GotoBottom()
			m.currentLine = m.log.Size() - 1
			return updateResult{needsRender: true}, true
		}
		return updateResult{
			cmd: func() tea.Msg { return tui.ReconnectLogcatCmd{} },
		}, true
	}

	return updateResult{}, false
}

// handleNormalModeKey handles keys specific to normal (non-visual) mode
func (m *LogcatViewModel) handleNormalModeKey(key string) updateResult {
	switch key {
	case "W":
		m.softWrap = !m.softWrap
		return updateResult{
			cmd: func() tea.Msg { return tui.ReconnectLogcatCmd{} },
		}

	case "L":
		m.filter.Level = m.filter.Level.Next()
		return updateResult{
			cmd: func() tea.Msg { return tui.ReconnectLogcatCmd{} },
		}

	case "C":
		m.log.Clear()
		return updateResult{needsRender: true}

	case "ctrl+p":
		m.showCommandDialog = !m.showCommandDialog
		if m.showCommandDialog {
			m.commandTable, m.commandSkipRows = newCommandTable(m.filter, m.format, m.softWrap)
		}
		return updateResult{needsRender: true}
	}

	return updateResult{}
}

// handleVisualModeKey handles keys specific to visual mode
func (m *LogcatViewModel) handleVisualModeKey(key string) updateResult {
	switch key {
	case "V":
		if m.startSelected >= 0 {
			m.startSelected = -1
		} else {
			m.startSelected = m.currentLine
		}
		return updateResult{needsRender: true}

	case "esc":
		if m.startSelected >= 0 {
			m.startSelected = -1
			return updateResult{needsRender: true}
		}
		m.visualMode = false
		m.startSelected = -1
		return updateResult{
			cmd: func() tea.Msg { return tui.ReconnectLogcatCmd{} },
		}

	case "y":
		if m.currentLine >= 0 && m.currentLine < m.log.Size() {
			var err error
			if m.startSelected >= 0 {
				start := min(m.currentLine, m.startSelected)
				end := max(m.currentLine, m.startSelected)
				var lines []string
				logs := m.log.Recent(m.log.Size() - start)
				for i := 0; i <= end-start; i++ {
					lines = append(lines, strings.TrimSpace(logs[i]))
				}
				err = util.CopyToClipboard(lines...)
				m.startSelected = -1
				return updateResult{needsRender: true}
			} else {
				logs := m.log.Recent(m.log.Size() - m.currentLine)
				lineText := strings.TrimSpace(logs[0])
				err = util.CopyToClipboard(lineText)
			}
			if err != nil {
				slog.Error("Failed to copy to clipboard", "error", err)
			}
		}
		return updateResult{}

	case "j", "down":
		if m.currentLine < m.log.Size()-1 {
			m.currentLine++
			m.ensureLineVisible()
			return updateResult{needsRender: true}
		}

	case "k", "up":
		if m.currentLine > 0 {
			m.currentLine--
			m.ensureLineVisible()
			return updateResult{needsRender: true}
		}
	}

	return updateResult{}
}

func (m *LogcatViewModel) ensureLineVisible() {
	if !m.visualMode || m.currentLine < 0 || m.currentLine >= m.log.Size() {
		return
	}

	min := min(m.currentLine, m.startSelected)
	max := max(m.currentLine, m.startSelected)

	logs := m.log.All()
	linesUpToCurrent := 0
	for i := 0; i <= m.currentLine; i++ {
		line := strings.TrimSuffix(logs[i], "\n")
		if m.softWrap || (m.startSelected != -1 && i >= min && i <= max) {
			linesUpToCurrent += lipgloss.Height(lipgloss.NewStyle().Width(m.viewport.Width).Render(line))
		} else {
			linesUpToCurrent++
		}
	}

	if linesUpToCurrent < m.viewport.YOffset+2 {
		m.viewport.HalfPageUp()
	} else if linesUpToCurrent > m.viewport.YOffset+m.viewport.Height-1 {
		m.viewport.HalfPageDown()
	}
}

func (m LogcatViewModel) renderBaseView() string {
	return fmt.Sprintf(
		"%s\n%s\n%s",
		m.headerView(),
		m.viewport.View(),
		m.footerView(),
	)
}

func (m LogcatViewModel) overlayDialog(baseView, dialog string) string {
	// Ensure base view fills the entire parent size
	background := lipgloss.Place(
		m.parentSize.Width,
		m.parentSize.Height,
		lipgloss.Left,
		lipgloss.Top,
		baseView,
	)

	// Split background into lines
	bgLines := strings.Split(background, "\n")

	// Calculate dialog dimensions
	dialogLines := strings.Split(dialog, "\n")
	dialogHeight := len(dialogLines)
	dialogWidth := 0
	for _, line := range dialogLines {
		w := ansi.StringWidth(line)
		if w > dialogWidth {
			dialogWidth = w
		}
	}

	// Calculate center position
	x := (m.parentSize.Width - dialogWidth) / 2
	y := (m.parentSize.Height - dialogHeight) / 2

	// Ensure we don't go out of bounds
	if y < 0 {
		y = 0
	}
	if x < 0 {
		x = 0
	}

	// Overlay dialog onto background
	var result strings.Builder
	for i := 0; i < len(bgLines); i++ {
		// Check if this line should have dialog content overlaid
		dialogLineIdx := i - y
		if dialogLineIdx >= 0 && dialogLineIdx < dialogHeight {
			// This line needs dialog overlay
			bgLine := bgLines[i]
			dialogLine := dialogLines[dialogLineIdx]

			// Overlay the dialog line at position x
			overlaidLine := m.overlayLine(bgLine, dialogLine, x)
			result.WriteString(overlaidLine)
		} else {
			// No overlay needed, use background as-is
			result.WriteString(bgLines[i])
		}

		if i < len(bgLines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// overlayLine overlays foreground onto background at position x (in visual character positions)
func (m LogcatViewModel) overlayLine(background, foreground string, x int) string {
	bgWidth := ansi.StringWidth(background)
	fgWidth := ansi.StringWidth(foreground)

	// If the overlay position is beyond the background width, just return background
	if x >= bgWidth {
		return background
	}

	// Truncate background to make space for foreground, then append foreground and remainder
	// We need to work with visual positions, not byte positions
	var result strings.Builder

	// Add the part before the overlay (0 to x)
	if x > 0 {
		prefix := ansi.Truncate(background, x, "")
		result.WriteString(prefix)
	}

	// Add the foreground
	result.WriteString(foreground)

	// Add the part after the overlay
	endPos := x + fgWidth
	if endPos < bgWidth {
		// We need to skip the first 'endPos' characters and take the rest
		// Since ansi.Truncate doesn't support offset, we'll do a simpler approach:
		// Just pad if needed, as the dialog will cover the middle part
		remaining := bgWidth - endPos
		if remaining > 0 {
			result.WriteString(strings.Repeat(" ", remaining))
		}
	}

	return result.String()
}

func truncateMiddle(s string, maxWidth int) string {
	w := ansi.StringWidth(s)
	if w <= maxWidth {
		return s
	}
	// Reserve 1 char for the ellipsis
	left := (maxWidth - 1) / 2
	right := maxWidth - 1 - left

	// Take `right` visual-width chars from the end
	runes := []rune(s)
	var suffix string
	suffixW := 0
	for i := len(runes) - 1; i >= 0 && suffixW < right; i-- {
		suffixW++
		suffix = string(runes[i]) + suffix
	}

	return ansi.Truncate(s, left, "") + "…" + suffix
}

func newCommandTable(filter model.Filter, format model.Format, softWrap bool) (table.Model, map[int]bool) {
	columns := []table.Column{
		{Title: "", Width: 16},
		{Title: "", Width: 10},
		{Title: "", Width: 10},
	}

	resolveValue := func(cmd commandui.Command) string {
		switch cmd {
		case commandui.CommandPackage:
			return filter.PackageName
		case commandui.CommandTag:
			return filter.Tag
		case commandui.CommandLevel:
			lvl := string(filter.Level)
			if lvl == "" {
				lvl = "V"
			}
			return lvl
		case commandui.CommandContent:
			return filter.Text
		case commandui.CommandFormat:
			return format.Value()
		case commandui.CommandModifiers:
			mods := format.Modifiers()
			switch len(mods) {
			case 0:
				return ""
			case 1:
				return mods[0]
			default:
				return fmt.Sprintf("[%d] mods", len(mods))
			}
		case commandui.CommandToggleWrap:
			if softWrap {
				return "on"
			}
			return "off"
		default:
			return ""
		}
	}

	skipRows := make(map[int]bool)
	var rows []table.Row
	for i, group := range commandui.Commands() {
		if i > 0 {
			skipRows[len(rows)] = true
			rows = append(rows, table.Row{"", "", ""})
		}
		skipRows[len(rows)] = true
		groupName := lipgloss.NewStyle().Bold(true).Render(group.Name)
		rows = append(rows, table.Row{groupName, "", ""})
		for _, cmd := range group.Commands {
			value := truncateMiddle(resolveValue(cmd.Command), 10)
			rows = append(rows, table.Row{cmd.Name, value, cmd.Shortcut})
		}
	}

	km := table.DefaultKeyMap()
	km.GotoTop.SetEnabled(false)
	km.GotoBottom.SetEnabled(false)
	km.HalfPageUp.SetEnabled(false)
	km.HalfPageDown.SetEnabled(false)
	km.PageDown.SetEnabled(false)

	s := table.Styles{
		Header:   lipgloss.NewStyle(),
		Cell:     lipgloss.NewStyle().Padding(0, 1),
		Selected: lipgloss.NewStyle().Bold(true).Foreground(theme.FGSelected).Background(theme.BGCursor),
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(len(rows)),
		table.WithFocused(true),
		table.WithKeyMap(km),
	)
	t.SetStyles(s)
	t.SetCursor(1) // Skip the first group header

	return t, skipRows
}

func (m LogcatViewModel) renderDialog() string {
	title := lipgloss.NewStyle().Bold(true).Render("Command List")
	footer := lipgloss.NewStyle().Foreground(theme.FGHelp).Render("esc to close")
	content := title + "\n" + m.commandTable.View() + "\n" + footer
	return dialogStyle().Render(content)
}

func (m LogcatViewModel) View() string {
	if m.err != nil {
		slog.Error("Logcat view error", "error", m.err)
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	baseView := m.renderBaseView()

	if m.showCommandDialog {
		dimmedBaseView := tui.DimView(baseView)
		dialogContent := m.renderDialog()
		return m.overlayDialog(dimmedBaseView, dialogContent)
	}

	return baseView
}

func (m LogcatViewModel) headerView() string {
	var filters []string
	if !m.filter.IsEmpty() {
		if m.filter.PackageName != "" {
			filters = append(filters, fmt.Sprintf("pkg:%s", m.filter.PackageName))
		}
		if m.filter.Level != "" && m.filter.Level != model.LvlV {
			filters = append(filters, fmt.Sprintf("level:%s", m.filter.Level))
		}
		if m.filter.Tag != "" {
			filters = append(filters, fmt.Sprintf("tag:%s", m.filter.Tag))
		}
		if m.filter.Text != "" {
			filters = append(filters, fmt.Sprintf("text:%s", m.filter.Text))
		}
	}

	format := m.format.Value()
	var mods []string
	var modsStr string

	if m.format.Color {
		mods = append(mods, "color")
	}
	if m.format.Descriptive {
		mods = append(mods, "descriptive")
	}
	if m.format.Epoch {
		mods = append(mods, "epoch")
	}
	if m.format.Monotonic {
		mods = append(mods, "monotonic")
	}
	if m.format.Printable {
		mods = append(mods, "printable")
	}
	if m.format.Uid {
		mods = append(mods, "uid")
	}
	if m.format.Usec {
		mods = append(mods, "usec")
	}
	if m.format.UTC {
		mods = append(mods, "UTC")
	}
	if m.format.Year {
		mods = append(mods, "year")
	}
	if m.format.Zone {
		mods = append(mods, "zone")
	}

	if len(mods) > 0 {
		modsStr = fmt.Sprintf(" | %s", strings.Join(mods, ","))
	}

	filtersStr := ""
	if len(filters) > 0 {
		filtersStr = fmt.Sprintf("\n%s", strings.Join(filters, " | "))
	}

	deviceName := lipgloss.NewStyle().Bold(true).Render(m.device.Name)
	headerText := titleStyle.Render(fmt.Sprintf("%s | %s%s%s", deviceName, format, modsStr, filtersStr))
	width := lipgloss.Width(headerText)

	if width > m.viewport.Width {
		headerText = titleStyle.Render(fmt.Sprintf("%s | ...", deviceName))
	}

	return headerText
}

func (m LogcatViewModel) footerView() string {
	var helpText string
	if m.visualMode {
		helpText = helpTextVisual
	} else {
		helpText = helpTextNormal
	}

	help := lipgloss.NewStyle().
		Foreground(theme.FGHelp).
		Width(m.viewport.Width).
		AlignHorizontal(lipgloss.Center).
		Padding(0, 2).
		Render(helpText)

	return help
}

func Close(m *LogcatViewModel) {
	util.CloseLogcat()
	m.log = util.NewRingBuffer(maxLogLines)
}
