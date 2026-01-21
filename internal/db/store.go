package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	sqlc "github.com/Joseda-hg/lazytask/internal/db/sqlc"
	"github.com/Joseda-hg/lazytask/internal/model"
)

type Store struct {
	DB      *sql.DB
	Queries *sqlc.Queries
}

type TaskInput struct {
	Title        string
	Description  string
	Status       string
	Priority     int64
	DueAt        *time.Time
	ParentTaskID *int64
	Tags         []string
}

func NewStore(db *sql.DB) *Store {
	return &Store{DB: db, Queries: sqlc.New(db)}
}

func (s *Store) CreateTask(ctx context.Context, input TaskInput) (model.Task, error) {
	status := normalizeStatus(input.Status)

	var dueAt sql.NullTime
	if input.DueAt != nil {
		dueAt = sql.NullTime{Time: *input.DueAt, Valid: true}
	}

	var parentTaskID sql.NullInt64
	if input.ParentTaskID != nil {
		parentTaskID = sql.NullInt64{Int64: *input.ParentTaskID, Valid: true}
	}

	created, err := s.Queries.CreateTask(ctx, sqlc.CreateTaskParams{
		Title:        input.Title,
		Description:  input.Description,
		Status:       status,
		Priority:     input.Priority,
		DueAt:        dueAt,
		ParentTaskID: parentTaskID,
	})
	if err != nil {
		return model.Task{}, err
	}

	if err := s.SetTaskTags(ctx, created.ID, input.Tags); err != nil {
		return model.Task{}, err
	}

	createdTask, err := s.GetTaskWithTags(ctx, created.ID)
	if err != nil {
		return model.Task{}, err
	}

	if _, err := s.Queries.AddHistory(ctx, sqlc.AddHistoryParams{
		TaskID:    created.ID,
		EventType: "created",
		Details:   formatCreatedDetails(createdTask),
	}); err != nil {
		return model.Task{}, err
	}

	return createdTask, nil
}

func (s *Store) UpdateTask(ctx context.Context, taskID int64, input TaskInput) (model.Task, error) {
	before, err := s.GetTaskWithTags(ctx, taskID)
	if err != nil {
		return model.Task{}, err
	}

	status := normalizeStatus(input.Status)

	var dueAt sql.NullTime
	if input.DueAt != nil {
		dueAt = sql.NullTime{Time: *input.DueAt, Valid: true}
	}

	var parentTaskID sql.NullInt64
	if input.ParentTaskID != nil {
		parentTaskID = sql.NullInt64{Int64: *input.ParentTaskID, Valid: true}
	}

	updated, err := s.Queries.UpdateTask(ctx, sqlc.UpdateTaskParams{
		Title:        input.Title,
		Description:  input.Description,
		Status:       status,
		Priority:     input.Priority,
		DueAt:        dueAt,
		ParentTaskID: parentTaskID,
		ID:           taskID,
	})
	if err != nil {
		return model.Task{}, err
	}

	if err := s.SetTaskTags(ctx, updated.ID, input.Tags); err != nil {
		return model.Task{}, err
	}

	after, err := s.GetTaskWithTags(ctx, updated.ID)
	if err != nil {
		return model.Task{}, err
	}

	if _, err := s.Queries.AddHistory(ctx, sqlc.AddHistoryParams{
		TaskID:    updated.ID,
		EventType: "updated",
		Details:   formatTaskDiff(before, after),
	}); err != nil {
		return model.Task{}, err
	}

	return after, nil
}

func (s *Store) DeleteTask(ctx context.Context, taskID int64) error {
	before, err := s.GetTaskWithTags(ctx, taskID)
	if err != nil {
		return err
	}

	if _, err := s.Queries.AddHistory(ctx, sqlc.AddHistoryParams{
		TaskID:    taskID,
		EventType: "deleted",
		Details:   formatDeletedDetails(before),
	}); err != nil {
		return err
	}

	return s.Queries.DeleteTask(ctx, taskID)
}

