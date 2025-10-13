package components

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/standardbeagle/brummer/internal/tui/messages"
	"github.com/standardbeagle/brummer/internal/tui/services"
	"github.com/standardbeagle/brummer/pkg/events"
)

// FileBrowserComponent manages file browsing dialogs and interactions
// This component migrates functionality from the old FileBrowserController
// to use the new service-based architecture
type FileBrowserComponent struct {
	BaseComponent

	// Service integration
	fileBrowserService *services.FileBrowserService

	// Component state
	activeDialogs   map[string]*DialogState
	currentDialogID string
	visibleDialogs  []string // Ordered list of visible dialog IDs

	// Event subscription
	subscriptionCh chan events.Event

	// Performance tracking
	lastActionTime time.Time
	actionCount    int

	// Rendering
	styles DialogStyles
}

// DialogState represents the state of a file browser dialog in the component
type DialogState struct {
	ID           string                           `json:"id"`
	IsVisible    bool                             `json:"is_visible"`
	CurrentPath  string                           `json:"current_path"`
	FileList     []*services.FileInfo             `json:"file_list"`
	SelectedPath string                           `json:"selected_path"`
	SelectedIndex int                            `json:"selected_index"`
	Config       services.FileBrowserDialogConfig `json:"config"`
	CreatedAt    time.Time                        `json:"created_at"`
	UpdatedAt    time.Time                        `json:"updated_at"`
	ScrollOffset int                             `json:"scroll_offset"`
	Filter       string                          `json:"filter"`
}

// DialogStyles contains styling information for file browser dialogs
type DialogStyles struct {
	Dialog      lipgloss.Style
	Title       lipgloss.Style
	Border      lipgloss.Style
	Directory   lipgloss.Style
	File        lipgloss.Style
	Selected    lipgloss.Style
	Highlight   lipgloss.Style
	Status      lipgloss.Style
	Error       lipgloss.Style
	Help        lipgloss.Style
}

// NewFileBrowserComponent creates a new file browser component
func NewFileBrowserComponent(fileBrowserService *services.FileBrowserService) *FileBrowserComponent {
	if fileBrowserService == nil {
		fileBrowserService = services.NewFileBrowserService(events.NewEventBus(), services.DefaultFileBrowserConfig())
	}

	component := &FileBrowserComponent{
		BaseComponent:      NewBaseComponent(),
		fileBrowserService: fileBrowserService,
		subscriptionCh:     make(chan events.Event, 100),
		activeDialogs:      make(map[string]*DialogState),
		visibleDialogs:     make([]string, 0),
		lastActionTime:     time.Now(),
		actionCount:        0,
		styles:             initializeDialogStyles(),
	}

	// Initialize component state from service
	component.syncFromService()

	// Subscribe to file browser service events
	component.subscribeToEvents()

	return component
}

// Init initializes the component
func (fbc *FileBrowserComponent) Init() tea.Cmd {
	// Get initial state from service
	return func() tea.Msg {
		// Get active dialogs from service
		serviceDialogs := fbc.fileBrowserService.GetActiveDialogs()
		for dialogID, serviceDialog := range serviceDialogs {
			dialogState := fbc.convertServiceDialogToState(serviceDialog)
			fbc.activeDialogs[dialogID] = dialogState
			if dialogState.IsVisible {
				fbc.visibleDialogs = append(fbc.visibleDialogs, dialogID)
			}
		}

		return messages.ComponentReadyMsg{
			ComponentID: "file_browser",
			Time:        time.Now(),
		}
	}
}

