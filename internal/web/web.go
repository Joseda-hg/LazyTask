package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Joseda-hg/lazytask/internal/db"
	"github.com/Joseda-hg/lazytask/internal/model"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var (
	indexTemplate = template.Must(template.ParseFS(templateFS, "templates/index.tmpl"))
	taskTemplate  = template.Must(template.ParseFS(templateFS, "templates/task.tmpl"))
)

type Server struct {
	store *db.Store
}

type taskRow struct {
	Task     model.Task
	IndentPx int
}

func NewServer(store *db.Store) *Server {
	return &Server{store: store}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.indexHandler)
	mux.HandleFunc("/tasks/", s.taskHandler)
	mux.HandleFunc("/api/tasks", s.apiTasksHandler)
	mux.HandleFunc("/api/tasks/", s.apiTaskHandler)
	return mux
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	filter := filterFromRequest(r)
	tasks, err := s.store.ListTasks(context.Background(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	rows := buildTaskRows(tasks)

	data := struct {
		Total int
		Rows  []taskRow
	}{Total: len(tasks), Rows: rows}

	if err := indexTemplate.Execute(w, data); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
}

func buildTaskRows(tasks []model.Task) []taskRow {
	if len(tasks) == 0 {
		return nil
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

	visible := make([]model.Task, 0, len(tasks))
	depthByID := make(map[int64]int, len(tasks))

	var walk func(parentID int64, depth int)
	walk = func(parentID int64, depth int) {
		for _, task := range childrenByParent[parentID] {
			visible = append(visible, task)
			depthByID[task.ID] = depth
			walk(task.ID, depth+1)
		}
	}
	walk(0, 0)

	rows := make([]taskRow, 0, len(visible))
	for _, task := range visible {
		rows = append(rows, taskRow{Task: task, IndentPx: depthByID[task.ID] * 20})
	}
	return rows
}

func (s *Server) taskHandler(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/tasks/")
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	task, err := s.store.GetTaskWithTags(context.Background(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	history, err := s.store.ListHistory(context.Background(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	data := struct {
		Task    model.Task
		History []model.HistoryEntry
	}{Task: task, History: history}

	if err := taskTemplate.Execute(w, data); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
}

func (s *Server) apiTasksHandler(w http.ResponseWriter, r *http.Request) {
	filter := filterFromRequest(r)
	tasks, err := s.store.ListTasks(context.Background(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, tasks)
}

func (s *Server) apiTaskHandler(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Path, "/api/tasks/")
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	task, err := s.store.GetTaskWithTags(context.Background(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}

	history, err := s.store.ListHistory(context.Background(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	payload := struct {
		Task    model.Task           `json:"task"`
		History []model.HistoryEntry `json:"history"`
	}{Task: task, History: history}

	writeJSON(w, payload)
}

func filterFromRequest(r *http.Request) model.Filter {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	var dueBefore *time.Time
	if value := strings.TrimSpace(r.URL.Query().Get("due_before")); value != "" {
		if parsed, err := time.Parse("2006-01-02", value); err == nil {
			dueBefore = &parsed
		}
	}

	var dueAfter *time.Time
	if value := strings.TrimSpace(r.URL.Query().Get("due_after")); value != "" {
		if parsed, err := time.Parse("2006-01-02", value); err == nil {
			dueAfter = &parsed
		}
	}

	var tags []string
	if value := strings.TrimSpace(r.URL.Query().Get("tags")); value != "" {
		for _, part := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}

	return model.Filter{Query: query, Status: status, Tags: tags, DueBefore: dueBefore, DueAfter: dueAfter}
}

func parseID(path, prefix string) (int64, error) {
	if !strings.HasPrefix(path, prefix) {
		return 0, fmt.Errorf("invalid path")
	}
	value := strings.TrimPrefix(path, prefix)
	value = strings.Trim(value, "/")
	if value == "" {
		return 0, fmt.Errorf("missing id")
	}
	return strconv.ParseInt(value, 10, 64)
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(err.Error()))
}
