package receiptcategory

import (
	"timetrack/internal/adapter/permission"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiberRouter fiber.Router, service Service, permission *permission.Client, prefix string) {
	handler := NewHandler(service)
	router := fiberRouter.Group("/receipt-categories")

	// читать может любой с базовым receipts:read — нужно для select при
	// ручной категоризации чека и для экрана настроек "Категории и слова"
	router.Get("",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "receipts", Action: "read"}),
		handler.ListCategories)
	router.Get("/keywords",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "receipts", Action: "read"}),
		handler.ListKeywords)
	router.Get("/merchants",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "receipts", Action: "read"}),
		handler.ListMerchants)

	// предпросмотр категории при сканировании, до сохранения чека —
	// receipts:create, как и сам POST /receipts/create
	router.Post("/preview",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "receipts", Action: "create"}),
		handler.Preview)

	// управление словарём — только с receipts.all:edit, как и остальное
	// администрирование чеков (см. receipt/route.go)
	router.Post("",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "receipts", Action: "edit", RequireAll: true}),
		handler.CreateCategory)
	router.Post("/keywords",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "receipts", Action: "edit", RequireAll: true}),
		handler.CreateKeyword)
	router.Put("/merchants/:inn",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "receipts", Action: "edit", RequireAll: true}),
		handler.UpdateMerchant)
	router.Put("/:id",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "receipts", Action: "edit", RequireAll: true}),
		handler.RenameCategory)
}
