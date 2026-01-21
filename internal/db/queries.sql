-- name: CreateTask :one
INSERT INTO tasks (title, description, status, priority, due_at, parent_task_id)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, parent_task_id, title, description, status, priority, due_at, created_at, updated_at;

-- name: UpdateTask :one
UPDATE tasks
SET title = ?,
    description = ?,
    status = ?,
    priority = ?,
    due_at = ?,
    parent_task_id = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, parent_task_id, title, description, status, priority, due_at, created_at, updated_at;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = ?;

-- name: GetTask :one
SELECT id, parent_task_id, title, description, status, priority, due_at, created_at, updated_at
FROM tasks
WHERE id = ?;

-- name: ListTasks :many
SELECT id, parent_task_id, title, description, status, priority, due_at, created_at, updated_at
FROM tasks
WHERE (sqlc.arg(query) = '' OR title LIKE '%' || sqlc.arg(query) || '%' OR description LIKE '%' || sqlc.arg(query) || '%')
  AND (sqlc.arg(status) = '' OR status = sqlc.arg(status))
  AND (sqlc.arg(due_before) IS NULL OR due_at <= sqlc.arg(due_before))
  AND (sqlc.arg(due_after) IS NULL OR due_at >= sqlc.arg(due_after))
ORDER BY created_at DESC;

-- name: ListTasksByTags :many
SELECT DISTINCT tasks.id, tasks.parent_task_id, tasks.title, tasks.description, tasks.status, tasks.priority, tasks.due_at, tasks.created_at, tasks.updated_at
FROM tasks
JOIN task_tags ON task_tags.task_id = tasks.id
JOIN tags ON tags.id = task_tags.tag_id
WHERE tags.name IN (sqlc.slice('tags'))
  AND (sqlc.arg(query) = '' OR tasks.title LIKE '%' || sqlc.arg(query) || '%' OR tasks.description LIKE '%' || sqlc.arg(query) || '%')
ORDER BY tasks.created_at DESC;

-- name: CreateTag :one
INSERT INTO tags (name)
VALUES (?)
ON CONFLICT(name) DO UPDATE SET name = excluded.name
RETURNING id, name, created_at;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?;

-- name: ListTags :many
SELECT id, name, created_at FROM tags ORDER BY name ASC;

-- name: GetTagByName :one
SELECT id, name, created_at FROM tags WHERE name = ?;

-- name: AssignTagToTask :exec
INSERT OR IGNORE INTO task_tags (task_id, tag_id)
VALUES (?, ?);

-- name: RemoveTagFromTask :exec
DELETE FROM task_tags WHERE task_id = ? AND tag_id = ?;

-- name: ClearTagsForTask :exec
DELETE FROM task_tags WHERE task_id = ?;

-- name: ListTagsForTask :many
SELECT tags.id, tags.name, tags.created_at
FROM tags
JOIN task_tags ON task_tags.tag_id = tags.id
WHERE task_tags.task_id = ?
ORDER BY tags.name ASC;

-- name: AddHistory :one
INSERT INTO task_history (task_id, event_type, details)
VALUES (?, ?, ?)
RETURNING id, task_id, event_type, details, created_at;

-- name: ListHistoryByTask :many
SELECT id, task_id, event_type, details, created_at
FROM task_history
WHERE task_id = ?
ORDER BY created_at DESC;

-- name: CreateView :one
INSERT INTO views (name, filter_json)
VALUES (?, ?)
RETURNING id, name, filter_json, created_at, updated_at;

-- name: UpdateView :one
UPDATE views
SET name = ?,
    filter_json = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, name, filter_json, created_at, updated_at;

-- name: DeleteView :exec
DELETE FROM views WHERE id = ?;

-- name: ListViews :many
SELECT id, name, filter_json, created_at, updated_at
FROM views
ORDER BY name ASC;

-- name: GetViewByName :one
SELECT id, name, filter_json, created_at, updated_at
FROM views
WHERE name = ?;
