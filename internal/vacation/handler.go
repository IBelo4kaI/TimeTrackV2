package vacation

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service     Service
	fileService *service.FileService
}

func NewHandler(service Service, fileService *service.FileService) *Handler {
	return &Handler{service: service, fileService: fileService}
}

func (h *Handler) CreateVacation(c fiber.Ctx) error {
	var body VacationCreateRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.Error(c, http.StatusBadRequest, err)
	}

	err := h.service.CreateVacationReport(c.RequestCtx(), body)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Created(c)
}

func (h *Handler) GetVacationsByYear(c fiber.Ctx) error {
	year, err := fiber.Params[int](c, "year"), error(nil)

	if err != nil {
		return response.BadRequest(c)
	}

	userId := c.Params("userId")

	vacations, err := h.service.GetVacationsByYear(c.RequestCtx(), userId, year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, vacations)
}

func (h *Handler) GetAllUserVacationsByYear(c fiber.Ctx) error {
	year, err := fiber.Params[int](c, "year"), error(nil)

	if err != nil {
		return response.BadRequest(c)
	}

	vacations, err := h.service.GetAllUserVacationsByYear(c.RequestCtx(), year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, vacations)
}

func (h *Handler) CalculateVacationDays(c fiber.Ctx) error {
	// Получаем параметры из query string
	startDateStr := c.Query("startDate")
	endDateStr := c.Query("endDate")

	// Проверяем, что параметры переданы
	if startDateStr == "" || endDateStr == "" {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Необходимо указать startDate и endDate параметры"))
	}

	// Парсим даты
	const dateLayout = "2006-01-02"
	startDate, err := time.Parse(dateLayout, startDateStr)
	if err != nil {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Некорректный формат startDate. Используйте YYYY-MM-DD"))
	}

	endDate, err := time.Parse(dateLayout, endDateStr)
	if err != nil {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Некорректный формат endDate. Используйте YYYY-MM-DD"))
	}

	// Проверяем, что startDate <= endDate
	if startDate.After(endDate) {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "startDate не может быть позже endDate"))
	}

	// Вызываем сервис
	result, err := h.service.CalculateVacationDays(c.RequestCtx(), startDate, endDate)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, result)
}

func (h *Handler) GetVacationStatistics(c fiber.Ctx) error {
	year, err := fiber.Params[int](c, "year"), error(nil)

	if err != nil {
		return response.BadRequest(c)
	}

	userId := c.Params("userId")

	stats, err := h.service.GetVacationsStats(c.RequestCtx(), userId, year)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, stats)
}

func (h *Handler) ApproveVacation(c fiber.Ctx) error {
	vacationID := c.Params("id")

	if vacationID == "" {
		return response.BadRequest(c)
	}

	// Используем UpdateVacationStatus с статусом "approved"
	err := h.service.UpdateVacationStatus(c.RequestCtx(), vacationID, repo.VacationsStatusApproved)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Отпуск подтвержден",
	})
}

func (h *Handler) UpdateVacationStatus(c fiber.Ctx) error {
	vacationID := c.Params("id")

	if vacationID == "" {
		return response.BadRequest(c)
	}

	var body struct {
		Status string `json:"status"`
	}

	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	// Валидация статуса
	var status repo.VacationsStatus
	switch body.Status {
	case "pending":
		status = repo.VacationsStatusPending
	case "approved":
		status = repo.VacationsStatusApproved
	case "rejected":
		status = repo.VacationsStatusRejected
	default:
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "Invalid status. Must be one of: pending, approved, rejected"))
	}

	err := h.service.UpdateVacationStatus(c.RequestCtx(), vacationID, status)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Статус обновлен",
	})
}

func (h *Handler) DeleteVacation(c fiber.Ctx) error {
	vacationID := c.Params("id")

	if vacationID == "" {
		return response.BadRequest(c)
	}

	err := h.service.DeleteVacation(c.RequestCtx(), vacationID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Заявка на отпуск удалена",
	})
}

// UploadVacationFile загружает файл для отпуска
func (h *Handler) UploadVacationFile(c fiber.Ctx) error {
	vacationID := c.Params("id")
	if vacationID == "" {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "ID отпуска не указан"))
	}

	// Получаем файл из формы
	file, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "Файл не найден в запросе"))
	}

	// Загружаем файл через сервис
	result, err := h.fileService.UploadFile(c.RequestCtx(), service.LegacyUploadFileParams{
		File:         file,
		SubDirectory: "vacations",
		FileName:     "",
	})
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}

	// Обновляем поле doc_file_name в базе данных
	err = h.service.UpdateVacationFileName(c.RequestCtx(), vacationID, result.FileName)
	if err != nil {
		// Если не удалось обновить базу данных, удаляем загруженный файл
		h.fileService.DeleteFile(c.RequestCtx(), result.FilePath)
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, fiber.Map{
		"message":    "Файл успешно загружен",
		"fileName":   result.FileName,
		"filePath":   result.FilePath,
		"vacationId": vacationID,
	})
}

// GetVacationFile возвращает файл отпуска
func (h *Handler) GetVacationFile(c fiber.Ctx) error {
	fileName := c.Query("fileName")
	if fileName == "" {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "Имя файла не указано"))
	}

	// Проверяем, что имя файла безопасное
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "Некорректное имя файла"))
	}

	// Формируем путь к файлу
	filePath := "docs/vacations/" + fileName

	// Отправляем файл клиенту
	return c.SendFile(filePath)
}

// DeleteVacationFile удаляет файл отпуска
func (h *Handler) DeleteVacationFile(c fiber.Ctx) error {
	fileName := c.Query("fileName")
	vacationID := c.Query("vacationId")

	if fileName == "" {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "Имя файла не указано"))
	}

	if vacationID == "" {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "ID отпуска не указан"))
	}

	// fmt.Printf("filename: %v id: %v", fileName, vacationID)

	// Проверяем, что имя файла безопасное
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "Некорректное имя файла"))
	}

	// Формируем путь к файлу
	filePath := "docs/vacations/" + fileName

	// Проверяем существование файла перед удалением
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return response.Error(c, http.StatusNotFound, fiber.NewError(http.StatusNotFound, "Файл не найден"))
	}

	// Проверяем, что файл принадлежит указанному отпуску
	// Для этого нужно получить информацию об отпуске и сравнить имя файла
	vacation, err := h.service.GetVacationByID(c.RequestCtx(), vacationID)
	if err != nil {
		fmt.Println("ERROR: GetVacationByID")
		return response.Error(c, http.StatusInternalServerError, err)
	}

	// Проверяем, что файл действительно принадлежит этому отпуску
	// Сравниваем имя файла с doc_file_name в базе данных
	if vacation.DocFileName != fileName {
		fmt.Println("ERROR: vacation.DocFileName != fileName")

		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "Файл не принадлежит указанному отпуску"))
	}

	// Удаляем файл через сервис
	err = h.fileService.DeleteFile(c.RequestCtx(), filePath)
	if err != nil {
		fmt.Println("ERROR: DeleteFile")
		return response.Error(c, http.StatusInternalServerError, err)
	}

	// Очищаем поле doc_file_name в базе данных
	err = h.service.UpdateVacationFileName(c.RequestCtx(), vacationID, "")
	if err != nil {
		fmt.Println("ERROR: UpdateVacationFileName")
		return response.Error(c, http.StatusInternalServerError, err)
	}

	return response.Success(c, fiber.Map{
		"message": "Файл успешно удален",
	})
}
