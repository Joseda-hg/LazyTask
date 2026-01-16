package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Joseda-hg/lazytask/internal/db"
	"github.com/Joseda-hg/lazytask/internal/model"
	goerrors "github.com/go-errors/errors"
	"github.com/jesseduffield/gocui"
)

const (
	viewHeader      = "header"
	viewFooter      = "footer"
	viewPending     = "pending"
	viewDone        = "done"
	viewTags        = "tags"
	viewHighlighted = "highlighted"
	viewEventually  = "eventually"
	viewHistory     = "history"
	viewSearch      = "search"
	viewForm        = "form"
	viewHelp        = "help"
	viewTagCreate   = "tagCreate"
)

type UI struct {
	store *db.Store
	gui   *gocui.Gui

	filter     model.Filter
	activeView *model.View

	pending            []model.Task
	pendingDepth       map[int64]int
	pendingHasChildren map[int64]bool

	done            []model.Task
	doneDepth       map[int64]int
	doneHasChildren map[int64]bool

	eventually            []model.Task
	eventuallyDepth       map[int64]int
	eventuallyHasChildren map[int64]bool

	tags    []tagCountEntry
	doing   []model.Task
	history []model.HistoryEntry

	collapsed map[int64]bool

	selectedPending    int
	selectedDone       int
	selectedEventually int
	selectedTags       int
	selectedHistory    int
	focus              string

	activeTags      map[string]struct{}
	form            *formState
	formEditor      *formEditor
	formTagIndex    int
	searchActive    bool
	helpActive      bool
	tagCreateActive bool
	tagCreateValue  string
	status          string
}

type formState struct {
	taskID       int64
	parentTaskID *int64
	fields       []formField
	index        int
}

type formEditor struct {
	ui *UI
}

func Run(store *db.Store) error {
	gui, err := gocui.NewGui(gocui.NewGuiOpts{OutputMode: gocui.OutputNormal})
	if err != nil {
		return err
	}
	defer gui.Close()

	ui := &UI{
		store:      store,
		gui:        gui,
		focus:      viewPending,
		activeTags: make(map[string]struct{}),
		collapsed:  make(map[int64]bool),
	}
	gui.Mouse = true
	ui.formEditor = &formEditor{ui: ui}

	gui.SetManagerFunc(ui.layout)
	if err := ui.bindKeys(gui); err != nil {
		return err
	}
	if err := ui.loadTasks(); err != nil {
		return err
	}

	if err := gui.MainLoop(); err != nil && err != gocui.ErrQuit {
		return err
	}

	return nil
}

func (u *UI) bindKeys(gui *gocui.Gui) error {
	if err := gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, u.quit); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", 'q', gocui.ModNone, u.quit); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", 'r', gocui.ModNone, u.reload); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", 'g', gocui.ModNone, u.clearFilters); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", 'a', gocui.ModNone, u.addTask); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", 's', gocui.ModNone, u.addSubtask); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", 'e', gocui.ModNone, u.editTask); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", 'd', gocui.ModNone, u.deleteTask); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", 'c', gocui.ModNone, u.toggleDoing); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", 'x', gocui.ModNone, u.toggleDone); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", 'v', gocui.ModNone, u.toggleEventually); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", 'h', gocui.ModNone, u.refreshHistory); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", '/', gocui.ModNone, u.startSearch); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", '?', gocui.ModNone, u.toggleHelp); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", gocui.KeyTab, gocui.ModNone, u.switchFocus); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", '1', gocui.ModNone, u.focusPending); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", '2', gocui.ModNone, u.focusDone); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", '3', gocui.ModNone, u.focusTags); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", '4', gocui.ModNone, u.focusHighlighted); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", '5', gocui.ModNone, u.focusEventually); err != nil {
		return err
	}
	if err := gui.SetKeybinding("", '6', gocui.ModNone, u.focusHistory); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewPending, gocui.KeyArrowDown, gocui.ModNone, u.moveDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewPending, 'j', gocui.ModNone, u.moveDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewPending, gocui.KeyArrowUp, gocui.ModNone, u.moveUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewPending, 'k', gocui.ModNone, u.moveUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewPending, gocui.KeyEnter, gocui.ModNone, u.toggleCollapse); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewDone, gocui.KeyArrowDown, gocui.ModNone, u.moveDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewDone, 'j', gocui.ModNone, u.moveDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewDone, gocui.KeyArrowUp, gocui.ModNone, u.moveUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewDone, 'k', gocui.ModNone, u.moveUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewDone, gocui.KeyEnter, gocui.ModNone, u.toggleCollapse); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewEventually, gocui.KeyArrowDown, gocui.ModNone, u.moveDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewEventually, 'j', gocui.ModNone, u.moveDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewEventually, gocui.KeyArrowUp, gocui.ModNone, u.moveUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewEventually, 'k', gocui.ModNone, u.moveUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewEventually, gocui.KeyEnter, gocui.ModNone, u.toggleCollapse); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewTags, gocui.KeyArrowDown, gocui.ModNone, u.moveDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewTags, 'j', gocui.ModNone, u.moveDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewTags, gocui.KeyArrowUp, gocui.ModNone, u.moveUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewTags, 'k', gocui.ModNone, u.moveUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewTags, gocui.KeySpace, gocui.ModNone, u.toggleTagFilter); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewTags, gocui.KeyEnter, gocui.ModNone, u.toggleTagFilter); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewTags, 'a', gocui.ModNone, u.openTagCreate); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewTags, 'd', gocui.ModNone, u.deleteTag); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewHistory, gocui.KeyArrowDown, gocui.ModNone, u.moveDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewHistory, 'j', gocui.ModNone, u.moveDown); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewHistory, gocui.KeyArrowUp, gocui.ModNone, u.moveUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewHistory, 'k', gocui.ModNone, u.moveUp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewSearch, gocui.KeyEnter, gocui.ModNone, u.submitSearch); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewSearch, gocui.KeyEsc, gocui.ModNone, u.cancelSearch); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewForm, gocui.KeyEnter, gocui.ModNone, u.submitFormNow); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewForm, gocui.KeyCtrlJ, gocui.ModNone, u.submitFormNow); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewForm, gocui.KeyTab, gocui.ModNone, u.nextFormField); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewForm, gocui.KeyBacktab, gocui.ModNone, u.prevFormField); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewForm, gocui.KeyArrowDown, gocui.ModNone, u.nextFormField); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewForm, gocui.KeyArrowUp, gocui.ModNone, u.prevFormField); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewForm, gocui.KeyEsc, gocui.ModNone, u.cancelForm); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewHelp, gocui.KeyEsc, gocui.ModNone, u.closeHelp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewHelp, 'q', gocui.ModNone, u.closeHelp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewHelp, '?', gocui.ModNone, u.closeHelp); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewTagCreate, gocui.KeyEnter, gocui.ModNone, u.submitTagCreate); err != nil {
		return err
	}
	if err := gui.SetKeybinding(viewTagCreate, gocui.KeyEsc, gocui.ModNone, u.cancelTagCreate); err != nil {
		return err
	}
	if err := gui.SetViewClickBinding(&gocui.ViewMouseBinding{ViewName: viewPending, Key: gocui.MouseLeft, Handler: func(opts gocui.ViewMouseBindingOpts) error {
		return u.onListClick(gui, viewPending, opts)
	}}); err != nil {
		return err
	}
	if err := gui.SetViewClickBinding(&gocui.ViewMouseBinding{ViewName: viewDone, Key: gocui.MouseLeft, Handler: func(opts gocui.ViewMouseBindingOpts) error {
		return u.onListClick(gui, viewDone, opts)
	}}); err != nil {
		return err
	}
	if err := gui.SetViewClickBinding(&gocui.ViewMouseBinding{ViewName: viewTags, Key: gocui.MouseLeft, Handler: func(opts gocui.ViewMouseBindingOpts) error {
		return u.onListClick(gui, viewTags, opts)
	}}); err != nil {
		return err
	}
	if err := gui.SetViewClickBinding(&gocui.ViewMouseBinding{ViewName: viewEventually, Key: gocui.MouseLeft, Handler: func(opts gocui.ViewMouseBindingOpts) error {
		return u.onListClick(gui, viewEventually, opts)
	}}); err != nil {
		return err
	}
	if err := gui.SetViewClickBinding(&gocui.ViewMouseBinding{ViewName: viewHistory, Key: gocui.MouseLeft, Handler: func(opts gocui.ViewMouseBindingOpts) error {
		return u.onListClick(gui, viewHistory, opts)
	}}); err != nil {
		return err
	}
	if err := u.bindMouseScroll(gui); err != nil {
		return err
	}
	return nil
}

