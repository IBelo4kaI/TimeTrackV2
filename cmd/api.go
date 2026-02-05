package main

import (
	"database/sql"
	"log/slog"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/handler"
	"timetrack/internal/service"

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

	fiberApp.Use(cors.New())
	fiberApp.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} | ${latency} | ${status} - ${method} ${path} \n",
	}))

	v1 := fiberApp.Group("v1")

	calendarService := service.NewCalendarService(repo.New(app.db))
	calendarRouter := v1.Group("/calendar")

	calendarRouter.Get("/test", func(c *fiber.Ctx) error {
		userTimeEntries, err := calendarService.GetCalendarDays(c.Context(), "b5b2e35e-01bd-11f1-bf7c-c21b751ece2a", 3, 2024)
		if err != nil {
			return err
		}

		return c.JSON(userTimeEntries)
	})

	dayTypeService := service.NewDayTypeService(repo.New(app.db))
	dayTypeHandler := handler.NewDayTypeHandler(dayTypeService)
	dayTypeRouter := v1.Group("/daytypes")
	dayTypeRouter.Get("", dayTypeHandler.GetDayTypes)

	return fiberApp
}

func (app *application) run(f *fiber.App) error {
	return f.Listen(app.config.addr)
}
