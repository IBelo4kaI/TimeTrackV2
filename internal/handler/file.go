package handler

import (
	"errors"
	"net/http"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v2"
)

type FileHandler struct {
	service *service.FileService
}

func NewFileHandler(fileService *service.FileService) *FileHandler {
	return &FileHandler{service: fileService}
}

// UploadFile godoc
// POST /v1/files/upload
// Form fields: file (required), entity_type (optional), entity_id (optional)
func (h *FileHandler) UploadFile(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "файл не найден в запросе"))
	}

	uploaderID := c.Locals("user_id")
	uploaderIDStr, _ := uploaderID.(string)

	f, err := h.service.Upload(c.Context(), service.UploadFileParams{
		File:       fileHeader,
		EntityType: c.FormValue("entity_type"),
		EntityID:   c.FormValue("entity_id"),
		UploaderID: uploaderIDStr,
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

// OpenFile godoc
// GET /v1/files/open/:id
func (h *FileHandler) OpenFile(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	f, err := h.service.GetFile(c.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			return response.Error(c, http.StatusNotFound, fiber.NewError(http.StatusNotFound, "файл не найден"))
		}
		return response.ServerError(c)
	}

	return c.SendFile(f.StoragePath)
}

// DeleteFile godoc
// DELETE /v1/files/:id
func (h *FileHandler) DeleteFile(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	if err := h.service.Delete(c.Context(), id); err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			return response.Error(c, http.StatusNotFound, fiber.NewError(http.StatusNotFound, "файл не найден"))
		}
		return response.ServerError(c)
	}

	return response.Deleted(c)
}

// ListFilesByEntity godoc
// GET /v1/files/entity/:entityType/:entityId
func (h *FileHandler) ListFilesByEntity(c *fiber.Ctx) error {
	entityType := c.Params("entityType")
	entityID := c.Params("entityId")

	if entityType == "" || entityID == "" {
		return response.BadRequest(c)
	}

	files, err := h.service.ListByEntity(c.Context(), entityType, entityID)
	if err != nil {
		return response.ServerError(c)
	}

	return response.Success(c, files)
}