func (u *UI) layout(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()
	if maxX <= 0 || maxY <= 0 {
		return nil
	}

	headerView, err := gui.SetView(viewHeader, 0, 0, maxX-1, 0, 0)
	if err != nil && !goerrors.Is(err, gocui.ErrUnknownView) {
		return err
	}
	if goerrors.Is(err, gocui.ErrUnknownView) {
		headerView.Frame = false
	}
	headerView.Frame = false
	headerView.Wrap = true
	headerView.FgColor = gocui.ColorDefault
	u.renderHeader(headerView)

	footerY1 := maxY - 2
	if footerY1 < 1 {
		footerY1 = 1
	}
	footerY0 := footerY1 - 2
	if footerY0 < 1 {
		footerY0 = 1
	}
	footerView, err := gui.SetView(viewFooter, 0, footerY0, maxX-1, footerY1, 0)
	if err != nil && !goerrors.Is(err, gocui.ErrUnknownView) {
		return err
	}

	footerView.Title = ""
	footerView.Frame = false
	footerView.Wrap = true
	footerView.FgColor = gocui.ColorDefault | gocui.AttrDim
	footerView.BgColor = gocui.ColorDefault
	u.renderFooter(footerView)

	bodyTop := 1
	bodyBottom := footerY0 - 1

	if bodyBottom < bodyTop {
		return nil
	}

	bodyHeight := bodyBottom - bodyTop + 1
	layout := computeLayout(maxX, bodyHeight)
	leftX0 := 0
	leftX1 := leftX0 + layout.leftWidth - 1
	gap := 1
	rightX0 := leftX1 + gap
	if rightX0 >= maxX {
		rightX0 = leftX1
	}
	rightX1 := maxX - 1

	pendingY0 := bodyTop
	pendingY1 := pendingY0 + layout.pendingHeight - 1
	doneY0 := pendingY1 + 1
	doneY1 := doneY0 + layout.doneHeight - 1
	tagsY0 := doneY1 + 1
	tagsY1 := bodyBottom

	highlightedY0 := bodyTop
	highlightedY1 := highlightedY0 + layout.highlighted - 1
	eventuallyY0 := highlightedY1 + 1
	eventuallyY1 := eventuallyY0 + layout.eventuallyHeight - 1
	historyY0 := eventuallyY1 + 1
	historyY1 := bodyBottom

	pendingView, err := gui.SetView(viewPending, leftX0, pendingY0, leftX1, pendingY1, 0)
	if err != nil && !goerrors.Is(err, gocui.ErrUnknownView) {
		return err
	}
	if goerrors.Is(err, gocui.ErrUnknownView) {
		pendingView.Title = "1 Pending"
		pendingView.TitleColor = gocui.ColorRed
	}
	applyViewStyle(pendingView, u.focus == viewPending, true)
	u.renderTaskList(pendingView, u.pending, u.selectedPending, u.focus == viewPending, u.pendingDepth, u.pendingHasChildren)

	doneView, err := gui.SetView(viewDone, leftX0, doneY0, leftX1, doneY1, 0)
	if err != nil && !goerrors.Is(err, gocui.ErrUnknownView) {
		return err
	}
	if goerrors.Is(err, gocui.ErrUnknownView) {
		doneView.Title = "2 Recently Done"
		doneView.TitleColor = gocui.ColorGreen
	}
	applyViewStyle(doneView, u.focus == viewDone, true)
	u.renderTaskList(doneView, u.done, u.selectedDone, u.focus == viewDone, u.doneDepth, u.doneHasChildren)

	tagsView, err := gui.SetView(viewTags, leftX0, tagsY0, leftX1, tagsY1, 0)
	if err != nil && !goerrors.Is(err, gocui.ErrUnknownView) {
		return err
	}
	if goerrors.Is(err, gocui.ErrUnknownView) {
		tagsView.Title = "3 Tags"
		tagsView.TitleColor = gocui.ColorCyan
	}
	applyViewStyle(tagsView, u.focus == viewTags, false)
	u.renderTags(tagsView)

	highlightedView, err := gui.SetView(viewHighlighted, rightX0, highlightedY0, rightX1, highlightedY1, 0)
	if err != nil && !goerrors.Is(err, gocui.ErrUnknownView) {
		return err
	}
	if goerrors.Is(err, gocui.ErrUnknownView) {
		highlightedView.Title = "4 Highlighted"
	}
	applyViewStyle(highlightedView, u.focus == viewHighlighted, false)
	u.renderHighlighted(highlightedView)

	eventuallyView, err := gui.SetView(viewEventually, rightX0, eventuallyY0, rightX1, eventuallyY1, 0)
	if err != nil && !goerrors.Is(err, gocui.ErrUnknownView) {
		return err
	}
	if goerrors.Is(err, gocui.ErrUnknownView) {
		eventuallyView.Title = "5 Eventually"
		eventuallyView.TitleColor = gocui.ColorYellow
	}
	applyViewStyle(eventuallyView, u.focus == viewEventually, true)
	u.renderTaskList(eventuallyView, u.eventually, u.selectedEventually, u.focus == viewEventually, u.eventuallyDepth, u.eventuallyHasChildren)

	historyView, err := gui.SetView(viewHistory, rightX0, historyY0, rightX1, historyY1, 0)
	if err != nil && !goerrors.Is(err, gocui.ErrUnknownView) {
		return err
	}
	if goerrors.Is(err, gocui.ErrUnknownView) {
		historyView.Title = "6 History"
	}

	applyViewStyle(historyView, u.focus == viewHistory, true)
	u.renderHistory(historyView, u.focus == viewHistory)

	_, _ = gui.SetViewOnTop(viewHeader)
	_, _ = gui.SetViewOnTop(viewFooter)

	if u.searchActive {
		if err := u.showSearch(gui); err != nil {
			return err
		}
	} else {
		_ = gui.DeleteView(viewSearch)
	}

	if u.form != nil {
		if err := u.showForm(gui); err != nil {
			return err
		}
	} else {
		_ = gui.DeleteView(viewForm)
	}

	if u.tagCreateActive {
		if err := u.showTagCreate(gui); err != nil {
			return err
		}
	} else {
		_ = gui.DeleteView(viewTagCreate)
	}

	if gui.CurrentView() == nil {
		_, _ = gui.SetCurrentView(u.focus)
	}

	gui.Cursor = u.searchActive || u.form != nil || u.tagCreateActive

	return nil
}

