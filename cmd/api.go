package main

import (
	"database/sql"
	"log/slog"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/handler"
	"timetrack/internal/service"

	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

type application struct {
	config config
	db     *sql.DB
	logger *slog.Logger
}

type config struct {
	addr string
	db   dbConfig
}

type dbConfig struct {
	dsn string
}

func (app *application) mount() *fiber.App {
	fiberApp := fiber.New(fiber.Config{
		// Prefork: true,
		// EnablePrintRoutes: true,
	})

	cfg := swagger.Config{
		BasePath: "/",
		FilePath: "./docs/swagger.json",
		Path:     "swagger",
		Title:    "Swagger API Docs",
	}

	fiberApp.Use(swagger.New(cfg))

	fiberApp.Use(cors.New())
	fiberApp.Use(logger.New(logger.Config{
		Format: "${time} | [${ip}]:${port} | ${latency} | ${status} - ${method} ${path} \n",
	}))

	v1 := fiberApp.Group("v1")

	calendarService := service.NewCalendarService(repo.New(app.db))
	calendarHandler := handler.NewCalendarHandler(calendarService)
	calendarRouter := v1.Group("/calendar")

	// permission calendar.all:read
	calendarRouter.Get("/:userId/:year/:month", calendarHandler.GetCalendarDaysWithUserId)

	dayTypeService := service.NewDayTypeService(repo.New(app.db))
	dayTypeHandler := handler.NewDayTypeHandler(dayTypeService)
	dayTypeRouter := v1.Group("/daytypes")

	// permission для всех, нужна только авторизация
	dayTypeRouter.Get("", dayTypeHandler.GetDayTypes)

	userTimeEntryService := service.NewUserTimeEntryService(repo.New(app.db), app.db)
	userTimeEntryHandler := handler.NewUserTimeEntryHandler(userTimeEntryService)
	userTimeEntryRouter := v1.Group("/usertimeentries")

	// permission usertime.all:edit
	userTimeEntryRouter.Post("/create", userTimeEntryHandler.CreateUserTimeEntry)
	userTimeEntryRouter.Post("/update", userTimeEntryHandler.UpdateUserTimeEntries)
	userTimeEntryRouter.Post("/delete", userTimeEntryHandler.DeleteUserTimeEntries)

	return fiberApp
}

func (app *application) run(f *fiber.App) error {
	return f.Listen(app.config.addr)
}
