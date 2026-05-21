package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"

	"task-management/internal/domain"
	"task-management/internal/repository"
	"task-management/pkg/utils/autotx"
	"task-management/pkg/utils/response"
)

type ListResult struct {
	Tasks []domain.Task
	Total int64
	Page  int
	Limit int
}

type TaskUsecase interface {
	CreateTask(ctx context.Context, userID string, input CreateTaskInput, idempotencyKey string) (*domain.Task, bool, error)
	ListTasks(ctx context.Context, userID string, status, search string, page, limit int) (*ListResult, error)
	GetTask(ctx context.Context, userID, id string) (*domain.Task, error)
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
	txManager       autotx.Manager
	taskRepo        repository.TaskRepository
	idempotencyRepo repository.IdempotencyRepository
	taskLogRepo     repository.TaskLogRepository
	userRepo        repository.UserRepository
}

func NewTaskUsecase(
	txManager autotx.Manager,
	taskRepo repository.TaskRepository,
	idempotencyRepo repository.IdempotencyRepository,
	taskLogRepo repository.TaskLogRepository,
	userRepo repository.UserRepository,
) TaskUsecase {
	return &taskUsecase{
		txManager:       txManager,
		taskRepo:        taskRepo,
		idempotencyRepo: idempotencyRepo,
		taskLogRepo:     taskLogRepo,
		userRepo:        userRepo,
	}
}

func (uc *taskUsecase) CreateTask(ctx context.Context, userID string, input CreateTaskInput, idempotencyKey string) (*domain.Task, bool, error) {

	var taskResult *domain.Task
	var created bool

	err := uc.txManager.Run(ctx, func(tx *sqlx.Tx) error {
		claimed, cached, err := uc.idempotencyRepo.ClaimKey(ctx, tx, idempotencyKey, userID)
		if err != nil {
			return response.Internal("failed to check idempotency key", err)
		}

		if !claimed {
			var t domain.Task
			if err := json.Unmarshal([]byte(cached.Body), &t); err != nil {
				return response.Internal("failed to read cached response", err)
			}
			taskResult = &t
			return nil
		}

		task := &domain.Task{
			UserID:      userID,
			Title:       input.Title,
			Description: input.Description,
			Status:      domain.TaskStatusPending,
		}
		if err := uc.taskRepo.Create(ctx, tx, task); err != nil {
			return response.Internal("failed to create task", err)
		}

		body, err := json.Marshal(task)
		if err != nil {
			return response.Internal("failed to serialize task", err)
		}

		if err := uc.idempotencyRepo.StoreResponse(ctx, tx, idempotencyKey, 201, string(body)); err != nil {
			return response.Internal("failed to store idempotency response", err)
		}

		taskResult = task
		created = true
		return nil
	})

	if err != nil {
		return nil, false, err
	}

	return taskResult, created, nil
}

func (uc *taskUsecase) ListTasks(ctx context.Context, userID string, status, search string, page, limit int) (*ListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	filter := repository.TaskFilter{
		Search: search,
		Page:   page,
		Limit:  limit,
	}
	if status != "" {
		s := domain.TaskStatus(status)
		if !s.Valid() {
			return nil, response.ValidationError("invalid task status: " + status)
		}
		filter.Status = s
	}

	tasks, total, err := uc.taskRepo.FindByUserID(ctx, userID, filter)
	if err != nil {
		return nil, response.Internal("failed to list tasks", err)
	}

	if tasks == nil {
		tasks = []domain.Task{}
	}

	return &ListResult{
		Tasks: tasks,
		Total: total,
		Page:  page,
		Limit: limit,
	}, nil
}

func (uc *taskUsecase) GetTask(ctx context.Context, userID, id string) (*domain.Task, error) {
	task, err := uc.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, response.Internal("failed to get task", err)
	}
	if task == nil {
		return nil, response.NotFound("task not found")
	}
	if task.UserID != userID && (task.AssigneeID == nil || *task.AssigneeID != userID) {
		return nil, response.NotFound("task not found")
	}
	return task, nil
}