type layout struct {
	leftWidth        int
	pendingHeight    int
	doneHeight       int
	tagsHeight       int
	highlighted      int
	eventuallyHeight int
	historyHeight    int
}

func computeLayout(width, height int) layout {
	safeWidth := max(width-2, 20)
	safeHeight := max(height, 8)

	leftWidth := safeWidth / 3
	if leftWidth < 26 {
		leftWidth = 26
	}
	if leftWidth > safeWidth-18 {
		leftWidth = safeWidth / 2
	}

	pendingHeight := int(float64(safeHeight) * 0.45)
	if pendingHeight < 4 {
		pendingHeight = 4
	}
	doneHeight := int(float64(safeHeight) * 0.3)
	if doneHeight < 4 {
		doneHeight = 4
	}
	tagsHeight := safeHeight - pendingHeight - doneHeight - 2
	if tagsHeight < 4 {
		tagsHeight = 4
		doneHeight = max(safeHeight-pendingHeight-tagsHeight-2, 4)
	}

	highlighted := int(float64(safeHeight) * 0.4)
	if highlighted < 4 {
		highlighted = 4
	}
	eventuallyHeight := int(float64(safeHeight) * 0.2)
	if eventuallyHeight < 3 {
		eventuallyHeight = 3
	}
	historyHeight := safeHeight - highlighted - eventuallyHeight - 2
	if historyHeight < 4 {
		historyHeight = 4
		eventuallyHeight = max(safeHeight-highlighted-historyHeight-2, 3)
	}

	return layout{
		leftWidth:        leftWidth,
		pendingHeight:    pendingHeight,
		doneHeight:       doneHeight,
		tagsHeight:       tagsHeight,
		highlighted:      highlighted,
		eventuallyHeight: eventuallyHeight,
		historyHeight:    historyHeight,
	}
}

func (u *UI) loadTasks() error {
	tasks, err := u.store.ListTasks(context.Background(), u.filter)
	if err != nil {
		return err
	}

	allTags, err := u.store.ListTags(context.Background())
	if err != nil {
		return err
	}

	pending := make([]model.Task, 0, len(tasks))
	done := make([]model.Task, 0, len(tasks))
	eventually := make([]model.Task, 0, len(tasks))
	doing := make([]model.Task, 0, len(tasks))
	tagCounts := make(map[string]int)
	for _, task := range tasks {
		for _, tag := range task.Tags {
			tagCounts[tag.Name]++
		}
		switch task.Status {
		case "done":
			done = append(done, task)
		case "doing":
			doing = append(doing, task)
			pending = append(pending, task)
		case "eventually":
			eventually = append(eventually, task)
		default:
			pending = append(pending, task)
		}
	}

	sort.Slice(done, func(i, j int) bool {
		return done[i].UpdatedAt.After(done[j].UpdatedAt)
	})
	if len(done) > 10 {
		done = done[:10]
	}

	entries := make([]tagCountEntry, 0, len(allTags))
	for _, tag := range allTags {
		entries = append(entries, tagCountEntry{ID: tag.ID, Name: tag.Name, Count: tagCounts[tag.Name]})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count == entries[j].Count {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Count > entries[j].Count
	})

	pendingVisible, pendingDepth, pendingHasChildren := buildVisibleTaskTree(pending, u.collapsed)
	doneVisible, doneDepth, doneHasChildren := buildVisibleTaskTree(done, u.collapsed)
	eventuallyVisible, eventuallyDepth, eventuallyHasChildren := buildVisibleTaskTree(eventually, u.collapsed)

	u.pending = pendingVisible
	u.pendingDepth = pendingDepth
	u.pendingHasChildren = pendingHasChildren

	u.done = doneVisible
	u.doneDepth = doneDepth
	u.doneHasChildren = doneHasChildren

	u.eventually = eventuallyVisible
	u.eventuallyDepth = eventuallyDepth
	u.eventuallyHasChildren = eventuallyHasChildren

	u.tags = entries
	u.doing = doing

	if u.selectedPending >= len(u.pending) {
		u.selectedPending = max(len(u.pending)-1, 0)
	}
	if u.selectedDone >= len(u.done) {
		u.selectedDone = max(len(u.done)-1, 0)
	}
	if u.selectedEventually >= len(u.eventually) {
		u.selectedEventually = max(len(u.eventually)-1, 0)
	}
	if u.selectedTags >= len(u.tags) {
		u.selectedTags = max(len(u.tags)-1, 0)
	}
	if u.formTagIndex >= len(u.tags) {
		u.formTagIndex = max(len(u.tags)-1, 0)
	}

	return u.loadHistory()
}

