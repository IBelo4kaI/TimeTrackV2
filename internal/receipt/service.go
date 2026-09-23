package receipt

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
	receiptcategory "timetrack/internal/receipt_category"

	"github.com/google/uuid"
)

var (
	ErrNotFound           = errors.New("чек не найден")
	ErrDuplicate          = errors.New("этот чек уже был добавлен ранее")
	ErrUserRequired       = errors.New("не указан пользователь, отсканировавший чек")
	ErrFiscalDataRequired = errors.New("не переданы фискальные реквизиты чека (ФН/ФД/ФПД)")
	ErrTotalSumInvalid    = errors.New("некорректная сумма чека")
	ErrNoItems            = errors.New("в чеке нет ни одной позиции")
	ErrNewOwnerRequired   = errors.New("не указан сотрудник, которому передаётся чек")
	ErrSameOwner          = errors.New("чек уже принадлежит этому сотруднику")
)

type Service interface {
	Create(ctx context.Context, req CreateReceiptRequest) (ReceiptWithItems, error)
	GetByID(ctx context.Context, id string) (ReceiptWithItems, error)
	ListByUser(ctx context.Context, userID string) ([]repo.Receipt, error)
	ListAll(ctx context.Context) ([]repo.Receipt, error)
	Delete(ctx context.Context, id string) error
	Transfer(ctx context.Context, id, newUserID string) (ReceiptWithItems, error)
	// SetCategory — ручная правка категории чека (categoryID == nil снимает
	// категорию). При непустом categoryID запоминает выбор как
	// user_override в словаре продавцов (см. receiptcategory.Service).
	SetCategory(ctx context.Context, id string, categoryID *int32) (ReceiptWithItems, error)
	// BackfillCategories — перепрогоняет через классификацию ВСЕ уже
	// сохранённые чеки (не только "Без категории" — и те, что уже
	// классифицированы, тоже: словари категорий/ключевых слов меняются со
	// временем, старая категория чека могла устареть). Тот же алгоритм и
	// то же самообучение словаря продавцов, что и в Create. Возвращает,
	// сколько чеков реально сменили категорию.
	BackfillCategories(ctx context.Context) (updated int, total int, err error)
}

type receiptService struct {
	repo            *repo.Queries
	db              *sql.DB
	categoryService receiptcategory.Service
}

func NewService(repo *repo.Queries, db *sql.DB, categoryService receiptcategory.Service) Service {
	return &receiptService{repo: repo, db: db, categoryService: categoryService}
}

func (s *receiptService) Create(ctx context.Context, req CreateReceiptRequest) (ReceiptWithItems, error) {
	if err := validate(req); err != nil {
		return ReceiptWithItems{}, err
	}

	// Проверка на дубликат — только когда есть фискальные реквизиты (скан):
	// у ручных чеков их нет, сравнивать/дедуплицировать по ним нечего.
	if req.FiscalDriveNumber != nil {
		if _, err := s.repo.GetReceiptByFiscalKey(ctx, repo.GetReceiptByFiscalKeyParams{
			FiscalDriveNumber:    nullString(req.FiscalDriveNumber),
			FiscalDocumentNumber: nullString(req.FiscalDocumentNumber),
			FiscalSign:           nullString(req.FiscalSign),
		}); err == nil {
			return ReceiptWithItems{}, ErrDuplicate
		} else if !errors.Is(err, sql.ErrNoRows) {
			return ReceiptWithItems{}, err
		}
	}

	sellerINN := ""
	if req.SellerINN != nil {
		sellerINN = *req.SellerINN
	}

	// Классификация — локальными словарями (ИНН продавца, потом ключевые
	// слова позиций), см. internal/receipt_category. До открытия транзакции
	// ниже: сама может писать в merchant_category (самообучение), это
	// отдельная операция, чек ещё не обязан существовать в БД. Пустой ИНН
	// (ручной ввод) Classify обрабатывает штатно — просто пропускает шаг
	// поиска по словарю продавцов.
	classifyItems := make([]receiptcategory.ItemForClassify, len(req.Items))
	for i, item := range req.Items {
		classifyItems[i] = receiptcategory.ItemForClassify{Name: item.Name}
	}
	classified, err := s.categoryService.Classify(ctx, sellerINN, classifyItems)
	if err != nil {
		return ReceiptWithItems{}, fmt.Errorf("classify receipt: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return ReceiptWithItems{}, err
	}
	defer tx.Rollback()

	qtx := s.repo.WithTx(tx)

	id := uuid.NewString()
	now := time.Now().UTC()
	if err := qtx.CreateReceipt(ctx, repo.CreateReceiptParams{
		ID:                      id,
		UserID:                  req.UserID,
		FiscalDriveNumber:       nullString(req.FiscalDriveNumber),
		FiscalDocumentNumber:    nullString(req.FiscalDocumentNumber),
		FiscalSign:              nullString(req.FiscalSign),
		TicketDate:              req.TicketDate,
		TotalSum:                req.TotalSum,
		SellerInn:               nullString(req.SellerINN),
		SellerName:              nullString(req.SellerName),
		OperationType:           req.OperationType,
		HasPaper:                req.HasPaper,
		CategoryID:              nullInt32(classified.CategoryID),
		RetailPlaceAddress:      nullString(req.RetailPlaceAddress),
		RequestNumber:           nullString(req.RequestNumber),
		CashTotalSum:            nullInt64(req.CashTotalSum),
		EcashTotalSum:           nullInt64(req.ECashTotalSum),
		TaxationType:            nullInt32(req.TaxationType),
		ShiftNumber:             nullInt32(req.ShiftNumber),
		KktRegID:                nullString(req.KktRegID),
		FiscalDocumentFormatVer: nullInt32(req.FiscalDocumentFormatVer),
		MachineNumber:           nullString(req.MachineNumber),
		RetailPlace:             nullString(req.RetailPlace),
		Operator:                nullString(req.Operator),
		PrepaidSum:              nullInt64(req.PrepaidSum),
		Nds20:                   req.Nds20,
		Nds10:                   req.Nds10,
		Nds0:                    req.Nds0,
		NdsNo:                   req.NdsNo,
		Nds22:                   req.Nds22,
		RawQr:                   nullString(req.RawQR),
		CreatedAt:               now,
		UpdatedAt:               now,
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
			NdsCode:    nullInt32(item.NdsCode),
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

func (s *receiptService) Transfer(ctx context.Context, id, newUserID string) (ReceiptWithItems, error) {
	if newUserID == "" {
		return ReceiptWithItems{}, ErrNewOwnerRequired
	}

	r, err := s.repo.GetReceiptByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ReceiptWithItems{}, ErrNotFound
		}
		return ReceiptWithItems{}, err
	}

	if r.UserID == newUserID {
		return ReceiptWithItems{}, ErrSameOwner
	}

	if err := s.repo.UpdateReceiptOwner(ctx, repo.UpdateReceiptOwnerParams{
		UserID:    newUserID,
		UpdatedAt: time.Now().UTC(),
		ID:        id,
	}); err != nil {
		return ReceiptWithItems{}, fmt.Errorf("update receipt owner: %w", err)
	}

	return s.GetByID(ctx, id)
}

func (s *receiptService) SetCategory(ctx context.Context, id string, categoryID *int32) (ReceiptWithItems, error) {
	r, err := s.repo.GetReceiptByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ReceiptWithItems{}, ErrNotFound
		}
		return ReceiptWithItems{}, err
	}

	if err := s.repo.UpdateReceiptCategoryID(ctx, repo.UpdateReceiptCategoryIDParams{
		CategoryID: nullInt32(categoryID),
		UpdatedAt:  time.Now().UTC(),
		ID:         id,
	}); err != nil {
		return ReceiptWithItems{}, fmt.Errorf("update receipt category: %w", err)
	}

	// Снятие категории (categoryID == nil) — только у этого чека, словарь
	// продавца не трогаем (это не значит "этот продавец вообще без
	// категории"). Непустой выбор — как и правка через "Продавцы" в
	// настройках (UpdateMerchant): запоминаем override И сразу переносим
	// категорию на ВСЕ чеки этого продавца, а не только на будущие —
	// иначе "Категоризировать" не подхватит уже категоризированные чеки
	// с устаревшей категорией (Backfill трогает только "Без категории").
	// Без ИНН (ручной чек) обновлять словарь продавцов нечем — только сам
	// чек, UpdateMerchant тут не вызываем (ему обязателен непустой ИНН).
	if categoryID != nil && r.SellerInn.Valid && r.SellerInn.String != "" {
		if _, err := s.categoryService.UpdateMerchant(ctx, r.SellerInn.String, *categoryID); err != nil {
			fmt.Printf("update merchant category override: %v\n", err)
		}
	}

	return s.GetByID(ctx, id)
}

