-- created_at/updated_at были DATETIME без долей секунды. При добавлении
-- нескольких чеков подряд (например, кнопкой "Добавить все" в
-- ReceiptScan.vue) они получали одинаковый created_at с точностью до
-- секунды — сортировка чеков по дате добавления на фронте (см.
-- filterReceipts в stores/receipt.js) не могла разрешить эту "ничью" и
-- молча падала обратно на исходный порядок с бэка (ORDER BY ticket_date
-- DESC), выглядя так, будто сортировка не работает.

ALTER TABLE `receipts`
  MODIFY COLUMN `created_at` DATETIME(6) NOT NULL,
  MODIFY COLUMN `updated_at` DATETIME(6) NOT NULL;
