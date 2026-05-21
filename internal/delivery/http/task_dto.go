package http

import (
	"time"

	"task-management/internal/domain"
)

type CreateTaskRequest struct {
	Title       string `json:"title" validate:"required,min=1,max=255"`
	Description string `json:"description" validate:"max=1000"`
}

type UpdateTaskRequest struct {
	Title       *string            `json:"title" validate:"omitempty,min=1,max=255"`
	Description *string            `json:"description" validate:"omitempty,max=1000"`
	Status      *domain.TaskStatus `json:"status" validate:"omitempty,oneof=pending in_progress completed"`
}

type TaskResponse struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	AssigneeID  *string           `json:"assignee_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      domain.TaskStatus `json:"status"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

type AssignTaskRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

func ToTaskResponse(t *domain.Task) TaskResponse {
	createdAt := ""
	updatedAt := ""
	if !t.CreatedAt.IsZero() {
		createdAt = t.CreatedAt.UTC().Format(time.RFC3339)
	}
	if !t.UpdatedAt.IsZero() {
		updatedAt = t.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return TaskResponse{
		ID:          t.ID,
		UserID:      t.UserID,
		AssigneeID:  t.AssigneeID,
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}
