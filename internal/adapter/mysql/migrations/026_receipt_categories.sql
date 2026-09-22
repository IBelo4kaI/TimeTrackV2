-- Авто-категоризация чеков по спецификации (локальные словари, без внешних
-- API): категории, словарь "ИНН продавца -> категория" (самообучающийся) и
-- словарь "ключевое слово -> категория" для классификации по позициям чека,
-- когда продавец ещё не встречался. См. internal/receipt_category.
--
-- id — INT AUTO_INCREMENT (не UUID, как у остальных таблиц в проекте) —
-- сознательное отклонение от общего стиля: это внешние ключи внутри одной
-- фичи, не идентификаторы сущностей, которыми где-то ещё оперируют по UUID.
CREATE TABLE IF NOT EXISTS `categories` (
  `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `name` VARCHAR(64) NOT NULL,
  -- true — предзаполненная (эта миграция), false — добавлена пользователем
  -- на экране настроек "Категории и слова"
  `is_system` TINYINT(1) NOT NULL DEFAULT 1,
  UNIQUE KEY `uq_categories_name` (`name`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- ИНН -> категория. source: 'seed' (заведено вручную заранее, сейчас пусто
-- — конкретных ИНН реальных продавцов в спецификации нет) | 'keyword_match'
-- (нашлось по ключевым словам позиций, записывается автоматически — см.
-- ClassifyReceipt) | 'user_override' (сотрудник поправил категорию чека
-- вручную — перезаписывает то, что было раньше, и с этого момента все чеки
-- от этого продавца идут по этому маппингу напрямую, шаг 1 алгоритма).
CREATE TABLE IF NOT EXISTS `merchant_category` (
  `inn` VARCHAR(20) NOT NULL PRIMARY KEY,
  `category_id` INT NOT NULL,
  `source` VARCHAR(20) NOT NULL,
  CONSTRAINT `fk_merchant_category_category` FOREIGN KEY (`category_id`) REFERENCES `categories` (`id`) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- Слово (уже в нормальной форме) -> категория, для классификации по
-- позициям чека, когда продавца ещё нет в merchant_category.
CREATE TABLE IF NOT EXISTS `keyword_category` (
  `keyword` VARCHAR(64) NOT NULL PRIMARY KEY,
  `category_id` INT NOT NULL,
  `is_system` TINYINT(1) NOT NULL DEFAULT 1,
  CONSTRAINT `fk_keyword_category_category` FOREIGN KEY (`category_id`) REFERENCES `categories` (`id`) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

INSERT IGNORE INTO `categories` (`name`) VALUES
  ('Продукты'), ('Кафе и рестораны'), ('Транспорт'), ('АЗС'),
  ('Здоровье и аптеки'), ('Одежда и обувь'), ('Дом и быт'), ('Электроника'),
  ('Развлечения'), ('Связь и интернет'), ('Коммунальные услуги'),
  ('Красота и уход'), ('Спорт'), ('Дети'), ('Животные'), ('Образование'),
  ('Автомобиль (сервис)'), ('Подарки и цветы'), ('Путешествия'),
  ('Алкоголь и табак'), ('Стройматериалы'), ('Инструменты'),
  ('Мебель и интерьер'), ('Канцтовары и хобби'), ('Банковские услуги'),
  ('Прочее');

-- Seed ключевых слов — пары (слово_или_фраза, категория), джойним по имени
-- категории, чтобы не завязываться на конкретные id AUTO_INCREMENT.
INSERT IGNORE INTO `keyword_category` (`keyword`, `category_id`, `is_system`)
SELECT k.keyword, c.id, 1
FROM (
  SELECT 'хлеб' AS keyword, 'Продукты' AS category UNION ALL
  SELECT 'батон', 'Продукты' UNION ALL
  SELECT 'булка', 'Продукты' UNION ALL
  SELECT 'молоко', 'Продукты' UNION ALL
  SELECT 'кефир', 'Продукты' UNION ALL
  SELECT 'йогурт', 'Продукты' UNION ALL
  SELECT 'творог', 'Продукты' UNION ALL
  SELECT 'сметана', 'Продукты' UNION ALL
  SELECT 'сыр', 'Продукты' UNION ALL
  SELECT 'масло', 'Продукты' UNION ALL
  SELECT 'яйцо', 'Продукты' UNION ALL
  SELECT 'мясо', 'Продукты' UNION ALL
  SELECT 'курица', 'Продукты' UNION ALL
  SELECT 'говядина', 'Продукты' UNION ALL
  SELECT 'свинина', 'Продукты' UNION ALL
  SELECT 'фарш', 'Продукты' UNION ALL
  SELECT 'колбаса', 'Продукты' UNION ALL
  SELECT 'сосиска', 'Продукты' UNION ALL
  SELECT 'рыба', 'Продукты' UNION ALL
  SELECT 'картофель', 'Продукты' UNION ALL
  SELECT 'картошка', 'Продукты' UNION ALL
  SELECT 'морковь', 'Продукты' UNION ALL
  SELECT 'лук', 'Продукты' UNION ALL
  SELECT 'капуста', 'Продукты' UNION ALL
  SELECT 'помидор', 'Продукты' UNION ALL
  SELECT 'огурец', 'Продукты' UNION ALL
  SELECT 'яблоко', 'Продукты' UNION ALL
  SELECT 'банан', 'Продукты' UNION ALL
  SELECT 'апельсин', 'Продукты' UNION ALL
  SELECT 'груша', 'Продукты' UNION ALL
  SELECT 'сахар', 'Продукты' UNION ALL
  SELECT 'соль', 'Продукты' UNION ALL
  SELECT 'мука', 'Продукты' UNION ALL
  SELECT 'крупа', 'Продукты' UNION ALL
  SELECT 'рис', 'Продукты' UNION ALL
  SELECT 'гречка', 'Продукты' UNION ALL
  SELECT 'макароны', 'Продукты' UNION ALL
  SELECT 'чай', 'Продукты' UNION ALL
  SELECT 'кофе', 'Продукты' UNION ALL
  SELECT 'вода', 'Продукты' UNION ALL
  SELECT 'сок', 'Продукты' UNION ALL
  SELECT 'конфета', 'Продукты' UNION ALL
  SELECT 'шоколад', 'Продукты' UNION ALL
  SELECT 'печенье', 'Продукты' UNION ALL
  SELECT 'овощи', 'Продукты' UNION ALL
  SELECT 'фрукты', 'Продукты' UNION ALL

  SELECT 'кофейня', 'Кафе и рестораны' UNION ALL
  SELECT 'кафе', 'Кафе и рестораны' UNION ALL
  SELECT 'ресторан', 'Кафе и рестораны' UNION ALL
  SELECT 'бар', 'Кафе и рестораны' UNION ALL
  SELECT 'пицца', 'Кафе и рестораны' UNION ALL
  SELECT 'суши', 'Кафе и рестораны' UNION ALL
  SELECT 'бургер', 'Кафе и рестораны' UNION ALL
  SELECT 'кофе то-го', 'Кафе и рестораны' UNION ALL
  SELECT 'шаурма', 'Кафе и рестораны' UNION ALL
  SELECT 'столовая', 'Кафе и рестораны' UNION ALL
  SELECT 'фастфуд', 'Кафе и рестораны' UNION ALL

  SELECT 'такси', 'Транспорт' UNION ALL
  SELECT 'метро', 'Транспорт' UNION ALL
  SELECT 'автобус', 'Транспорт' UNION ALL
  SELECT 'электричка', 'Транспорт' UNION ALL
  SELECT 'каршеринг', 'Транспорт' UNION ALL
  SELECT 'парковка', 'Транспорт' UNION ALL
  SELECT 'проезд', 'Транспорт' UNION ALL
  SELECT 'проездной', 'Транспорт' UNION ALL
  SELECT 'ласточка', 'Транспорт' UNION ALL
  SELECT 'поезд', 'Транспорт' UNION ALL
  SELECT 'билет', 'Транспорт' UNION ALL

  SELECT 'бензин', 'АЗС' UNION ALL
  SELECT 'дт', 'АЗС' UNION ALL
  SELECT 'дизель', 'АЗС' UNION ALL
  SELECT 'азс', 'АЗС' UNION ALL
  SELECT 'топливо', 'АЗС' UNION ALL
  SELECT 'заправка', 'АЗС' UNION ALL
  SELECT 'аи-92', 'АЗС' UNION ALL
  SELECT 'аи-95', 'АЗС' UNION ALL

  SELECT 'аптека', 'Здоровье и аптеки' UNION ALL
  SELECT 'лекарство', 'Здоровье и аптеки' UNION ALL
  SELECT 'таблетка', 'Здоровье и аптеки' UNION ALL
  SELECT 'витамин', 'Здоровье и аптеки' UNION ALL
  SELECT 'бинт', 'Здоровье и аптеки' UNION ALL
  SELECT 'маска', 'Здоровье и аптеки' UNION ALL
  SELECT 'тест', 'Здоровье и аптеки' UNION ALL
  SELECT 'градусник', 'Здоровье и аптеки' UNION ALL
  SELECT 'клиника', 'Здоровье и аптеки' UNION ALL
  SELECT 'стоматология', 'Здоровье и аптеки' UNION ALL
  SELECT 'анализы', 'Здоровье и аптеки' UNION ALL
  SELECT 'приём врача', 'Здоровье и аптеки' UNION ALL

  SELECT 'футболка', 'Одежда и обувь' UNION ALL
  SELECT 'брюки', 'Одежда и обувь' UNION ALL
  SELECT 'джинсы', 'Одежда и обувь' UNION ALL
  SELECT 'куртка', 'Одежда и обувь' UNION ALL
  SELECT 'платье', 'Одежда и обувь' UNION ALL
  SELECT 'обувь', 'Одежда и обувь' UNION ALL
  SELECT 'кроссовки', 'Одежда и обувь' UNION ALL
  SELECT 'ботинки', 'Одежда и обувь' UNION ALL
  SELECT 'носки', 'Одежда и обувь' UNION ALL
  SELECT 'бельё', 'Одежда и обувь' UNION ALL
  SELECT 'шапка', 'Одежда и обувь' UNION ALL
  SELECT 'перчатки', 'Одежда и обувь' UNION ALL

  SELECT 'порошок', 'Дом и быт' UNION ALL
  SELECT 'средство', 'Дом и быт' UNION ALL
  SELECT 'губка', 'Дом и быт' UNION ALL
  SELECT 'мешок', 'Дом и быт' UNION ALL
  SELECT 'туалетная бумага', 'Дом и быт' UNION ALL
  SELECT 'салфетка', 'Дом и быт' UNION ALL
  SELECT 'лампочка', 'Дом и быт' UNION ALL
  SELECT 'батарейка', 'Дом и быт' UNION ALL
  SELECT 'посуда', 'Дом и быт' UNION ALL
  SELECT 'полотенце', 'Дом и быт' UNION ALL
  SELECT 'моющее', 'Дом и быт' UNION ALL

  SELECT 'кабель', 'Электроника' UNION ALL
  SELECT 'зарядка', 'Электроника' UNION ALL
  SELECT 'наушники', 'Электроника' UNION ALL
  SELECT 'флешка', 'Электроника' UNION ALL
  SELECT 'аккумулятор', 'Электроника' UNION ALL
  SELECT 'чехол', 'Электроника' UNION ALL
  SELECT 'адаптер', 'Электроника' UNION ALL

  SELECT 'кино', 'Развлечения' UNION ALL
  SELECT 'билет в кино', 'Развлечения' UNION ALL
  SELECT 'театр', 'Развлечения' UNION ALL
  SELECT 'концерт', 'Развлечения' UNION ALL
  SELECT 'боулинг', 'Развлечения' UNION ALL
  SELECT 'каток', 'Развлечения' UNION ALL
  SELECT 'музей', 'Развлечения' UNION ALL
  SELECT 'парк развлечений', 'Развлечения' UNION ALL
  SELECT 'игра', 'Развлечения' UNION ALL
  SELECT 'подписка', 'Развлечения' UNION ALL

  SELECT 'мобильная связь', 'Связь и интернет' UNION ALL
  SELECT 'интернет', 'Связь и интернет' UNION ALL
  SELECT 'sim', 'Связь и интернет' UNION ALL
  SELECT 'пополнение баланса', 'Связь и интернет' UNION ALL
  SELECT 'тариф', 'Связь и интернет' UNION ALL

  SELECT 'жкх', 'Коммунальные услуги' UNION ALL
  SELECT 'электроэнергия', 'Коммунальные услуги' UNION ALL
  SELECT 'отопление', 'Коммунальные услуги' UNION ALL
  SELECT 'водоснабжение', 'Коммунальные услуги' UNION ALL
  SELECT 'газ', 'Коммунальные услуги' UNION ALL
  SELECT 'квартплата', 'Коммунальные услуги' UNION ALL

  SELECT 'шампунь', 'Красота и уход' UNION ALL
  SELECT 'крем', 'Красота и уход' UNION ALL
  SELECT 'парфюм', 'Красота и уход' UNION ALL
  SELECT 'косметика', 'Красота и уход' UNION ALL
  SELECT 'маникюр', 'Красота и уход' UNION ALL
  SELECT 'стрижка', 'Красота и уход' UNION ALL
  SELECT 'салон красоты', 'Красота и уход' UNION ALL
  SELECT 'бритва', 'Красота и уход' UNION ALL

  SELECT 'абонемент', 'Спорт' UNION ALL
  SELECT 'фитнес', 'Спорт' UNION ALL
  SELECT 'спортзал', 'Спорт' UNION ALL
  SELECT 'тренировка', 'Спорт' UNION ALL
  SELECT 'экипировка', 'Спорт' UNION ALL
  SELECT 'мяч', 'Спорт' UNION ALL
  SELECT 'гантель', 'Спорт' UNION ALL
  SELECT 'тренажёр', 'Спорт' UNION ALL
  SELECT 'велосипед', 'Спорт' UNION ALL
  SELECT 'самокат', 'Спорт' UNION ALL
  SELECT 'коньки', 'Спорт' UNION ALL
  SELECT 'лыжи', 'Спорт' UNION ALL

  SELECT 'памперс', 'Дети' UNION ALL
  SELECT 'подгузник', 'Дети' UNION ALL
  SELECT 'соска', 'Дети' UNION ALL
  SELECT 'коляска', 'Дети' UNION ALL
  SELECT 'детское питание', 'Дети' UNION ALL
  SELECT 'смесь', 'Дети' UNION ALL
  SELECT 'игрушка', 'Дети' UNION ALL
  SELECT 'конструктор', 'Дети' UNION ALL
  SELECT 'канцелярия школьная', 'Дети' UNION ALL
  SELECT 'пенал', 'Дети' UNION ALL
  SELECT 'рюкзак школьный', 'Дети' UNION ALL
  SELECT 'форма школьная', 'Дети' UNION ALL
  SELECT 'развивающие занятия', 'Дети' UNION ALL
  SELECT 'кружок', 'Дети' UNION ALL

  SELECT 'корм', 'Животные' UNION ALL
  SELECT 'наполнитель', 'Животные' UNION ALL
  SELECT 'ветеринар', 'Животные' UNION ALL
  SELECT 'поводок', 'Животные' UNION ALL
  SELECT 'миска', 'Животные' UNION ALL
  SELECT 'когтеточка', 'Животные' UNION ALL
  SELECT 'вакцинация', 'Животные' UNION ALL
  SELECT 'зоомагазин', 'Животные' UNION ALL

  SELECT 'курс', 'Образование' UNION ALL
  SELECT 'учебник', 'Образование' UNION ALL
  SELECT 'репетитор', 'Образование' UNION ALL
  SELECT 'тетрадь', 'Образование' UNION ALL
  SELECT 'обучение', 'Образование' UNION ALL
  SELECT 'вебинар', 'Образование' UNION ALL
  SELECT 'семинар', 'Образование' UNION ALL
  SELECT 'подписка на курс', 'Образование' UNION ALL
  SELECT 'языковая школа', 'Образование' UNION ALL

  SELECT 'шиномонтаж', 'Автомобиль (сервис)' UNION ALL
  SELECT 'автомойка', 'Автомобиль (сервис)' UNION ALL
  SELECT 'мойка', 'Автомобиль (сервис)' UNION ALL
  SELECT 'то', 'Автомобиль (сервис)' UNION ALL
  SELECT 'техосмотр', 'Автомобиль (сервис)' UNION ALL
  SELECT 'запчасть', 'Автомобиль (сервис)' UNION ALL
  SELECT 'масло моторное', 'Автомобиль (сервис)' UNION ALL
  SELECT 'автосервис', 'Автомобиль (сервис)' UNION ALL
  SELECT 'страховка осаго', 'Автомобиль (сервис)' UNION ALL
  SELECT 'каско', 'Автомобиль (сервис)' UNION ALL

  SELECT 'цветы', 'Подарки и цветы' UNION ALL
  SELECT 'букет', 'Подарки и цветы' UNION ALL
  SELECT 'подарок', 'Подарки и цветы' UNION ALL
  SELECT 'открытка', 'Подарки и цветы' UNION ALL
  SELECT 'подарочная карта', 'Подарки и цветы' UNION ALL
  SELECT 'сертификат', 'Подарки и цветы' UNION ALL

  SELECT 'авиабилет', 'Путешествия' UNION ALL
  SELECT 'отель', 'Путешествия' UNION ALL
  SELECT 'гостиница', 'Путешествия' UNION ALL
  SELECT 'хостел', 'Путешествия' UNION ALL
  SELECT 'бронирование', 'Путешествия' UNION ALL
  SELECT 'тур', 'Путешествия' UNION ALL
  SELECT 'виза', 'Путешествия' UNION ALL
  SELECT 'аренда авто', 'Путешествия' UNION ALL
  SELECT 'чемодан', 'Путешествия' UNION ALL

  SELECT 'пиво', 'Алкоголь и табак' UNION ALL
  SELECT 'вино', 'Алкоголь и табак' UNION ALL
  SELECT 'водка', 'Алкоголь и табак' UNION ALL
  SELECT 'виски', 'Алкоголь и табак' UNION ALL
  SELECT 'сигареты', 'Алкоголь и табак' UNION ALL
  SELECT 'вейп', 'Алкоголь и табак' UNION ALL
  SELECT 'табак', 'Алкоголь и табак' UNION ALL

  SELECT 'цемент', 'Стройматериалы' UNION ALL
  SELECT 'песок', 'Стройматериалы' UNION ALL
  SELECT 'щебень', 'Стройматериалы' UNION ALL
  SELECT 'гравий', 'Стройматериалы' UNION ALL
  SELECT 'кирпич', 'Стройматериалы' UNION ALL
  SELECT 'блок газобетонный', 'Стройматериалы' UNION ALL
  SELECT 'пеноблок', 'Стройматериалы' UNION ALL
  SELECT 'арматура', 'Стройматериалы' UNION ALL
  SELECT 'сетка армирующая', 'Стройматериалы' UNION ALL
  SELECT 'гипсокартон', 'Стройматериалы' UNION ALL
  SELECT 'профиль', 'Стройматериалы' UNION ALL
  SELECT 'утеплитель', 'Стройматериалы' UNION ALL
  SELECT 'пенопласт', 'Стройматериалы' UNION ALL
  SELECT 'минвата', 'Стройматериалы' UNION ALL
  SELECT 'гидроизоляция', 'Стройматериалы' UNION ALL
  SELECT 'пароизоляция', 'Стройматериалы' UNION ALL
  SELECT 'плёнка строительная', 'Стройматериалы' UNION ALL
  SELECT 'доска', 'Стройматериалы' UNION ALL
  SELECT 'брус', 'Стройматериалы' UNION ALL
  SELECT 'фанера', 'Стройматериалы' UNION ALL
  SELECT 'осб', 'Стройматериалы' UNION ALL
  SELECT 'вагонка', 'Стройматериалы' UNION ALL
  SELECT 'штукатурка', 'Стройматериалы' UNION ALL
  SELECT 'шпаклёвка', 'Стройматериалы' UNION ALL
  SELECT 'грунтовка', 'Стройматериалы' UNION ALL
  SELECT 'краска', 'Стройматериалы' UNION ALL
  SELECT 'эмаль', 'Стройматериалы' UNION ALL
  SELECT 'лак', 'Стройматериалы' UNION ALL
  SELECT 'обои', 'Стройматериалы' UNION ALL
  SELECT 'клей плиточный', 'Стройматериалы' UNION ALL
  SELECT 'клей монтажный', 'Стройматериалы' UNION ALL
  SELECT 'герметик', 'Стройматериалы' UNION ALL
  SELECT 'пена монтажная', 'Стройматериалы' UNION ALL
  SELECT 'плитка керамическая', 'Стройматериалы' UNION ALL
  SELECT 'плитка тротуарная', 'Стройматериалы' UNION ALL
  SELECT 'ламинат', 'Стройматериалы' UNION ALL
  SELECT 'линолеум', 'Стройматериалы' UNION ALL
  SELECT 'паркет', 'Стройматериалы' UNION ALL
  SELECT 'затирка', 'Стройматериалы' UNION ALL
  SELECT 'кровля', 'Стройматериалы' UNION ALL
  SELECT 'черепица', 'Стройматериалы' UNION ALL
  SELECT 'металлочерепица', 'Стройматериалы' UNION ALL
  SELECT 'профнастил', 'Стройматериалы' UNION ALL
  SELECT 'труба пвх', 'Стройматериалы' UNION ALL
  SELECT 'труба металлическая', 'Стройматериалы' UNION ALL
  SELECT 'кабель электрический', 'Стройматериалы' UNION ALL
  SELECT 'провод', 'Стройматериалы' UNION ALL
  SELECT 'розетка', 'Стройматериалы' UNION ALL
  SELECT 'выключатель', 'Стройматериалы' UNION ALL
  SELECT 'светильник', 'Стройматериалы' UNION ALL
  SELECT 'смеситель', 'Стройматериалы' UNION ALL
  SELECT 'унитаз', 'Стройматериалы' UNION ALL
  SELECT 'раковина', 'Стройматериалы' UNION ALL
  SELECT 'ванна', 'Стройматериалы' UNION ALL
  SELECT 'батарея отопления', 'Стройматериалы' UNION ALL
  SELECT 'радиатор', 'Стройматериалы' UNION ALL
  SELECT 'теплоизоляция', 'Стройматериалы' UNION ALL

  SELECT 'дрель', 'Инструменты' UNION ALL
  SELECT 'перфоратор', 'Инструменты' UNION ALL
  SELECT 'болгарка', 'Инструменты' UNION ALL
  SELECT 'шуруповёрт', 'Инструменты' UNION ALL
  SELECT 'пила', 'Инструменты' UNION ALL
  SELECT 'лобзик', 'Инструменты' UNION ALL
  SELECT 'ножовка', 'Инструменты' UNION ALL
  SELECT 'молоток', 'Инструменты' UNION ALL
  SELECT 'отвёртка', 'Инструменты' UNION ALL
  SELECT 'гаечный ключ', 'Инструменты' UNION ALL
  SELECT 'уровень', 'Инструменты' UNION ALL
  SELECT 'рулетка', 'Инструменты' UNION ALL
  SELECT 'шпатель', 'Инструменты' UNION ALL
  SELECT 'кисть малярная', 'Инструменты' UNION ALL
  SELECT 'валик', 'Инструменты' UNION ALL
  SELECT 'стремянка', 'Инструменты' UNION ALL
  SELECT 'леса строительные', 'Инструменты' UNION ALL
  SELECT 'сварочный аппарат', 'Инструменты' UNION ALL
  SELECT 'компрессор', 'Инструменты' UNION ALL
  SELECT 'генератор', 'Инструменты' UNION ALL
  SELECT 'бетономешалка', 'Инструменты' UNION ALL
  SELECT 'диск отрезной', 'Инструменты' UNION ALL
  SELECT 'сверло', 'Инструменты' UNION ALL
  SELECT 'саморез', 'Инструменты' UNION ALL
  SELECT 'гвоздь', 'Инструменты' UNION ALL
  SELECT 'болт', 'Инструменты' UNION ALL
  SELECT 'дюбель', 'Инструменты' UNION ALL
  SELECT 'перчатки рабочие', 'Инструменты' UNION ALL
  SELECT 'спецодежда', 'Инструменты' UNION ALL
  SELECT 'каска', 'Инструменты' UNION ALL

  SELECT 'диван', 'Мебель и интерьер' UNION ALL
  SELECT 'стол', 'Мебель и интерьер' UNION ALL
  SELECT 'стул', 'Мебель и интерьер' UNION ALL
  SELECT 'шкаф', 'Мебель и интерьер' UNION ALL
  SELECT 'кровать', 'Мебель и интерьер' UNION ALL
  SELECT 'матрас', 'Мебель и интерьер' UNION ALL
  SELECT 'комод', 'Мебель и интерьер' UNION ALL
  SELECT 'зеркало', 'Мебель и интерьер' UNION ALL
  SELECT 'штора', 'Мебель и интерьер' UNION ALL
  SELECT 'ковёр', 'Мебель и интерьер' UNION ALL
  SELECT 'декор', 'Мебель и интерьер' UNION ALL
  SELECT 'ваза', 'Мебель и интерьер' UNION ALL
  SELECT 'рамка для фото', 'Мебель и интерьер' UNION ALL

  SELECT 'ручка', 'Канцтовары и хобби' UNION ALL
  SELECT 'карандаш', 'Канцтовары и хобби' UNION ALL
  SELECT 'блокнот', 'Канцтовары и хобби' UNION ALL
  SELECT 'бумага', 'Канцтовары и хобби' UNION ALL
  SELECT 'принтер', 'Канцтовары и хобби' UNION ALL
  SELECT 'картридж', 'Канцтовары и хобби' UNION ALL
  SELECT 'пряжа', 'Канцтовары и хобби' UNION ALL
  SELECT 'ткань', 'Канцтовары и хобби' UNION ALL
  SELECT 'краски для рисования', 'Канцтовары и хобби' UNION ALL
  SELECT 'кисть', 'Канцтовары и хобби' UNION ALL
  SELECT 'рамка', 'Канцтовары и хобби' UNION ALL

  SELECT 'комиссия', 'Банковские услуги' UNION ALL
  SELECT 'обслуживание счёта', 'Банковские услуги' UNION ALL
  SELECT 'перевод', 'Банковские услуги' UNION ALL
  SELECT 'снятие наличных', 'Банковские услуги'
) k
JOIN `categories` c ON c.name = k.category;
