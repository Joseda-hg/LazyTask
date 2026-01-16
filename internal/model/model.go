package model

import "time"

type Task struct {
	ID           int64
	ParentTaskID *int64
	Title        string
	Description  string
	Status       string
	Priority     int64
	DueAt        *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Tags         []Tag
}

type Tag struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

type HistoryEntry struct {
	ID        int64
	TaskID    int64
	EventType string
	Details   string
	CreatedAt time.Time
}

type View struct {
	ID        int64
	Name      string
	Filter    Filter
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Filter struct {
	Query     string     `json:"query"`
	Status    string     `json:"status"`
	Tags      []string   `json:"tags"`
	DueBefore *time.Time `json:"due_before"`
	DueAfter  *time.Time `json:"due_after"`
}