// Update handles incoming messages and file browser events
func (fbc *FileBrowserComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.FileBrowserShownMsg:
		return fbc.handleFileBrowserShown(msg)

	case messages.FileBrowserHiddenMsg:
		return fbc.handleFileBrowserHidden(msg)

	case messages.FileSelectionChangedMsg:
		return fbc.handleFileSelectionChanged(msg)

	case messages.FileBrowserNavigatedMsg:
		return fbc.handleFileBrowserNavigated(msg)

	case messages.FileBrowserFilterChangedMsg:
		return fbc.handleFileBrowserFilterChanged(msg)

	case tea.KeyMsg:
		return fbc.handleKeyMsg(msg)

	case events.Event:
		return fbc.handleServiceEvent(msg)

	default:
		return fbc, nil
	}
}

// View renders the file browser component
func (fbc *FileBrowserComponent) View() string {
	if !fbc.IsVisible() || len(fbc.visibleDialogs) == 0 {
		return ""
	}

	var dialogs []string

	// Render each visible dialog
	for _, dialogID := range fbc.visibleDialogs {
		if dialog, exists := fbc.activeDialogs[dialogID]; exists && dialog.IsVisible {
			rendered := fbc.renderDialog(dialog)
			dialogs = append(dialogs, rendered)
		}
	}

	if len(dialogs) == 0 {
		return ""
	}

	// Stack dialogs (for now, show the topmost one)
	// In a more complex implementation, we might handle multiple dialogs
	return dialogs[len(dialogs)-1]
}

// handleFileBrowserShown processes dialog display
func (fbc *FileBrowserComponent) handleFileBrowserShown(msg messages.FileBrowserShownMsg) (tea.Model, tea.Cmd) {
	dialogState := &DialogState{
		ID:           msg.DialogID,
		IsVisible:    true,
		CurrentPath:  msg.InitialPath,
		FileList:     []*services.FileInfo{},
		SelectedPath: "",
		SelectedIndex: -1,
		Config: services.FileBrowserDialogConfig{
			InitialPath: msg.InitialPath,
			CallbackID:  msg.CallbackID,
		},
		CreatedAt:    msg.Time,
		UpdatedAt:    msg.Time,
		ScrollOffset: 0,
		Filter:       "",
	}

	// Get initial file listing
	listing := fbc.fileBrowserService.ListDirectoryPath(msg.InitialPath)
	if listing.Error == "" {
		dialogState.FileList = listing.Files
	}

	// Add to active dialogs
	fbc.activeDialogs[msg.DialogID] = dialogState

	// Add to visible dialogs
	fbc.visibleDialogs = append(fbc.visibleDialogs, msg.DialogID)

	// Set as current dialog if no current dialog
	if fbc.currentDialogID == "" {
		fbc.currentDialogID = msg.DialogID
	}

	// Update tracking
	fbc.lastActionTime = msg.Time
	fbc.actionCount++

	return fbc, func() tea.Msg {
		return messages.FileBrowserShownMsg{
			DialogID:    msg.DialogID,
			InitialPath: msg.InitialPath,
			CallbackID:  msg.CallbackID,
			Time:        msg.Time,
		}
	}
}

// handleFileBrowserHidden processes dialog hiding
func (fbc *FileBrowserComponent) handleFileBrowserHidden(msg messages.FileBrowserHiddenMsg) (tea.Model, tea.Cmd) {
	if dialog, exists := fbc.activeDialogs[msg.DialogID]; exists {
		dialog.IsVisible = false
		dialog.UpdatedAt = msg.Time

		// Remove from visible dialogs
		for i, id := range fbc.visibleDialogs {
			if id == msg.DialogID {
				fbc.visibleDialogs = append(fbc.visibleDialogs[:i], fbc.visibleDialogs[i+1:]...)
				break
			}
		}

		// Update current dialog if needed
		if fbc.currentDialogID == msg.DialogID {
			if len(fbc.visibleDialogs) > 0 {
				fbc.currentDialogID = fbc.visibleDialogs[len(fbc.visibleDialogs)-1]
			} else {
				fbc.currentDialogID = ""
			}
		}
	}

	return fbc, func() tea.Msg {
		return messages.FileBrowserHiddenMsg{
			DialogID: msg.DialogID,
			Trigger:  msg.Trigger,
			Time:     msg.Time,
		}
	}
}

