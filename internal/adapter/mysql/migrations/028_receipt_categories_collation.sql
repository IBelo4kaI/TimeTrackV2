-- 026 создала categories/merchant_category/keyword_category с
-- DEFAULT CHARSET=utf8mb4 без явного COLLATE — сервер подставил свою
-- дефолтную коллацию (utf8mb4_0900_ai_ci), а не utf8mb4_unicode_ci, которой
-- явно создана receipts (см. 022_add_receipts.sql). Из-за этого сравнение
-- receipts.seller_inn = merchant_category.inn в ListMerchantCategories
-- падало с "Illegal mix of collations". Приводим все три таблицы к той же
-- коллации, что и receipts.
ALTER TABLE `categories` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
ALTER TABLE `merchant_category` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
ALTER TABLE `keyword_category` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
