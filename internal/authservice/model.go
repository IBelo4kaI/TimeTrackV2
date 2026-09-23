package authservice

import "time"

// UserStatus — локальная копия enum-типа сервиса авторизации (там это
// repo.UsersStatus, свой sqlc-тип того сервиса, импортировать напрямую
// нельзя — это отдельный модуль). Значение приходит строкой в JSON как есть.
type UserStatus string

// Gender — в отличие от UserStatus, на самом деле объект {id, name}, не
// строка (несмотря на repo.Gender в сигнатуре, которую прислали как образец).
type Gender struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RoleResponse/UserResponse — повторяют JSON-контракт GET /api/as/users/all
// сервиса авторизации (см. authservice.Service.GetAllUsers).
type RoleResponse struct {
	ID               string    `json:"id"`
	ServiceID        *string   `json:"service_id,omitempty"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	IsGlobal         bool      `json:"is_global"`
	ServiceName      string    `json:"service_name,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UserCount        int64     `json:"user_count"`
	PermissionsCount int64     `json:"permissions_count"`
}

type UserResponse struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Surname    string         `json:"surname"`
	Patronymic *string        `json:"patronymic"`
	Username   string         `json:"username"`
	Birthday   time.Time      `json:"birthday"`
	Status     UserStatus     `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	Gender     Gender         `json:"gender"`
	Roles      []RoleResponse `json:"roles"`
	RolesCount int            `json:"roles_count"`
}
