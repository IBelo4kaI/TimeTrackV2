package filecategory

import "time"

// FileCategoryNode — категория файлов с вычисленным деревом дочерних категорий.
// SystemName заполнен только у категорий, заведённых кодом (миграцией) —
// по нему бэкенд автоматически раскладывает файлы при загрузке
// (см. internal/service/file.go). Такие категории защищены от удаления.
type FileCategoryNode struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	SystemName *string             `json:"systemName"`
	ParentID   *string             `json:"parentId"`
	ColorCode  *string             `json:"colorCode"`
	SortOrder  int32               `json:"sortOrder"`
	CreatedAt  time.Time           `json:"createdAt"`
	UpdatedAt  time.Time           `json:"updatedAt"`
	Children   []*FileCategoryNode `json:"children"`
}

type CreateFileCategoryRequest struct {
	Name      string  `json:"name"`
	ParentID  *string `json:"parentId"`
	ColorCode *string `json:"colorCode"`
	SortOrder int32   `json:"sortOrder"`
}

type UpdateFileCategoryRequest struct {
	Name      string  `json:"name"`
	ParentID  *string `json:"parentId"`
	ColorCode *string `json:"colorCode"`
	SortOrder int32   `json:"sortOrder"`
}
