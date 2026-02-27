package main

import (
	"database/sql"
	"log/slog"
	"timetrack/internal/adapter/grpc"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/handler"
	"timetrack/internal/middleware"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

type application struct {
	config     config
	db         *sql.DB
	grpcClient *grpc.Client
	logger     *slog.Logger
}

type config struct {
	addr   string
	db     dbConfig
	prefix string
}

type dbConfig struct {
	dsn string
}

func (app *application) mount() *fiber.App {
	fiberApp := fiber.New(fiber.Config{
		Prefork: true,
		// EnablePrintRoutes: true,
	})

	fiberApp.Use(cors.New(cors.Config{
		AllowOrigins:     "http://192.168.88.147:5173,http://localhost:5173,http://localhost:8080,http://192.168.88.147:5176,http://192.168.88.147:8080",
		AllowCredentials: true,
	}))
	fiberApp.Use(logger.New(logger.Config{
		Format: "${time} | [${ip}]:${port} | ${latency} | ${status} - ${method} ${path} \n",
	}))

	v1 := fiberApp.Group("v1")

	calendarService := service.NewCalendarService(repo.New(app.db))
	calendarHandler := handler.NewCalendarHandler(calendarService)
	calendarRouter := v1.Group("/calendar")

	// permission calendar.all:read
	calendarRouter.Get("/:userId/:year/:month",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "calendar", Action: "read"}),
		calendarHandler.GetCalendarDaysWithUserId)

	dayTypeService := service.NewDayTypeService(repo.New(app.db))
	dayTypeHandler := handler.NewDayTypeHandler(dayTypeService)
	dayTypeRouter := v1.Group("/daytypes")

	// permission для всех, нужна только авторизация
	dayTypeRouter.Get("", dayTypeHandler.GetDayTypes)

	userTimeEntryService := service.NewUserTimeEntryService(repo.New(app.db), app.db)
	userTimeEntryHandler := handler.NewUserTimeEntryHandler(userTimeEntryService, app.logger)
	userTimeEntryRouter := v1.Group("/usertimeentries")

	// permission usertime:edit
	userTimeEntryRouter.Post("/create",
		middleware.RequireFromBody(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "calendar", Action: "create"}),
		userTimeEntryHandler.CreateUserTimeEntry)

	userTimeEntryRouter.Post("/update",
		middleware.RequireFromBody(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "calendar", Action: "edit"}),
		userTimeEntryHandler.UpdateUserTimeEntries)

	userTimeEntryRouter.Post("/delete",
		middleware.RequireFromBody(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "calendar", Action: "delete"}),
		userTimeEntryHandler.DeleteUserTimeEntries)

	// Report statistics route
	userTimeEntryRouter.Get("/statistics/:userId/:year/:month/:gender",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "calendar", Action: "read"}),
		userTimeEntryHandler.GetReportStatistics)

	// Vacation calculation routes
	vacationService := service.NewVacationService(repo.New(app.db), app.db, userTimeEntryService)
	fileService := service.NewFileService("docs")
	vacationHandler := handler.NewVacationHandler(vacationService, fileService)
	vacationRouter := v1.Group("/vacation")

	// permission vacation:read
	vacationRouter.Get("/calculate",
		vacationHandler.CalculateVacationDays)

	vacationRouter.Get("/stats/:userId/:year",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "vacation", Action: "read"}),
		vacationHandler.GetVacationStatistics)

	vacationRouter.Get("/all/:year",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "vacation", Action: "read"}),
		vacationHandler.GetAllUserVacationsByYear)

	vacationRouter.Get("/:userId/:year",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "vacation", Action: "read"}),
		vacationHandler.GetVacationsByYear)

	vacationRouter.Post("/create",
		middleware.RequireFromBody(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "vacation", Action: "create"}),
		vacationHandler.CreateVacation)

	vacationRouter.Put("/:id/approve",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "vacation", Action: "edit"}),
		vacationHandler.ApproveVacation)

	vacationRouter.Put("/:id/status",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "vacation", Action: "edit"}),
		vacationHandler.UpdateVacationStatus)

	// File routes for vacations
	vacationRouter.Post("/:id/file",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "vacation", Action: "edit"}),
		vacationHandler.UploadVacationFile)

	vacationRouter.Get("/file",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "vacation", Action: "read"}),
		vacationHandler.GetVacationFile)

	vacationRouter.Delete("/file",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "vacation", Action: "file_delete"}),
		vacationHandler.DeleteVacationFile)

	vacationRouter.Delete("/:id",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "vacation", Action: "delete"}),
		vacationHandler.DeleteVacation)

	// System settings routes
	systemSettingsService := service.NewSystemSettingsService(repo.New(app.db))
	systemSettingsHandler := handler.NewSystemSettingsHandler(systemSettingsService)
	systemSettingsRouter := v1.Group("/system-settings")

	// permission system_settings:read
	systemSettingsRouter.Get("/:key",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "system_settings", Action: "read"}),
		systemSettingsHandler.GetSystemSettingByKey)

	// permission system_settings:edit
	systemSettingsRouter.Post("/value",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "system_settings", Action: "edit"}),
		systemSettingsHandler.UpdateSystemSettingValue)

	// Work standards routes
	workStandardService := service.NewWorkStandardService(repo.New(app.db))
	workStandardHandler := handler.NewWorkStandardHandler(workStandardService)
	workStandardRouter := v1.Group("/work-standards")

	// permission work_standards:create
	workStandardRouter.Post("",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "work_standards", Action: "create"}),
		workStandardHandler.CreateWorkStandard)

	// permission work_standards:read
	workStandardRouter.Get("/month/:month/year/:year",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "work_standards", Action: "read"}),
		workStandardHandler.GetWorkStandardsByMonth)

	// permission work_standards:read
	workStandardRouter.Get("/year/:year",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "work_standards", Action: "read"}),
		workStandardHandler.GetWorkStandardsByYear)

	// permission work_standards:read
	workStandardRouter.Get("/year/:year/grouped",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "work_standards", Action: "read"}),
		workStandardHandler.GetWorkStandardsByYearGrouped)

	// permission work_standards:edit
	workStandardRouter.Put("/:id",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "work_standards", Action: "edit"}),
		workStandardHandler.UpdateWorkStandard)

	// permission work_standards:delete
	workStandardRouter.Delete("/:id",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "work_standards", Action: "delete"}),
		workStandardHandler.DeleteWorkStandard)

	return fiberApp
}

func (app *application) run(f *fiber.App) error {
	return f.Listen(app.config.addr)
}