func (s *Store) GetTaskWithTags(ctx context.Context, taskID int64) (model.Task, error) {
	row, err := s.Queries.GetTask(ctx, taskID)
	if err != nil {
		return model.Task{}, err
	}

	tags, err := s.Queries.ListTagsForTask(ctx, taskID)
	if err != nil {
		return model.Task{}, err
	}

	return mapTask(row, tags), nil
}

func (s *Store) ListTasks(ctx context.Context, filter model.Filter) ([]model.Task, error) {
	var dueBefore sql.NullTime
	if filter.DueBefore != nil {
		dueBefore = sql.NullTime{Time: *filter.DueBefore, Valid: true}
	}

	var dueAfter sql.NullTime
	if filter.DueAfter != nil {
		dueAfter = sql.NullTime{Time: *filter.DueAfter, Valid: true}
	}

	query := strings.TrimSpace(filter.Query)
	status := strings.TrimSpace(filter.Status)

	var rows []sqlc.Task
	var err error
	if len(filter.Tags) > 0 {
		rows, err = s.Queries.ListTasksByTags(ctx, sqlc.ListTasksByTagsParams{
			Tags:  filter.Tags,
			Query: query,
		})
	} else {
		rows, err = s.Queries.ListTasks(ctx, sqlc.ListTasksParams{
			Query:     query,
			Status:    status,
			DueBefore: dueBefore,
			DueAfter:  dueAfter,
		})
	}
	if err != nil {
		return nil, err
	}

	result := make([]model.Task, 0, len(rows))
	for _, row := range rows {
		tags, err := s.Queries.ListTagsForTask(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, mapTask(row, tags))
	}

	return result, nil
}