func (u *UI) loadHistory() error {
	selected := u.selectedTask()
	if selected == nil {
		u.history = nil
		return nil
	}

	history, err := u.store.ListHistory(context.Background(), selected.ID)
	if err != nil {
		return err
	}
	u.history = history
	if u.selectedHistory >= len(u.history) {
		u.selectedHistory = max(len(u.history)-1, 0)
	}
	return nil
}

func (u *UI) renderHeader(view *gocui.View) {
	view.Clear()
	query := strings.TrimSpace(u.filter.Query)
	if query == "" {
		query = "type / to search"
	}

	viewLabel := "none"
	if u.activeView != nil {
		viewLabel = u.activeView.Name
	}

	statusLabel := u.filter.Status
	if statusLabel == "" {
		statusLabel = "any"
	}

	tagsLabel := "none"
	if len(u.filter.Tags) > 0 {
		tagsLabel = strings.Join(u.filter.Tags, ",")
	}

	dueLabel := "any"
	if u.filter.DueBefore != nil || u.filter.DueAfter != nil {
		before := "-"
		after := "-"
		if u.filter.DueBefore != nil {
			before = u.filter.DueBefore.Format("2006-01-02")
		}
		if u.filter.DueAfter != nil {
			after = u.filter.DueAfter.Format("2006-01-02")
		}
		dueLabel = fmt.Sprintf("%s..%s", after, before)
	}

	fmt.Fprintf(view, "Search: %s | View: %s | Status: %s | Tags: %s | Due: %s", query, viewLabel, statusLabel, tagsLabel, dueLabel)
}

func (u *UI) renderFooter(view *gocui.View) {
	view.Clear()
	view.SetOrigin(0, 0)
	view.SetCursor(0, 0)

	fmt.Fprintln(view, "a add | s subtask | e edit | d delete | enter collapse/save | c current | x done | v eventually")
	fmt.Fprintln(view, "/ search | space tag | tab field | h history | r reload | g clear | tab cycle | 1-6 panes | q quit")
	if u.status != "" {
		fmt.Fprint(view, u.status)
	}
}

func (u *UI) renderTaskList(view *gocui.View, tasks []model.Task, selected int, focused bool, depthByID map[int64]int, hasChildrenByID map[int64]bool) {
	view.Clear()
	for i, task := range tasks {
		prefix := " "
		if i == selected {
			if focused {
				prefix = ">"
			} else {
				prefix = "*"
			}
		}

		depth := 0
		if depthByID != nil {
			depth = depthByID[task.ID]
		}
		indent := strings.Repeat("  ", depth)

		marker := " "
		if hasChildrenByID != nil && hasChildrenByID[task.ID] {
			if u.collapsed != nil && u.collapsed[task.ID] {
				marker = "+"
			} else {
				marker = "-"
			}
		}

		fmt.Fprintf(view, "%s %s%s %s\n", prefix, indent, marker, formatTaskSummary(task))
	}
	if focused {
		view.SetCursor(0, min(selected, len(tasks)-1))
	}
}

func (u *UI) renderTags(view *gocui.View) {
	view.Clear()
	for index, entry := range u.tags {
		prefix := " "
		if index == u.selectedTags {
			prefix = ">"
		}
		marker := " "
		if u.isTagActive(entry.Name) {
			marker = "x"
		}
		fmt.Fprintf(view, "%s [%s] %s (%d)\n", prefix, marker, entry.Name, entry.Count)
	}
	if u.focus == viewTags {
		view.SetCursor(0, min(u.selectedTags, len(u.tags)-1))
	}
}

func (u *UI) onListClick(gui *gocui.Gui, viewName string, opts gocui.ViewMouseBindingOpts) error {
	if u.inputActive() {
		return nil
	}
	view, err := gui.View(viewName)
	if err != nil {
		return nil
	}

	_, y0, _, _ := view.Dimensions()
	_, oy := view.Origin()
	row := opts.Y - y0 - 1 + oy
	if row < 0 {
		row = 0
	}

	switch viewName {
	case viewPending:
		u.selectedPending = min(row, len(u.pending)-1)
		return u.setFocus(gui, viewPending)
	case viewDone:
		u.selectedDone = min(row, len(u.done)-1)
		return u.setFocus(gui, viewDone)
	case viewEventually:
		u.selectedEventually = min(row, len(u.eventually)-1)
		return u.setFocus(gui, viewEventually)
	case viewTags:
		u.selectedTags = min(row, len(u.tags)-1)
		return u.setFocus(gui, viewTags)
	case viewHistory:
		u.selectedHistory = min(row, len(u.history)-1)
		return u.setFocus(gui, viewHistory)
	default:
		return nil
	}
}

func (u *UI) bindMouseScroll(gui *gocui.Gui) error {
	views := []string{viewPending, viewDone, viewTags, viewEventually, viewHistory, viewHighlighted}
	for _, name := range views {
		if err := gui.SetKeybinding(name, gocui.MouseWheelUp, gocui.ModNone, u.scrollUp); err != nil {
			return err
		}
		if err := gui.SetKeybinding(name, gocui.MouseWheelDown, gocui.ModNone, u.scrollDown); err != nil {
			return err
		}
	}
	return nil
}

func (u *UI) scrollUp(gui *gocui.Gui, view *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	if view == nil {
		view = gui.CurrentView()
	}
	if view == nil {
		return nil
	}
	view.ScrollUp(1)
	return nil
}

