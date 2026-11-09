package vacation

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"timetrack/internal/adapter/grpc"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/middleware"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service     Service
	fileService *service.FileService
	grpc        *grpc.Client
	prefix      string
}

func NewHandler(service Service, fileService *service.FileService, grpc *grpc.Client, prefix string) *Handler {
	return &Handler{service: service, fileService: fileService, grpc: grpc, prefix: prefix}
}

func (h *Handler) CreateVacation(c fiber.Ctx) error {
	var body VacationCreateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}

	if body.StartDate.IsZero() || body.EndDate.IsZero() || body.StartDate.After(body.EndDate) {
		return response.Error(c, 400, errors.New("Некорректный период отпуска"))
	}
	if body.Status == "" {
		body.Status = repo.VacationsStatusPending
	}
	if body.Status != repo.VacationsStatusPending && !middleware.HasAll(c, h.grpc, middleware.Params{Service: h.prefix, Entity: "vacation", Action: "edit"}) {
		return response.Error(c, 403, errors.New("Сотрудник может создать только заявку на рассмотрении"))
	}
	if body.Status != repo.VacationsStatusPending && body.Status != repo.VacationsStatusApproved {
		return response.Error(c, 400, errors.New("Некорректный начальный статус"))
	}
	err := h.service.CreateVacationReport(actionContext(c, ""), body)
	if err != nil {
		return workflowResponseError(c, err)
	}

	return response.Created(c)
}

func (h *Handler) GetVacationsByYear(c fiber.Ctx) error {
	year, err := strconv.Atoi(c.Params("year"))

	if err != nil || year < 1 || year > 9999 {
		return response.BadRequest(c)
	}

	userId := c.Params("userId")

	vacations, err := h.service.GetVacationsByYear(c.RequestCtx(), userId, year)
	if err != nil {
		return workflowResponseError(c, err)
	}

	return response.Success(c, vacations)
}

// GetVacation godoc
// GET /v1/vacation/:id — карточка отдельной заявки (для страницы заявления).
//
// Роут без :userId в пути (запись ищется по её собственному id), поэтому
// middleware.Require на этот роут проверяет только базовое "vacation:read" —
// permission-сервис не знает заранее, чья это заявка, и не может сам
// подставить ".all". Владельца узнаём только после того, как достали запись
// из БД, и если она чужая — довалидируем через middleware.RequireOwnerOrAll.
func (h *Handler) GetVacation(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	vacation, err := h.service.GetVacationByID(c.RequestCtx(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return response.Error(c, http.StatusNotFound, fiber.NewError(http.StatusNotFound, "заявка на отпуск не найдена"))
		}
		return workflowResponseError(c, err)
	}

	callerID, _ := c.Locals("user_id").(string)
	allowed := middleware.RequireOwnerOrAll(
		c,
		h.grpc,
		middleware.Params{Service: h.prefix, Entity: "vacation", Action: "read"},
		callerID,
		vacation.UserID,
	)
	if !allowed {
		return response.Error(c, http.StatusForbidden, fiber.NewError(http.StatusForbidden, "нет доступа к этой заявке"))
	}

	return response.Success(c, vacation)
}

func (h *Handler) GetAllUserVacationsByYear(c fiber.Ctx) error {
	year, err := strconv.Atoi(c.Params("year"))

	if err != nil || year < 1 || year > 9999 {
		return response.BadRequest(c)
	}

	vacations, err := h.service.GetAllUserVacationsByYear(c.RequestCtx(), year)
	if err != nil {
		return workflowResponseError(c, err)
	}

	return response.Success(c, vacations)
}

// ListVacationCalendar godoc
// GET /vacation/calendar/:year
// Урезанный (без description) список отпусков ВСЕХ сотрудников — для
// виджета "отпуска коллег". Отдельное разрешение time:vacation_calendar:read,
// см. комментарий в route.go.
func (h *Handler) ListVacationCalendar(c fiber.Ctx) error {
	year, err := strconv.Atoi(c.Params("year"))
	if err != nil || year < 1 || year > 9999 {
		return response.BadRequest(c)
	}

	vacations, err := h.service.ListVacationCalendarByYear(c.RequestCtx(), year)
	if err != nil {
		return workflowResponseError(c, err)
	}

	return response.Success(c, vacations)
}

func (h *Handler) CalculateVacationDays(c fiber.Ctx) error {
	// Получаем параметры из query string
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	// Проверяем, что параметры переданы
	if startDateStr == "" || endDateStr == "" {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Необходимо указать startDate и endDate параметры"))
	}

	// Парсим даты
	const dateLayout = "2006-01-02"
	startDate, err := time.Parse(dateLayout, startDateStr)
	if err != nil {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Некорректный формат startDate. Используйте YYYY-MM-DD"))
	}

	endDate, err := time.Parse(dateLayout, endDateStr)
	if err != nil {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Некорректный формат endDate. Используйте YYYY-MM-DD"))
	}

	// Проверяем, что startDate <= endDate
	if startDate.After(endDate) {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "startDate не может быть позже endDate"))
	}

	// Вызываем сервис
	result, err := h.service.CalculateVacationDays(c.RequestCtx(), startDate, endDate)
	if err != nil {
		return workflowResponseError(c, err)
	}

	return response.Success(c, result)
}