// handleFileSelectionChanged processes file selection changes
func (fbc *FileBrowserComponent) handleFileSelectionChanged(msg messages.FileSelectionChangedMsg) (tea.Model, tea.Cmd) {
	if dialog, exists := fbc.activeDialogs[msg.DialogID]; exists {
		dialog.SelectedPath = msg.SelectedPath
		dialog.SelectedIndex = msg.SelectedIndex
		dialog.UpdatedAt = msg.Time

		// Update tracking
		fbc.lastActionTime = msg.Time
		fbc.actionCount++
	}

	return fbc, func() tea.Msg {
		return messages.FileSelectionChangedMsg{
			DialogID:      msg.DialogID,
			SelectedPath:  msg.SelectedPath,
			SelectedIndex: msg.SelectedIndex,
			FileInfo:      msg.FileInfo,
			Time:          msg.Time,
		}
	}
}

// handleFileBrowserNavigated processes directory navigation
func (fbc *FileBrowserComponent) handleFileBrowserNavigated(msg messages.FileBrowserNavigatedMsg) (tea.Model, tea.Cmd) {
	if dialog, exists := fbc.activeDialogs[msg.DialogID]; exists {
		dialog.CurrentPath = msg.NewPath
		dialog.SelectedPath = ""
		dialog.SelectedIndex = -1
		dialog.ScrollOffset = 0
		dialog.UpdatedAt = msg.Time

		// Update file list
		if fileList, ok := msg.FileList.([]*services.FileInfo); ok {
			dialog.FileList = fileList
		}

		// Update tracking
		fbc.lastActionTime = msg.Time
		fbc.actionCount++
	}

	return fbc, func() tea.Msg {
		return messages.FileBrowserNavigatedMsg{
			DialogID: msg.DialogID,
			OldPath:  msg.OldPath,
			NewPath:  msg.NewPath,
			FileList: msg.FileList,
			Time:     msg.Time,
		}
	}
}

// handleFileBrowserFilterChanged processes filter changes
func (fbc *FileBrowserComponent) handleFileBrowserFilterChanged(msg messages.FileBrowserFilterChangedMsg) (tea.Model, tea.Cmd) {
	if dialog, exists := fbc.activeDialogs[msg.DialogID]; exists {
		dialog.Filter = msg.Filter
		dialog.ScrollOffset = 0
		dialog.UpdatedAt = msg.Time

		// Apply filter to file list
		dialog.FileList = fbc.applyFilter(dialog.FileList, msg.Filter)

		// Update tracking
		fbc.lastActionTime = msg.Time
		fbc.actionCount++
	}

	return fbc, func() tea.Msg {
		return messages.FileBrowserFilterChangedMsg{
			DialogID:   msg.DialogID,
			FilterType: msg.FilterType,
			Filter:     msg.Filter,
			Time:       msg.Time,
		}
	}
}

