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
    retail_place_address,
    request_number,
    cash_total_sum,
    ecash_total_sum,
    taxation_type,
    nds20,
    nds10,
    nds0,
    nds_no,
    raw_qr,
    created_at,
    updated_at
  )
VALUES
  (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, UTC_TIMESTAMP(), UTC_TIMESTAMP());

-- name: CreateReceiptItem :exec
INSERT INTO
  receipt_items (receipt_id, position_no, name, price, quantity, sum)
VALUES
  (?, ?, ?, ?, ?, ?);

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
