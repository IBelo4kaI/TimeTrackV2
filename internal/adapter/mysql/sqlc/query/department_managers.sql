-- name: ListDepartmentManagers :many
SELECT * FROM department_managers ORDER BY company, department, created_at, id;

-- name: GetDepartmentManager :one
SELECT * FROM department_managers WHERE id = ?;

-- name: ListDepartmentManagersByScope :many
SELECT * FROM department_managers WHERE company = ? AND department = ? ORDER BY id;

-- name: HasDepartmentManager :one
SELECT COUNT(*) FROM department_managers WHERE user_id = ? AND company = ? AND department = ?;

-- name: CreateDepartmentManager :exec
INSERT INTO department_managers (id, company, department, user_id, created_by) VALUES (?, ?, ?, ?, ?);

-- name: DeleteDepartmentManager :execrows
DELETE FROM department_managers WHERE id = ?;

-- name: SetDepartmentManagerRoleSync :exec
UPDATE department_managers SET role_sync_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;
