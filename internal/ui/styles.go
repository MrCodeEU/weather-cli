// Package ui contains shared terminal styles.
package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Title highlights application headings.
	Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
	// Subtle renders low-emphasis hints and status text.
	Subtle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	// Accent renders active tabs and primary labels.
	Accent = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	// Warn renders warning and error text.
	Warn = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	// Panel frames page content.
	Panel = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	// Selected renders selected list rows.
	Selected = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true)
)
