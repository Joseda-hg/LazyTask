package tui

import (
	"context"
	"testing"

	"github.com/Joseda-hg/lazytask/internal/db"
	"github.com/Joseda-hg/lazytask/internal/model"
)

func TestDeleteTagRemovesTagAndFilter(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	created, err := store.CreateTask(context.Background(), db.TaskInput{
		Title:       "Tag cleanup",
		Description: "",
		Status:      "todo",
		Priority:    1,
		Tags:        []string{"Work"},
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	ui := &UI{
		store:      store,
		focus:      viewTags,
		activeTags: map[string]struct{}{"Work": {}},
	}
	if err := ui.loadTasks(); err != nil {
		t.Fatalf("load tasks: %v", err)
	}
	if len(ui.tags) != 1 {
		t.Fatalf("expected 1 tag entry, got %d", len(ui.tags))
	}
	ui.selectedTags = 0

	if err := ui.deleteTag(nil, nil); err != nil {
		t.Fatalf("delete tag: %v", err)
	}

	remainingTags, err := store.ListTags(context.Background())
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(remainingTags) != 0 {
		t.Fatalf("expected tags to be deleted, got %d", len(remainingTags))
	}

	updated, err := store.GetTaskWithTags(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if len(updated.Tags) != 0 {
		t.Fatalf("expected task tags to be cleared, got %d", len(updated.Tags))
	}
	if len(ui.activeTags) != 0 {
		t.Fatalf("expected active tags to be cleared")
	}
	if len(ui.filter.Tags) != 0 {
		t.Fatalf("expected filter tags to be cleared")
	}
}

func TestToggleTaskStates(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.CreateTask(context.Background(), db.TaskInput{
		Title:    "Toggle status",
		Status:   "todo",
		Priority: 1,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	t.Run("toggle doing", func(t *testing.T) {
		ui := newTestUI(store)
		ui.focus = viewPending
		if err := ui.loadTasks(); err != nil {
			t.Fatalf("load tasks: %v", err)
		}
		ui.selectedPending = 0

		if err := ui.toggleDoing(nil, nil); err != nil {
			t.Fatalf("toggle doing: %v", err)
		}
		status := taskStatus(t, store)
		if status != "doing" {
			t.Fatalf("expected status 'doing', got %q", status)
		}

		if err := ui.toggleDoing(nil, nil); err != nil {
			t.Fatalf("toggle doing again: %v", err)
		}
		status = taskStatus(t, store)
		if status != "todo" {
			t.Fatalf("expected status 'todo', got %q", status)
		}
	})

	t.Run("toggle done", func(t *testing.T) {
		ui := newTestUI(store)
		ui.focus = viewPending
		if err := ui.loadTasks(); err != nil {
			t.Fatalf("load tasks: %v", err)
		}
		ui.selectedPending = 0

		if err := ui.toggleDone(nil, nil); err != nil {
			t.Fatalf("toggle done: %v", err)
		}
		status := taskStatus(t, store)
		if status != "done" {
			t.Fatalf("expected status 'done', got %q", status)
		}

		ui.focus = viewDone
		if err := ui.loadTasks(); err != nil {
			t.Fatalf("load tasks: %v", err)
		}
		ui.selectedDone = 0
		if err := ui.toggleDone(nil, nil); err != nil {
			t.Fatalf("toggle done again: %v", err)
		}
		status = taskStatus(t, store)
		if status != "todo" {
			t.Fatalf("expected status 'todo', got %q", status)
		}
	})

	t.Run("toggle eventually", func(t *testing.T) {
		ui := newTestUI(store)
		ui.focus = viewPending
		if err := ui.loadTasks(); err != nil {
			t.Fatalf("load tasks: %v", err)
		}
		ui.selectedPending = 0

		if err := ui.toggleEventually(nil, nil); err != nil {
			t.Fatalf("toggle eventually: %v", err)
		}
		status := taskStatus(t, store)
		if status != "eventually" {
			t.Fatalf("expected status 'eventually', got %q", status)
		}

		ui.focus = viewEventually
		if err := ui.loadTasks(); err != nil {
			t.Fatalf("load tasks: %v", err)
		}
		ui.selectedEventually = 0
		if err := ui.toggleEventually(nil, nil); err != nil {
			t.Fatalf("toggle eventually again: %v", err)
		}
		status = taskStatus(t, store)
		if status != "todo" {
			t.Fatalf("expected status 'todo', got %q", status)
		}
	})
}

func taskStatus(t *testing.T, store *db.Store) string {
	t.Helper()
	tasks, err := store.ListTasks(context.Background(), model.Filter{})
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	return tasks[0].Status
}

func newTestUI(store *db.Store) *UI {
	return &UI{
		store:      store,
		activeTags: make(map[string]struct{}),
	}
}

func newTestStore(t *testing.T) (*db.Store, func()) {
	t.Helper()
	dbConn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db.NewStore(dbConn), func() {
		_ = dbConn.Close()
	}
}
