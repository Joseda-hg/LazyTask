package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Joseda-hg/lazytask/internal/db"
	"github.com/Joseda-hg/lazytask/internal/model"
)

type formField struct {
	Label string
	Value string
}

const (
	fieldTitle = iota
	fieldDescription
	fieldStatus
	fieldTags
	fieldDue
	fieldPriority
)

func buildFormFields(task *model.Task) []formField {
	fields := []formField{
		{Label: "Title"},
		{Label: "Description"},
		{Label: "Status (space/←→)"},
		{Label: "Tags (space/←→)"},
		{Label: "Due (YYYY-MM-DD)"},
		{Label: "Priority"},
	}

	if task == nil {
		fields[fieldStatus].Value = "todo"
		fields[fieldPriority].Value = "0"
		return fields
	}

	fields[fieldTitle].Value = task.Title
	fields[fieldDescription].Value = task.Description
	fields[fieldStatus].Value = task.Status
	fields[fieldPriority].Value = strconv.FormatInt(task.Priority, 10)
	if task.DueAt != nil {
		fields[fieldDue].Value = task.DueAt.Format("2006-01-02")
	}
	fields[fieldTags].Value = joinTags(task.Tags)

	return fields
}

func parseFormFields(fields []formField) (db.TaskInput, error) {
	priority, err := parsePriority(fields[fieldPriority].Value)
	if err != nil {
		return db.TaskInput{}, err
	}

	dueAt, err := parseDue(fields[fieldDue].Value)
	if err != nil {
		return db.TaskInput{}, err
	}

	return db.TaskInput{
		Title:       strings.TrimSpace(fields[fieldTitle].Value),
		Description: strings.TrimSpace(fields[fieldDescription].Value),
		Status:      strings.TrimSpace(fields[fieldStatus].Value),
		Priority:    priority,
		DueAt:       dueAt,
		Tags:        parseTags(fields[fieldTags].Value),
	}, nil
}

func parsePriority(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid priority")
	}
	return parsed, nil
}

func parseDue(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid due date")
	}
	return &parsed, nil
}

func parseTags(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func joinTags(tags []model.Tag) string {
	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		names = append(names, tag.Name)
	}
	return strings.Join(names, ",")
}

func taskInputFromTask(task model.Task) db.TaskInput {
	return db.TaskInput{
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		Priority:    task.Priority,
		DueAt:       task.DueAt,
		Tags:        parseTags(joinTags(task.Tags)),
	}
}
