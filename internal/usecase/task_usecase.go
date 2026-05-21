package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"task-management/internal/apperror"
	"task-management/internal/domain"
	"task-management/internal/repository"
)

type TaskUsecase interface {
	CreateTask(ctx context.Context, userID string, input CreateTaskInput, idempotencyKey string) (*domain.Task, bool, error)
	ListTasks(ctx context.Context, userID string, status, search string, page, limit int) ([]domain.Task, int64, error)
	GetTask(ctx context.Context, id string) (*domain.Task, error)
	UpdateTask(ctx context.Context, userID, id string, input UpdateTaskInput) (*domain.Task, error)
	DeleteTask(ctx context.Context, userID, id string) error
	AssignTask(ctx context.Context, userID, taskID, newAssigneeID string) error
}

type CreateTaskInput struct {
	Title       string
	Description string
}

type UpdateTaskInput struct {
	Title       *string
	Description *string
	Status      *domain.TaskStatus
}

type taskUsecase struct {
	db              *sqlx.DB
	taskRepo        repository.TaskRepository
	idempotencyRepo repository.IdempotencyRepository
	taskLogRepo     repository.TaskLogRepository
	userRepo        repository.UserRepository
}

func NewTaskUsecase(
	db *sqlx.DB,
	taskRepo repository.TaskRepository,
	idempotencyRepo repository.IdempotencyRepository,
	taskLogRepo repository.TaskLogRepository,
	userRepo repository.UserRepository,
) TaskUsecase {
	return &taskUsecase{
		db:              db,
		taskRepo:        taskRepo,
		idempotencyRepo: idempotencyRepo,
		taskLogRepo:     taskLogRepo,
		userRepo:        userRepo,
	}
}

func (uc *taskUsecase) CreateTask(ctx context.Context, userID string, input CreateTaskInput, idempotencyKey string) (*domain.Task, bool, error) {
	if idempotencyKey == "" {
		return uc.createTaskSimple(ctx, userID, input)
	}
	return uc.createTaskIdempotent(ctx, userID, input, idempotencyKey)
}

func (uc *taskUsecase) createTaskSimple(ctx context.Context, userID string, input CreateTaskInput) (*domain.Task, bool, error) {
	task := &domain.Task{
		UserID:      userID,
		Title:       input.Title,
		Description: input.Description,
		Status:      domain.TaskStatusPending,
	}
	if err := uc.taskRepo.Create(ctx, nil, task); err != nil {
		return nil, false, apperror.Internal("failed to create task", err)
	}
	return task, true, nil
}

func (uc *taskUsecase) createTaskIdempotent(ctx context.Context, userID string, input CreateTaskInput, idempotencyKey string) (*domain.Task, bool, error) {
	tx, err := uc.beginTx(ctx)
	if err != nil {
		return nil, false, apperror.Internal("failed to begin transaction", err)
	}
	if tx != nil {
		defer tx.Rollback()
	}

	claimed, cached, err := uc.idempotencyRepo.ClaimKey(ctx, tx, idempotencyKey, userID)
	if err != nil {
		return nil, false, apperror.Internal("failed to check idempotency key", err)
	}

	if !claimed {
		var task domain.Task
		if err := json.Unmarshal([]byte(cached.Body), &task); err != nil {
			return nil, false, apperror.Internal("failed to read cached response", err)
		}
		return &task, false, nil
	}

	task := &domain.Task{
		UserID:      userID,
		Title:       input.Title,
		Description: input.Description,
		Status:      domain.TaskStatusPending,
	}
	if err := uc.taskRepo.Create(ctx, tx, task); err != nil {
		return nil, false, apperror.Internal("failed to create task", err)
	}

	body, err := json.Marshal(task)
	if err != nil {
		return nil, false, apperror.Internal("failed to serialize task", err)
	}

	if err := uc.idempotencyRepo.StoreResponse(ctx, tx, idempotencyKey, 201, string(body)); err != nil {
		return nil, false, apperror.Internal("failed to store idempotency response", err)
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, false, apperror.Internal("failed to commit transaction", err)
		}
	}

	return task, true, nil
}

func (uc *taskUsecase) beginTx(ctx context.Context) (*sqlx.Tx, error) {
	if uc.db == nil {
		return nil, nil
	}
	return uc.db.BeginTxx(ctx, nil)
}

