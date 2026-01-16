package db

import (
	"context"
	"testing"
)

func TestCreateTaskPersistsTagsAndHistory(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	created, err := store.CreateTask(context.Background(), TaskInput{
		Title:       "Write tests",
		Description: "Add coverage",
		Status:      "ToDo",
		Priority:    2,
		Tags:        []string{"Work", "Home"},
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("expected task ID to be set")
	}
	if created.Status != "todo" {
		t.Fatalf("expected status 'todo', got %q", created.Status)
	}
	if len(created.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(created.Tags))
	}
	nameSet := map[string]struct{}{}
	for _, tag := range created.Tags {
		nameSet[tag.Name] = struct{}{}
	}
	if _, ok := nameSet["Work"]; !ok {
		t.Fatalf("expected tag 'Work' to be assigned")
	}
	if _, ok := nameSet["Home"]; !ok {
		t.Fatalf("expected tag 'Home' to be assigned")
	}

	tags, err := store.ListTags(context.Background())
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags in store, got %d", len(tags))
	}

	history, err := store.ListHistory(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
	if history[0].EventType != "created" {
		t.Fatalf("expected history event 'created', got %q", history[0].EventType)
	}
}

func TestNestedTasksKeepChildrenOnDelete(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	parent, err := store.CreateTask(context.Background(), TaskInput{Title: "Parent"})
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	child, err := store.CreateTask(context.Background(), TaskInput{Title: "Child", ParentTaskID: &parent.ID})
	if err != nil {
		t.Fatalf("create child: %v", err)
	}
	if child.ParentTaskID == nil || *child.ParentTaskID != parent.ID {
		t.Fatalf("expected child.ParentTaskID to be %d, got %v", parent.ID, child.ParentTaskID)
	}

	if err := store.DeleteTask(context.Background(), parent.ID); err != nil {
		t.Fatalf("delete parent: %v", err)
	}

	reloaded, err := store.GetTaskWithTags(context.Background(), child.ID)
	if err != nil {
		t.Fatalf("get child after parent delete: %v", err)
	}
	if reloaded.ParentTaskID != nil {
		t.Fatalf("expected child.ParentTaskID to be nil after parent delete")
	}
}

func newTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return NewStore(db), func() {
		_ = db.Close()
	}
}
