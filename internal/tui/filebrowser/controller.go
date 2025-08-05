package filebrowser

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FileItem represents a file or directory in the file browser
type FileItem struct {
	Name  string
	IsDir bool
	Path  string
}

// fileBrowserItem implements list.Item for file browser
type fileBrowserItem struct {
	name  string
	path  string
	isDir bool
}

func (i fileBrowserItem) FilterValue() string { return i.name }
func (i fileBrowserItem) Title() string {
	if i.isDir {
		return "📁 " + i.name
	}
	return "📄 " + i.name
}
func (i fileBrowserItem) Description() string {
	if i.isDir {
		return "Directory"
	}
	return "File"
}

// Controller manages file browser state and operations
type Controller struct {
	showing     bool
	currentPath string
	fileList    []FileItem
	listModel   list.Model // For list navigation

	// Callbacks
	onFileSelect func(path string)
	onCancel     func()
}

// NewController creates a new file browser controller
func NewController() *Controller {
	homeDir, _ := os.UserHomeDir()

	listModel := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	listModel.Title = "File Browser"
	listModel.SetShowStatusBar(false)

	return &Controller{
		showing:     false,
		currentPath: homeDir,
		fileList:    []FileItem{},
		listModel:   listModel,
	}
}

// SetCallbacks sets the callback functions
func (c *Controller) SetCallbacks(onFileSelect func(string), onCancel func()) {
	c.onFileSelect = onFileSelect
	c.onCancel = onCancel
}

// Show displays the file browser
func (c *Controller) Show() {
	c.showing = true
	c.LoadFileList()
}

// Hide hides the file browser
func (c *Controller) Hide() {
	c.showing = false
}

// IsShowing returns whether the file browser is visible
func (c *Controller) IsShowing() bool {
	return c.showing
}

// GetCurrentPath returns the current directory path
func (c *Controller) GetCurrentPath() string {
	return c.currentPath
}

// GetFileList returns the current file list
func (c *Controller) GetFileList() []FileItem {
	return c.fileList
}

// GetListModel returns the list model for external manipulation
func (c *Controller) GetListModel() *list.Model {
	return &c.listModel
}

// GetSelectedIndex returns the currently selected index in the list
func (c *Controller) GetSelectedIndex() int {
	return c.listModel.Index()
}

// SetListSize updates the size of the list model
func (c *Controller) SetListSize(width, height int) {
	c.listModel.SetSize(width, height)
}

// LoadFileList loads files and directories for the current path
func (c *Controller) LoadFileList() {
	files, err := os.ReadDir(c.currentPath)
	if err != nil {
		return
	}

	c.fileList = []FileItem{}

	// Add parent directory option if not at root
	if c.currentPath != "/" {
		c.fileList = append(c.fileList, FileItem{
			Name:  "..",
			IsDir: true,
			Path:  filepath.Dir(c.currentPath),
		})
	}

	// Add directories first
	var dirs []FileItem
	var regularFiles []FileItem

	for _, file := range files {
		// Skip hidden files
		if strings.HasPrefix(file.Name(), ".") {
			continue
		}

		fullPath := filepath.Join(c.currentPath, file.Name())
		item := FileItem{
			Name:  file.Name(),
			IsDir: file.IsDir(),
			Path:  fullPath,
		}

		if file.IsDir() {
			dirs = append(dirs, item)
		} else {
			regularFiles = append(regularFiles, item)
		}
	}

	// Sort directories and files separately
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(regularFiles, func(i, j int) bool {
		return strings.ToLower(regularFiles[i].Name) < strings.ToLower(regularFiles[j].Name)
	})

	// Combine the lists
	c.fileList = append(c.fileList, dirs...)
	c.fileList = append(c.fileList, regularFiles...)

	// Update the list model
	c.updateListModel()
}

// updateListModel updates the list model with current file list
func (c *Controller) updateListModel() {
	items := []list.Item{}
	for _, file := range c.fileList {
		items = append(items, fileBrowserItem{
			name:  file.Name,
			path:  file.Path,
			isDir: file.IsDir,
		})
	}
	c.listModel.SetItems(items)
}

// HandleInput processes keyboard input for the file browser
func (c *Controller) HandleInput(msg tea.KeyMsg) tea.Cmd {
	selectedIndex := c.listModel.Index()

	switch msg.String() {
	case "esc":
		c.Hide()
		if c.onCancel != nil {
			c.onCancel()
		}
		return nil

	case "enter":
		if selectedIndex >= 0 && selectedIndex < len(c.fileList) {
			selected := c.fileList[selectedIndex]
			if selected.IsDir {
				c.currentPath = selected.Path
				c.LoadFileList()
			} else if c.onFileSelect != nil {
				c.Hide()
				c.onFileSelect(selected.Path)
			}
		}
		return nil
	}

	return nil
}

// Render renders the file browser UI
func (c *Controller) Render(width, height int) string {
	if !c.showing {
		return ""
	}

	selectedIndex := c.listModel.Index()

	// Create styles
	overlayStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Background(lipgloss.Color("0"))

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(width - 20).
		MaxHeight(height - 10).
		Background(lipgloss.Color("235"))

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		MarginBottom(1)

	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginBottom(1)

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230"))

	dirStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33"))

	// Build content
	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render("Select MCP Configuration File"))
	content.WriteString("\n")

	// Current path
	content.WriteString(pathStyle.Render(fmt.Sprintf("📁 %s", c.currentPath)))
	content.WriteString("\n\n")

	// File list
	for i, item := range c.fileList {
		line := item.Name
		if item.IsDir {
			line = "📁 " + line
		} else {
			line = "📄 " + line
		}

		if i == selectedIndex {
			content.WriteString(selectedStyle.Render(line))
		} else if item.IsDir {
			content.WriteString(dirStyle.Render(line))
		} else {
			content.WriteString(line)
		}
		content.WriteString("\n")
	}

	// Help text
	content.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	content.WriteString(helpStyle.Render("↑/↓: Navigate • Enter: Select • Esc: Cancel"))

	// Create dialog
	dialog := dialogStyle.Render(content.String())

	// Center the dialog
	dialogLines := strings.Split(dialog, "\n")
	dialogHeight := len(dialogLines)
	topPadding := (height - dialogHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	// Build final overlay
	var overlay strings.Builder
	for i := 0; i < topPadding; i++ {
		overlay.WriteString("\n")
	}

	leftPadding := (width - lipgloss.Width(dialogLines[0])) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	for _, line := range dialogLines {
		overlay.WriteString(strings.Repeat(" ", leftPadding))
		overlay.WriteString(line)
		overlay.WriteString("\n")
	}

	return overlayStyle.Render(overlay.String())
}
