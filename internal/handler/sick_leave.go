package handler

import (
	"net/http"
	"os"
	"strings"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v2"
)

type SickLeaveHandler struct {
	service     service.SickLeaveService
	fileService *service.FileService
}

func NewSickLeaveHandler(svc service.SickLeaveService, fileService *service.FileService) *SickLeaveHandler {
	return &SickLeaveHandler{service: svc, fileService: fileService}
}

func (h *SickLeaveHandler) CreateSickLeave(c *fiber.Ctx) error {
	var body struct {
		UserID      string           `json:"userId"`
		StartDate   time.Time        `json:"startDate"`
		EndDate     time.Time        `json:"endDate"`
		Description string           `json:"description"`
		Status      string           `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c)
	}

	if body.StartDate.After(body.EndDate) {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "startDate не может быть позже endDate"))
	}

	status := repo.SickLeavesStatus(body.Status)
	if status != repo.SickLeavesStatusOfficial && status != repo.SickLeavesStatusUnofficial {
		status = repo.SickLeavesStatusUnofficial
	}

	if err := h.service.CreateSickLeave(c.Context(), service.CreateSickLeaveParams{
		UserID:      body.UserID,
		StartDate:   body.StartDate,
		EndDate:     body.EndDate,
		Description: body.Description,
		Status:      status,
	}); err != nil {
		return response.ServerError(c)
	}

	return response.Created(c)
}

func (h *SickLeaveHandler) GetSickLeavesByYear(c *fiber.Ctx) error {
	year, err := c.ParamsInt("year")
	if err != nil {
		return response.BadRequest(c)
	}
	userID := c.Params("userId")

	rows, err := h.service.GetSickLeavesByYear(c.Context(), userID, year)
	if err != nil {
		return response.ServerError(c)
	}
	return response.Success(c, rows)
}

func (h *SickLeaveHandler) GetAllUsersSickLeavesByYear(c *fiber.Ctx) error {
	year, err := c.ParamsInt("year")
	if err != nil {
		return response.BadRequest(c)
	}

	rows, err := h.service.GetAllUsersSickLeavesByYear(c.Context(), year)
	if err != nil {
		return response.ServerError(c)
	}
	return response.Success(c, rows)
}

func (h *SickLeaveHandler) UpdateSickLeaveStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c)
	}

	status := repo.SickLeavesStatus(body.Status)
	if status != repo.SickLeavesStatusOfficial && status != repo.SickLeavesStatusUnofficial {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "допустимые статусы: official, unofficial"))
	}

	if err := h.service.UpdateSickLeaveStatus(c.Context(), id, status); err != nil {
		return response.ServerError(c)
	}
	return response.Updated(c)
}

func (h *SickLeaveHandler) DeleteSickLeave(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	if err := h.service.DeleteSickLeave(c.Context(), id); err != nil {
		return response.ServerError(c)
	}
	return response.Deleted(c)
}

func (h *SickLeaveHandler) UploadSickLeaveFile(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "ID больничного не указан"))
	}

	file, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "файл не найден в запросе"))
	}

	result, err := h.fileService.UploadFile(c.Context(), service.LegacyUploadFileParams{
		File:         file,
		SubDirectory: "sick-leaves",
	})
	if err != nil {
		return response.ServerError(c)
	}

	if err := h.service.UpdateSickLeaveFileName(c.Context(), id, result.FileName); err != nil {
		h.fileService.DeleteFile(c.Context(), result.FilePath)
		return response.ServerError(c)
	}

	return response.Success(c, fiber.Map{
		"fileName":   result.FileName,
		"sickLeaveId": id,
	})
}

func (h *SickLeaveHandler) GetSickLeaveFile(c *fiber.Ctx) error {
	fileName := c.Query("fileName")
	if fileName == "" {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "имя файла не указано"))
	}
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "некорректное имя файла"))
	}

	return c.SendFile("docs/sick-leaves/" + fileName)
}

func (h *SickLeaveHandler) DeleteSickLeaveFile(c *fiber.Ctx) error {
	fileName := c.Query("fileName")
	id := c.Query("sickLeaveId")

	if fileName == "" || id == "" {
		return response.BadRequest(c)
	}
	if strings.Contains(fileName, "..") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "некорректное имя файла"))
	}

	sl, err := h.service.GetSickLeaveByID(c.Context(), id)
	if err != nil {
		return response.ServerError(c)
	}
	if sl.DocFileName != fileName {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "файл не принадлежит указанному больничному"))
	}

	filePath := "docs/sick-leaves/" + fileName
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return response.Error(c, http.StatusNotFound, fiber.NewError(http.StatusNotFound, "файл не найден"))
	}

	if err := h.fileService.DeleteFile(c.Context(), filePath); err != nil {
		return response.ServerError(c)
	}
	if err := h.service.UpdateSickLeaveFileName(c.Context(), id, ""); err != nil {
		return response.ServerError(c)
	}

	return response.Deleted(c)
}
