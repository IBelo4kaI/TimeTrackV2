package businesscard

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"

	"github.com/google/uuid"
)

var (
	ErrNotFound       = errors.New("карта не найдена")
	ErrNumberRequired = errors.New("не указан номер карты")
	ErrStatusInvalid  = errors.New("некорректный статус карты")
	ErrUserRequired   = errors.New("не указан сотрудник")
	ErrSameOwner      = errors.New("карта уже выдана этому сотруднику")
	ErrNotAssigned    = errors.New("карта никому не выдана")
)

type Service interface {
	Create(ctx context.Context, req CardRequest) (CardView, error)
	Update(ctx context.Context, id string, req CardRequest) (CardView, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (CardView, error)
	ListAll(ctx context.Context) ([]CardView, error)
	ListByUser(ctx context.Context, userID string) ([]CardView, error)
	Number(ctx context.Context, id string) (string, error)
	Assign(ctx context.Context, id, userID, assignedBy string) (CardView, error)
	Release(ctx context.Context, id string) (CardView, error)
	History(ctx context.Context, id string) ([]Assignment, error)
}

type service struct {
	repo          *repo.Queries
	db            *sql.DB
	encryptionKey string
}

func NewService(r *repo.Queries, db *sql.DB, encryptionKey string) Service {
	return &service{repo: r, db: db, encryptionKey: encryptionKey}
}

func (s *service) Create(ctx context.Context, req CardRequest) (CardView, error) {
	number := normalizeNumber(req.Number)
	if number == "" {
		return CardView{}, ErrNumberRequired
	}
	status := req.Status
	if status == "" {
		status = StatusActive
	}
	if !validStatus(status) {
		return CardView{}, ErrStatusInvalid
	}

	enc, err := encrypt(s.encryptionKey, number)
	if err != nil {
		return CardView{}, fmt.Errorf("encrypt card number: %w", err)
	}

	id := uuid.NewString()
	now := time.Now().UTC()
	if err := s.repo.CreateBusinessCard(ctx, repo.CreateBusinessCardParams{
		ID:            id,
		CardNumberEnc: enc,
		Last4:         last4(number),
		Label:         nullString(req.Label),
		Bank:          nullString(req.Bank),
		HolderName:    nullString(req.HolderName),
		Expiry:        nullString(req.Expiry),
		CardLimit:     nullInt64(req.CardLimit),
		Status:        status,
		CreatedAt:     now,
		UpdatedAt:     now,
	}); err != nil {
		return CardView{}, fmt.Errorf("create business card: %w", err)
	}

	return s.GetByID(ctx, id)
}

// Update — пустой Number оставляет номер прежним.
func (s *service) Update(ctx context.Context, id string, req CardRequest) (CardView, error) {
	card, err := s.get(ctx, id)
	if err != nil {
		return CardView{}, err
	}

	status := req.Status
	if status == "" {
		status = card.Status
	}
	if !validStatus(status) {
		return CardView{}, ErrStatusInvalid
	}

	enc, l4 := card.CardNumberEnc, card.Last4
	if number := normalizeNumber(req.Number); number != "" {
		if enc, err = encrypt(s.encryptionKey, number); err != nil {
			return CardView{}, fmt.Errorf("encrypt card number: %w", err)
		}
		l4 = last4(number)
	}

	if err := s.repo.UpdateBusinessCard(ctx, repo.UpdateBusinessCardParams{
		CardNumberEnc: enc,
		Last4:         l4,
		Label:         nullString(req.Label),
		Bank:          nullString(req.Bank),
		HolderName:    nullString(req.HolderName),
		Expiry:        nullString(req.Expiry),
		CardLimit:     nullInt64(req.CardLimit),
		Status:        status,
		UpdatedAt:     time.Now().UTC(),
		ID:            id,
	}); err != nil {
		return CardView{}, fmt.Errorf("update business card: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id string) error {
	if _, err := s.get(ctx, id); err != nil {
		return err
	}
	return s.repo.DeleteBusinessCard(ctx, id)
}

func (s *service) GetByID(ctx context.Context, id string) (CardView, error) {
	card, err := s.get(ctx, id)
	if err != nil {
		return CardView{}, err
	}

	view := CardView{
		ID:         card.ID,
		Last4:      card.Last4,
		Label:      ptrString(card.Label),
		Bank:       ptrString(card.Bank),
		HolderName: ptrString(card.HolderName),
		Expiry:     ptrString(card.Expiry),
		CardLimit:  ptrInt64(card.CardLimit),
		Status:     card.Status,
		CreatedAt:  card.CreatedAt,
		UpdatedAt:  card.UpdatedAt,
	}

	a, err := s.repo.GetActiveBusinessCardAssignment(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return CardView{}, err
	}
	if err == nil {
		view.OwnerID = &a.UserID
		view.AssignedAt = &a.AssignedAt
	}
	return view, nil
}

func (s *service) ListAll(ctx context.Context) ([]CardView, error) {
	rows, err := s.repo.ListAllBusinessCards(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CardView, len(rows))
	for i, r := range rows {
		out[i] = viewFromRow(r)
	}
	return out, nil
}

func (s *service) ListByUser(ctx context.Context, userID string) ([]CardView, error) {
	rows, err := s.repo.ListBusinessCardsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]CardView, len(rows))
	for i, r := range rows {
		out[i] = CardView{
			ID:         r.ID,
			Last4:      r.Last4,
			Label:      ptrString(r.Label),
			Bank:       ptrString(r.Bank),
			HolderName: ptrString(r.HolderName),
			Expiry:     ptrString(r.Expiry),
			CardLimit:  ptrInt64(r.CardLimit),
			Status:     r.Status,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
			OwnerID:    &r.OwnerID,
			AssignedAt: &r.AssignedAt,
		}
	}
	return out, nil
}

func (s *service) Number(ctx context.Context, id string) (string, error) {
	card, err := s.get(ctx, id)
	if err != nil {
		return "", err
	}
	return decrypt(s.encryptionKey, card.CardNumberEnc)
}

// Assign закрывает текущую выдачу (если есть) и открывает новую.
func (s *service) Assign(ctx context.Context, id, userID, assignedBy string) (CardView, error) {
	if userID == "" {
		return CardView{}, ErrUserRequired
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CardView{}, err
	}
	defer tx.Rollback()
	qtx := s.repo.WithTx(tx)

	if _, err := qtx.GetBusinessCardForUpdate(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CardView{}, ErrNotFound
		}
		return CardView{}, err
	}

	cur, err := qtx.GetActiveBusinessCardAssignment(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return CardView{}, err
	}
	if err == nil && cur.UserID == userID {
		return CardView{}, ErrSameOwner
	}

	now := time.Now().UTC()
	if err := qtx.ReleaseBusinessCardAssignment(ctx, repo.ReleaseBusinessCardAssignmentParams{
		ReleasedAt: sql.NullTime{Time: now, Valid: true},
		CardID:     id,
	}); err != nil {
		return CardView{}, fmt.Errorf("release assignment: %w", err)
	}
	if err := qtx.CreateBusinessCardAssignment(ctx, repo.CreateBusinessCardAssignmentParams{
		CardID:     id,
		UserID:     userID,
		AssignedBy: assignedBy,
		AssignedAt: now,
	}); err != nil {
		return CardView{}, fmt.Errorf("create assignment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return CardView{}, err
	}
	return s.GetByID(ctx, id)
}

func (s *service) Release(ctx context.Context, id string) (CardView, error) {
	if _, err := s.get(ctx, id); err != nil {
		return CardView{}, err
	}
	if _, err := s.repo.GetActiveBusinessCardAssignment(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CardView{}, ErrNotAssigned
		}
		return CardView{}, err
	}

	if err := s.repo.ReleaseBusinessCardAssignment(ctx, repo.ReleaseBusinessCardAssignmentParams{
		ReleasedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true},
		CardID:     id,
	}); err != nil {
		return CardView{}, fmt.Errorf("release assignment: %w", err)
	}
	return s.GetByID(ctx, id)
}

