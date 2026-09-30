package businesscard

import (
	"errors"
	"net/http"
	"timetrack/internal/adapter/grpc"
	"timetrack/internal/middleware"
	"timetrack/internal/response"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service Service
	grpc    *grpc.Client
	prefix  string
}

func NewHandler(service Service, grpc *grpc.Client, prefix string) Handler {
	return Handler{service: service, grpc: grpc, prefix: prefix}
}

// GET /v1/business-cards/my — карты, выданные вызывающему.
func (h Handler) GetMy(c fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	cards, err := h.service.ListByUser(c.RequestCtx(), userID)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, cards)
}

// GET /v1/business-cards/all — все карты (business_cards.all:read).
func (h Handler) GetAll(c fiber.Ctx) error {
	cards, err := h.service.ListAll(c.RequestCtx())
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, err)
	}
	return response.Success(c, cards)
}

// GET /v1/business-cards/:id — своя карта или любая с business_cards.all:read.
func (h Handler) GetCard(c fiber.Ctx) error {
	card, err := h.service.GetByID(c.RequestCtx(), c.Params("id"))
	if err != nil {
		return mapError(c, err)
	}

	callerID, _ := c.Locals("user_id").(string)
	owner := ""
	if card.OwnerID != nil {
		owner = *card.OwnerID
	}
	// карта без владельца — только для .all, поэтому подставляем заведомо чужого
	if owner == "" {
		owner = "unassigned"
	}
	if !middleware.RequireOwnerOrAll(c, h.grpc,
		middleware.Params{Service: h.prefix, Entity: "business_cards", Action: "read"},
		callerID, owner) {
		return response.Error(c, http.StatusForbidden, errors.New("нет доступа к этой карте"))
	}

	return response.Success(c, card)
}

func (h Handler) Create(c fiber.Ctx) error {
	var body CardRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}
	card, err := h.service.Create(c.RequestCtx(), body)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, card)
}

func (h Handler) Update(c fiber.Ctx) error {
	var body CardRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}
	card, err := h.service.Update(c.RequestCtx(), c.Params("id"), body)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, card)
}

func (h Handler) Delete(c fiber.Ctx) error {
	if err := h.service.Delete(c.RequestCtx(), c.Params("id")); err != nil {
		return mapError(c, err)
	}
	return response.Deleted(c)
}

// GET /v1/business-cards/:id/number — полный номер, только .all:read.
func (h Handler) GetNumber(c fiber.Ctx) error {
	number, err := h.service.Number(c.RequestCtx(), c.Params("id"))
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, NumberResponse{Number: number})
}

func (h Handler) GetHistory(c fiber.Ctx) error {
	history, err := h.service.History(c.RequestCtx(), c.Params("id"))
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, history)
}

func (h Handler) Assign(c fiber.Ctx) error {
	var body AssignRequest
	if err := c.Bind().Body(&body); err != nil {
		return response.BadRequest(c)
	}
	adminID, _ := c.Locals("user_id").(string)
	card, err := h.service.Assign(c.RequestCtx(), c.Params("id"), body.UserID, adminID)
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, card)
}

func (h Handler) Release(c fiber.Ctx) error {
	card, err := h.service.Release(c.RequestCtx(), c.Params("id"))
	if err != nil {
		return mapError(c, err)
	}
	return response.Success(c, card)
}

func mapError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return response.Error(c, http.StatusNotFound, err)
	case errors.Is(err, ErrNumberRequired),
		errors.Is(err, ErrStatusInvalid),
		errors.Is(err, ErrUserRequired),
		errors.Is(err, ErrSameOwner),
		errors.Is(err, ErrNotAssigned):
		return response.Error(c, http.StatusBadRequest, err)
	default:
		return response.Error(c, http.StatusInternalServerError, err)
	}
}
