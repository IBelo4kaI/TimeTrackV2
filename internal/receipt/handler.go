package receipt

import (
	"errors"
	"fmt"
	"net/http"
	"timetrack/internal/adapter/grpc"
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

func NewHandler(service Service, fileService *service.FileService, grpc *grpc.Client, prefix string) Handler {
	return Handler{service: service, fileService: fileService, grpc: grpc, prefix: prefix}
}

func (h Handler) CreateReceipt(c fiber.Ctx) error {
	var body CreateReceiptRequest
	if err := c.Bind().Body(&body); err != nil {
		fmt.Printf("%+v", err.Error())
		return response.BadRequest(c)
	}

	r, err := h.service.Create(c.RequestCtx(), body)
	if err != nil {
		fmt.Printf("%+v", err.Error())
		return mapError(c, err)
	}

	return response.Success(c, r)
}

// GetReceiptsByUser godoc
// GET /v1/receipts/user/:userId — список чеков сотрудника (без позиций,
// для таблицы/списка на фронте; карточка с позициями — GetReceipt).
func (h Handler) GetReceiptsByUser(c fiber.Ctx) error {
	userId := c.Params("userId")
	if userId == "" {
		return response.BadRequest(c)
	}

	receipts, err := h.service.ListByUser(c.RequestCtx(), userId)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, receipts)
}

// GetAllReceipts godoc
// GET /v1/receipts/all — чеки ВСЕХ сотрудников, для бухгалтерии.
func (h Handler) GetAllReceipts(c fiber.Ctx) error {
	receipts, err := h.service.ListAll(c.RequestCtx())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, receipts)
}

// GetReceipt godoc
// GET /v1/receipts/:id — карточка чека с позициями.
//
// Как и vacation.GetVacation: роут без :userId в пути, поэтому базовый
// middleware.Require проверяет только "receipts:read", владельца узнаём
// после чтения записи и довалидируем через RequireOwnerOrAll.
func (h Handler) GetReceipt(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	r, err := h.service.GetByID(c.RequestCtx(), id)
	if err != nil {
		return mapError(c, err)
	}

	callerID, _ := c.Locals("user_id").(string)
	allowed := middleware.RequireOwnerOrAll(
		c,
		h.grpc,
		middleware.Params{Service: h.prefix, Entity: "receipts", Action: "read"},
		callerID,
		r.UserID,
	)
	if !allowed {
		return response.Error(c, http.StatusForbidden, errors.New("нет доступа к этому чеку"))
	}

	return response.Success(c, r)
}

// UploadReceiptFile godoc
// POST /v1/receipts/:id/file — прикрепляет файл (фото/скан чека) через
// общий file-сервис (entityType="receipt"), см. vacation.UploadVacationFile.
// Просмотр: GET /v1/files/entity/receipt/:id, удаление: DELETE /v1/files/:id.
//
// Свою запись можно дополнить файлом всегда, чужую — только с
// receipts.all:edit (RequireOwnerOrAll), как и у vacation.
func (h Handler) UploadReceiptFile(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	r, err := h.service.GetByID(c.RequestCtx(), id)
	if err != nil {
		return mapError(c, err)
	}

	uploaderID, _ := c.Locals("user_id").(string)
	allowed := middleware.RequireOwnerOrAll(
		c,
		h.grpc,
		middleware.Params{Service: h.prefix, Entity: "receipts", Action: "edit"},
		uploaderID,
		r.UserID,
	)
	if !allowed {
		return response.Error(c, http.StatusForbidden, errors.New("нет доступа к этому чеку"))
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, errors.New("файл не найден в запросе"))
	}

	f, err := h.fileService.Upload(c.RequestCtx(), service.UploadFileParams{
		File:       fileHeader,
		EntityType: "receipt",
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

// DeleteReceipt godoc
// DELETE /v1/receipts/:id — владелец довалидируется в хендлере, как и в
// GetReceipt/vacation.DeleteVacation.
func (h Handler) DeleteReceipt(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	r, err := h.service.GetByID(c.RequestCtx(), id)
	if err != nil {
		return mapError(c, err)
	}

	callerID, _ := c.Locals("user_id").(string)
	allowed := middleware.RequireOwnerOrAll(
		c,
		h.grpc,
		middleware.Params{Service: h.prefix, Entity: "receipts", Action: "delete"},
		callerID,
		r.UserID,
	)
	if !allowed {
		return response.Error(c, http.StatusForbidden, errors.New("нет доступа к этому чеку"))
	}

	if err := h.service.Delete(c.RequestCtx(), id); err != nil {
		return mapError(c, err)
	}

	if err := h.fileService.DeleteByEntity(c.RequestCtx(), "receipt", id); err != nil {
		fmt.Printf("delete receipt files: %v\n", err)
	}

	return response.Deleted(c)
}

func mapError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return response.Error(c, http.StatusNotFound, err)
	case errors.Is(err, ErrDuplicate):
		return response.Error(c, http.StatusConflict, err)
	case errors.Is(err, ErrUserRequired),
		errors.Is(err, ErrFiscalDataRequired),
		errors.Is(err, ErrTotalSumInvalid),
		errors.Is(err, ErrSellerInnRequired),
		errors.Is(err, ErrNoItems):
		return response.Error(c, http.StatusBadRequest, err)
	default:
		return response.Error(c, http.StatusInternalServerError, err)
	}
}