// handleKeyMsg processes keyboard input for file browser operations
func (fbc *FileBrowserComponent) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if fbc.currentDialogID == "" {
		return fbc, nil
	}

	dialog, exists := fbc.activeDialogs[fbc.currentDialogID]
	if !exists || !dialog.IsVisible {
		return fbc, nil
	}

	switch msg.Type {
	case tea.KeyEscape:
		// Close dialog
		return fbc.hideDialog(fbc.currentDialogID, "escape")

	case tea.KeyEnter:
		// Select file or navigate into directory
		if dialog.SelectedIndex >= 0 && dialog.SelectedIndex < len(dialog.FileList) {
			selectedFile := dialog.FileList[dialog.SelectedIndex]
			if selectedFile.IsDir {
				// Navigate into directory
				return fbc.navigateToDirectory(fbc.currentDialogID, selectedFile.Path)
			} else {
				// Select file (call callback)
				return fbc.selectFile(fbc.currentDialogID, selectedFile.Path, dialog.SelectedIndex)
			}
		}

	case tea.KeyUp:
		// Move selection up
		return fbc.moveSelectionUp(fbc.currentDialogID)

	case tea.KeyDown:
		// Move selection down
		return fbc.moveSelectionDown(fbc.currentDialogID)

	case tea.KeyLeft:
		// Navigate to parent directory
		return fbc.navigateToParentDirectory(fbc.currentDialogID)

	case tea.KeyRight:
		// Navigate into directory (if directory is selected)
		if dialog.SelectedIndex >= 0 && dialog.SelectedIndex < len(dialog.FileList) {
			selectedFile := dialog.FileList[dialog.SelectedIndex]
			if selectedFile.IsDir {
				return fbc.navigateToDirectory(fbc.currentDialogID, selectedFile.Path)
			}
		}

	default:
		// Check for specific key strings
		keyStr := msg.String()
		if keyStr == "q" || keyStr == "ctrl+c" {
			return fbc.hideDialog(fbc.currentDialogID, "cancel")
		}

		// Handle typing for filter
		if len(msg.Runes) > 0 {
			return fbc.handleTyping(fbc.currentDialogID, string(msg.Runes))
		}
	}

	return fbc, nil
}

// handleServiceEvent handles events from the file browser service
func (fbc *FileBrowserComponent) handleServiceEvent(event events.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case "file_browser.dialog_shown":
		return fbc.handleDialogShownEvent(event)
	case "file_browser.dialog_hidden":
		return fbc.handleDialogHiddenEvent(event)
	case "file_browser.selection_changed":
		return fbc.handleSelectionChangedEvent(event)
	case "file_browser.navigated":
		return fbc.handleNavigatedEvent(event)
	default:
		return fbc, nil
	}
}

// handleDialogShownEvent handles dialog shown events from service
func (fbc *FileBrowserComponent) handleDialogShownEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data

	dialogID, _ := data["dialog_id"].(string)
	initialPath, _ := data["initial_path"].(string)
	callbackID, _ := data["callback_id"].(string)

	dialogState := &DialogState{
		ID:           dialogID,
		IsVisible:    true,
		CurrentPath:  initialPath,
		FileList:     []*services.FileInfo{},
		SelectedPath: "",
		SelectedIndex: -1,
		Config: services.FileBrowserDialogConfig{
			InitialPath: initialPath,
			CallbackID:  callbackID,
		},
		CreatedAt:    event.Timestamp,
		UpdatedAt:    event.Timestamp,
		ScrollOffset: 0,
		Filter:       "",
	}

	// Get initial file listing
	listing := fbc.fileBrowserService.ListDirectoryPath(initialPath)
	if listing.Error == "" {
		dialogState.FileList = listing.Files
	}

	// Add to active dialogs
	fbc.activeDialogs[dialogID] = dialogState

	// Add to visible dialogs
	fbc.visibleDialogs = append(fbc.visibleDialogs, dialogID)

	// Set as current dialog if no current dialog
	if fbc.currentDialogID == "" {
		fbc.currentDialogID = dialogID
	}

	return fbc, func() tea.Msg {
		return messages.FileBrowserShownMsg{
			DialogID:    dialogID,
			InitialPath: initialPath,
			CallbackID:  callbackID,
			Time:        event.Timestamp,
		}
	}
}