func (s *service) History(ctx context.Context, id string) ([]Assignment, error) {
	if _, err := s.get(ctx, id); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListBusinessCardAssignments(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]Assignment, len(rows))
	for i, r := range rows {
		out[i] = Assignment{
			ID:         r.ID,
			CardID:     r.CardID,
			UserID:     r.UserID,
			AssignedBy: r.AssignedBy,
			AssignedAt: r.AssignedAt,
		}
		if r.ReleasedAt.Valid {
			out[i].ReleasedAt = &r.ReleasedAt.Time
		}
	}
	return out, nil
}

func (s *service) get(ctx context.Context, id string) (repo.BusinessCard, error) {
	card, err := s.repo.GetBusinessCardByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repo.BusinessCard{}, ErrNotFound
		}
		return repo.BusinessCard{}, err
	}
	return card, nil
}

func viewFromRow(r repo.ListAllBusinessCardsRow) CardView {
	return CardView{
		ID:         r.ID,
		Last4:      r.Last4,
		Label:      ptrString(r.Label),
		Bank:       ptrString(r.Bank),
		HolderName: ptrString(r.HolderName),
		Expiry:     ptrString(r.Expiry),
		CardLimit:  ptrInt64(r.CardLimit),
		Status:     r.Status,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
		OwnerID:    ptrString(r.OwnerID),
		AssignedAt: ptrTime(r.AssignedAt),
	}
}

func validStatus(s string) bool {
	return s == StatusActive || s == StatusBlocked
}

// Пробелы и дефисы из номера убираем — храним и сверяем без форматирования.
func normalizeNumber(s string) string {
	return strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(s))
}

func last4(number string) string {
	r := []rune(number)
	if len(r) <= 4 {
		return number
	}
	return string(r[len(r)-4:])
}

func nullString(v *string) sql.NullString {
	if v == nil || *v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}

func nullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func ptrString(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	return &v.String
}

func ptrInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func ptrTime(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	return &v.Time
}
