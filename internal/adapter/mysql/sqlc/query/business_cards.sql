-- ============================================
-- business_cards / business_card_assignments queries
-- ============================================

-- name: CreateBusinessCard :exec
INSERT INTO
  business_cards (
    id,
    card_number_enc,
    last4,
    label,
    bank,
    holder_name,
    expiry,
    card_limit,
    status,
    created_at,
    updated_at
  )
VALUES
  (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetBusinessCardByID :one
SELECT
  *
FROM
  business_cards
WHERE
  id = ?;

-- name: GetBusinessCardForUpdate :one
SELECT
  id
FROM
  business_cards
WHERE
  id = ?
FOR UPDATE;

-- name: UpdateBusinessCard :exec
UPDATE business_cards
SET
  card_number_enc = ?,
  last4 = ?,
  label = ?,
  bank = ?,
  holder_name = ?,
  expiry = ?,
  card_limit = ?,
  status = ?,
  updated_at = ?
WHERE
  id = ?;

-- name: DeleteBusinessCard :exec
DELETE FROM business_cards
WHERE
  id = ?;

-- name: ListAllBusinessCards :many
SELECT
  c.id,
  c.last4,
  c.label,
  c.bank,
  c.holder_name,
  c.expiry,
  c.card_limit,
  c.status,
  c.created_at,
  c.updated_at,
  a.user_id AS owner_id,
  a.assigned_at AS assigned_at
FROM
  business_cards c
  LEFT JOIN business_card_assignments a ON a.card_id = c.id
  AND a.released_at IS NULL
ORDER BY
  c.created_at DESC;

-- name: ListBusinessCardsByUser :many
SELECT
  c.id,
  c.last4,
  c.label,
  c.bank,
  c.holder_name,
  c.expiry,
  c.card_limit,
  c.status,
  c.created_at,
  c.updated_at,
  a.user_id AS owner_id,
  a.assigned_at AS assigned_at
FROM
  business_cards c
  JOIN business_card_assignments a ON a.card_id = c.id
  AND a.released_at IS NULL
WHERE
  a.user_id = ?
ORDER BY
  c.created_at DESC;

-- name: GetActiveBusinessCardAssignment :one
SELECT
  *
FROM
  business_card_assignments
WHERE
  card_id = ?
  AND released_at IS NULL;

-- name: ReleaseBusinessCardAssignment :exec
UPDATE business_card_assignments
SET
  released_at = ?
WHERE
  card_id = ?
  AND released_at IS NULL;

-- name: CreateBusinessCardAssignment :exec
INSERT INTO
  business_card_assignments (card_id, user_id, assigned_by, assigned_at)
VALUES
  (?, ?, ?, ?);

-- name: ListBusinessCardAssignments :many
SELECT
  *
FROM
  business_card_assignments
WHERE
  card_id = ?
ORDER BY
  assigned_at DESC,
  id DESC;

-- name: ListReceiptsByBusinessCard :many
SELECT
  *
FROM
  receipts
WHERE
  business_card_id = ?
ORDER BY
  ticket_date DESC;