// handleDialogHiddenEvent handles dialog hidden events from service
func (fbc *FileBrowserComponent) handleDialogHiddenEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data

	dialogID, _ := data["dialog_id"].(string)
	trigger, _ := data["trigger"].(string)

	if dialog, exists := fbc.activeDialogs[dialogID]; exists {
		dialog.IsVisible = false
		dialog.UpdatedAt = event.Timestamp

		// Remove from visible dialogs
		for i, id := range fbc.visibleDialogs {
			if id == dialogID {
				fbc.visibleDialogs = append(fbc.visibleDialogs[:i], fbc.visibleDialogs[i+1:]...)
				break
			}
		}

		// Update current dialog if needed
		if fbc.currentDialogID == dialogID {
			if len(fbc.visibleDialogs) > 0 {
				fbc.currentDialogID = fbc.visibleDialogs[len(fbc.visibleDialogs)-1]
			} else {
				fbc.currentDialogID = ""
			}
		}
	}

	return fbc, func() tea.Msg {
		return messages.FileBrowserHiddenMsg{
			DialogID: dialogID,
			Trigger:  trigger,
			Time:     event.Timestamp,
		}
	}
}

// handleSelectionChangedEvent handles selection changed events from service
func (fbc *FileBrowserComponent) handleSelectionChangedEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data

	dialogID, _ := data["dialog_id"].(string)
	selectedPath, _ := data["selected_path"].(string)
	selectedIndex, _ := data["selected_index"].(int)

	if dialog, exists := fbc.activeDialogs[dialogID]; exists {
		dialog.SelectedPath = selectedPath
		dialog.SelectedIndex = selectedIndex
		dialog.UpdatedAt = event.Timestamp
	}

	return fbc, func() tea.Msg {
		return messages.FileSelectionChangedMsg{
			DialogID:      dialogID,
			SelectedPath:  selectedPath,
			SelectedIndex: selectedIndex,
			FileInfo:      data["file_info"],
			Time:          event.Timestamp,
		}
	}
}

// handleNavigatedEvent handles navigation events from service
func (fbc *FileBrowserComponent) handleNavigatedEvent(event events.Event) (tea.Model, tea.Cmd) {
	data := event.Data

	dialogID, _ := data["dialog_id"].(string)
	newPath, _ := data["new_path"].(string)

	if dialog, exists := fbc.activeDialogs[dialogID]; exists {
		dialog.CurrentPath = newPath
		dialog.SelectedPath = ""
		dialog.SelectedIndex = -1
		dialog.ScrollOffset = 0
		dialog.UpdatedAt = event.Timestamp

		// Update file list
		if fileList, ok := data["file_list"].([]*services.FileInfo); ok {
			dialog.FileList = fileList
		}
	}

	return fbc, func() tea.Msg {
		return messages.FileBrowserNavigatedMsg{
			DialogID: dialogID,
			OldPath:  "", // Not provided in service event
			NewPath:  newPath,
			FileList: data["file_list"],
			Time:     event.Timestamp,
		}
	}
}

// syncFromService synchronizes component state from the service
func (fbc *FileBrowserComponent) syncFromService() {
	// Get active dialogs from service
	serviceDialogs := fbc.fileBrowserService.GetActiveDialogs()
	for dialogID, serviceDialog := range serviceDialogs {
		dialogState := fbc.convertServiceDialogToState(serviceDialog)
		fbc.activeDialogs[dialogID] = dialogState
		if dialogState.IsVisible {
			fbc.visibleDialogs = append(fbc.visibleDialogs, dialogID)
		}
	}
}

// convertServiceDialogToState converts a service dialog to component state
func (fbc *FileBrowserComponent) convertServiceDialogToState(serviceDialog *services.FileBrowserDialog) *DialogState {
	return &DialogState{
		ID:            serviceDialog.ID,
		IsVisible:     serviceDialog.IsVisible,
		CurrentPath:   serviceDialog.CurrentPath,
		FileList:      serviceDialog.FileList,
		SelectedPath:  serviceDialog.SelectedPath,
		SelectedIndex: serviceDialog.SelectedIndex,
		Config:        serviceDialog.Config,
		CreatedAt:     serviceDialog.CreatedAt,
		UpdatedAt:     serviceDialog.UpdatedAt,
		ScrollOffset:  0,
		Filter:        "",
	}
}

