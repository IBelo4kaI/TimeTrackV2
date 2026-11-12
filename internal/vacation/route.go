package vacation

import (
	"timetrack/internal/adapter/permission"
	"timetrack/internal/middleware"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v3"
)

func SetupRoutes(fiber fiber.Router, service Service, fileService *service.FileService, permission *permission.Client, prefix string) {
	handler := NewHandler(service, fileService, permission, prefix)
	router := fiber.Group("/vacation")

	// расчёт дней отпуска — только авторизация
	router.Get("/calculate", handler.CalculateVacationDays)
	router.Get("/calculate-end", handler.CalculateVacationEndDate)

	// permission vacation:read
	router.Get("/stats/:userId/:year",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "vacation", Action: "read"}),
		handler.GetVacationStatistics)

	// эндпоинт отдаёт отпуска ВСЕХ сотрудников — без RequireAll middleware
	// проверил бы обычный "vacation:read" вместо "vacation.all:read" (в пути
	// нет :userId, permission-сервис сам ".all" не добавит)
	router.Get("/all/:year",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "vacation", Action: "read", RequireAll: true}),
		handler.GetAllUserVacationsByYear)

	// Урезанный (без description) список отпусков всех сотрудников — для
	// виджета "отпуска коллег" на дашборде. Специально ОТДЕЛЬНОЕ разрешение
	// (time:vacation_calendar:read), а не vacation.all:read: последнее даёт
	// доступ к полным данным заявок + предполагает менеджерские права
	// (approve/reject), его нельзя было раздавать всем сотрудникам просто
	// чтобы они видели календарь коллег — см. историю в readme.md.
	router.Get("/calendar/:year",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "vacation_calendar", Action: "read"}),
		handler.ListVacationCalendar)

	// Конкретный суффикс должен предшествовать маршруту с двумя параметрами.
	router.Get("/:id/history", middleware.Require(permission, middleware.Params{Service: prefix, Entity: "vacation", Action: "read"}), handler.History)

	router.Get("/:userId/:year",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "vacation", Action: "read"}),
		handler.GetVacationsByYear)

	router.Put("/:id/application", middleware.Require(permission, middleware.Params{Service: prefix, Entity: "vacation", Action: "edit"}), handler.EditApplication)
	router.Put("/:id/withdraw", middleware.Require(permission, middleware.Params{Service: prefix, Entity: "vacation", Action: "edit"}), handler.Withdraw)

	// карточка отдельной заявки, по id отпуска (страница заявления)
	router.Get("/:id",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "vacation", Action: "read"}),
		handler.GetVacation)

	// permission vacation:create
	router.Post("/create",
		middleware.RequireFromBody(permission, middleware.Params{Service: prefix, Entity: "vacation", Action: "create"}),
		handler.CreateVacation)

	// permission vacation.all:edit — утверждение/отклонение/смена типа это
	// чисто административные действия, self-service тут смысла не имеет
	// (сотрудник не должен мочь утвердить/отклонить свою же заявку), поэтому
	// RequireAll:true без исключения для владельца, в отличие от upload/delete.
	router.Put("/:id/approve",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "vacation", Action: "edit", RequireAll: true}),
		handler.ApproveVacation)

	router.Put("/:id/status",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "vacation", Action: "edit", RequireAll: true}),
		handler.UpdateVacationStatus)

	router.Put("/:id/type",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "vacation", Action: "edit"}),
		handler.UpdateVacationType)

	// загрузка файла; просмотр: GET /v1/files/entity/vacation/:id, удаление: DELETE /v1/files/:id
	router.Post("/:id/file",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "vacation", Action: "edit"}),
		handler.UploadVacationFile)

	// permission vacation:delete; владелец/статус довалидируются в хендлере
	// (RequireOwnerOrAll), см. комментарий над DeleteVacation
	router.Delete("/:id",
		middleware.Require(permission, middleware.Params{Service: prefix, Entity: "vacation", Action: "delete"}),
		handler.DeleteVacation)
}