func (u *UI) scrollDown(gui *gocui.Gui, view *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	if view == nil {
		view = gui.CurrentView()
	}
	if view == nil {
		return nil
	}
	view.ScrollDown(1)
	return nil
}

func (u *UI) renderHighlighted(view *gocui.View) {
	view.Clear()
	selected := u.selectedTask()
	if selected == nil {
		fmt.Fprint(view, "No task selected")
		return
	}

	due := "n/a"
	if selected.DueAt != nil {
		due = selected.DueAt.Format("2006-01-02")
	}

	lines := []string{}
	if u.focus == viewHistory {
		if entry := u.selectedHistoryEntry(); entry != nil {
			lines = append(lines,
				"History Detail",
				fmt.Sprintf("When: %s", entry.CreatedAt.Format("2006-01-02 15:04:05")),
				fmt.Sprintf("Type: %s", entry.EventType),
				fmt.Sprintf("Details: %s", entry.Details),
				"",
				"Task",
			)
		} else {
			lines = append(lines, "No history selected", "", "Task")
		}
	}

	lines = append(lines,
		selected.Title,
		fmt.Sprintf("Status: %s", selected.Status),
		fmt.Sprintf("Priority: %d", selected.Priority),
		fmt.Sprintf("Due: %s", due),
		fmt.Sprintf("Tags: %s", formatTags(selected.Tags)),
		"",
		selected.Description,
	)

	others := u.otherDoingTasks(selected.ID)
	if len(others) > 0 {
		lines = append(lines, "", "Also doing:")
		for _, task := range others {
			dueLabel := "n/a"
			if task.DueAt != nil {
				dueLabel = task.DueAt.Format("2006-01-02")
			}
			lines = append(lines,
				fmt.Sprintf("- %s", task.Title),
				fmt.Sprintf("  Status: %s", task.Status),
				fmt.Sprintf("  Due: %s", dueLabel),
				fmt.Sprintf("  Tags: %s", formatTags(task.Tags)),
				fmt.Sprintf("  %s", strings.TrimSpace(task.Description)),
			)
		}
	}

	fmt.Fprint(view, strings.Join(lines, "\n"))
}

func (u *UI) renderHistory(view *gocui.View, focused bool) {
	view.Clear()
	for index, entry := range u.history {
		prefix := " "
		if index == u.selectedHistory {
			if focused {
				prefix = ">"
			} else {
				prefix = "*"
			}
		}
		fmt.Fprintf(view, "%s %s | %s | %s\n", prefix, entry.CreatedAt.Format("2006-01-02 15:04"), entry.EventType, entry.Details)
	}
	if focused {
		view.SetCursor(0, min(u.selectedHistory, len(u.history)-1))
	}
}

func (u *UI) selectedHistoryEntry() *model.HistoryEntry {
	if u.selectedHistory >= 0 && u.selectedHistory < len(u.history) {
		return &u.history[u.selectedHistory]
	}
	return nil
}

func (u *UI) otherDoingTasks(selectedID int64) []model.Task {
	if len(u.doing) == 0 {
		return nil
	}

	others := make([]model.Task, 0, len(u.doing))
	for _, task := range u.doing {
		if task.ID == selectedID {
			continue
		}
		others = append(others, task)
	}
	return others
}

func (u *UI) selectedTask() *model.Task {
	switch u.focus {
	case viewDone:
		if u.selectedDone >= 0 && u.selectedDone < len(u.done) {
			return &u.done[u.selectedDone]
		}
	case viewEventually:
		if u.selectedEventually >= 0 && u.selectedEventually < len(u.eventually) {
			return &u.eventually[u.selectedEventually]
		}
	default:
		if u.selectedPending >= 0 && u.selectedPending < len(u.pending) {
			return &u.pending[u.selectedPending]
		}
	}
	return nil
}

func (u *UI) switchFocus(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}

	switch u.focus {
	case viewPending:
		u.focus = viewDone
	case viewDone:
		u.focus = viewTags
	case viewTags:
		u.focus = viewEventually
	default:
		u.focus = viewPending
	}
	_, _ = gui.SetCurrentView(u.focus)
	return u.reload(gui, nil)
}

func (u *UI) focusPending(gui *gocui.Gui, _ *gocui.View) error {
	return u.setFocus(gui, viewPending)
}

func (u *UI) focusDone(gui *gocui.Gui, _ *gocui.View) error {
	return u.setFocus(gui, viewDone)
}

func (u *UI) focusTags(gui *gocui.Gui, _ *gocui.View) error {
	return u.setFocus(gui, viewTags)
}

func (u *UI) focusHighlighted(gui *gocui.Gui, _ *gocui.View) error {
	return u.setFocus(gui, viewHighlighted)
}

func (u *UI) focusEventually(gui *gocui.Gui, _ *gocui.View) error {
	return u.setFocus(gui, viewEventually)
}

func (u *UI) focusHistory(gui *gocui.Gui, _ *gocui.View) error {
	return u.setFocus(gui, viewHistory)
}

func (u *UI) setFocus(gui *gocui.Gui, name string) error {
	if u.inputActive() {
		return nil
	}
	u.focus = name
	_, _ = gui.SetCurrentView(name)
	return u.reload(gui, nil)
}

func (u *UI) moveDown(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	switch u.focus {
	case viewDone:
		if u.selectedDone < len(u.done)-1 {
			u.selectedDone++
			return u.loadHistory()
		}
	case viewPending:
		if u.selectedPending < len(u.pending)-1 {
			u.selectedPending++
			return u.loadHistory()
		}
	case viewHistory:
		if u.selectedHistory < len(u.history)-1 {
			u.selectedHistory++
		}
	case viewTags:
		if u.selectedTags < len(u.tags)-1 {
			u.selectedTags++
		}
	case viewEventually:
		if u.selectedEventually < len(u.eventually)-1 {
			u.selectedEventually++
			return u.loadHistory()
		}
	}
	return nil
}

func (u *UI) moveUp(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	switch u.focus {
	case viewDone:
		if u.selectedDone > 0 {
			u.selectedDone--
			return u.loadHistory()
		}
	case viewPending:
		if u.selectedPending > 0 {
			u.selectedPending--
			return u.loadHistory()
		}
	case viewHistory:
		if u.selectedHistory > 0 {
			u.selectedHistory--
		}
	case viewTags:
		if u.selectedTags > 0 {
			u.selectedTags--
		}
	case viewEventually:
		if u.selectedEventually > 0 {
			u.selectedEventually--
			return u.loadHistory()
		}
	}
	return nil
}

