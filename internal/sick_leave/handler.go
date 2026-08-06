package sickleave

import (
	"net/http"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service     Service
	fileService *service.FileService
}

func NewHandler(svc Service, fileService *service.FileService) *Handler {
	return &Handler{service: svc, fileService: fileService}
}

func (h *Handler) CreateSickLeave(c fiber.Ctx) error {
	var body struct {
		UserID      string    `json:"userId"`
		StartDate   time.Time `json:"startDate"`
		EndDate     time.Time `json:"endDate"`
		Description string    `json:"description"`
		Status      string    `json:"status"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	if body.StartDate.After(body.EndDate) {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "startDate не может быть позже endDate"))
	}

	status := repo.SickLeavesStatus(body.Status)
	if status != repo.SickLeavesStatusOfficial && status != repo.SickLeavesStatusUnofficial {
		status = repo.SickLeavesStatusUnofficial
	}

	if err := h.service.CreateSickLeave(c.RequestCtx(), CreateSickLeaveParams{
		UserID:      body.UserID,
		StartDate:   body.StartDate,
		EndDate:     body.EndDate,
		Description: body.Description,
		Status:      status,
	}); err != nil {
		return response.ServerError(c)
	}

	return response.Created(c)
}

func (h *Handler) GetSickLeavesByYear(c fiber.Ctx) error {
	year, err := fiber.Params[int](c, "year"), error(nil)
	if err != nil {
		return response.BadRequest(c)
	}
	userID := c.Params("userId")

	rows, err := h.service.GetSickLeavesByYear(c.RequestCtx(), userID, year)
	if err != nil {
		return response.ServerError(c)
	}
	return response.Success(c, rows)
}

func (h *Handler) GetAllUsersSickLeavesByYear(c fiber.Ctx) error {
	year, err := fiber.Params[int](c, "year"), error(nil)
	if err != nil {
		return response.BadRequest(c)
	}

	rows, err := h.service.GetAllUsersSickLeavesByYear(c.RequestCtx(), year)
	if err != nil {
		return response.ServerError(c)
	}
	return response.Success(c, rows)
}

func (h *Handler) UpdateSickLeaveStatus(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	status := repo.SickLeavesStatus(body.Status)
	if status != repo.SickLeavesStatusOfficial && status != repo.SickLeavesStatusUnofficial {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "допустимые статусы: official, unofficial"))
	}

	if err := h.service.UpdateSickLeaveStatus(c.RequestCtx(), id, status); err != nil {
		return response.ServerError(c)
	}
	return response.Updated(c)
}

func (h *Handler) DeleteSickLeave(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	if err := h.service.DeleteSickLeave(c.RequestCtx(), id); err != nil {
		return response.ServerError(c)
	}
	return response.Deleted(c)
}

// UploadSickLeaveFile загружает файл и привязывает его к больничному через file_entity_refs.
// Файлы доступны через GET /v1/files/open/:id и листаются через GET /v1/files/entity/sick_leave/:id.
func (h *Handler) UploadSickLeaveFile(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "ID больничного не указан"))
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "файл не найден в запросе"))
	}

	uploaderID, _ := c.Locals("user_id").(string)

	f, err := h.fileService.Upload(c.RequestCtx(), service.UploadFileParams{
		File:       fileHeader,
		EntityType: "sick_leave",
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