func (s *receiptService) BackfillCategories(ctx context.Context) (int, int, error) {
	receipts, err := s.repo.ListAllReceipts(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("list receipts: %w", err)
	}

	updated := 0
	for _, r := range receipts {
		items, err := s.repo.ListReceiptItemsByReceipt(ctx, r.ID)
		if err != nil {
			fmt.Printf("backfill category: list items for receipt %s: %v\n", r.ID, err)
			continue
		}

		classifyItems := make([]receiptcategory.ItemForClassify, len(items))
		for i, item := range items {
			classifyItems[i] = receiptcategory.ItemForClassify{Name: item.Name}
		}

		classified, err := s.categoryService.Classify(ctx, r.SellerInn.String, classifyItems)
		if err != nil {
			fmt.Printf("backfill category: classify receipt %s: %v\n", r.ID, err)
			continue
		}
		if classified.CategoryID == nil {
			// Не определилось — старую категорию (если была) не трогаем:
			// это не сигнал "снять категорию", а просто "нечего сказать".
			continue
		}
		if r.CategoryID.Valid && r.CategoryID.Int32 == *classified.CategoryID {
			continue // уже такая же — писать нечего
		}

		if err := s.repo.UpdateReceiptCategoryID(ctx, repo.UpdateReceiptCategoryIDParams{
			CategoryID: nullInt32(classified.CategoryID),
			UpdatedAt:  time.Now().UTC(),
			ID:         r.ID,
		}); err != nil {
			fmt.Printf("backfill category: update receipt %s: %v\n", r.ID, err)
			continue
		}
		updated++
	}

	return updated, len(receipts), nil
}

// validate — ИНН продавца больше не обязателен (см. CreateReceiptRequest):
// чек можно ввести вручную, когда QR нет/не читается. Фискальные реквизиты
// в этом случае тоже не заполняются, но смешивать нельзя — либо все три
// указаны (скан), либо все три отсутствуют (ручной ввод).
func validate(req CreateReceiptRequest) error {
	if req.UserID == "" {
		return ErrUserRequired
	}
	fiscalFieldsPresent := 0
	for _, v := range []*string{req.FiscalDriveNumber, req.FiscalDocumentNumber, req.FiscalSign} {
		if v != nil && *v != "" {
			fiscalFieldsPresent++
		}
	}
	if fiscalFieldsPresent != 0 && fiscalFieldsPresent != 3 {
		return ErrFiscalDataRequired
	}
	if req.TotalSum <= 0 {
		return ErrTotalSumInvalid
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
