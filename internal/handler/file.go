package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v3"
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
func (h *FileHandler) UploadFile(c fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "файл не найден в запросе"))
	}

	uploaderID := c.Locals("user_id")
	uploaderIDStr, _ := uploaderID.(string)

	f, err := h.service.Upload(c.RequestCtx(), service.UploadFileParams{
		File:       fileHeader,
		EntityType: c.FormValue("entity_type"),
		EntityID:   c.FormValue("entity_id"),
		CategoryID: c.FormValue("category_id"),
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
		"categoryId":   nullableString(f.CategoryID),
		"sizeBytes":    f.SizeBytes,
	})
}

// OpenFile godoc
// GET /v1/files/open/:id
func (h *FileHandler) OpenFile(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	f, err := h.service.GetFile(c.RequestCtx(), id)
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
func (h *FileHandler) DeleteFile(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	if err := h.service.Delete(c.RequestCtx(), id); err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			return response.Error(c, http.StatusNotFound, fiber.NewError(http.StatusNotFound, "файл не найден"))
		}
		return response.ServerError(c)
	}

	return response.Deleted(c)
}

// ListFilesByEntity godoc
// GET /v1/files/entity/:entityType/:entityId?year=2026
// year — необязательный query-параметр; без него возвращаются файлы за все годы.
func (h *FileHandler) ListFilesByEntity(c fiber.Ctx) error {
	entityType := c.Params("entityType")
	entityID := c.Params("entityId")

	if entityType == "" || entityID == "" {
		return response.BadRequest(c)
	}

	year, err := queryYear(c)
	if err != nil {
		return response.BadRequest(c)
	}

	files, err := h.service.ListByEntity(c.RequestCtx(), entityType, entityID, year)
	if err != nil {
		return response.ServerError(c)
	}

	return response.Success(c, files)
}

// ListFilesByEntityType godoc
// GET /v1/files/entity/:entityType?year=2026
// year — необязательный query-параметр; без него возвращаются файлы за все годы.
func (h *FileHandler) ListFilesByEntityType(c fiber.Ctx) error {
	entityType := c.Params("entityType")

	if entityType == "" {
		return response.BadRequest(c)
	}

	year, err := queryYear(c)
	if err != nil {
		return response.BadRequest(c)
	}

	files, err := h.service.ListByEntityType(c.RequestCtx(), entityType, year)
	if err != nil {
		return response.ServerError(c)
	}

	return response.Success(c, files)
}

// ListFilesByCategory godoc
// GET /v1/files/category/:categoryId?year=2026
// year — необязательный query-параметр; без него возвращаются файлы за все годы.
func (h *FileHandler) ListFilesByCategory(c fiber.Ctx) error {
	categoryID := c.Params("categoryId")

	if categoryID == "" {
		return response.BadRequest(c)
	}

	year, err := queryYear(c)
	if err != nil {
		return response.BadRequest(c)
	}

	files, err := h.service.ListByCategory(c.RequestCtx(), categoryID, year)
	if err != nil {
		return response.ServerError(c)
	}

	return response.Success(c, files)
}

// queryYear парсит необязательный query-параметр ?year=. Пустая строка (параметр
// не передан) — это 0, «без фильтра». Непустое, но нечисловое значение — ошибка.
func queryYear(c fiber.Ctx) (int, error) {
	raw := c.Query("year")
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}

// SetFileCategory godoc
// PUT /v1/files/:id/category
// Body: { "categoryId": "..." } — пустая строка убирает файл из категории.
func (h *FileHandler) SetFileCategory(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	var body struct {
		CategoryID string `json:"categoryId"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	if err := h.service.SetCategory(c.RequestCtx(), id, body.CategoryID); err != nil {
		if errors.Is(err, service.ErrFileNotFound) {
			return response.Error(c, http.StatusNotFound, fiber.NewError(http.StatusNotFound, "файл не найден"))
		}
		return response.ServerError(c)
	}

	return response.Updated(c)
}

func nullableString(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

// fiber:context-methods migrated
