package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"timetrack/internal/adapter/grpc"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/middleware"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v3"
)

type FileHandler struct {
	service *service.FileService
	grpc    *grpc.Client
	prefix  string
}

func NewFileHandler(fileService *service.FileService, grpc *grpc.Client, prefix string) *FileHandler {
	return &FileHandler{service: fileService, grpc: grpc, prefix: prefix}
}

// hasAllFor — проверка права <entity>.all:<action> у вызывающего (ленивая:
// запрос к сервису прав только когда владелец не совпал)
func (h *FileHandler) hasAllFor(c fiber.Ctx, action string) service.HasAllFunc {
	return func(entity string) bool {
		return middleware.HasAll(c, h.grpc, middleware.Params{Service: h.prefix, Entity: entity, Action: action})
	}
}

var errFileForbidden = errors.New("нет доступа к этому файлу")

// authorizeFile — nil, если вызывающему можно работать с файлом; иначе уже
// отправленный ответ 403/500.
func (h *FileHandler) authorizeFile(c fiber.Ctx, fileID, action string) error {
	callerID, _ := c.Locals("user_id").(string)
	ok, err := h.service.CanAccessFile(c.RequestCtx(), fileID, callerID, h.hasAllFor(c, action))
	if err != nil {
		return response.ServerError(c)
	}
	if !ok {
		return response.Error(c, http.StatusForbidden, errFileForbidden)
	}
	return nil
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

	entityType, entityID := c.FormValue("entity_type"), c.FormValue("entity_id")
	if entityType != "" && entityID != "" {
		ok, err := h.service.CanAccessEntity(c.RequestCtx(), entityType, entityID, uploaderIDStr, h.hasAllFor(c, "edit"))
		if err != nil {
			return response.ServerError(c)
		}
		if !ok {
			return response.Error(c, http.StatusForbidden, errFileForbidden)
		}
	}

	f, err := h.service.Upload(c.RequestCtx(), service.UploadFileParams{
		File:       fileHeader,
		EntityType: entityType,
		EntityID:   entityID,
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

	if err := h.authorizeFile(c, id, "read"); err != nil {
		return err
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

	// удаление файла — это правка сущности, к которой он привязан
	if err := h.authorizeFile(c, id, "edit"); err != nil {
		return err
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

	callerID, _ := c.Locals("user_id").(string)
	ok, err := h.service.CanAccessEntity(c.RequestCtx(), entityType, entityID, callerID, h.hasAllFor(c, "read"))
	if err != nil {
		return response.ServerError(c)
	}
	if !ok {
		return response.Error(c, http.StatusForbidden, errFileForbidden)
	}

	files, err := h.service.ListByEntity(c.RequestCtx(), entityType, entityID, year)
	if err != nil {
		return response.ServerError(c)
	}

	return response.Success(c, files)
}

// ListFilesByEntityType godoc
// GET /v1/files/entity/:entityType?year=2026&scope=my|all
// year — необязательный query-параметр; без него возвращаются файлы за все годы.
// scope=my — только привязанные к сущностям вызывающего (его отпуск/чек/больничный);
// без scope или scope=all — все, если есть право <entity>.all:read (см.
// service.RestrictedEntityPermissions; открытые типы — всем), иначе свои.
func (h *FileHandler) ListFilesByEntityType(c fiber.Ctx) error {
	entityType := c.Params("entityType")

	if entityType == "" {
		return response.BadRequest(c)
	}

	year, err := queryYear(c)
	if err != nil {
		return response.BadRequest(c)
	}

	// Список всех вложений чатов отдавать некому: доступ только по конкретному
	// сообщению (см. CanAccessEntity)
	if entityType == service.EntityTypeChatMessage {
		return response.Error(c, http.StatusForbidden, errFileForbidden)
	}

	scope := c.Query("scope")
	if scope != "" && scope != "my" && scope != "all" {
		return response.BadRequest(c)
	}

	allEntity, restricted := service.RestrictedEntityPermissions[entityType]
	// Открытые типы сущностей видны всем — фильтровать по владельцу нечем
	canSeeAll := !restricted ||
		middleware.HasAll(c, h.grpc, middleware.Params{Service: h.prefix, Entity: allEntity, Action: "read"})

	if scope == "all" && !canSeeAll {
		return response.Error(c, http.StatusForbidden, errors.New("нет доступа к документам других сотрудников"))
	}

	if scope == "all" || (scope == "" && canSeeAll) {
		files, err := h.service.ListByEntityType(c.RequestCtx(), entityType, year)
		if err != nil {
			return response.ServerError(c)
		}
		return response.Success(c, files)
	}

	callerID, _ := c.Locals("user_id").(string)
	files, err := h.service.ListByEntityTypeForUser(c.RequestCtx(), entityType, callerID, year)
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

	// Категории общие, а файлы в них могут быть привязаны к чужим сущностям
	callerID, _ := c.Locals("user_id").(string)
	hasAll := h.hasAllFor(c, "read")
	visible := make([]repo.ListFilesByCategoryRow, 0, len(files))
	for _, f := range files {
		ok, err := h.service.CanAccessFile(c.RequestCtx(), f.ID, callerID, hasAll)
		if err != nil {
			return response.ServerError(c)
		}
		if ok {
			visible = append(visible, f)
		}
	}

	return response.Success(c, visible)
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

	if err := h.authorizeFile(c, id, "edit"); err != nil {
		return err
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