func (u *UI) reload(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	u.status = ""
	return u.loadTasks()
}

func (u *UI) clearFilters(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	u.filter.Query = ""
	u.filter.Tags = nil
	u.filter.Status = ""
	u.filter.DueAfter = nil
	u.filter.DueBefore = nil
	u.activeTags = make(map[string]struct{})
	return u.reload(gui, nil)
}

func (u *UI) refreshHistory(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	return u.loadHistory()
}

func (u *UI) startSearch(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	u.searchActive = true
	return nil
}

func (u *UI) toggleHelp(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() && !u.helpActive {
		return nil
	}
	u.helpActive = !u.helpActive
	return nil
}

func (u *UI) closeHelp(gui *gocui.Gui, _ *gocui.View) error {
	u.helpActive = false
	_ = gui.DeleteView(viewHelp)
	_, _ = gui.SetCurrentView(u.focus)
	return nil
}

func (u *UI) openTagCreate(gui *gocui.Gui, _ *gocui.View) error {
	if u.tagCreateActive || u.focus != viewTags {
		return nil
	}
	u.tagCreateActive = true
	u.tagCreateValue = ""
	return nil
}

func (u *UI) submitTagCreate(gui *gocui.Gui, view *gocui.View) error {
	if !u.tagCreateActive {
		return nil
	}
	name := strings.TrimSpace(view.Buffer())
	if name != "" {
		if _, err := u.store.Queries.CreateTag(context.Background(), name); err != nil {
			u.status = err.Error()
			return nil
		}
	}
	return u.closeTagCreate(gui)
}

func (u *UI) cancelTagCreate(gui *gocui.Gui, _ *gocui.View) error {
	if !u.tagCreateActive {
		return nil
	}
	return u.closeTagCreate(gui)
}

func (u *UI) closeTagCreate(gui *gocui.Gui) error {
	u.tagCreateActive = false
	_ = gui.DeleteView(viewTagCreate)
	_, _ = gui.SetCurrentView(u.focus)
	return u.loadTasks()
}

func (u *UI) showTagCreate(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()
	width := max(40, maxX/3)
	height := 3
	x0 := (maxX - width) / 2
	y0 := (maxY - height) / 2
	x1 := x0 + width
	y1 := y0 + height

	view, err := gui.SetView(viewTagCreate, x0, y0, x1, y1, 0)
	if err != nil && !goerrors.Is(err, gocui.ErrUnknownView) {
		return err
	}
	if goerrors.Is(err, gocui.ErrUnknownView) {
		view.Title = "New Tag"
		view.Wrap = true
		view.Editable = true
		view.Editor = gocui.DefaultEditor
		view.Clear()
	}
	view.Editable = true
	view.Editor = gocui.DefaultEditor
	_, _ = gui.SetCurrentView(viewTagCreate)
	return nil
}

func (u *UI) tagOptions() []string {
	result := make([]string, 0, len(u.tags))
	for _, entry := range u.tags {
		result = append(result, entry.Name)
	}
	return result
}

func (u *UI) showHelp(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()
	width := max(60, maxX/2)
	height := 12
	x0 := (maxX - width) / 2
	y0 := (maxY - height) / 2
	x1 := x0 + width
	y1 := y0 + height

	view, err := gui.SetView(viewHelp, x0, y0, x1, y1, 0)
	if err != nil && !goerrors.Is(err, gocui.ErrUnknownView) {
		return err
	}
	if goerrors.Is(err, gocui.ErrUnknownView) {
		view.Title = "Help"
		view.Wrap = true
	}
	view.Clear()
	fmt.Fprint(view, helpText())
	_, _ = gui.SetCurrentView(viewHelp)
	return nil
}

func (u *UI) showSearch(gui *gocui.Gui) error {
	maxX, maxY := gui.Size()
	width := max(30, maxX/2)
	height := 3
	x0 := (maxX - width) / 2
	y0 := (maxY - height) / 2
	x1 := x0 + width
	y1 := y0 + height

	view, err := gui.SetView(viewSearch, x0, y0, x1, y1, 0)
	if err != nil && !goerrors.Is(err, gocui.ErrUnknownView) {
		return err
	}
	if goerrors.Is(err, gocui.ErrUnknownView) {
		view.Title = "Search"
		view.Wrap = true
		view.Clear()
		fmt.Fprint(view, u.filter.Query)
	}
	view.Editable = true
	view.Editor = gocui.DefaultEditor
	_, _ = gui.SetCurrentView(viewSearch)
	return nil
}

func (u *UI) submitSearch(gui *gocui.Gui, view *gocui.View) error {
	value := strings.TrimSpace(view.Buffer())
	u.filter.Query = value
	u.searchActive = false
	u.status = ""
	_ = gui.DeleteView(viewSearch)
	_, _ = gui.SetCurrentView(u.focus)
	return u.loadTasks()
}

func (u *UI) cancelSearch(gui *gocui.Gui, _ *gocui.View) error {
	u.searchActive = false
	_ = gui.DeleteView(viewSearch)
	_, _ = gui.SetCurrentView(u.focus)
	return nil
}

func (u *UI) addTask(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	fields := buildFormFields(nil)
	if u.focus == viewEventually {
		fields[fieldStatus].Value = "eventually"
	}
	u.form = &formState{fields: fields}
	u.formTagIndex = 0
	return nil
}

func (u *UI) addSubtask(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	selected := u.selectedTask()
	if selected == nil {
		return nil
	}

	fields := buildFormFields(nil)
	fields[fieldStatus].Value = selected.Status
	fields[fieldTags].Value = joinTags(selected.Tags)
	parentID := selected.ID
	u.form = &formState{fields: fields, parentTaskID: &parentID}
	u.formTagIndex = 0
	return nil
}

func (u *UI) editTask(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	selected := u.selectedTask()
	if selected == nil {
		return nil
	}
	fields := buildFormFields(selected)
	u.form = &formState{taskID: selected.ID, fields: fields}
	u.formTagIndex = 0
	return nil
}

