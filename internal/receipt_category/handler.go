package receiptcategory

import (
	"errors"
	"net/http"
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

func mapError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrCategoryDuplicate), errors.Is(err, ErrKeywordDuplicate):
		return response.Error(c, http.StatusConflict, err)
	case errors.Is(err, ErrNameRequired), errors.Is(err, ErrKeywordRequired):
		return response.Error(c, http.StatusBadRequest, err)
	default:
		return response.Error(c, http.StatusInternalServerError, err)
	}
}
