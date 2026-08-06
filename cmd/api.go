package main

import (
	"database/sql"
	"log/slog"
	"timetrack/internal/adapter/grpc"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/calendar"
	calendarevent "timetrack/internal/calendar_event"
	daytype "timetrack/internal/day_type"
	"timetrack/internal/handler"
	"timetrack/internal/middleware"
	"timetrack/internal/service"
	sickleave "timetrack/internal/sick_leave"
	systemsetting "timetrack/internal/system_setting"
	usertimeentry "timetrack/internal/user_time_entry"
	vacation "timetrack/internal/vacation"
	workstandard "timetrack/internal/work_standard"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
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
	fiberApp := fiber.New(fiber.Config{})

	fiberApp.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://192.168.88.147:5173", "http://localhost:5173", "http://localhost:8080", "http://192.168.88.147:5176", "http://192.168.88.147:8080"},
		AllowCredentials: true,
	}))

	fiberApp.Use(logger.New(logger.Config{
		Format: "${time} | [${ip}]:${port} | ${latency} | ${status} - ${method} ${path} \n",
	}))

	v1 := fiberApp.Group("v1")

	calendarService := calendar.NewService(repo.New(app.db))
	calendar.SetupRoutes(v1, calendarService, app.grpcClient, app.config.prefix)

	calendarEventService := calendarevent.NewService(repo.New(app.db))
	calendarevent.SetupRoutes(v1, calendarEventService, app.grpcClient, app.config.prefix)

	dayTypeService := daytype.NewService(repo.New(app.db))
	daytype.SetupRoutes(v1, dayTypeService, app.grpcClient, app.config.prefix)

	userTimeEntryService := usertimeentry.NewService(repo.New(app.db), app.db)
	usertimeentry.SetupRoutes(v1, userTimeEntryService, app.logger, app.grpcClient, app.config.prefix)

	fileService := service.NewFileService(app.db, "docs")

	// Vacation routes
	vacationService := vacation.NewService(repo.New(app.db), app.db, userTimeEntryService)
	vacation.SetupRoutes(v1, vacationService, fileService, app.grpcClient, app.config.prefix)

	// Sick leave routes
	sickLeaveService := sickleave.NewService(repo.New(app.db), userTimeEntryService)
	sickleave.SetupRoutes(v1, sickLeaveService, fileService, app.grpcClient, app.config.prefix)

	// File routes
	fileHandler := handler.NewFileHandler(fileService)
	fileRouter := v1.Group("/files")

	// permission files:create
	fileRouter.Post("/upload",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "files", Action: "create"}),
		fileHandler.UploadFile)

	// permission files:read
	fileRouter.Get("/open/:id",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "files", Action: "read"}),
		fileHandler.OpenFile)

	// permission files:read
	fileRouter.Get("/entity/:entityType/:entityId",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "files", Action: "read"}),
		fileHandler.ListFilesByEntity)

	// permission files:delete
	fileRouter.Delete("/:id",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "files", Action: "delete"}),
		fileHandler.DeleteFile)

	// System settings routes
	systemSettingService := systemsetting.NewService(repo.New(app.db))
	systemsetting.SetupRoutes(v1, systemSettingService, app.grpcClient, app.config.prefix)

	// Work standards routes
	workStandardService := workstandard.NewService(repo.New(app.db))
	workstandard.SetupRoutes(v1, workStandardService, app.grpcClient, app.config.prefix)

	return fiberApp
}

func (app *application) run(f *fiber.App) error {
	return f.Listen(app.config.addr, fiber.ListenConfig{EnablePrefork: true})
}
