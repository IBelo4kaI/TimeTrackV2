-- name: ListDepartmentManagers :many
SELECT * FROM department_managers ORDER BY department, created_at, id;

-- name: ListDepartmentManagersByUser :many
SELECT * FROM department_managers WHERE user_id = ? ORDER BY department, id;

-- name: GetDepartmentManager :one
SELECT * FROM department_managers WHERE id = ?;

-- name: ListDepartmentManagersByScope :many
SELECT * FROM department_managers WHERE department = ? ORDER BY id;

-- name: HasDepartmentManager :one
SELECT COUNT(*) FROM department_managers WHERE user_id = ? AND department = ?;

-- name: CreateDepartmentManager :exec
INSERT INTO department_managers (id, company, department, user_id, created_by) VALUES (?, ?, ?, ?, ?);

-- name: DeleteDepartmentManager :execrows
DELETE FROM department_managers WHERE id = ?;

-- name: SetDepartmentManagerRoleSync :exec
UPDATE department_managers SET role_sync_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: LockDepartmentManager :one
SELECT * FROM department_managers WHERE id = ? FOR UPDATE;

-- name: LockDepartmentManagerRoleSetting :one
SELECT setting_value FROM system_settings WHERE setting_key = 'department_manager_role_id' FOR UPDATE;

-- name: CountOtherDepartmentManagersWithRole :one
SELECT
  (SELECT COUNT(*) FROM department_managers d WHERE d.user_id = sqlc.arg(user_id) AND d.auth_role_id = sqlc.arg(auth_role_id) AND d.id <> sqlc.arg(id)) +
  (SELECT COUNT(*) FROM department_manager_merged_assignments a
   JOIN department_managers m ON m.id = a.retained_id
   WHERE a.user_id = sqlc.arg(user_id) AND a.auth_role_id = sqlc.arg(auth_role_id)
     AND a.retained_id <> sqlc.arg(id) AND a.role_released = FALSE) AS count;

-- name: UpdateDepartmentManagerRole :exec
UPDATE department_managers SET role_sync_status = ?, auth_role_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: ListMergedDepartmentManagerRoles :many
SELECT id, auth_role_id FROM department_manager_merged_assignments
WHERE retained_id = ? AND role_released = FALSE ORDER BY id;

-- name: ReleaseMergedDepartmentManagerRole :exec
UPDATE department_manager_merged_assignments SET role_released = TRUE WHERE id = ?;