// subscribeToEvents subscribes to file browser service events
func (fbc *FileBrowserComponent) subscribeToEvents() {
	// Subscribe to file browser service events through the event bus
	// This would typically use the event bus subscription mechanism
	// For now, we'll simulate this with direct channel subscription
}

// Rendering methods

func (fbc *FileBrowserComponent) renderDialog(dialog *DialogState) string {
	// Create dialog content
	content := fbc.styles.Dialog.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			fbc.renderTitle(dialog),
			fbc.renderPath(dialog),
			fbc.renderFileList(dialog),
			fbc.renderStatus(dialog),
			fbc.renderHelp(),
		),
	)

	// Center the dialog
	return lipgloss.Place(
		fbc.Width(), fbc.Height(),
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

func (fbc *FileBrowserComponent) renderTitle(dialog *DialogState) string {
	title := "File Browser"
	if dialog.Config.Title != "" {
		title = dialog.Config.Title
	}

	return fbc.styles.Title.Render(title)
}

func (fbc *FileBrowserComponent) renderPath(dialog *DialogState) string {
	return fbc.styles.Status.Render("Path: " + dialog.CurrentPath)
}

func (fbc *FileBrowserComponent) renderFileList(dialog *DialogState) string {
	if len(dialog.FileList) == 0 {
		return fbc.styles.Error.Render("No files found")
	}

	var files []string
	maxHeight := 15 // Limit height of file list

	// Apply filter if set
	filteredFiles := dialog.FileList
	if dialog.Filter != "" {
		filteredFiles = fbc.applyFilter(dialog.FileList, dialog.Filter)
	}

	// Determine visible range
	startIndex := dialog.ScrollOffset
	endIndex := min(startIndex+maxHeight, len(filteredFiles))

	if endIndex > len(filteredFiles) {
		endIndex = len(filteredFiles)
		if startIndex > endIndex-maxHeight {
			startIndex = max(0, endIndex-maxHeight)
		}
	}

	// Render visible files
	for i := startIndex; i < endIndex; i++ {
		if i >= len(filteredFiles) {
			break
		}

		file := filteredFiles[i]
		isSelected := i == dialog.SelectedIndex

		var style lipgloss.Style
		if isSelected {
			style = fbc.styles.Selected
		} else if file.IsDir {
			style = fbc.styles.Directory
		} else {
			style = fbc.styles.File
		}

		// Format file entry
		icon := "📄"
		if file.IsDir {
			icon = "📁"
		}

		fileText := icon + " " + file.Name
		if file.IsDir {
			fileText += "/"
		}

		files = append(files, style.Render(fileText))
	}

	return lipgloss.JoinVertical(lipgloss.Left, files...)
}

func (fbc *FileBrowserComponent) renderStatus(dialog *DialogState) string {
	var statusText string

	if dialog.SelectedIndex >= 0 && dialog.SelectedIndex < len(dialog.FileList) {
		selectedFile := dialog.FileList[dialog.SelectedIndex]
		if selectedFile.IsDir {
			statusText = "Directory: " + selectedFile.Name
		} else {
			statusText = "File: " + selectedFile.Name + " (" + fbc.formatFileSize(selectedFile.Size) + ")"
		}
	} else {
		statusText = "Use arrow keys to navigate, Enter to select"
	}

	return fbc.styles.Status.Render(statusText)
}

func (fbc *FileBrowserComponent) renderHelp() string {
	help := "↑↓ Navigate  Enter Select  ← Parent  Esc Close"
	return fbc.styles.Help.Render(help)
}

// Navigation helpers

func (fbc *FileBrowserComponent) moveSelectionUp(dialogID string) (tea.Model, tea.Cmd) {
	dialog, exists := fbc.activeDialogs[dialogID]
	if !exists {
		return fbc, nil
	}

	if dialog.SelectedIndex > 0 {
		dialog.SelectedIndex--
		if dialog.SelectedIndex < dialog.ScrollOffset {
			dialog.ScrollOffset = dialog.SelectedIndex
		}
		dialog.UpdatedAt = time.Now()
	}

	return fbc, nil
}

func (fbc *FileBrowserComponent) moveSelectionDown(dialogID string) (tea.Model, tea.Cmd) {
	dialog, exists := fbc.activeDialogs[dialogID]
	if !exists {
		return fbc, nil
	}

	filteredFiles := dialog.FileList
	if dialog.Filter != "" {
		filteredFiles = fbc.applyFilter(dialog.FileList, dialog.Filter)
	}

	if dialog.SelectedIndex < len(filteredFiles)-1 {
		dialog.SelectedIndex++
		maxVisible := 15
		if dialog.SelectedIndex >= dialog.ScrollOffset+maxVisible {
			dialog.ScrollOffset = dialog.SelectedIndex - maxVisible + 1
		}
		dialog.UpdatedAt = time.Now()
	}

	return fbc, nil
}

func (fbc *FileBrowserComponent) navigateToDirectory(dialogID, path string) (tea.Model, tea.Cmd) {
	return fbc, func() tea.Msg {
		ctx := context.Background()
		err := fbc.fileBrowserService.NavigateToDirectory(ctx, dialogID, path)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "FileBrowserComponent",
				Message: "Failed to navigate to directory: " + err.Error(),
				Time:    time.Now(),
			}
		}

		return messages.FileBrowserNavigatedMsg{
			DialogID: dialogID,
			OldPath:  "", // Will be filled by service
			NewPath:  path,
			FileList: nil, // Will be filled by service
			Time:     time.Now(),
		}
	}
}

