package sickleave

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"
	"timetrack/internal/adapter/grpc"
	repo "timetrack/internal/adapter/mysql/sqlc"
	"timetrack/internal/middleware"
	"timetrack/internal/response"
	"timetrack/internal/service"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service     Service
	fileService *service.FileService
	grpc        *grpc.Client
	prefix      string
}

func NewHandler(svc Service, fileService *service.FileService, grpc *grpc.Client, prefix string) *Handler {
	return &Handler{service: svc, fileService: fileService, grpc: grpc, prefix: prefix}
}

func (h *Handler) CreateSickLeave(c fiber.Ctx) error {
	var body struct {
		UserID      string    `json:"userId"`
		StartDate   time.Time `json:"startDate"`
		EndDate     time.Time `json:"endDate"`
		Description string    `json:"description"`
		Status      string    `json:"status"`
		// ФИО заявителя — фронт уже знает его (userStore.usersAll), передаёт
		// только для текста уведомления админам, нигде не хранится.
		ApplicantName string `json:"applicantName"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	if body.StartDate.After(body.EndDate) {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "startDate не может быть позже endDate"))
	}

	status := repo.SickLeavesStatus(body.Status)
	if status != repo.SickLeavesStatusOfficial && status != repo.SickLeavesStatusUnofficial {
		status = repo.SickLeavesStatusUnofficial
	}

	if err := h.service.CreateSickLeave(c.RequestCtx(), CreateSickLeaveParams{
		UserID:        body.UserID,
		StartDate:     body.StartDate,
		EndDate:       body.EndDate,
		Description:   body.Description,
		Status:        status,
		ApplicantName: body.ApplicantName,
	}); err != nil {
		return response.ServerError(c)
	}

	return response.Created(c)
}

func (h *Handler) GetSickLeavesByYear(c fiber.Ctx) error {
	year, err := fiber.Params[int](c, "year"), error(nil)
	if err != nil {
		return response.BadRequest(c)
	}
	userID := c.Params("userId")

	rows, err := h.service.GetSickLeavesByYear(c.RequestCtx(), userID, year)
	if err != nil {
		return response.ServerError(c)
	}
	return response.Success(c, rows)
}

func (h *Handler) GetSickLeaveStatistics(c fiber.Ctx) error {
	year, err := fiber.Params[int](c, "year"), error(nil)
	if err != nil {
		return response.BadRequest(c)
	}
	userID := c.Params("userId")

	stats, err := h.service.GetSickLeaveStats(c.RequestCtx(), userID, year)
	if err != nil {
		return response.ServerError(c)
	}
	return response.Success(c, stats)
}

func (h *Handler) GetAllUsersSickLeavesByYear(c fiber.Ctx) error {
	year, err := fiber.Params[int](c, "year"), error(nil)
	if err != nil {
		return response.BadRequest(c)
	}

	rows, err := h.service.GetAllUsersSickLeavesByYear(c.RequestCtx(), year)
	if err != nil {
		return response.ServerError(c)
	}
	return response.Success(c, rows)
}

func (h *Handler) UpdateSickLeaveStatus(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}

	status := repo.SickLeavesStatus(body.Status)
	if status != repo.SickLeavesStatusOfficial && status != repo.SickLeavesStatusUnofficial {
		return response.Error(c, http.StatusBadRequest,
			fiber.NewError(http.StatusBadRequest, "допустимые статусы: official, unofficial"))
	}

	if err := h.service.UpdateSickLeaveStatus(c.RequestCtx(), id, status); err != nil {
		return response.ServerError(c)
	}
	return response.Updated(c)
}

// authorizeOwnerOrAll — доступ к больничному: свой — по базовому праву
// (его уже проверил Require на роуте), чужой — только с sick_leaves.all:<action>
// (тот же приём, что у чеков и отпусков).
func (h *Handler) authorizeOwnerOrAll(c fiber.Ctx, id, action string) error {
	row, err := h.service.GetSickLeaveByID(c.RequestCtx(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return response.Error(c, http.StatusNotFound, errors.New("больничный не найден"))
		}
		return response.ServerError(c)
	}

	callerID, _ := c.Locals("user_id").(string)
	if !middleware.RequireOwnerOrAll(c, h.grpc,
		middleware.Params{Service: h.prefix, Entity: "sick_leaves", Action: action},
		callerID, row.UserID) {
		return response.Error(c, http.StatusForbidden, errors.New("нет доступа к этому больничному"))
	}
	return nil
}

func (h *Handler) DeleteSickLeave(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c)
	}

	if err := h.authorizeOwnerOrAll(c, id, "delete"); err != nil {
		return err
	}

	if err := h.service.DeleteSickLeave(c.RequestCtx(), id); err != nil {
		return response.ServerError(c)
	}

	if err := h.fileService.DeleteByEntity(c.RequestCtx(), "sick_leave", id); err != nil {
		fmt.Printf("delete sick leave files: %v\n", err)
	}

	return response.Deleted(c)
}

// UploadSickLeaveFile загружает файл и привязывает его к больничному через file_entity_refs.
// Файлы доступны через GET /v1/files/open/:id и листаются через GET /v1/files/entity/sick_leave/:id.
func (h *Handler) UploadSickLeaveFile(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "ID больничного не указан"))
	}

	if err := h.authorizeOwnerOrAll(c, id, "edit"); err != nil {
		return err
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return response.Error(c, http.StatusBadRequest, fiber.NewError(http.StatusBadRequest, "файл не найден в запросе"))
	}

	uploaderID, _ := c.Locals("user_id").(string)

	f, err := h.fileService.Upload(c.RequestCtx(), service.UploadFileParams{
		File:       fileHeader,
		EntityType: "sick_leave",
		EntityID:   id,
		UploaderID: uploaderID,
	})
	if err != nil {
		return response.ServerError(c)
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"id":           f.ID,
		"originalName": f.OriginalName,
		"mimeType":     f.MimeType,
		"fileType":     f.FileType,
		"sizeBytes":    f.SizeBytes,
	})
}