func (uc *taskUsecase) UpdateTask(ctx context.Context, userID, id string, input UpdateTaskInput) (*domain.Task, error) {
	task, err := uc.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, response.Internal("failed to get task", err)
	}
	if task == nil {
		return nil, response.NotFound("task not found")
	}

	isOwner := task.UserID == userID
	isAssignee := task.AssigneeID != nil && *task.AssigneeID == userID
	if !isOwner && !isAssignee {
		return nil, response.Forbidden("you can only update your own tasks")
	}

	if input.Title != nil {
		task.Title = *input.Title
	}
	if input.Description != nil {
		task.Description = *input.Description
	}
	if input.Status != nil {
		if !input.Status.Valid() {
			return nil, response.ValidationError("invalid task status")
		}
		task.Status = *input.Status
	}

	if err := uc.taskRepo.Update(ctx, nil, task); err != nil {
		return nil, response.Internal("failed to update task", err)
	}

	return task, nil
}

func (uc *taskUsecase) DeleteTask(ctx context.Context, userID, id string) error {
	task, err := uc.taskRepo.FindByID(ctx, id)
	if err != nil {
		return response.Internal("failed to get task", err)
	}
	if task == nil {
		return response.NotFound("task not found")
	}
	if task.UserID != userID {
		return response.Forbidden("you can only delete your own tasks")
	}

	if err := uc.taskRepo.SoftDelete(ctx, nil, id); err != nil {
		return response.Internal("failed to delete task", err)
	}

	return nil
}

func (uc *taskUsecase) AssignTask(ctx context.Context, userID, taskID, newAssigneeID string) error {
	return uc.txManager.Run(ctx, func(tx *sqlx.Tx) error {
		return uc.assignTask(ctx, userID, taskID, newAssigneeID, tx)
	})
}

func (uc *taskUsecase) assignTask(ctx context.Context, userID, taskID, newAssigneeID string, tx *sqlx.Tx) error {
	task, err := uc.taskRepo.FindByIDForUpdate(ctx, tx, taskID)
	if err != nil {
		return response.Internal("failed to get task", err)
	}
	if task == nil {
		return response.NotFound("task not found")
	}
	if task.UserID != userID {
		return response.Forbidden("you can only assign your own tasks")
	}

	if newAssigneeID == userID {
		return response.Forbidden("cannot assign task to yourself")
	}

	assignee, err := uc.userRepo.FindByID(ctx, newAssigneeID)
	if err != nil {
		return response.Internal("failed to find assignee", err)
	}
	if assignee == nil {
		return response.NotFound("assignee not found")
	}

	taskOwner, err := uc.userRepo.FindByID(ctx, task.UserID)
	if err != nil {
		return response.Internal("failed to find task owner", err)
	}
	if taskOwner == nil {
		return response.NotFound("task owner not found")
	}

	if taskOwner.TeamID == nil {
		return response.Forbidden("you must join a team before assigning tasks")
	}

	if assignee.TeamID == nil {
		return response.Forbidden("assignee must join a team first")
	}

	if *taskOwner.TeamID != *assignee.TeamID {
		return response.Forbidden("assignee must be in the same team")
	}

	oldAssigneeID := task.AssigneeID
	if err := uc.taskRepo.UpdateAssignee(ctx, tx, taskID, &newAssigneeID); err != nil {
		return response.Internal("failed to update assignee", err)
	}

	logEntry := &domain.TaskLog{
		TaskID:    taskID,
		Action:    "assign",
		ChangedBy: userID,
	}
	if oldAssigneeID != nil {
		v := fmt.Sprintf(`{"assignee_id":"%s"}`, *oldAssigneeID)
		logEntry.OldValue = &v
	}
	nv := fmt.Sprintf(`{"assignee_id":"%s"}`, newAssigneeID)
	logEntry.NewValue = &nv

	if err := uc.taskLogRepo.Create(ctx, tx, logEntry); err != nil {
		return response.Internal("failed to create task log", err)
	}

	from := "unassigned"
	if oldAssigneeID != nil {
		from = *oldAssigneeID
	}

	assignedByName := userID
	assignedToName := newAssigneeID
	if assigner, _ := uc.userRepo.FindByID(ctx, userID); assigner != nil {
		assignedByName = assigner.Name
	}
	if assigneeUser, _ := uc.userRepo.FindByID(ctx, newAssigneeID); assigneeUser != nil {
		assignedToName = assigneeUser.Name
	}

	slog.Info("TASK_ASSIGNED",
		slog.String("task_id", taskID),
		slog.String("task_title", task.Title),
		slog.String("assigned_by", assignedByName),
		slog.String("assigned_to", assignedToName),
		slog.String("previous_assignee", from),
		slog.String("timestamp", time.Now().UTC().Format(time.RFC3339)),
	)

	return nil
}
