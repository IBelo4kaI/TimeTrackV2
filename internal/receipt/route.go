package receipt

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, fileService *service.FileService, grpc *grpc.Client, prefix string) {
	handler := NewHandler(service, fileService, grpc, prefix)
	router := fiber.Group("/receipts")

	// permission receipts:create — userId берём из тела (фронт сканирует QR,
	// сам ходит во внешнее API и уже с готовым ответом идёт сюда), как и
	// vacation/create (см. middleware.RequireFromBody).
	router.Post("/create",
		middleware.RequireFromBody(grpc, middleware.Params{Service: prefix, Entity: "receipts", Action: "create"}),
		handler.CreateReceipt)

	// чеки ВСЕХ сотрудников (бухгалтерия) — RequireAll: без :userId в пути
	// permission-сервис сам ".all" не подставит, см. комментарий в
	// vacation/route.go про "/all/:year".
	router.Get("/all",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "receipts", Action: "read", RequireAll: true}),
		handler.GetAllReceipts)

	// список чеков конкретного сотрудника
	router.Get("/user/:userId",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "receipts", Action: "read"}),
		handler.GetReceiptsByUser)

	// карточка отдельного чека (с позициями); владелец довалидируется в
	// хендлере через RequireOwnerOrAll
	router.Get("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "receipts", Action: "read"}),
		handler.GetReceipt)

	// permission receipts:delete; владелец довалидируется в хендлере
	router.Delete("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "receipts", Action: "delete"}),
		handler.DeleteReceipt)

	// прикрепление файла (фото/скан чека); просмотр: GET
	// /v1/files/entity/receipt/:id, удаление: DELETE /v1/files/:id.
	// permission receipts:edit; владелец довалидируется в хендлере
	router.Post("/:id/file",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "receipts", Action: "edit"}),
		handler.UploadReceiptFile)
}
