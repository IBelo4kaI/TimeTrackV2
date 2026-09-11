package receipt

import (
	"time"
	repo "timetrack/internal/adapter/mysql/sqlc"
)

// ReceiptItemRequest — одна позиция чека, как её присылает фронт (уже
// разобранный ответ стороннего API "Проверка чека онлайн").
type ReceiptItemRequest struct {
	Name     string  `json:"name"`
	Price    int64   `json:"price"`    // в копейках
	Quantity float64 `json:"quantity"` // может быть дробным (вес)
	Sum      int64   `json:"sum"`      // в копейках
}

// CreateReceiptRequest — тело POST /receipts/create. Фронт сканирует QR,
// сам ходит во внешнее API и присылает сюда уже готовый разобранный ответ +
// userId сотрудника, который отсканировал чек.
type CreateReceiptRequest struct {
	UserID string `json:"userId"`

	FiscalDriveNumber    string `json:"fiscalDriveNumber"`
	FiscalDocumentNumber string `json:"fiscalDocumentNumber"`
	FiscalSign           string `json:"fiscalSign"`

	TicketDate    time.Time `json:"ticketDate"`
	TotalSum      int64     `json:"totalSum"` // в копейках
	SellerINN     string    `json:"sellerInn"`
	SellerName    *string   `json:"sellerName"`
	OperationType int32     `json:"operationType"` // 1-Приход,2-Возврат прихода,3-Расход,4-Возврат расхода

	RetailPlaceAddress *string `json:"retailPlaceAddress"`
	RequestNumber      *string `json:"requestNumber"`
	CashTotalSum       *int64  `json:"cashTotalSum"`  // в копейках
	ECashTotalSum      *int64  `json:"ecashTotalSum"` // в копейках
	TaxationType       *int32  `json:"taxationType"`  // 1-ОСН,2-УСН,4-УСН(доход-расход),8-ЕНВД,16-ЕСХН,32-ПСН

	// НДС в разрезе ставок, в копейках
	Nds20 int64 `json:"nds20"`
	Nds10 int64 `json:"nds10"`
	Nds0  int64 `json:"nds0"`
	NdsNo int64 `json:"ndsNo"`

	RawQR *string `json:"rawQr"` // исходная qr-строка, для отладки

	Items []ReceiptItemRequest `json:"items"`
}

// ReceiptWithItems — карточка чека с позициями. Отдаём при создании и на
// GET /receipts/:id — фронту сразу нужны позиции для отображения.
type ReceiptWithItems struct {
	repo.Receipt
	Items []repo.ReceiptItem `json:"items"`
}