func (u *UI) showForm(gui *gocui.Gui) error {
	if u.form == nil {
		return nil
	}

	maxX, maxY := gui.Size()
	width := max(60, maxX/2)
	height := min(12, max(8, maxY/2))
	x0 := (maxX - width) / 2
	y0 := (maxY - height) / 2
	x1 := x0 + width
	y1 := y0 + height

	view, err := gui.SetView(viewForm, x0, y0, x1, y1, 0)
	if err != nil && !goerrors.Is(err, gocui.ErrUnknownView) {
		return err
	}
	if goerrors.Is(err, gocui.ErrUnknownView) {
		view.Wrap = true
		view.Editable = true
		view.Editor = u.formEditor
	}
	view.Title = "Task Editor"
	if u.form.taskID != 0 {
		view.Title = "Edit Task"
	} else if u.form.parentTaskID != nil {
		view.Title = "New Subtask"
	} else {
		view.Title = "New Task"
	}
	view.Editable = true
	view.KeybindOnEdit = true
	view.Editor = u.formEditor
	u.renderForm(view)
	_, _ = gui.SetCurrentView(viewForm)
	return nil
}

func (u *UI) submitFormNow(gui *gocui.Gui, view *gocui.View) error {
	if u.form == nil {
		return nil
	}

	input, err := parseFormFields(u.form.fields)
	if err != nil {
		u.status = err.Error()
		return nil
	}
	input.ParentTaskID = u.form.parentTaskID

	if u.form.taskID == 0 {
		if _, err := u.store.CreateTask(context.Background(), input); err != nil {
			u.status = err.Error()
			return nil
		}
	} else {
		if _, err := u.store.UpdateTask(context.Background(), u.form.taskID, input); err != nil {
			u.status = err.Error()
			return nil
		}
	}

	u.form = nil
	u.status = ""
	_ = gui.DeleteView(viewForm)
	_, _ = gui.SetCurrentView(u.focus)
	return u.loadTasks()
}

func (u *UI) cancelForm(gui *gocui.Gui, _ *gocui.View) error {
	u.form = nil
	_ = gui.DeleteView(viewForm)
	_, _ = gui.SetCurrentView(u.focus)
	return nil
}

func (u *UI) nextFormField(gui *gocui.Gui, view *gocui.View) error {
	if u.form == nil {
		return nil
	}
	if u.form.index < len(u.form.fields)-1 {
		u.form.index++
	}
	u.renderForm(view)
	return nil
}

func (u *UI) prevFormField(gui *gocui.Gui, view *gocui.View) error {
	if u.form == nil {
		return nil
	}
	if u.form.index > 0 {
		u.form.index--
	}
	u.renderForm(view)
	return nil
}

func (u *UI) renderForm(view *gocui.View) {
	if u.form == nil || view == nil {
		return
	}
	view.Clear()
	for index, field := range u.form.fields {
		prefix := "  "
		if index == u.form.index {
			prefix = "> "
		}
		value := field.Value
		if isTagsField(field.Label) {
			candidate := u.currentTagOption()
			if candidate != "" {
				value = fmt.Sprintf("%s [pick: %s]", value, candidate)
			}
		}
		fmt.Fprintf(view, "%s%s: %s\n", prefix, field.Label, value)
	}
	label := u.form.fields[u.form.index].Label + ": "
	cursorX := len([]rune(label)) + len([]rune(u.form.fields[u.form.index].Value)) + 2
	view.SetCursor(cursorX, u.form.index)
}

func (e *formEditor) Edit(view *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) bool {
	ui := e.ui
	if ui == nil || ui.form == nil || view == nil {
		return false
	}
	field := &ui.form.fields[ui.form.index]

	if isStatusField(field.Label) {
		switch key {
		case gocui.KeyArrowRight, gocui.KeySpace:
			field.Value = nextStatus(field.Value)
		case gocui.KeyArrowLeft:
			field.Value = prevStatus(field.Value)
		}
		ui.renderForm(view)
		return true
	}

	if isTagsField(field.Label) {
		switch key {
		case gocui.KeyArrowRight:
			ui.formTagIndex = min(ui.formTagIndex+1, len(ui.tagOptions())-1)
		case gocui.KeyArrowLeft:
			ui.formTagIndex = max(ui.formTagIndex-1, 0)
		case gocui.KeySpace:
			ui.toggleTagInField(field)
		}
		ui.renderForm(view)
		return true
	}

	switch key {
	case gocui.KeyBackspace, gocui.KeyBackspace2:
		runes := []rune(field.Value)
		if len(runes) > 0 {
			field.Value = string(runes[:len(runes)-1])
		}
	case gocui.KeySpace:
		field.Value += " "
	case gocui.KeyCtrlU:
		field.Value = ""
	}

	if ch != 0 && ch != '\n' && ch != '\r' && mod == 0 {
		field.Value += string(ch)
	}

	ui.renderForm(view)
	return true
}

func isStatusField(label string) bool {
	return strings.HasPrefix(label, "Status")
}

func isTagsField(label string) bool {
	return strings.HasPrefix(label, "Tags")
}

func nextStatus(current string) string {
	order := []string{"todo", "doing", "eventually", "done"}
	return cycleStatus(order, current, 1)
}

func prevStatus(current string) string {
	order := []string{"todo", "doing", "eventually", "done"}
	return cycleStatus(order, current, -1)
}

func cycleStatus(order []string, current string, delta int) string {
	value := strings.TrimSpace(strings.ToLower(current))
	index := 0
	for i, status := range order {
		if status == value {
			index = i
			break
		}
	}
	index = (index + delta + len(order)) % len(order)
	return order[index]
}

func (u *UI) currentTagOption() string {
	options := u.tagOptions()
	if len(options) == 0 {
		return ""
	}
	if u.formTagIndex < 0 {
		u.formTagIndex = 0
	}
	if u.formTagIndex >= len(options) {
		u.formTagIndex = len(options) - 1
	}
	return options[u.formTagIndex]
}

func (u *UI) toggleTagInField(field *formField) {
	options := u.tagOptions()
	if len(options) == 0 {
		return
	}

	current := u.currentTagOption()
	if current == "" {
		return
	}

	selected := make(map[string]struct{})
	for _, name := range parseTags(field.Value) {
		selected[name] = struct{}{}
	}

	if _, ok := selected[current]; ok {
		delete(selected, current)
	} else {
		selected[current] = struct{}{}
	}

	ordered := make([]string, 0, len(selected))
	for name := range selected {
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)
	field.Value = strings.Join(ordered, ", ")
}

