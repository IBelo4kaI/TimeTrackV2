-- ============================================
-- receipts / receipt_items queries
-- ============================================

-- name: CreateReceipt :exec
INSERT INTO
  receipts (
    id,
    user_id,
    fiscal_drive_number,
    fiscal_document_number,
    fiscal_sign,
    ticket_date,
    total_sum,
    seller_inn,
    seller_name,
    operation_type,
    has_paper,
    categories_manual,
    object_id,
    retail_place_address,
    request_number,
    cash_total_sum,
    ecash_total_sum,
    taxation_type,
    shift_number,
    kkt_reg_id,
    fiscal_document_format_ver,
    machine_number,
    retail_place,
    operator,
    prepaid_sum,
    nds20,
    nds10,
    nds0,
    nds_no,
    nds22,
    raw_qr,
    created_at,
    updated_at
  )
VALUES
  -- created_at/updated_at приходят из Go (time.Now().UTC()), а не
  -- UTC_TIMESTAMP() — та отдаёт только целые секунды (DATETIME(6) без долей
  -- секунды в значении бесполезен), а sqlc к тому же не знает сигнатуру
  -- UTC_TIMESTAMP(6) с аргументом (см. 023_receipts_timestamp_precision.sql).
  (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CreateReceiptItem :exec
INSERT INTO
  receipt_items (receipt_id, position_no, name, price, quantity, sum, nds_code)
VALUES
  (?, ?, ?, ?, ?, ?, ?);

-- name: GetReceiptByID :one
SELECT
  *
FROM
  receipts
WHERE
  id = ?;

-- name: GetReceiptByFiscalKey :one
SELECT
  *
FROM
  receipts
WHERE
  fiscal_drive_number = ?
  AND fiscal_document_number = ?
  AND fiscal_sign = ?;

-- name: ListReceiptItemsByReceipt :many
SELECT
  *
FROM
  receipt_items
WHERE
  receipt_id = ?
ORDER BY
  position_no,
  id;

-- name: ListReceiptsByUser :many
SELECT
  *
FROM
  receipts
WHERE
  user_id = ?
ORDER BY
  ticket_date DESC;

-- name: ListAllReceipts :many
SELECT
  *
FROM
  receipts
ORDER BY
  ticket_date DESC;

-- name: DeleteReceipt :exec
DELETE FROM receipts
WHERE
  id = ?;

-- name: UpdateReceiptObjectID :exec
UPDATE receipts
SET
  object_id = ?,
  updated_at = ?
WHERE
  id = ?;

-- name: UpdateReceiptOwner :exec
UPDATE receipts
SET
  user_id = ?,
  updated_at = ?
WHERE
  id = ?;