func (h *Handler) CalculateVacationEndDate(c fiber.Ctx) error {
	startDateStr := c.Query("startDate")
	daysStr := c.Query("days")

	if startDateStr == "" || daysStr == "" {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Необходимо указать startDate и days параметры"))
	}

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Некорректный формат startDate. Используйте YYYY-MM-DD"))
	}

	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "days должен быть положительным целым числом"))
	}

	result, err := h.service.CalculateVacationEndDate(c.RequestCtx(), startDate, days)
	if err != nil {
		return workflowResponseError(c, err)
	}

	return response.Success(c, result)
}

func (h *Handler) GetVacationStatistics(c fiber.Ctx) error {
	year, err := strconv.Atoi(c.Params("year"))

	if err != nil || year < 1 || year > 9999 {
		return response.BadRequest(c)
	}

	userId := c.Params("userId")

	stats, err := h.service.GetVacationsStats(c.RequestCtx(), userId, year)
	if err != nil {
		return workflowResponseError(c, err)
	}

	return response.Success(c, stats)
}

func (h *Handler) ApproveVacation(c fiber.Ctx) error {
	vacationID := c.Params("id")

	if vacationID == "" {
		return response.BadRequest(c)
	}

	// Используем UpdateVacationStatus с статусом "approved"
	err := h.service.UpdateVacationStatus(actionContext(c, c.Query("comment")), vacationID, repo.VacationsStatusApproved)
	if err != nil {
		return workflowResponseError(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Отпуск подтвержден",
	})
}

func (h *Handler) UpdateVacationStatus(c fiber.Ctx) error {
	vacationID := c.Params("id")

	if vacationID == "" {
		return response.BadRequest(c)
	}

	var body struct {
		Status  string `json:"status"`
		Comment string `json:"comment"`
	}

	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	// Валидация статуса
	var status repo.VacationsStatus
	switch body.Status {
	case "pending":
		status = repo.VacationsStatusPending
	case "approved":
		status = repo.VacationsStatusApproved
	case "rejected":
		status = repo.VacationsStatusRejected
	default:
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Invalid status. Must be one of: pending, approved, rejected"))
	}

	err := h.service.UpdateVacationStatus(actionContext(c, body.Comment), vacationID, status)
	if err != nil {
		return workflowResponseError(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Статус обновлен",
	})
}

func (h *Handler) UpdateVacationType(c fiber.Ctx) error {
	if _, err := h.applicationAccess(c, "edit"); err != nil {
		return workflowResponseError(c, err)
	}
	vacationID := c.Params("id")

	if vacationID == "" {
		return response.BadRequest(c)
	}

	var body struct {
		VacationTypeID string `json:"vacationTypeId"`
	}

	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	if body.VacationTypeID == "" {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "vacationTypeId обязателен"))
	}

	err := h.service.UpdateVacationType(actionContext(c, ""), vacationID, body.VacationTypeID)
	if err != nil {
		return workflowResponseError(c, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Тип отпуска обновлён",
	})
}

// DeleteVacation godoc
// DELETE /v1/vacation/:id
//
// Как и GetVacation, роут без :userId в пути — базовый middleware.Require
// проверяет только "vacation:delete", владельца узнаём после чтения записи
// и довалидируем через RequireOwnerOrAll. Дополнительно: свою заявку можно
// удалить только пока она "на рассмотрении" — чужую (т.е. только через
// .all, значит уже подтверждённый админ) можно удалить в любом статусе.
func (h *Handler) DeleteVacation(c fiber.Ctx) error {
	vacationID := c.Params("id")

	if vacationID == "" {
		return response.BadRequest(c)
	}

	vacation, err := h.service.GetVacationByID(c.RequestCtx(), vacationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return response.Error(c, http.StatusNotFound, fiber.NewError(http.StatusNotFound, "заявка на отпуск не найдена"))
		}
		return workflowResponseError(c, err)
	}

	callerID, _ := c.Locals("user_id").(string)
	allowed := middleware.RequireOwnerOrAll(
		c,
		h.grpc,
		middleware.Params{Service: h.prefix, Entity: "vacation", Action: "delete"},
		callerID,
		vacation.UserID,
	)
	if !allowed {
		return response.Error(c, http.StatusForbidden, fiber.NewError(http.StatusForbidden, "нет доступа к этой заявке"))
	}

	if vacation.UserID == callerID && vacation.Status != repo.VacationsStatusPending {
		return response.Error(c, http.StatusForbidden, fiber.NewError(http.StatusForbidden, "удалить можно только заявку на рассмотрении"))
	}

	err = h.service.DeleteVacation(c.RequestCtx(), vacationID)
	if err != nil {
		return workflowResponseError(c, err)
	}

	if err := h.fileService.DeleteByEntity(c.RequestCtx(), "vacation", vacationID); err != nil {
		fmt.Printf("delete vacation files: %v\n", err)
	}

	return response.Success(c, fiber.Map{
		"message": "Заявка на отпуск удалена",
	})
}

