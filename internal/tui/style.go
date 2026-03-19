package tui

import "github.com/charmbracelet/lipgloss"

var (
	appStyle          = lipgloss.NewStyle().Margin(1, 2).Padding(1, 2).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#569CD6"))
	titleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#4EC9B0")).Bold(true).MarginBottom(1)
	focusedLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#CE9178")).Bold(true)
	blurredLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#75715E"))
	dividerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#3E3D32")).Margin(1, 0)
	errorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#F44747")).Bold(true)
	successStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#B5CEA8")).Bold(true)
	infoStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#DCDCAA"))
	responseBoxStyle  = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#808080")).Padding(0, 1)
	curlPreviewStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#C586C0")).Italic(true)
)
