package handler

import (
	"net/http"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v2"
)

type SickLeaveHandler struct {
	service     service.SickLeaveService
	fileService *service.FileService
}

func NewSickLeaveHandler(svc service.SickLeaveService, fileService *service.FileService) *SickLeaveHandler {
	return &SickLeaveHandler{service: svc, fileService: fileService}
}

func (h *SickLeaveHandler) CreateSickLeave(c *fiber.Ctx) error {
	var body struct {
		UserID      string    `json:"userId"`
		StartDate   time.Time `json:"startDate"`
		EndDate     time.Time `json:"endDate"`
		Description string    `json:"description"`
		Status      string    `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c)
	}

	if body.StartDate.After(body.EndDate) {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "startDate не может быть позже endDate"))
	}

	status := repo.SickLeavesStatus(body.Status)
	if status != repo.SickLeavesStatusOfficial && status != repo.SickLeavesStatusUnofficial {
		status = repo.SickLeavesStatusUnofficial
	}

	if err := h.service.CreateSickLeave(c.Context(), service.CreateSickLeaveParams{
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

func (h *SickLeaveHandler) GetSickLeavesByYear(c *fiber.Ctx) error {
	year, err := c.ParamsInt("year")
	if err != nil {
		return response.BadRequest(c)
	}
	userID := c.Params("userId")

	rows, err := h.service.GetSickLeavesByYear(c.Context(), userID, year)
	if err != nil {
		return response.ServerError(c)
	}
	return response.Success(c, rows)
}

func (h *SickLeaveHandler) GetAllUsersSickLeavesByYear(c *fiber.Ctx) error {
	year, err := c.ParamsInt("year")
	if err != nil {
		return response.BadRequest(c)
	}

	rows, err := h.service.GetAllUsersSickLeavesByYear(c.Context(), year)
	if err != nil {
		return response.ServerError(c)
	}
	return response.Success(c, rows)
}

func (h *SickLeaveHandler) UpdateSickLeaveStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c)
	}

	status := repo.SickLeavesStatus(body.Status)
	if status != repo.SickLeavesStatusOfficial && status != repo.SickLeavesStatusUnofficial {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "допустимые статусы: official, unofficial"))
	}

	if err := h.service.UpdateSickLeaveStatus(c.Context(), id, status); err != nil {
		return response.ServerError(c)
	}
	return response.Updated(c)
}

func (h *SickLeaveHandler) DeleteSickLeave(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	if err := h.service.DeleteSickLeave(c.Context(), id); err != nil {
		return response.ServerError(c)
	}
	return response.Deleted(c)
}

// UploadSickLeaveFile загружает файл и привязывает его к больничному через file_entity_refs.
// Файлы доступны через GET /v1/files/open/:id и листаются через GET /v1/files/entity/sick_leave/:id.
func (h *SickLeaveHandler) UploadSickLeaveFile(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "ID больничного не указан"))
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "файл не найден в запросе"))
	}

	uploaderID, _ := c.Locals("user_id").(string)

	f, err := h.fileService.Upload(c.Context(), service.UploadFileParams{
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
