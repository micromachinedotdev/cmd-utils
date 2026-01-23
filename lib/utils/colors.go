package utils

import (
	"log"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var Success = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
var Fail = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
var Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff9300")) // yellow
var Info = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))         // blue
var Muted = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))       // gray
var Gray = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))          // gray
var Magenta = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
var Cyan = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
var Default = lipgloss.NewStyle()

func LogWithColor(color lipgloss.Style, text string) {
	log.SetFlags(0)
	log.Printf("%s %s\n", Gray.Render(time.Now().Format(time.TimeOnly)), color.Render(text))
}
