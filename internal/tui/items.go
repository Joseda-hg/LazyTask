package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Joseda-hg/lazytask/internal/model"
)

type tagCountEntry struct {
	ID    int64
	Name  string
	Count int
}

func formatTags(tags []model.Tag) string {
	if len(tags) == 0 {
		return "no tags"
	}
	parts := make([]string, 0, len(tags))
	for _, tag := range tags {
		parts = append(parts, tag.Name)
	}
	return strings.Join(parts, ",")
}

func formatTaskSummary(task model.Task) string {
	return fmt.Sprintf("%s | %s | p%d | %s", task.Title, task.Status, task.Priority, formatTags(task.Tags))
}

func buildVisibleTaskTree(tasks []model.Task, collapsed map[int64]bool) ([]model.Task, map[int64]int, map[int64]bool) {
	if len(tasks) == 0 {
		return nil, map[int64]int{}, map[int64]bool{}
	}

	indexByID := make(map[int64]int, len(tasks))
	existsByID := make(map[int64]struct{}, len(tasks))
	for i, task := range tasks {
		indexByID[task.ID] = i
		existsByID[task.ID] = struct{}{}
	}

	childrenByParent := make(map[int64][]model.Task)
	for _, task := range tasks {
		parentID := int64(0)
		if task.ParentTaskID != nil {
			if _, ok := existsByID[*task.ParentTaskID]; ok {
				parentID = *task.ParentTaskID
			}
		}
		childrenByParent[parentID] = append(childrenByParent[parentID], task)
	}

	for parentID := range childrenByParent {
		children := childrenByParent[parentID]
		sort.Slice(children, func(i, j int) bool {
			return indexByID[children[i].ID] < indexByID[children[j].ID]
		})
		childrenByParent[parentID] = children
	}

	hasChildren := make(map[int64]bool, len(childrenByParent))
	for parentID, children := range childrenByParent {
		if parentID == 0 {
			continue
		}
		if len(children) > 0 {
			hasChildren[parentID] = true
		}
	}

	visible := make([]model.Task, 0, len(tasks))
	depthByID := make(map[int64]int, len(tasks))

	var walk func(parentID int64, depth int)
	walk = func(parentID int64, depth int) {
		for _, task := range childrenByParent[parentID] {
			visible = append(visible, task)
			depthByID[task.ID] = depth
			if hasChildren[task.ID] && collapsed != nil && collapsed[task.ID] {
				continue
			}
			walk(task.ID, depth+1)
		}
	}

	walk(0, 0)
	return visible, depthByID, hasChildren
}
