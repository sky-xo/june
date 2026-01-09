package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Task represents a persistent task in the database
type Task struct {
	ID        string
	ParentID  *string // nil for root tasks
	Title     string
	Status    string // "open", "in_progress", "closed"
	Notes     *string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // nil = not deleted
	RepoPath  string
	Branch    string
}

// ErrTaskNotFound is returned when a task doesn't exist
var ErrTaskNotFound = fmt.Errorf("task not found")

// CreateTask inserts a new task into the database
func (d *DB) CreateTask(t Task) error {
	_, err := d.Exec(`
		INSERT INTO tasks (id, parent_id, title, status, notes, created_at, updated_at, deleted_at, repo_path, branch)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ParentID, t.Title, t.Status, t.Notes,
		t.CreatedAt.Format(time.RFC3339),
		t.UpdatedAt.Format(time.RFC3339),
		nil, // deleted_at
		t.RepoPath, t.Branch,
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

// GetTask retrieves a task by ID
func (d *DB) GetTask(id string) (*Task, error) {
	row := d.QueryRow(`
		SELECT id, parent_id, title, status, notes, created_at, updated_at, deleted_at, repo_path, branch
		FROM tasks WHERE id = ?`, id)

	var t Task
	var parentID, notes, deletedAt *string
	var createdAt, updatedAt string

	err := row.Scan(&t.ID, &parentID, &t.Title, &t.Status, &notes,
		&createdAt, &updatedAt, &deletedAt, &t.RepoPath, &t.Branch)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("scan task: %w", err)
	}

	t.ParentID = parentID
	t.Notes = notes
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if deletedAt != nil {
		dt, _ := time.Parse(time.RFC3339, *deletedAt)
		t.DeletedAt = &dt
	}

	return &t, nil
}

// TaskUpdate holds optional fields for updating a task
type TaskUpdate struct {
	Title  *string
	Status *string
	Notes  *string
}

// UpdateTask updates specified fields of a task
func (d *DB) UpdateTask(id string, update TaskUpdate) error {
	// Build dynamic query based on which fields are set
	setParts := []string{"updated_at = ?"}
	args := []any{time.Now().Format(time.RFC3339)}

	if update.Title != nil {
		setParts = append(setParts, "title = ?")
		args = append(args, *update.Title)
	}
	if update.Status != nil {
		setParts = append(setParts, "status = ?")
		args = append(args, *update.Status)
	}
	if update.Notes != nil {
		setParts = append(setParts, "notes = ?")
		args = append(args, *update.Notes)
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE tasks SET %s WHERE id = ? AND deleted_at IS NULL",
		strings.Join(setParts, ", "))

	result, err := d.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return ErrTaskNotFound
	}

	return nil
}
