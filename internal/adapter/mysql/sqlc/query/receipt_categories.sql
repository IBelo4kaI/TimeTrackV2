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
-- с чека этого продавца, просто для отображения в UI (см. экран настроек
-- "Категории и слова" -> "Продавцы"). Сортируем по ticket_date (дата самой
-- покупки), а не updated_at — тот меняется у всех чеков разом при массовой
-- перекатегоризации (BackfillCategories/UpdateMerchant), из-за чего "самый
-- свежий" чек и его seller_name раньше менялись от одного этого, а не от
-- новых покупок. seller_name IS NULL — в конец, чтобы не показывать пусто,
-- когда есть чек с реальным именем продавца.
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
      (r.seller_name IS NULL) ASC,
      r.ticket_date DESC
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

-- name: ListReceiptCategoryIDs :many
SELECT
  category_id
FROM
  receipt_categories
WHERE
  receipt_id = ?
ORDER BY
  category_id;

-- name: ListReceiptCategoryLinksByUser :many
SELECT
  rc.receipt_id,
  rc.category_id
FROM
  receipt_categories rc
  JOIN receipts r ON r.id = rc.receipt_id
WHERE
  r.user_id = ?
ORDER BY
  rc.category_id;

-- name: ListAllReceiptCategoryLinks :many
SELECT
  receipt_id,
  category_id
FROM
  receipt_categories
ORDER BY
  category_id;

-- name: DeleteReceiptCategoryLinks :exec
DELETE FROM receipt_categories
WHERE
  receipt_id = ?;

-- name: InsertReceiptCategoryLink :exec
INSERT IGNORE INTO
  receipt_categories (receipt_id, category_id)
VALUES
  (?, ?);

-- name: SetReceiptCategoriesManual :exec
UPDATE receipts
SET
  categories_manual = ?,
  updated_at = ?
WHERE
  id = ?;

-- name: DeleteAutoCategoryLinksBySellerInn :exec
-- Ретроактивное обновление по продавцу (см. receiptcategory.Service.
-- UpdateMerchant) — только чеки, где категории не выбраны вручную.
DELETE FROM receipt_categories
WHERE
  receipt_id IN (
    SELECT
      id
    FROM
      receipts
    WHERE
      seller_inn = ?
      AND categories_manual = FALSE
  );

-- name: InsertAutoCategoryLinkBySellerInn :execrows
INSERT IGNORE INTO
  receipt_categories (receipt_id, category_id)
SELECT
  id,
  ?
FROM
  receipts
WHERE
  seller_inn = ?
  AND categories_manual = FALSE;
