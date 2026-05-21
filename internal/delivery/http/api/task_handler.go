package api

import (
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"task-management/pkg/utils/response"
	"task-management/internal/delivery/http/dto"
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
// @Param       request body dto.CreateTaskRequest true "Create task payload"
// @Success     201 {object} object
// @Failure     400 {object} object
// @Failure     401 {object} object
// @Failure     422 {object} object
// @Router      /api/v1/tasks [post]
func (h *TaskHandler) Create(c echo.Context) error {
	var req dto.CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest("invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return response.ValidationError(err.Error())
	}

	userID := middleware.GetUserID(c)
	idempotencyKey := c.Request().Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		return response.BadRequest("Idempotency-Key header is required")
	}

	input := usecase.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
	}

	task, _, err := h.taskUC.CreateTask(c.Request().Context(), userID, input, idempotencyKey)
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusCreated, "Task created successfully", dto.ToTaskResponse(task))
}

// @Summary     List tasks
// @Description List tasks with optional filtering by status, search by title, and pagination
// @Tags        tasks
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       status query string false "Filter: pending, in_progress, completed"
// @Param       search query string false "Search by title"
// @Param       page query int false "Page number" default(1)
// @Param       limit query int false "Items per page" default(10)
// @Success     200 {object} object
// @Failure     401 {object} object
// @Router      /api/v1/tasks [get]
func (h *TaskHandler) List(c echo.Context) error {
	userID := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	status := c.QueryParam("status")
	search := c.QueryParam("search")

	result, err := h.taskUC.ListTasks(c.Request().Context(), userID, status, search, page, limit)
	if err != nil {
		return err
	}

	taskList := make([]dto.TaskResponse, len(result.Tasks))
	for i, t := range result.Tasks {
		taskList[i] = dto.ToTaskResponse(&t)
	}

	return response.Paginated(c, "Tasks retrieved successfully", taskList, result.Page, result.Limit, result.Total)
}

// @Summary     Get task
// @Description Get a single task by ID
// @Tags        tasks
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id path string true "Task ID"
// @Success     200 {object} object
// @Failure     404 {object} object
// @Router      /api/v1/tasks/{id} [get]
func (h *TaskHandler) Get(c echo.Context) error {
	task, err := h.taskUC.GetTask(c.Request().Context(), middleware.GetUserID(c), c.Param("id"))
	if err != nil {
		return err
	}
	return response.Success(c, http.StatusOK, "Task updated successfully", dto.ToTaskResponse(task))
}

// @Summary     Update task
// @Description Partial update — only send fields you want to change. Status: pending, in_progress, completed. Example: update just status without title/description.
// @Tags        tasks
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id path string true "Task ID"
// @Param       request body dto.UpdateTaskRequest true "Status: pending, in_progress, completed"
// @Success     200 {object} object
// @Failure     403 {object} object
// @Failure     404 {object} object
// @Router      /api/v1/tasks/{id} [put]
func (h *TaskHandler) Update(c echo.Context) error {
	var req dto.UpdateTaskRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest("invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return response.ValidationError(err.Error())
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

	return response.Success(c, http.StatusOK, "Task updated successfully", dto.ToTaskResponse(task))
}

// @Summary     Delete task
// @Description Soft-delete a task. Only the task owner can delete.
// @Tags        tasks
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id path string true "Task ID"
// @Success     200 {object} object
// @Failure     403 {object} object
// @Failure     404 {object} object
// @Router      /api/v1/tasks/{id} [delete]
func (h *TaskHandler) Delete(c echo.Context) error {
	userID := middleware.GetUserID(c)
	if err := h.taskUC.DeleteTask(c.Request().Context(), userID, c.Param("id")); err != nil {
		return err
	}
	return response.Success(c, http.StatusOK, "Task deleted successfully", nil)
}

// @Summary     Assign task
// @Description Assign a task to another user in the same team. Single database transaction.
// @Tags        tasks
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       id path string true "Task ID"
// @Param       request body dto.AssignTaskRequest true "Assign payload"
// @Success     200 {object} object
// @Failure     403 {object} object
// @Failure     404 {object} object
// @Router      /api/v1/tasks/{id}/assign [post]
func (h *TaskHandler) Assign(c echo.Context) error {
	var req dto.AssignTaskRequest
	if err := c.Bind(&req); err != nil {
		return response.BadRequest("invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return response.ValidationError(err.Error())
	}

	userID := middleware.GetUserID(c)
	if err := h.taskUC.AssignTask(c.Request().Context(), userID, c.Param("id"), req.UserID); err != nil {
		return err
	}

	task, err := h.taskUC.GetTask(c.Request().Context(), middleware.GetUserID(c), c.Param("id"))
	if err != nil {
		return err
	}

	return response.Success(c, http.StatusOK, "Task updated successfully", dto.ToTaskResponse(task))
}
