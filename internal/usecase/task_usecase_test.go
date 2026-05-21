package usecase_test

import (
	"context"
	"sync"
	"testing"

	"task-management/internal/domain"
	"task-management/internal/repository"
	"task-management/internal/usecase"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubTxManager struct{}

func (m *stubTxManager) Run(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	return fn(nil)
}

type stubTaskRepo struct {
	tasks map[string]*domain.Task
	mu    sync.Mutex
}

func newStubTaskRepo() *stubTaskRepo {
	return &stubTaskRepo{tasks: make(map[string]*domain.Task)}
}

func (s *stubTaskRepo) Create(ctx context.Context, tx *sqlx.Tx, task *domain.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	task.ID = "task-" + string(rune(len(s.tasks)+1+'0'))
	s.tasks[task.ID] = task
	return nil
}

func (s *stubTaskRepo) FindByID(ctx context.Context, id string) (*domain.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok || task.DeletedAt != nil {
		return nil, nil
	}
	return task, nil
}

func (s *stubTaskRepo) FindByUserID(ctx context.Context, userID string, filter repository.TaskFilter) ([]domain.Task, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []domain.Task
	for _, t := range s.tasks {
		if t.UserID == userID && t.DeletedAt == nil {
			result = append(result, *t)
		}
	}
	return result, int64(len(result)), nil
}

func (s *stubTaskRepo) Update(ctx context.Context, tx *sqlx.Tx, task *domain.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
	return nil
}

func (s *stubTaskRepo) SoftDelete(ctx context.Context, tx *sqlx.Tx, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return nil
	}
	now := task.CreatedAt
	task.DeletedAt = &now
	return nil
}

func (s *stubTaskRepo) FindByIDForUpdate(ctx context.Context, tx *sqlx.Tx, id string) (*domain.Task, error) {
	return s.FindByID(ctx, id)
}

func (s *stubTaskRepo) UpdateAssignee(ctx context.Context, tx *sqlx.Tx, taskID string, assigneeID *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return nil
	}
	task.AssigneeID = assigneeID
	return nil
}

type stubIdempotencyRepo struct {
	keys    map[string]*repository.StoredResponse
	mu      sync.Mutex
	barrier chan struct{}
}

func newStubIdempotencyRepo() *stubIdempotencyRepo {
	return &stubIdempotencyRepo{
		keys:    make(map[string]*repository.StoredResponse),
		barrier: make(chan struct{}, 1),
	}
}

func (s *stubIdempotencyRepo) ClaimKey(ctx context.Context, tx *sqlx.Tx, key, userID string) (bool, *repository.StoredResponse, error) {
	s.mu.Lock()

	cached, exists := s.keys[key]
	if !exists {
		s.keys[key] = &repository.StoredResponse{}
		s.mu.Unlock()
		return true, nil, nil
	}

	if cached.Status != 0 {
		s.mu.Unlock()
		return false, cached, nil
	}

	s.mu.Unlock()
	<-s.barrier

	s.mu.Lock()
	cached = s.keys[key]
	s.mu.Unlock()
	return false, cached, nil
}

func (s *stubIdempotencyRepo) StoreResponse(ctx context.Context, tx *sqlx.Tx, key string, status int, body string) error {
	s.mu.Lock()
	s.keys[key] = &repository.StoredResponse{Status: status, Body: body}
	s.mu.Unlock()
	close(s.barrier)
	return nil
}

func (s *stubIdempotencyRepo) PurgeExpired(ctx context.Context) error { return nil }

type stubTaskLogRepoForTask struct{}

func (s *stubTaskLogRepoForTask) Create(ctx context.Context, tx *sqlx.Tx, log *domain.TaskLog) error {
	return nil
}

type stubUserRepoForTask struct {
	users map[string]*domain.User
}

func newStubUserRepoForTask() *stubUserRepoForTask {
	return &stubUserRepoForTask{
		users: map[string]*domain.User{
			"user-1": {ID: "user-1", TeamID: strPtr("team-1")},
			"user-2": {ID: "user-2", TeamID: strPtr("team-1")},
			"user-3": {ID: "user-3", TeamID: strPtr("team-2")},
		},
	}
}

