package receipt

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	repo "timetrack/internal/adapter/mysql/sqlc"

	"github.com/google/uuid"
)

var (
	ErrNotFound           = errors.New("чек не найден")
	ErrDuplicate          = errors.New("этот чек уже был добавлен ранее")
	ErrUserRequired       = errors.New("не указан пользователь, отсканировавший чек")
	ErrFiscalDataRequired = errors.New("не переданы фискальные реквизиты чека (ФН/ФД/ФПД)")
	ErrTotalSumInvalid    = errors.New("некорректная сумма чека")
	ErrSellerInnRequired  = errors.New("не указан ИНН продавца")
	ErrNoItems            = errors.New("в чеке нет ни одной позиции")
)

type Service interface {
	Create(ctx context.Context, req CreateReceiptRequest) (ReceiptWithItems, error)
	GetByID(ctx context.Context, id string) (ReceiptWithItems, error)
	ListByUser(ctx context.Context, userID string) ([]repo.Receipt, error)
	ListAll(ctx context.Context) ([]repo.Receipt, error)
	Delete(ctx context.Context, id string) error
}

type receiptService struct {
	repo *repo.Queries
	db   *sql.DB
}

func NewService(repo *repo.Queries, db *sql.DB) Service {
	return &receiptService{repo: repo, db: db}
}

func (s *receiptService) Create(ctx context.Context, req CreateReceiptRequest) (ReceiptWithItems, error) {
	if err := validate(req); err != nil {
		return ReceiptWithItems{}, err
	}

	if _, err := s.repo.GetReceiptByFiscalKey(ctx, repo.GetReceiptByFiscalKeyParams{
		FiscalDriveNumber:    req.FiscalDriveNumber,
		FiscalDocumentNumber: req.FiscalDocumentNumber,
		FiscalSign:           req.FiscalSign,
	}); err == nil {
		return ReceiptWithItems{}, ErrDuplicate
	} else if !errors.Is(err, sql.ErrNoRows) {
		return ReceiptWithItems{}, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return ReceiptWithItems{}, err
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	id := uuid.NewString()
	if err := qtx.CreateReceipt(ctx, repo.CreateReceiptParams{
		ID:                   id,
		UserID:               req.UserID,
		FiscalDriveNumber:    req.FiscalDriveNumber,
		FiscalDocumentNumber: req.FiscalDocumentNumber,
		FiscalSign:           req.FiscalSign,
		TicketDate:           req.TicketDate,
		TotalSum:             req.TotalSum,
		SellerInn:            req.SellerINN,
		SellerName:           nullString(req.SellerName),
		OperationType:        req.OperationType,
		RetailPlaceAddress:   nullString(req.RetailPlaceAddress),
		RequestNumber:        nullString(req.RequestNumber),
		CashTotalSum:         nullInt64(req.CashTotalSum),
		EcashTotalSum:        nullInt64(req.ECashTotalSum),
		TaxationType:         nullInt32(req.TaxationType),
		Nds20:                req.Nds20,
		Nds10:                req.Nds10,
		Nds0:                 req.Nds0,
		NdsNo:                req.NdsNo,
		RawQr:                nullString(req.RawQR),
	}); err != nil {
		return ReceiptWithItems{}, fmt.Errorf("create receipt: %w", err)
	}

	for i, item := range req.Items {
		positionNo := int32(i + 1)
		if err := qtx.CreateReceiptItem(ctx, repo.CreateReceiptItemParams{
			ReceiptID:  id,
			PositionNo: sql.NullInt32{Int32: positionNo, Valid: true},
			Name:       item.Name,
			Price:      item.Price,
			Quantity:   strconv.FormatFloat(item.Quantity, 'f', 3, 64),
			Sum:        item.Sum,
		}); err != nil {
			return ReceiptWithItems{}, fmt.Errorf("create receipt item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return ReceiptWithItems{}, err
	}

	return s.GetByID(ctx, id)
}

func (s *receiptService) GetByID(ctx context.Context, id string) (ReceiptWithItems, error) {
	r, err := s.repo.GetReceiptByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ReceiptWithItems{}, ErrNotFound
		}
		return ReceiptWithItems{}, err
	}

	items, err := s.repo.ListReceiptItemsByReceipt(ctx, id)
	if err != nil {
		return ReceiptWithItems{}, err
	}

	return ReceiptWithItems{Receipt: r, Items: items}, nil
}

func (s *receiptService) ListByUser(ctx context.Context, userID string) ([]repo.Receipt, error) {
	return s.repo.ListReceiptsByUser(ctx, userID)
}

func (s *receiptService) ListAll(ctx context.Context) ([]repo.Receipt, error) {
	return s.repo.ListAllReceipts(ctx)
}

func (s *receiptService) Delete(ctx context.Context, id string) error {
	if _, err := s.repo.GetReceiptByID(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	// receipt_items удалятся каскадом (fk_receipt_items_receipt ON DELETE CASCADE)
	if err := s.repo.DeleteReceipt(ctx, id); err != nil {
		return fmt.Errorf("delete receipt: %w", err)
	}
	return nil
}

func validate(req CreateReceiptRequest) error {
	if req.UserID == "" {
		return ErrUserRequired
	}
	if req.FiscalDriveNumber == "" || req.FiscalDocumentNumber == "" || req.FiscalSign == "" {
		return ErrFiscalDataRequired
	}
	if req.TotalSum <= 0 {
		return ErrTotalSumInvalid
	}
	if req.SellerINN == "" {
		return ErrSellerInnRequired
	}
	if len(req.Items) == 0 {
		return ErrNoItems
	}
	return nil
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

func nullInt32(v *int32) sql.NullInt32 {
	if v == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *v, Valid: true}
}
