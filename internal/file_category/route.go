package filecategory

import (
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, grpc *grpc.Client, prefix string) {
	handler := NewHandler(service)
	router := fiber.Group("/file-categories")

	// доступно всем авторизованным — нужно для выбора категории при загрузке файла
	router.Get("", handler.GetFileCategories)

	// permission file_categories:create — управление деревом категорий только у админа
	router.Post("",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "file_categories", Action: "create"}),
		handler.CreateFileCategory)

	// permission file_categories:edit
	router.Put("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "file_categories", Action: "edit"}),
		handler.UpdateFileCategory)

	// permission file_categories:delete
	router.Delete("/:id",
		middleware.Require(grpc, middleware.Params{Service: prefix, Entity: "file_categories", Action: "delete"}),
		handler.DeleteFileCategory)
}