func (uc *taskUsecase) ListTasks(ctx context.Context, userID string, status, search string, page, limit int) ([]domain.Task, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	filter := repository.TaskFilter{
		Search: search,
		Page:   page,
		Limit:  limit,
	}
	if status != "" {
		s := domain.TaskStatus(status)
		if !s.Valid() {
			return nil, 0, apperror.ValidationError("invalid task status: " + status)
		}
		filter.Status = s
	}

	tasks, total, err := uc.taskRepo.FindByUserID(ctx, userID, filter)
	if err != nil {
		return nil, 0, apperror.Internal("failed to list tasks", err)
	}

	if tasks == nil {
		tasks = []domain.Task{}
	}

	return tasks, total, nil
}

func (uc *taskUsecase) GetTask(ctx context.Context, id string) (*domain.Task, error) {
	task, err := uc.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.Internal("failed to get task", err)
	}
	if task == nil {
		return nil, apperror.NotFound("task not found")
	}
	return task, nil
}

func (uc *taskUsecase) UpdateTask(ctx context.Context, userID, id string, input UpdateTaskInput) (*domain.Task, error) {
	task, err := uc.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperror.Internal("failed to get task", err)
	}
	if task == nil {
		return nil, apperror.NotFound("task not found")
	}
	if task.UserID != userID {
		return nil, apperror.Forbidden("you can only update your own tasks")
	}

	if input.Title != nil {
		task.Title = *input.Title
	}
	if input.Description != nil {
		task.Description = *input.Description
	}
	if input.Status != nil {
		if !input.Status.Valid() {
			return nil, apperror.ValidationError("invalid task status")
		}
		task.Status = *input.Status
	}

	if err := uc.taskRepo.Update(ctx, nil, task); err != nil {
		return nil, apperror.Internal("failed to update task", err)
	}

	return task, nil
}

func (uc *taskUsecase) DeleteTask(ctx context.Context, userID, id string) error {
	task, err := uc.taskRepo.FindByID(ctx, id)
	if err != nil {
		return apperror.Internal("failed to get task", err)
	}
	if task == nil {
		return apperror.NotFound("task not found")
	}
	if task.UserID != userID {
		return apperror.Forbidden("you can only delete your own tasks")
	}

	if err := uc.taskRepo.SoftDelete(ctx, nil, id); err != nil {
		return apperror.Internal("failed to delete task", err)
	}

	return nil
}

func (uc *taskUsecase) AssignTask(ctx context.Context, userID, taskID, newAssigneeID string) error {
	tx, err := uc.db.BeginTxx(ctx, nil)
	if err != nil {
		return apperror.Internal("failed to begin transaction", err)
	}
	defer tx.Rollback()

	task, err := uc.taskRepo.FindByIDForUpdate(ctx, tx, taskID)
	if err != nil {
		return apperror.Internal("failed to get task", err)
	}
	if task == nil {
		return apperror.NotFound("task not found")
	}
	if task.UserID != userID {
		return apperror.Forbidden("you can only assign your own tasks")
	}

	assignee, err := uc.userRepo.FindByID(ctx, newAssigneeID)
	if err != nil {
		return apperror.Internal("failed to find assignee", err)
	}
	if assignee == nil {
		return apperror.NotFound("assignee not found")
	}

	taskOwner, err := uc.userRepo.FindByID(ctx, task.UserID)
	if err != nil {
		return apperror.Internal("failed to find task owner", err)
	}
	if taskOwner == nil {
		return apperror.NotFound("task owner not found")
	}

	if taskOwner.TeamID == nil || assignee.TeamID == nil || *taskOwner.TeamID != *assignee.TeamID {
		return apperror.Forbidden("assignee must be in the same team")
	}

	if err := func() error {
		oldAssigneeID := task.AssigneeID
		if err := uc.taskRepo.UpdateAssignee(ctx, tx, taskID, &newAssigneeID); err != nil {
			return apperror.Internal("failed to update assignee", err)
		}

		logEntry := &domain.TaskLog{
			TaskID:    taskID,
			Action:    "assign",
			ChangedBy: userID,
		}
		if oldAssigneeID != nil {
			v := *oldAssigneeID
			logEntry.OldValue = &v
		}
		logEntry.NewValue = &newAssigneeID

		if err := uc.taskLogRepo.Create(ctx, tx, logEntry); err != nil {
			return apperror.Internal("failed to create task log", err)
		}

		slog.Info("notification: task assigned",
			"task_id", taskID,
			"from", fmt.Sprintf("%v", oldAssigneeID),
			"to", newAssigneeID,
		)

		return nil
	}(); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return apperror.Internal("failed to commit transaction", err)
	}

	return nil
}
