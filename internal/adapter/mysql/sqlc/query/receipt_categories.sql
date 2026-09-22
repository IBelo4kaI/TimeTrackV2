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

-- name: RenameCategory :exec
UPDATE categories
SET
  name = ?
WHERE
  id = ?;

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

-- name: ListMerchantCategories :many
-- seller_name — не своя колонка (merchant_category знает только ИНН), берём
-- с последнего по времени чека от этого продавца, просто для отображения
-- в UI (см. экран настроек "Категории и слова" -> "Продавцы").
SELECT
  mc.inn,
  mc.category_id,
  mc.source,
  (
    SELECT
      r.seller_name
    FROM
      receipts r
    WHERE
      r.seller_inn = mc.inn
    ORDER BY
      r.updated_at DESC
    LIMIT
      1
  ) AS seller_name
FROM
  merchant_category mc
ORDER BY
  mc.inn;

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