func (fbc *FileBrowserComponent) navigateToParentDirectory(dialogID string) (tea.Model, tea.Cmd) {
	dialog, exists := fbc.activeDialogs[dialogID]
	if !exists {
		return fbc, nil
	}

	parentPath := filepath.Dir(dialog.CurrentPath)
	if parentPath != dialog.CurrentPath {
		return fbc.navigateToDirectory(dialogID, parentPath)
	}

	return fbc, nil
}

func (fbc *FileBrowserComponent) selectFile(dialogID, path string, index int) (tea.Model, tea.Cmd) {
	return fbc, func() tea.Msg {
		ctx := context.Background()
		err := fbc.fileBrowserService.SelectFile(ctx, dialogID, path, index)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "FileBrowserComponent",
				Message: "Failed to select file: " + err.Error(),
				Time:    time.Now(),
			}
		}

		// Hide dialog after selection
		fbc.fileBrowserService.HideFileBrowser(ctx, dialogID)

		return messages.FileSelectionChangedMsg{
			DialogID:      dialogID,
			SelectedPath:  path,
			SelectedIndex: index,
			FileInfo:      nil, // Will be filled by service
			Time:          time.Now(),
		}
	}
}

func (fbc *FileBrowserComponent) hideDialog(dialogID, trigger string) (tea.Model, tea.Cmd) {
	return fbc, func() tea.Msg {
		ctx := context.Background()
		err := fbc.fileBrowserService.HideFileBrowser(ctx, dialogID)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "FileBrowserComponent",
				Message: "Failed to hide dialog: " + err.Error(),
				Time:    time.Now(),
			}
		}

		return messages.FileBrowserHiddenMsg{
			DialogID: dialogID,
			Trigger:  trigger,
			Time:     time.Now(),
		}
	}
}

func (fbc *FileBrowserComponent) handleTyping(dialogID, char string) (tea.Model, tea.Cmd) {
	dialog, exists := fbc.activeDialogs[dialogID]
	if !exists {
		return fbc, nil
	}

	dialog.Filter += char
	dialog.UpdatedAt = time.Now()

	return fbc, func() tea.Msg {
		return messages.FileBrowserFilterChangedMsg{
			DialogID:   dialogID,
			FilterType: "text",
			Filter:     dialog.Filter,
			Time:       time.Now(),
		}
	}
}

