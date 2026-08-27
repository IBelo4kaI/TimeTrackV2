package vacation

import (
	"database/sql"
	"errors"
	"net/http"
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

	err := h.service.CreateVacationReport(c.RequestCtx(), body)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Created(c)
}

func (h *Handler) GetVacationsByYear(c fiber.Ctx) error {
	year, err := fiber.Params[int](c, "year"), error(nil)

	if err != nil {
		return response.BadRequest(c)
	}

	userId := c.Params("userId")

	vacations, err := h.service.GetVacationsByYear(c.RequestCtx(), userId, year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
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
		return response.Error(c, http.StatusInternalServerError, err)
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
	year, err := fiber.Params[int](c, "year"), error(nil)

	if err != nil {
		return response.BadRequest(c)
	}

	vacations, err := h.service.GetAllUserVacationsByYear(c.RequestCtx(), year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, vacations)
}

// ListVacationCalendar godoc
// GET /vacation/calendar/:year
// Урезанный (без description) список отпусков ВСЕХ сотрудников — для
// виджета "отпуска коллег". Отдельное разрешение time:vacation_calendar:read,
// см. комментарий в route.go.
func (h *Handler) ListVacationCalendar(c fiber.Ctx) error {
	year, err := fiber.Params[int](c, "year"), error(nil)
	if err != nil {
		return response.BadRequest(c)
	}

	vacations, err := h.service.ListVacationCalendarByYear(c.RequestCtx(), year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
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
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, result)
}

func (h *Handler) GetVacationStatistics(c fiber.Ctx) error {
	year, err := fiber.Params[int](c, "year"), error(nil)

	if err != nil {
		return response.BadRequest(c)
	}

	userId := c.Params("userId")

	stats, err := h.service.GetVacationsStats(c.RequestCtx(), userId, year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, stats)
}

func (h *Handler) ApproveVacation(c fiber.Ctx) error {
	vacationID := c.Params("id")

	if vacationID == "" {
		return response.BadRequest(c)
	}

	// Используем UpdateVacationStatus с статусом "approved"
	err := h.service.UpdateVacationStatus(c.RequestCtx(), vacationID, repo.VacationsStatusApproved)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
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
		Status string `json:"status"`
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

	err := h.service.UpdateVacationStatus(c.RequestCtx(), vacationID, status)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Статус обновлен",
	})
}

func (h *Handler) UpdateVacationType(c fiber.Ctx) error {
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

	err := h.service.UpdateVacationType(c.RequestCtx(), vacationID, body.VacationTypeID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
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
		return response.Error(c, http.StatusInternalServerError, err)
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
		return response.Error(c, http.StatusInternalServerError, err)
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
		return response.Error(c, http.StatusInternalServerError, err)
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
		return response.ServerError(c)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"id":           f.ID,
		"originalName": f.OriginalName,
		"mimeType":     f.MimeType,
		"fileType":     f.FileType,
		"sizeBytes":    f.SizeBytes,
	})
}