// UploadVacationFile загружает файл и привязывает его к отпуску через file_entity_refs.
// Файлы доступны через GET /v1/files/open/:id и листаются через GET /v1/files/entity/vacation/:id.
//
// Как и GetVacation/DeleteVacation, роут без :userId в пути — базовый
// middleware.Require проверяет только "vacation:edit", владельца узнаём
// после чтения записи и довалидируем через RequireOwnerOrAll: свою заявку
// можно дополнить файлом всегда, чужую — только с vacation.all:edit.
func (h *Handler) UploadVacationFile(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "ID отпуска не указан"))
	}

	vacation, err := h.service.GetVacationByID(c.RequestCtx(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return response.Error(c, http.StatusNotFound, fiber.NewError(http.StatusNotFound, "заявка на отпуск не найдена"))
		}
		return workflowResponseError(c, err)
	}

	uploaderID, _ := c.Locals("user_id").(string)
	allowed := middleware.RequireOwnerOrAll(
		c,
		h.grpc,
		middleware.Params{Service: h.prefix, Entity: "vacation", Action: "edit"},
		uploaderID,
		vacation.UserID,
	)
	if !allowed {
		return response.Error(c, http.StatusForbidden, fiber.NewError(http.StatusForbidden, "нет доступа к этой заявке"))
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "файл не найден в запросе"))
	}

	f, err := h.fileService.Upload(c.RequestCtx(), service.UploadFileParams{
		File:       fileHeader,
		EntityType: "vacation",
		EntityID:   id,
		UploaderID: uploaderID,
	})
	if err != nil {
		return workflowResponseError(c, err)
	}

	// Если заявка уже утверждена — этот файл может быть тем самым сканом,
	// которого не хватало для письма (см. SendApprovalEmailIfReady).
	// Best-effort — не должно ронять сам аплоад.
	if err := h.service.SendApprovalEmailIfReady(c.RequestCtx(), id); err != nil {
		fmt.Printf("vacation: send approval email failed: %v\n", err)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"id":           f.ID,
		"originalName": f.OriginalName,
		"mimeType":     f.MimeType,
		"fileType":     f.FileType,
		"sizeBytes":    f.SizeBytes,
	})
}

func workflowResponseError(c fiber.Ctx, err error) error {
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return response.Error(c, fe.Code, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return response.Error(c, 404, errors.New("Заявка не найдена"))
	}
	return response.Error(c, 500, err)
}
func (h *Handler) applicationAccess(c fiber.Ctx, action string) (*repo.GetVacationByIDRow, error) {
	v, err := h.service.GetVacationByID(c.RequestCtx(), c.Params("id"))
	if err != nil {
		return nil, err
	}
	caller, _ := c.Locals("user_id").(string)
	if !middleware.RequireOwnerOrAll(c, h.grpc, middleware.Params{Service: h.prefix, Entity: "vacation", Action: action}, caller, v.UserID) {
		return nil, fiber.NewError(403, "Нет доступа к заявке")
	}
	return v, nil
}
func (h *Handler) History(c fiber.Ctx) error {
	if _, err := h.applicationAccess(c, "read"); err != nil {
		return workflowResponseError(c, err)
	}
	events, err := h.service.GetHistory(c.RequestCtx(), c.Params("id"))
	if err != nil {
		return workflowResponseError(c, err)
	}
	return response.Success(c, historyResponse(events))
}
func (h *Handler) EditApplication(c fiber.Ctx) error {
	if _, err := h.applicationAccess(c, "edit"); err != nil {
		return workflowResponseError(c, err)
	}
	var body VacationCreateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}
	if err := h.service.EditApplication(actionContext(c, ""), c.Params("id"), body); err != nil {
		return workflowResponseError(c, err)
	}
	return response.Updated(c)
}
func (h *Handler) Withdraw(c fiber.Ctx) error {
	v, err := h.applicationAccess(c, "edit")
	if err != nil {
		return workflowResponseError(c, err)
	}
	caller, _ := c.Locals("user_id").(string)
	if caller != v.UserID {
		return response.Error(c, 403, errors.New("Отозвать заявку может только её автор"))
	}
	if err := h.service.UpdateVacationStatus(actionContext(c, ""), v.ID, repo.VacationsStatusWithdrawn); err != nil {
		return workflowResponseError(c, err)
	}
	return response.Updated(c)
}
