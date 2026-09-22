package receiptcategory

// CreateCategoryRequest — тело POST /receipt-categories.
type CreateCategoryRequest struct {
	Name string `json:"name"`
}

// CreateKeywordRequest — тело POST /receipt-categories/keywords.
type CreateKeywordRequest struct {
	Keyword    string `json:"keyword"`
	CategoryID int32  `json:"categoryId"`
}

// PreviewRequest — тело POST /receipt-categories/preview. Отправляется до
// сохранения чека (см. ReceiptScan.vue) — показать сотруднику, какая
// категория определится, ещё до нажатия "Добавить".
type PreviewRequest struct {
	SellerInn string             `json:"sellerInn"`
	Items     []PreviewItemInput `json:"items"`
}

type PreviewItemInput struct {
	Name string `json:"name"`
}

// PreviewResponse — categoryId == nil, если ничего не определилось
// ("Без категории").
type PreviewResponse struct {
	CategoryID *int32 `json:"categoryId"`
}