func (s *stubUserRepoForTask) Create(ctx context.Context, user *domain.User) error { return nil }
func (s *stubUserRepoForTask) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil
}
func (s *stubUserRepoForTask) FindByID(ctx context.Context, id string) (*domain.User, error) {
	u, ok := s.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (s *stubUserRepoForTask) UpdateTeamID(ctx context.Context, userID string, teamID *string) error {
	if u, ok := s.users[userID]; ok {
		u.TeamID = teamID
	}
	return nil
}

func (s *stubUserRepoForTask) FindTeamByCode(ctx context.Context, code string) (*domain.Team, error) {
	return &domain.Team{ID: "team-1", Code: code, Name: "Test"}, nil
}

func strPtr(s string) *string { return &s }

func setupTaskUsecase() (usecase.TaskUsecase, *stubTaskRepo, *stubIdempotencyRepo) {
	taskRepo := newStubTaskRepo()
	idempRepo := newStubIdempotencyRepo()
	taskLogRepo := &stubTaskLogRepoForTask{}
	userRepo := newStubUserRepoForTask()
	uc := usecase.NewTaskUsecase(&stubTxManager{}, taskRepo, idempRepo, taskLogRepo, userRepo)
	return uc, taskRepo, idempRepo
}

func TestCreateTask_Success(t *testing.T) {
	uc, _, _ := setupTaskUsecase()

	task, isNew, err := uc.CreateTask(context.Background(), "user-1", usecase.CreateTaskInput{
		Title: "Test Task", Description: "Test",
	}, "550e8400-e29b-41d4-a716-446655440001")

	require.NoError(t, err)
	assert.True(t, isNew)
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, "Test Task", task.Title)
	assert.Equal(t, domain.TaskStatusPending, task.Status)
}

func TestCreateTask_SequentialIdempotency(t *testing.T) {
	uc, _, _ := setupTaskUsecase()

	key := "550e8400-e29b-41d4-a716-446655440000"
	input := usecase.CreateTaskInput{Title: "Idempotent", Description: "Test"}
	userID := "user-1"

	task1, isNew1, err1 := uc.CreateTask(context.Background(), userID, input, key)
	require.NoError(t, err1)
	assert.True(t, isNew1)

	task2, isNew2, err2 := uc.CreateTask(context.Background(), userID, input, key)
	require.NoError(t, err2)
	assert.False(t, isNew2)

	assert.Equal(t, task1.ID, task2.ID)
	assert.Equal(t, task1.Title, task2.Title)
}

func TestCreateTask_ConcurrentIdempotency(t *testing.T) {
	taskRepo := newStubTaskRepo()
	idempRepo := newStubIdempotencyRepo()

	uc := usecase.NewTaskUsecase(&stubTxManager{}, taskRepo, idempRepo, &stubTaskLogRepoForTask{}, newStubUserRepoForTask())

	key := "concurrent-key"
	input := usecase.CreateTaskInput{Title: "Concurrent", Description: "Test"}
	userID := "user-1"

	const numGoroutines = 100
	var wg sync.WaitGroup
	results := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			task, _, err := uc.CreateTask(context.Background(), userID, input, key)
			if err != nil {
				results <- "error:" + err.Error()
				return
			}
			results <- task.ID
		}()
	}

	wg.Wait()
	close(results)

	var taskIDs []string
	var firstID string
	for id := range results {
		assert.NotContains(t, id, "error:")
		if firstID == "" {
			firstID = id
		}
		taskIDs = append(taskIDs, id)
	}

	assert.Len(t, taskIDs, numGoroutines)
	assert.Equal(t, 1, len(taskRepo.tasks))
	for _, id := range taskIDs {
		assert.Equal(t, firstID, id)
	}
}

func TestListTasks_Empty(t *testing.T) {
	uc, _, _ := setupTaskUsecase()

	result, err := uc.ListTasks(context.Background(), "user-1", "", "", 1, 10)
	require.NoError(t, err)
	assert.Empty(t, result.Tasks)
	assert.Equal(t, int64(0), result.Total)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.Limit)
}

