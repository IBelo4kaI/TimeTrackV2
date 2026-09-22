-- ============================================
-- categories / merchant_category / keyword_category
-- (авто-категоризация чеков, см. internal/receipt_category)
-- ============================================

-- name: ListCategories :many
SELECT
  *
FROM
  categories
ORDER BY
  name;

-- name: CreateCategory :execlastid
INSERT INTO
  categories (name, is_system)
VALUES
  (?, 0);

-- name: GetMerchantCategory :one
SELECT
  *
FROM
  merchant_category
WHERE
  inn = ?;

-- name: UpsertMerchantCategory :exec
INSERT INTO
  merchant_category (inn, category_id, source)
VALUES
  (?, ?, ?) ON DUPLICATE KEY
UPDATE category_id =
VALUES
  (category_id),
  source =
VALUES
  (source);

-- name: ListKeywordCategories :many
SELECT
  *
FROM
  keyword_category;

-- name: CreateKeyword :exec
INSERT INTO
  keyword_category (keyword, category_id, is_system)
VALUES
  (?, ?, 0);

-- name: UpdateReceiptCategoryID :exec
UPDATE receipts
SET
  category_id = ?,
  updated_at = ?
WHERE
  id = ?;
