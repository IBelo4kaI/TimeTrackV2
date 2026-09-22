package receiptcategory

import (
	"errors"
	"net/http"
	"strconv"
	"timetrack/internal/response"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) Handler {
	return Handler{service: service}
}

// ListCategories godoc
// GET /v1/receipt-categories — справочник категорий (системных + добавленных
// пользователем), нужен и для select при ручной правке категории чека, и
// для экрана настроек "Категории и слова".
func (h Handler) ListCategories(c fiber.Ctx) error {
	categories, err := h.service.ListCategories(c.RequestCtx())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, categories)
}

// CreateCategory godoc
// POST /v1/receipt-categories — добавляет пользовательскую категорию
// (is_system=false).
func (h Handler) CreateCategory(c fiber.Ctx) error {
	var body CreateCategoryRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	category, err := h.service.CreateCategory(c.RequestCtx(), body.Name)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, category)
}

// RenameCategory godoc
// PUT /v1/receipt-categories/:id — переименовывает категорию (системную или
// пользовательскую — is_system не меняется, правится только название).
func (h Handler) RenameCategory(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 32)
	if err != nil {
		return response.BadRequest(c)
	}

	var body CreateCategoryRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	category, err := h.service.RenameCategory(c.RequestCtx(), int32(id), body.Name)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, category)
}

// ListKeywords godoc
// GET /v1/receipt-categories/keywords — весь словарь ключевых слов, для
// экрана настроек "Категории и слова".
func (h Handler) ListKeywords(c fiber.Ctx) error {
	keywords, err := h.service.ListKeywords(c.RequestCtx())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, keywords)
}

// CreateKeyword godoc
// POST /v1/receipt-categories/keywords — добавляет пользовательскую пару
// слово -> категория (is_system=false).
func (h Handler) CreateKeyword(c fiber.Ctx) error {
	var body CreateKeywordRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	keyword, err := h.service.CreateKeyword(c.RequestCtx(), body.Keyword, body.CategoryID)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, keyword)
}

// Preview godoc
// POST /v1/receipt-categories/preview — определяет категорию по ИНН
// продавца/позициям без сохранения чека и без самообучения (см. Preview в
// service.go) — показать сотруднику ДО нажатия "Добавить" в очереди
// сканирования (ReceiptScan.vue).
func (h Handler) Preview(c fiber.Ctx) error {
	var body PreviewRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	items := make([]ItemForClassify, len(body.Items))
	for i, item := range body.Items {
		items[i] = ItemForClassify{Name: item.Name}
	}

	result, err := h.service.Preview(c.RequestCtx(), body.SellerInn, items)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, PreviewResponse{CategoryID: result.CategoryID})
}

// ListMerchants godoc
// GET /v1/receipt-categories/merchants — словарь "ИНН продавца -> категория"
// целиком, для экрана настроек "Категории и слова" (вкладка "Продавцы").
func (h Handler) ListMerchants(c fiber.Ctx) error {
	merchants, err := h.service.ListMerchants(c.RequestCtx())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, merchants)
}

// UpdateMerchant godoc
// PUT /v1/receipt-categories/merchants/:inn — правит связь продавец ->
// категория и сразу переносит новую категорию на все уже сохранённые чеки
// этого продавца (см. UpdateMerchant в service.go).
func (h Handler) UpdateMerchant(c fiber.Ctx) error {
	inn := c.Params("inn")
	if inn == "" {
		return response.BadRequest(c)
	}

	var body UpdateMerchantRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	updated, err := h.service.UpdateMerchant(c.RequestCtx(), inn, body.CategoryID)
	if err != nil {
		return mapError(c, err)
	}

	return response.Success(c, fiber.Map{"updatedReceipts": updated})
}

func mapError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrCategoryNotFound):
		return response.Error(c, http.StatusNotFound, err)
	case errors.Is(err, ErrCategoryDuplicate), errors.Is(err, ErrKeywordDuplicate):
		return response.Error(c, http.StatusConflict, err)
	case errors.Is(err, ErrNameRequired),
		errors.Is(err, ErrKeywordRequired),
		errors.Is(err, ErrSellerInnRequired):
		return response.Error(c, http.StatusBadRequest, err)
	default:
		return response.Error(c, http.StatusInternalServerError, err)
	}
}