func TestGetTask_NotFound(t *testing.T) {
	uc, _, _ := setupTaskUsecase()

	_, err := uc.GetTask(context.Background(), "user-1", "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestUpdateTask_Forbidden(t *testing.T) {
	uc, taskRepo, _ := setupTaskUsecase()

	taskRepo.Create(context.Background(), nil, &domain.Task{
		ID: "task-1", UserID: "owner", Title: "Mine", Status: domain.TaskStatusPending,
	})

	_, err := uc.UpdateTask(context.Background(), "someone-else", "task-1", usecase.UpdateTaskInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "you can only update your own tasks")
}

func TestUpdateTask_Success(t *testing.T) {
	uc, taskRepo, _ := setupTaskUsecase()

	taskRepo.Create(context.Background(), nil, &domain.Task{
		ID: "task-1", UserID: "user-1", Title: "Original", Status: domain.TaskStatusPending,
	})

	newTitle := "Updated"
	status := domain.TaskStatusInProgress
	updated, err := uc.UpdateTask(context.Background(), "user-1", "task-1", usecase.UpdateTaskInput{
		Title:  &newTitle,
		Status: &status,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Title)
	assert.Equal(t, domain.TaskStatusInProgress, updated.Status)
}

func TestDeleteTask_Success(t *testing.T) {
	uc, taskRepo, _ := setupTaskUsecase()

	taskRepo.Create(context.Background(), nil, &domain.Task{
		ID: "task-1", UserID: "user-1", Title: "Delete me", Status: domain.TaskStatusPending,
	})

	err := uc.DeleteTask(context.Background(), "user-1", "task-1")
	require.NoError(t, err)

	_, err = uc.GetTask(context.Background(), "user-1", "task-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestAssignTask_Success(t *testing.T) {
	uc, taskRepo, _ := setupTaskUsecase()

	taskRepo.Create(context.Background(), nil, &domain.Task{
		ID: "task-1", UserID: "user-1", Title: "Assign me", Status: domain.TaskStatusPending,
	})

	err := uc.AssignTask(context.Background(), "user-1", "task-1", "user-2")
	require.NoError(t, err)

	task, _ := uc.GetTask(context.Background(), "user-1", "task-1")
	require.NotNil(t, task)
	assert.Equal(t, "user-2", *task.AssigneeID)
}

func TestAssignTask_WrongTeam(t *testing.T) {
	uc, taskRepo, _ := setupTaskUsecase()

	taskRepo.Create(context.Background(), nil, &domain.Task{
		ID: "task-1", UserID: "user-1", Title: "Assign me", Status: domain.TaskStatusPending,
	})

	err := uc.AssignTask(context.Background(), "user-1", "task-1", "user-3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "assignee must be in the same team")
}

func TestAssignTask_OwnerHasNoTeam(t *testing.T) {
	taskRepo := newStubTaskRepo()
	userRepo := &stubUserRepoForTask{
		users: map[string]*domain.User{
			"user-1": {ID: "user-1", TeamID: nil},
			"user-2": {ID: "user-2", TeamID: strPtr("team-1")},
		},
	}
	uc := usecase.NewTaskUsecase(&stubTxManager{}, taskRepo, newStubIdempotencyRepo(), &stubTaskLogRepoForTask{}, userRepo)

	taskRepo.Create(context.Background(), nil, &domain.Task{
		ID: "task-1", UserID: "user-1", Title: "No team", Status: domain.TaskStatusPending,
	})

	err := uc.AssignTask(context.Background(), "user-1", "task-1", "user-2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "you must join a team before assigning tasks")
}

func TestAssignTask_SelfAssign(t *testing.T) {
	uc, taskRepo, _ := setupTaskUsecase()

	taskRepo.Create(context.Background(), nil, &domain.Task{
		ID: "task-1", UserID: "user-1", Title: "Assign me", Status: domain.TaskStatusPending,
	})

	err := uc.AssignTask(context.Background(), "user-1", "task-1", "user-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot assign task to yourself")
}

func TestAssignTask_TaskNotFound(t *testing.T) {
	uc, _, _ := setupTaskUsecase()

	err := uc.AssignTask(context.Background(), "user-1", "nonexistent", "user-2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestAssignTask_AssigneeNotFound(t *testing.T) {
	uc, taskRepo, _ := setupTaskUsecase()

	taskRepo.Create(context.Background(), nil, &domain.Task{
		ID: "task-1", UserID: "user-1", Title: "Assign me", Status: domain.TaskStatusPending,
	})

	err := uc.AssignTask(context.Background(), "user-1", "task-1", "user-999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "assignee not found")
}