func nextTagValue(current string, options []string) string {
	return cycleTag(options, current, 1)
}

func prevTagValue(current string, options []string) string {
	return cycleTag(options, current, -1)
}

func cycleTag(options []string, current string, delta int) string {
	if len(options) == 0 {
		return ""
	}
	value := strings.TrimSpace(current)
	index := 0
	for i, option := range options {
		if option == value {
			index = i
			break
		}
	}
	index = (index + delta + len(options)) % len(options)
	return options[index]
}

func (u *UI) deleteTask(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	selected := u.selectedTask()
	if selected == nil {
		return nil
	}
	if err := u.store.DeleteTask(context.Background(), selected.ID); err != nil {
		u.status = err.Error()
		return nil
	}
	u.status = ""
	return u.loadTasks()
}

func (u *UI) deleteTag(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() || u.focus != viewTags {
		return nil
	}
	if u.selectedTags < 0 || u.selectedTags >= len(u.tags) {
		return nil
	}
	entry := u.tags[u.selectedTags]
	if err := u.store.DeleteTag(context.Background(), entry.ID); err != nil {
		u.status = err.Error()
		return nil
	}
	delete(u.activeTags, entry.Name)
	u.filter.Tags = u.activeTagList()
	u.status = ""
	return u.loadTasks()
}

func (u *UI) toggleCollapse(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	if u.focus != viewPending && u.focus != viewDone && u.focus != viewEventually {
		return nil
	}

	selected := u.selectedTask()
	if selected == nil {
		return nil
	}

	hasChildren := false
	switch u.focus {
	case viewDone:
		hasChildren = u.doneHasChildren != nil && u.doneHasChildren[selected.ID]
	case viewEventually:
		hasChildren = u.eventuallyHasChildren != nil && u.eventuallyHasChildren[selected.ID]
	default:
		hasChildren = u.pendingHasChildren != nil && u.pendingHasChildren[selected.ID]
	}
	if !hasChildren {
		return nil
	}

	u.collapsed[selected.ID] = !u.collapsed[selected.ID]
	return u.loadTasks()
}

func (u *UI) toggleDoing(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	selected := u.selectedTask()
	if selected == nil {
		return nil
	}
	input := taskInputFromTask(*selected)
	if selected.Status == "doing" {
		input.Status = "todo"
	} else {
		input.Status = "doing"
	}
	if _, err := u.store.UpdateTask(context.Background(), selected.ID, input); err != nil {
		u.status = err.Error()
		return nil
	}
	u.status = ""
	return u.loadTasks()
}

func (u *UI) toggleDone(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	selected := u.selectedTask()
	if selected == nil {
		return nil
	}
	input := taskInputFromTask(*selected)
	if selected.Status == "done" {
		input.Status = "todo"
	} else {
		input.Status = "done"
	}
	if _, err := u.store.UpdateTask(context.Background(), selected.ID, input); err != nil {
		u.status = err.Error()
		return nil
	}
	u.status = ""
	return u.loadTasks()
}

func (u *UI) toggleEventually(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() {
		return nil
	}
	selected := u.selectedTask()
	if selected == nil {
		return nil
	}
	input := taskInputFromTask(*selected)
	if selected.Status == "eventually" {
		input.Status = "todo"
	} else {
		input.Status = "eventually"
	}
	if _, err := u.store.UpdateTask(context.Background(), selected.ID, input); err != nil {
		u.status = err.Error()
		return nil
	}
	u.status = ""
	return u.loadTasks()
}

func (u *UI) toggleTagFilter(gui *gocui.Gui, _ *gocui.View) error {
	if u.inputActive() || u.focus != viewTags {
		return nil
	}
	if u.selectedTags < 0 || u.selectedTags >= len(u.tags) {
		return nil
	}
	name := u.tags[u.selectedTags].Name
	if u.isTagActive(name) {
		delete(u.activeTags, name)
	} else {
		u.activeTags[name] = struct{}{}
	}
	u.filter.Tags = u.activeTagList()
	return u.reload(gui, nil)
}

func (u *UI) activeTagList() []string {
	result := make([]string, 0, len(u.activeTags))
	for name := range u.activeTags {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func (u *UI) isTagActive(name string) bool {
	_, ok := u.activeTags[name]
	return ok
}

func (u *UI) inputActive() bool {
	return u.searchActive || u.form != nil || u.helpActive || u.tagCreateActive
}

func (u *UI) quit(_ *gocui.Gui, _ *gocui.View) error {
	return gocui.ErrQuit
}

func helpText() string {
	return strings.Join([]string{
		"Navigation:",
		"  Tab cycle panes (pending/done/tags/eventually)",
		"  1 Pending | 2 Done | 3 Tags | 4 Highlighted | 5 Eventually | 6 History",
		"  j/k or arrows move selection",
		"  mouse click to focus/select",
		"  mouse wheel scrolls hovered pane",
		"",
		"Actions:",
		"  a add task | s add subtask | e edit task | d delete task/tag",
		"  c current | x toggle done | v eventually",
		"  enter collapse/expand (lists) | enter save (form) | tab next field",
		"",
		"Search/Filter:",
		"  / search | g clear filters",
		"",
		"Tags:",
		"  space toggle tag filter (Tags pane)",
		"  a add tag (Tags pane)",
		"  d delete tag (Tags pane)",
		"  space/left/right cycle tags (form)",
		"",
		"Status:",
		"  space/left/right cycle status (form)",
		"",
		"Other:",
		"  h refresh history | r reload | ? help | esc/q close help | q quit",
	}, "\n")
}

func applyViewStyle(view *gocui.View, focused bool, highlight bool) {
	view.Frame = true
	view.Highlight = focused && highlight
	view.HighlightInactive = false
	view.SelBgColor = gocui.ColorBlue
	view.SelFgColor = gocui.ColorBlack
	view.InactiveViewSelBgColor = gocui.ColorDefault
	if focused {
		view.FrameColor = gocui.ColorCyan
		view.TitleColor = gocui.ColorCyan
	} else {
		view.FrameColor = gocui.ColorDefault
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