// Helper methods

func (fbc *FileBrowserComponent) applyFilter(files []*services.FileInfo, filter string) []*services.FileInfo {
	if filter == "" {
		return files
	}

	var filtered []*services.FileInfo
	filterLower := strings.ToLower(filter)

	for _, file := range files {
		if strings.Contains(strings.ToLower(file.Name), filterLower) {
			filtered = append(filtered, file)
		}
	}

	return filtered
}

func (fbc *FileBrowserComponent) formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

// Public API methods

// ShowFileBrowser displays a file browser dialog
func (fbc *FileBrowserComponent) ShowFileBrowser(config services.FileBrowserDialogConfig) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		dialogID, err := fbc.fileBrowserService.ShowFileBrowser(ctx, config)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "FileBrowserComponent",
				Message: "Failed to show file browser: " + err.Error(),
				Time:    time.Now(),
			}
		}

		return messages.FileBrowserShownMsg{
			DialogID:    dialogID,
			InitialPath: config.InitialPath,
			CallbackID:  config.CallbackID,
			Time:        time.Now(),
		}
	}
}

// HideFileBrowser hides a file browser dialog
func (fbc *FileBrowserComponent) HideFileBrowser(dialogID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := fbc.fileBrowserService.HideFileBrowser(ctx, dialogID)
		if err != nil {
			return messages.ErrorMessage{
				Source:  "FileBrowserComponent",
				Message: "Failed to hide file browser: " + err.Error(),
				Time:    time.Now(),
			}
		}

		return messages.FileBrowserHiddenMsg{
			DialogID: dialogID,
			Trigger:  "manual",
			Time:     time.Now(),
		}
	}
}

// GetActiveDialogs returns current active dialogs
func (fbc *FileBrowserComponent) GetActiveDialogs() map[string]*DialogState {
	result := make(map[string]*DialogState)
	for id, dialog := range fbc.activeDialogs {
		result[id] = dialog
	}
	return result
}

// GetVisibleDialogs returns visible dialog IDs
func (fbc *FileBrowserComponent) GetVisibleDialogs() []string {
	result := make([]string, len(fbc.visibleDialogs))
	copy(result, fbc.visibleDialogs)
	return result
}

// GetCurrentDialogID returns the current active dialog ID
func (fbc *FileBrowserComponent) GetCurrentDialogID() string {
	return fbc.currentDialogID
}

// GetDialogStats returns dialog processing statistics
func (fbc *FileBrowserComponent) GetDialogStats() (int, time.Time) {
	return fbc.actionCount, fbc.lastActionTime
}

// Focus gives keyboard focus to the file browser component
func (fbc *FileBrowserComponent) Focus() tea.Cmd {
	fbc.BaseComponent.Focus()
	return nil
}

// Blur removes keyboard focus from the file browser component
func (fbc *FileBrowserComponent) Blur() tea.Cmd {
	fbc.BaseComponent.Blur()
	return nil
}

// Cleanup performs cleanup when component is destroyed
func (fbc *FileBrowserComponent) Cleanup() error {
	return fbc.fileBrowserService.Stop()
}

// Utility functions

func initializeDialogStyles() DialogStyles {
	return DialogStyles{
		Dialog: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(1, 2).
			Background(lipgloss.Color("235")),

		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Bold(true).
			MarginBottom(1),

		Border: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")),

		Directory: lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")). // Blue
			Bold(true),

		File: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")), // White

		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).   // Black
			Background(lipgloss.Color("12")). // Blue
			Bold(true),

		Highlight: lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Yellow
			Bold(true),

		Status: lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")). // Silver
			MarginTop(1),

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // Red
			Italic(true),

		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")). // Gray
			Italic(true).
			MarginTop(1),
	}
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