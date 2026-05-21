package http

import (
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"task-management/internal/apperror"
	"task-management/internal/delivery/middleware"
	"task-management/internal/usecase"
)

type TaskHandler struct {
	taskUC    usecase.TaskUsecase
	validator *validator.Validate
}

func NewTaskHandler(taskUC usecase.TaskUsecase) *TaskHandler {
	return &TaskHandler{
		taskUC:    taskUC,
		validator: validator.New(),
	}
}

// @Summary     Create task
// @Description Create a new task with optional idempotency key for safe retries
// @Tags        tasks
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       Idempotency-Key header string false "Idempotency key (UUID v4)"
// @Param       request body CreateTaskRequest true "Create task payload"
// @Success     201 {object} SuccessResponse{data=TaskResponse}
// @Failure     400 {object} apperror.ErrorResponse
// @Failure     401 {object} apperror.ErrorResponse
// @Failure     422 {object} apperror.ErrorResponse
// @Router      /api/v1/tasks [post]
func (h *TaskHandler) Create(c echo.Context) error {
	var req CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return apperror.BadRequest("invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return apperror.ValidationError(err.Error())
	}

	userID := middleware.GetUserID(c)
	idempotencyKey := c.Request().Header.Get("Idempotency-Key")

	input := usecase.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
	}

	task, _, err := h.taskUC.CreateTask(c.Request().Context(), userID, input, idempotencyKey)
	if err != nil {
		return err
	}

	return Success(c, http.StatusCreated, "Task created successfully", ToTaskResponse(task))
}

// @Summary     List tasks
// @Description List tasks with optional filtering by status, search by title, and pagination
// @Tags        tasks
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       status query string false "Filter by status (pending, in_progress, completed)"
// @Param       search query string false "Search by title"
// @Param       page query int false "Page number" default(1)
// @Param       limit query int false "Items per page" default(10)
// @Success     200 {object} PaginatedResponse{data=[]TaskResponse}
// @Failure     401 {object} apperror.ErrorResponse
// @Router      /api/v1/tasks [get]
func (h *TaskHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	status := c.QueryParam("status")
	search := c.QueryParam("search")

	tasks, total, err := h.taskUC.ListTasks(c.Request().Context(), userID, status, search, page, limit)
	if err != nil {
		return err
	}

	result := make([]TaskResponse, len(tasks))
	for i, t := range tasks {
		result[i] = ToTaskResponse(&t)
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	return Paginated(c, "Tasks retrieved successfully", result, page, limit, total)
}

// @Summary     Get task
// @Description Get a single task by ID
// @Tags        tasks
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id path string true "Task ID"
// @Success     200 {object} SuccessResponse{data=TaskResponse}
// @Failure     404 {object} apperror.ErrorResponse
// @Router      /api/v1/tasks/{id} [get]
func (h *TaskHandler) Get(c echo.Context) error {
	task, err := h.taskUC.GetTask(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}
	return Success(c, http.StatusOK, "Task retrieved successfully", ToTaskResponse(task))
}

// @Summary     Update task
// @Description Update task fields (title, description, status). Only the task owner can update.
// @Tags        tasks
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id path string true "Task ID"
// @Param       request body UpdateTaskRequest true "Update payload"
// @Success     200 {object} SuccessResponse{data=TaskResponse}
// @Failure     403 {object} apperror.ErrorResponse
// @Failure     404 {object} apperror.ErrorResponse
// @Router      /api/v1/tasks/{id} [put]
func (h *TaskHandler) Update(c echo.Context) error {
	var req UpdateTaskRequest
	if err := c.Bind(&req); err != nil {
		return apperror.BadRequest("invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return apperror.ValidationError(err.Error())
	}

	userID := middleware.GetUserID(c)
	input := usecase.UpdateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	}

	task, err := h.taskUC.UpdateTask(c.Request().Context(), userID, c.Param("id"), input)
	if err != nil {
		return err
	}

	return Success(c, http.StatusOK, "Task updated successfully", ToTaskResponse(task))
}

// @Summary     Delete task
// @Description Soft-delete a task. Only the task owner can delete.
// @Tags        tasks
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id path string true "Task ID"
// @Success     200 {object} SuccessResponse
// @Failure     403 {object} apperror.ErrorResponse
// @Failure     404 {object} apperror.ErrorResponse
// @Router      /api/v1/tasks/{id} [delete]
func (h *TaskHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if err := h.taskUC.DeleteTask(c.Request().Context(), userID, c.Param("id")); err != nil {
		return err
	}
	return Success(c, http.StatusOK, "Task deleted successfully", nil)
}

// @Summary     Assign task
// @Description Assign a task to another user in the same team. Single database transaction.
// @Tags        tasks
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id path string true "Task ID"
// @Param       request body AssignTaskRequest true "Assign payload"
// @Success     200 {object} SuccessResponse{data=TaskResponse}
// @Failure     403 {object} apperror.ErrorResponse
// @Failure     404 {object} apperror.ErrorResponse
// @Router      /api/v1/tasks/{id}/assign [post]
func (h *TaskHandler) Assign(c echo.Context) error {
	var req AssignTaskRequest
	if err := c.Bind(&req); err != nil {
		return apperror.BadRequest("invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return apperror.ValidationError(err.Error())
	}

	userID := middleware.GetUserID(c)
	if err := h.taskUC.AssignTask(c.Request().Context(), userID, c.Param("id"), req.UserID); err != nil {
		return err
	}

	task, err := h.taskUC.GetTask(c.Request().Context(), c.Param("id"))
	if err != nil {
		return err
	}

	return Success(c, http.StatusOK, "Task assigned successfully", ToTaskResponse(task))
}