func (s *Store) SetTaskTags(ctx context.Context, taskID int64, tagNames []string) error {
	if err := s.Queries.ClearTagsForTask(ctx, taskID); err != nil {
		return err
	}

	for _, name := range normalizeTags(tagNames) {
		tag, err := s.Queries.CreateTag(ctx, name)
		if err != nil {
			return err
		}
		if err := s.Queries.AssignTagToTask(ctx, sqlc.AssignTagToTaskParams{TaskID: taskID, TagID: tag.ID}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) ListTags(ctx context.Context) ([]model.Tag, error) {
	rows, err := s.Queries.ListTags(ctx)
	if err != nil {
		return nil, err
	}

	tags := make([]model.Tag, 0, len(rows))
	for _, row := range rows {
		tags = append(tags, model.Tag{ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt})
	}
	return tags, nil
}

func (s *Store) DeleteTag(ctx context.Context, tagID int64) error {
	return s.Queries.DeleteTag(ctx, tagID)
}

func (s *Store) ListHistory(ctx context.Context, taskID int64) ([]model.HistoryEntry, error) {
	rows, err := s.Queries.ListHistoryByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	history := make([]model.HistoryEntry, 0, len(rows))
	for _, row := range rows {
		history = append(history, model.HistoryEntry{
			ID:        row.ID,
			TaskID:    row.TaskID,
			EventType: row.EventType,
			Details:   row.Details,
			CreatedAt: row.CreatedAt,
		})
	}
	return history, nil
}

func (s *Store) SaveView(ctx context.Context, view model.View) (model.View, error) {
	payload, err := json.Marshal(view.Filter)
	if err != nil {
		return model.View{}, err
	}

	if view.ID == 0 {
		created, err := s.Queries.CreateView(ctx, sqlc.CreateViewParams{Name: view.Name, FilterJson: string(payload)})
		if err != nil {
			return model.View{}, err
		}
		return mapView(created)
	}

	updated, err := s.Queries.UpdateView(ctx, sqlc.UpdateViewParams{
		ID:         view.ID,
		Name:       view.Name,
		FilterJson: string(payload),
	})
	if err != nil {
		return model.View{}, err
	}
	return mapView(updated)
}

func (s *Store) ListViews(ctx context.Context) ([]model.View, error) {
	rows, err := s.Queries.ListViews(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]model.View, 0, len(rows))
	for _, row := range rows {
		view, err := mapView(row)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}

	return views, nil
}

func (s *Store) DeleteView(ctx context.Context, viewID int64) error {
	return s.Queries.DeleteView(ctx, viewID)
}

func (s *Store) GetViewByName(ctx context.Context, name string) (model.View, error) {
	row, err := s.Queries.GetViewByName(ctx, name)
	if err != nil {
		return model.View{}, err
	}
	return mapView(row)
}

func mapTask(task sqlc.Task, tags []sqlc.Tag) model.Task {
	result := model.Task{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		Priority:    task.Priority,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
	if task.DueAt.Valid {
		result.DueAt = &task.DueAt.Time
	}
	if task.ParentTaskID.Valid {
		parentID := task.ParentTaskID.Int64
		result.ParentTaskID = &parentID
	}

	result.Tags = make([]model.Tag, 0, len(tags))
	for _, tag := range tags {
		result.Tags = append(result.Tags, model.Tag{ID: tag.ID, Name: tag.Name, CreatedAt: tag.CreatedAt})
	}

	return result
}

func mapView(row sqlc.View) (model.View, error) {
	var filter model.Filter
	if err := json.Unmarshal([]byte(row.FilterJson), &filter); err != nil {
		return model.View{}, err
	}
	return model.View{
		ID:        row.ID,
		Name:      row.Name,
		Filter:    filter,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

func normalizeStatus(status string) string {
	value := strings.TrimSpace(strings.ToLower(status))
	if value == "" {
		return "todo"
	}
	return value
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func formatCreatedDetails(task model.Task) string {
	return fmt.Sprintf("created: title='%s' status=%s priority=%d due=%s tags=%s", task.Title, task.Status, task.Priority, formatDue(task.DueAt), formatTags(task.Tags))
}

func formatDeletedDetails(task model.Task) string {
	return fmt.Sprintf("deleted: title='%s' status=%s priority=%d due=%s tags=%s", task.Title, task.Status, task.Priority, formatDue(task.DueAt), formatTags(task.Tags))
}

func formatTaskDiff(before, after model.Task) string {
	changes := []string{}
	if before.Title != after.Title {
		changes = append(changes, formatChange("title", before.Title, after.Title))
	}
	if before.Description != after.Description {
		changes = append(changes, formatChange("description", before.Description, after.Description))
	}
	if before.Status != after.Status {
		changes = append(changes, formatChange("status", before.Status, after.Status))
	}
	if before.Priority != after.Priority {
		changes = append(changes, formatChange("priority", fmt.Sprintf("%d", before.Priority), fmt.Sprintf("%d", after.Priority)))
	}
	if formatParent(before.ParentTaskID) != formatParent(after.ParentTaskID) {
		changes = append(changes, formatChange("parent", formatParent(before.ParentTaskID), formatParent(after.ParentTaskID)))
	}
	if formatDue(before.DueAt) != formatDue(after.DueAt) {
		changes = append(changes, formatChange("due", formatDue(before.DueAt), formatDue(after.DueAt)))
	}
	beforeTags := formatTags(before.Tags)
	afterTags := formatTags(after.Tags)
	if beforeTags != afterTags {
		changes = append(changes, formatChange("tags", beforeTags, afterTags))
	}

	if len(changes) == 0 {
		return "updated: no changes"
	}

	return "updated: " + strings.Join(changes, "; ")
}

func formatChange(field, before, after string) string {
	return fmt.Sprintf("%s: '%s' -> '%s'", field, valueOrNone(before), valueOrNone(after))
}

func valueOrNone(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "none"
	}
	return trimmed
}

func formatDue(value *time.Time) string {
	if value == nil {
		return "none"
	}
	return value.Format("2006-01-02")
}

func formatParent(parentID *int64) string {
	if parentID == nil || *parentID == 0 {
		return "none"
	}
	return fmt.Sprintf("%d", *parentID)
}

func formatTags(tags []model.Tag) string {
	if len(tags) == 0 {
		return "none"
	}

	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		names = append(names, tag.Name)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}
