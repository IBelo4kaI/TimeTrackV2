package businesscard

import "time"

const (
	StatusActive  = "active"
	StatusBlocked = "blocked"
)

// Number — строка: у сторонних API номер может прийти не только цифрами.
type CardRequest struct {
	Number     string  `json:"number"`
	Label      *string `json:"label"`
	Bank       *string `json:"bank"`
	HolderName *string `json:"holderName"`
	Expiry     *string `json:"expiry"`
	CardLimit  *int64  `json:"cardLimit"` // в копейках
	Status     string  `json:"status"`
}

// CardView — карта без полного номера.
type CardView struct {
	ID         string     `json:"id"`
	Last4      string     `json:"last4"`
	Label      *string    `json:"label"`
	Bank       *string    `json:"bank"`
	HolderName *string    `json:"holderName"`
	Expiry     *string    `json:"expiry"`
	CardLimit  *int64     `json:"cardLimit"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	OwnerID    *string    `json:"ownerId"`
	AssignedAt *time.Time `json:"assignedAt"`
}

type AssignRequest struct {
	UserID string `json:"userId"`
}

type NumberResponse struct {
	Number string `json:"number"`
}

type Assignment struct {
	ID         int64      `json:"id"`
	CardID     string     `json:"cardId"`
	UserID     string     `json:"userId"`
	AssignedBy string     `json:"assignedBy"`
	AssignedAt time.Time  `json:"assignedAt"`
	ReleasedAt *time.Time `json:"releasedAt"`
}
