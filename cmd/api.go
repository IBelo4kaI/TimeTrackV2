package main

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
	"timetrack/internal/adapter/grpc"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/calendar"
	calendarevent "timetrack/internal/calendar_event"
	"timetrack/internal/chat"
	daytype "timetrack/internal/day_type"
	filecategory "timetrack/internal/file_category"
	"timetrack/internal/handler"
	"timetrack/internal/middleware"
	"timetrack/internal/news"
	"timetrack/internal/notification"
	notificationtemplate "timetrack/internal/notification_template"
	"timetrack/internal/service"
	sickleave "timetrack/internal/sick_leave"
	systemsetting "timetrack/internal/system_setting"
	"timetrack/internal/timesheetreminder"
	usertimeentry "timetrack/internal/user_time_entry"
	vacation "timetrack/internal/vacation"
	vacationtype "timetrack/internal/vacation_type"
	"timetrack/internal/vk"
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
	addr        string
	db          dbConfig
	prefix      string
	frontendURL string
	vk          vkConfig
}

type dbConfig struct {
	dsn string
}

type vkConfig struct {
	groupToken          string
	confirmationString  string
	secretKey           string
	communityScreenName string
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

	// VK-бот и notifications нужны раньше vacation/sick_leave — те шлют в
	// них уведомления админам о новых заявках (см. notifyAdminsNewApplication
	// в соответствующих service.go).
	vkService := vk.NewService(repo.New(app.db), app.config.vk.groupToken, app.config.vk.communityScreenName, app.logger)
	vk.SetupRoutes(v1, vkService, app.grpcClient, app.config.prefix, vk.Config{
		ConfirmationString: app.config.vk.confirmationString,
		SecretKey:          app.config.vk.secretKey,
	})

	// Свой SSE-хаб, отдельный от chat.Hub — см. internal/notification/hub.go.
	notificationHub := notification.NewHub()
	notificationService := notification.NewService(repo.New(app.db), notificationHub, vkService, app.logger)
	notification.SetupRoutes(v1, notificationService, app.grpcClient, app.config.prefix)

	// Готовые шаблоны для ручной рассылки (см. notificationService.SendManual)
	notificationTemplateService := notificationtemplate.NewService(repo.New(app.db))
	notificationtemplate.SetupRoutes(v1, notificationTemplateService, app.grpcClient, app.config.prefix)

	// Новости/чейнджлог приложения (см. internal/news)
	newsService := news.NewService(repo.New(app.db))
	news.SetupRoutes(v1, newsService, app.grpcClient, app.config.prefix)

	// Vacation routes
	vacationService := vacation.NewService(repo.New(app.db), app.db, userTimeEntryService, notificationService, vkService, app.config.frontendURL)
	vacation.SetupRoutes(v1, vacationService, fileService, app.grpcClient, app.config.prefix)

	// Напоминание заполнить табель — свой тикер в процессе, без внешнего
	// cron (см. internal/timesheetreminder). Список сотрудников — только
	// локально известные (без похода в auth-сервис за полным списком).
	reminderService := timesheetreminder.NewService(repo.New(app.db), calendarService, notificationService, vkService, app.config.frontendURL, app.logger)
	timesheetreminder.SetupRoutes(v1, reminderService, app.grpcClient, app.config.prefix)
	go runTimesheetReminderTicker(reminderService)

	// Sick leave routes
	sickLeaveService := sickleave.NewService(repo.New(app.db), userTimeEntryService, notificationService, vkService, app.config.frontendURL)
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

	// permission files:read
	fileRouter.Get("/entity/:entityType",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "files", Action: "read"}),
		fileHandler.ListFilesByEntityType)

	// permission files:read
	fileRouter.Get("/category/:categoryId",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "files", Action: "read"}),
		fileHandler.ListFilesByCategory)

	// permission files:edit
	fileRouter.Put("/:id/category",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "files", Action: "edit"}),
		fileHandler.SetFileCategory)

	// permission files:delete
	fileRouter.Delete("/:id",
		middleware.Require(app.grpcClient, middleware.Params{Service: app.config.prefix, Entity: "files", Action: "delete"}),
		fileHandler.DeleteFile)

	// File category routes (дерево категорий файлов)
	fileCategoryService := filecategory.NewService(repo.New(app.db))
	filecategory.SetupRoutes(v1, fileCategoryService, app.grpcClient, app.config.prefix)

	// Vacation type routes
	vacationTypeService := vacationtype.NewService(repo.New(app.db))
	vacationtype.SetupRoutes(v1, vacationTypeService, app.grpcClient, app.config.prefix)

	// System settings routes
	systemSettingService := systemsetting.NewService(repo.New(app.db))
	systemsetting.SetupRoutes(v1, systemSettingService, app.grpcClient, app.config.prefix)

	// Work standards routes
	workStandardService := workstandard.NewService(repo.New(app.db))
	workstandard.SetupRoutes(v1, workStandardService, app.grpcClient, app.config.prefix)

	// Chat routes (SSE — требует единственного процесса, см. run() и
	// internal/chat/hub.go про отключённый prefork)
	chatHub := chat.NewHub()
	chatService := chat.NewService(repo.New(app.db), chatHub, fileService, vkService, notificationService, app.config.frontendURL)
	chat.SetupRoutes(v1, chatService, app.grpcClient, app.config.prefix)

	return fiberApp
}

// runTimesheetReminderTicker — раз в час дёргает Run (сама проверка внутри
// срабатывает не чаще раза в сутки, см. timesheetreminder.Service.Run).
// Первый вызов — сразу при старте, на случай если процесс поднялся ровно в
// нужный час; остальные — по тикеру. Живёт всё время работы процесса, без
// отдельного graceful shutdown — как и остальной фон в этом файле.
func runTimesheetReminderTicker(s *timesheetreminder.Service) {
	ctx := context.Background()
	s.Run(ctx, time.Now().UTC())

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for now := range ticker.C {
		s.Run(ctx, now.UTC())
	}
}

func (app *application) run(f *fiber.App) error {
	// Prefork отключён: чаты держат долгоживущие SSE-соединения в памяти
	// процесса (см. internal/chat/hub.go). При prefork'е сервер работает как
	// несколько ОС-процессов, разделяющих порт через SO_REUSEPORT — у каждого
	// процесса своя память, и хаб одного процесса не видит соединения,
	// принятые другим. Без единого процесса сообщение может просто не
	// дойти до части участников чата.
	return f.Listen(app.config.addr, fiber.ListenConfig{EnablePrefork: false})
}
