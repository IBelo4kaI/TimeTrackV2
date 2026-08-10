# TimeTrack API Endpoints Documentation

## Базовый URL

```
http://localhost:8080/v1
```

Все запросы требуют авторизации через заголовок `Authorization: Bearer <token>`, если не указано иное.

## 1. Календарь (Calendar)

### 1.1 Получить дни календаря для пользователя

**GET** `/calendar/:userId/:year/:month`

Получает информацию о днях календаря для указанного пользователя, месяца и года.

**Параметры пути:**

- `userId` (string) - ID пользователя
- `year` (int) - Год (например, 2026)
- `month` (int) - Месяц (1-12)

**Разрешение:** `time:calendar:read` или `time:calendar.all:read`

**Пример запроса:**

```bash
GET /v1/calendar/123e4567-e89b-12d3-a456-426614174000/2026/2
```

**Пример ответа:**

```json
{
   "success": true,
   "data": [
      {
         "id": "550e8400-e29b-41d4-a716-446655440000",
         "date": "2026-02-01T00:00:00Z",
         "day_type_id": "2cb03962-d661-11f0-b7e5-b05cda34b6c7",
         "day_type_name": "Рабочий день",
         "is_work_day": true,
         "color_code": "#39c684",
         "hours_worked": "8",
         "notes": null
      }
      // ... другие дни месяца
   ]
}
```

## 2. Типы дней (Day Types)

### 2.1 Получить все типы дней

**GET** `/daytypes`

Получает список всех доступных типов дней в системе.

**Разрешение:** Требуется только авторизация

**Пример запроса:**

```bash
GET /v1/daytypes
```

**Пример ответа:**

```json
{
   "success": true,
   "data": [
      {
         "id": "2cb03962-d661-11f0-b7e5-b05cda34b6c7",
         "name": "Рабочий день",
         "system_name": "work",
         "is_work_day": true,
         "affects_vacation": true,
         "is_user_select": true,
         "color_code": "#39c684",
         "created_at": "2026-02-05T11:49:13Z",
         "updated_at": "2026-02-07T17:56:17Z"
      },
      {
         "id": "3f631366-d661-11f0-b7e5-b05cda34b6c7",
         "name": "Отпуск",
         "system_name": "vacation",
         "is_work_day": false,
         "affects_vacation": true,
         "is_user_select": false,
         "color_code": "#e8a530",
         "created_at": "2026-02-05T11:49:13Z",
         "updated_at": "2026-02-08T16:52:06Z"
      }
      // ... другие типы дней
   ]
}
```

## 3. Учет времени (User Time Entries)

### 3.1 Создать записи учета времени

**POST** `/usertimeentries/create`

Создает одну или несколько записей учета рабочего времени.

**Разрешение:** `time:calendar:create` или `time:calendar.all:create`

**Тело запроса:**

```json
{
   "userId": "123e4567-e89b-12d3-a456-426614174000",
   "entities": [
      {
         "dayTypeId": "2cb03962-d661-11f0-b7e5-b05cda34b6c7",
         "hoursWorked": "8",
         "entryDate": "2026-02-15T00:00:00Z"
      },
      {
         "dayTypeId": "48f5ccec-d661-11f0-b7e5-b05cda34b6c7",
         "hoursWorked": "0",
         "entryDate": "2026-02-16T00:00:00Z"
      }
   ]
}
```

**Пример ответа:**

```json
{
   "success": true,
   "data": {
      "created_count": 2,
      "message": "Записи успешно созданы"
   }
}
```

### 3.2 Обновить записи учета времени

**POST** `/usertimeentries/update`

Обновляет существующие записи учета рабочего времени.

**Разрешение:** `time:calendar:edit` или `time:calendar.all:edit`

**Тело запроса:**

```json
{
   "userId": "123e4567-e89b-12d3-a456-426614174000",
   "entities": [
      {
         "id": "550e8400-e29b-41d4-a716-446655440000",
         "dayTypeId": "5543f231-d661-11f0-b7e5-b05cda34b6c7",
         "hoursWorked": "4",
         "entryDate": "2026-02-15T00:00:00Z"
      }
   ]
}
```

### 3.3 Удалить записи учета времени

**POST** `/usertimeentries/delete`

Удаляет записи учета рабочего времени.

**Разрешение:** `time:calendar:delete` или `time:calendar.all:delete`

**Тело запроса:**

```json
{
   "userId": "123e4567-e89b-12d3-a456-426614174000",
   "ids": [
      "550e8400-e29b-41d4-a716-446655440000",
      "550e8400-e29b-41d4-a716-446655440001"
   ]
}
```

### 3.4 Получить статистику отчетов

**GET** `/usertimeentries/statistics/:userId/:year/:month/:gender`

Получает статистику отчетов для пользователя за указанный месяц и год.

**Параметры пути:**

- `userId` (string) - ID пользователя
- `year` (int) - Год
- `month` (int) - Месяц
- `gender` (int) - Пол (1 - мужской, 2 - женский)

**Разрешение:** `time:calendar:read` или `time:calendar.all:read`

**Пример запроса:**

```bash
GET /v1/usertimeentries/statistics/123e4567-e89b-12d3-a456-426614174000/2026/2/1
```

**Пример ответа:**

```json
{
   "success": true,
   "data": {
      "userId": "123e4567-e89b-12d3-a456-426614174000",
      "year": 2026,
      "month": 2,
      "totalWorkHours": 152,
      "actualWorkHours": 140,
      "vacationDays": 5,
      "sickDays": 2,
      "timeOffDays": 1,
      "workEfficiency": 92.1
   }
}
```

## 4. Отпуска (Vacations)

### 4.1 Рассчитать дни отпуска

**GET** `/vacation/calculate`

Рассчитывает доступные дни отпуска на основе различных параметров.

**Параметры запроса:**

- `userId` (string) - ID пользователя
- `startDate` (string) - Дата начала расчета
- `endDate` (string) - Дата окончания расчета
- `includeHolidays` (bool) - Включать праздничные дни

**Разрешение:** `time:vacation:read` или `time:vacation.all:read`

**Пример запроса:**

```bash
GET /v1/vacation/calculate?userId=123e4567-e89b-12d3-a456-426614174000&startDate=2026-06-01&endDate=2026-06-15&includeHolidays=true
```

### 4.2 Получить статистику отпусков

**GET** `/vacation/stats/:userId/:year`

Получает статистику отпусков пользователя за указанный год.

**Параметры пути:**

- `userId` (string) - ID пользователя
- `year` (int) - Год

**Разрешение:** `time:vacation:read` или `time:vacation.all:read`

### 4.3 Получить все отпуска за год

**GET** `/vacation/all/:year`

Получает все отпуска всех пользователей за указанный год.

**Параметры пути:**

- `year` (int) - Год

**Разрешение:** `time:vacation.all:read`

### 4.4 Получить отпуска пользователя за год

**GET** `/vacation/:userId/:year`

Получает отпуска конкретного пользователя за указанный год.

**Параметры пути:**

- `userId` (string) - ID пользователя
- `year` (int) - Год

**Разрешение:** `time:vacation:read` или `time:vacation.all:read`

### 4.4a Получить заявку по ID

**GET** `/vacation/:id`

Карточка отдельной заявки на отпуск (для страницы заявления). Возвращает
то же самое, что и элементы списков `4.3`/`4.4` — включая данные типа
отпуска (`vacationTypeId`, `vacationTypeName`, `vacationTypeColor`,
`vacationTypeAffectsBalance`).

**Параметры пути:**

- `id` (string) - ID отпуска

**Разрешение:** `time:vacation:read` или `time:vacation.all:read`

**Ответ:** `404`, если заявка не найдена.

### 4.5 Создать отпуск

**POST** `/vacation/create`

Создает новую запись об отпуске.

**Разрешение:** `time:vacation:create` или `time:vacation.all:create`

**Тело запроса:**

```json
{
   "userId": "123e4567-e89b-12d3-a456-426614174000",
   "startDate": "2026-06-01T00:00:00Z",
   "endDate": "2026-06-15T00:00:00Z",
   "description": "Ежегодный оплачиваемый отпуск",
   "status": "pending",
   "vacationTypeId": "cd73f847-948d-11f1-b8fc-0a5eea63c1c5"
}
```

`vacationTypeId` — необязателен; если не передан, подставляется тип отпуска
с `system_name = "paid"` (см. [раздел 7](#7-типы-отпусков-vacation-types)). Список
отпусков (`4.3`, `4.4`) и получение по ID возвращают вместе с записью данные
типа: `vacationTypeId`, `vacationTypeName`, `vacationTypeColor`,
`vacationTypeAffectsBalance`. Только отпуска с типом, у которого
`affectsBalance = true`, учитываются в статистике (`4.2`) как использованные
дни отпуска.

### 4.6 Одобрить отпуск

**PUT** `/vacation/:id/approve`

Одобряет запрос на отпуск.

**Параметры пути:**

- `id` (string) - ID отпуска

**Разрешение:** `time:vacation:edit` или `time:vacation.all:edit`

### 4.7 Обновить статус отпуска

**PUT** `/vacation/:id/status`

Обновляет статус отпуска.

**Параметры пути:**

- `id` (string) - ID отпуска

**Тело запроса:**

```json
{
   "status": "approved",
   "reason": "Одобрено руководителем"
}
```

**Разрешение:** `time:vacation:edit` или `time:vacation.all:edit`

### 4.8 Загрузить файл для отпуска

**POST** `/vacation/:id/file`

Загружает файл и привязывает его к отпуску через `file_entity_refs` (например, заявление).

**Параметры пути:**

- `id` (string) - ID отпуска

**Формат запроса:** `multipart/form-data`

**Поля:**

- `file` - Файл для загрузки

**Ответ:** `201 Created`

```json
{
   "id": "550e8400-e29b-41d4-a716-446655440000",
   "originalName": "otpusk.pdf",
   "mimeType": "application/pdf",
   "fileType": "document",
   "sizeBytes": 12345
}
```

**Разрешение:** `time:vacation:edit` или `time:vacation.all:edit`

### 4.9 Получить файлы отпуска

Список файлов отпуска:
**GET** `/v1/files/entity/vacation/:id`

Открытие файла по ID (из ответа загрузки или списка):
**GET** `/v1/files/open/:id`

**Разрешение:** `time:files:read` или `time:files.all:read`

### 4.10 Удалить файл отпуска

**DELETE** `/v1/files/:id`

Удаляет файл по ID (из ответа загрузки или списка).

**Разрешение:** `time:files:delete` или `time:files.all:delete`

### 4.11 Удалить отпуск

**DELETE** `/vacation/:id`

Удаляет запись об отпуске.

**Параметры пути:**

- `id` (string) - ID отпуска

**Разрешение:** `time:vacation:delete` или `time:vacation.all:delete`

### 4.12 Изменить тип отпуска

**PUT** `/vacation/:id/type`

Меняет тип у существующей заявки на отпуск.

**Параметры пути:**

- `id` (string) - ID отпуска

**Тело запроса:**

```json
{
   "vacationTypeId": "cd740e87-948d-11f1-b8fc-0a5eea63c1c5"
}
```

**Разрешение:** `time:vacation:edit` или `time:vacation.all:edit`

## 5. Системные настройки (System Settings)

### 5.1 Получить системную настройку по ключу

**GET** `/system-settings/:key`

Получает значение системной настройки по ключу.

**Параметры пути:**

- `key` (string) - Ключ настройки

**Разрешение:** `time:system_settings:read`

**Пример запроса:**

```bash
GET /v1/system-settings/vacation_days_per_year
```

**Пример ответа:**

```json
{
   "success": true,
   "data": {
      "key": "vacation_days_per_year",
      "value": "28",
      "description": "Количество дней отпуска в год",
      "updated_at": "2026-02-10T10:30:00Z"
   }
}
```

### 5.2 Обновить значение системной настройки

**POST** `/system-settings/value`

Обновляет значение системной настройки.

**Разрешение:** `time:system_settings:edit`

**Тело запроса:**

```json
{
   "key": "vacation_days_per_year",
   "value": "30",
   "description": "Обновленное количество дней отпуска"
}
```

## 6. Рабочие нормативы (Work Standards)

### 6.1 Создать рабочий норматив

**POST** `/work-standards`

Создает новый рабочий норматив.

**Разрешение:** `time:work_standards:create`

**Тело запроса:**

```json
{
   "userId": null,
   "month": 3,
   "year": 2026,
   "standardHours": 168,
   "standardDays": 21,
   "gender": 1
}
```

### 6.2 Получить рабочие нормативы по месяцу

**GET** `/work-standards/month/:month/year/:year`

Получает рабочие нормативы для указанного месяца и года.

**Параметры пути:**

- `month` (int) - Месяц
- `year` (int) - Год

**Разрешение:** `time:work_standards:read`

### 6.3 Получить рабочие нормативы по году

**GET** `/work-standards/year/:year`

Получает все рабочие нормативы за указанный год.

**Параметры пути:**

- `year` (int) - Год

**Разрешение:** `time:work_standards:read`

### 6.4 Получить сгруппированные рабочие нормативы по году

**GET** `/work-standards/year/:year/grouped`

Получает рабочие нормативы за указанный год, сгруппированные по полу.

**Параметры пути:**

- `year` (int) - Год

**Разрешение:** `time:work_standards:read`

**Пример ответа:**

```json
{
   "success": true,
   "data": {
      "year": 2026,
      "male": [
         {
            "month": 1,
            "standardHours": 120,
            "standardDays": 15
         }
         // ... другие месяцы для мужчин
      ],
      "female": [
         {
            "month": 1,
            "standardHours": 114,
            "standardDays": 15
         }
         // ... другие месяцы для женщин
      ]
   }
}
```

### 6.5 Обновить рабочий норматив

**PUT** `/work-standards/:id`

Обновляет существующий рабочий норматив.

**Параметры пути:**

- `id` (string) - ID норматива

**Разрешение:** `time:work_standards:edit`

**Тело запроса:**

```json
{
   "standardHours": 170,
   "standardDays": 22,
   "gender": 1
}
```

### 6.6 Удалить рабочий норматив

**DELETE** `/work-standards/:id`

Удаляет рабочий норматив.

**Параметры пути:**

- `id` (string) - ID норматива

**Разрешение:** `time:work_standards:delete`

## 7. Типы отпусков (Vacation Types)

Справочник типов отпусков (аналог `day_types`). У каждого типа есть флаг
`affectsBalance` — списывается ли отпуск этого типа из годовой нормы дней
отпуска (используется при расчёте статистики, см. `4.2`). Базовый набор,
заводимый миграцией: `paid` (Основной оплачиваемый, влияет на баланс),
`unpaid` (За свой счёт, не влияет), `study` (Учебный, не влияет).

### 7.1 Получить все типы отпусков

**GET** `/vacation-types`

Возвращает все типы отпусков, включая неактивные (`isActive = false`) —
для админки.

**Разрешение:** Требуется только авторизация

**Пример ответа:**

```json
[
   {
      "id": "cd73f847-948d-11f1-b8fc-0a5eea63c1c5",
      "name": "Основной оплачиваемый",
      "systemName": "paid",
      "colorCode": "#2f80ed",
      "affectsBalance": true,
      "isActive": true,
      "sortOrder": 1,
      "createdAt": "2026-08-10T07:33:38Z",
      "updatedAt": "2026-08-10T07:33:38Z"
   }
]
```

### 7.2 Получить активные типы отпусков

**GET** `/vacation-types/active`

То же самое, но только `isActive = true` — используется для выбора типа
при подаче заявки на отпуск.

**Разрешение:** Требуется только авторизация

### 7.3 Создать тип отпуска

**POST** `/vacation-types`

**Разрешение:** `time:vacation_types:create` (только админ)

**Тело запроса:**

```json
{
   "name": "Дополнительный",
   "systemName": "additional",
   "colorCode": "#9b51e0",
   "affectsBalance": true,
   "isActive": true,
   "sortOrder": 4
}
```

### 7.4 Обновить тип отпуска

**PUT** `/vacation-types/:id`

**Параметры пути:**

- `id` (string) - ID типа отпуска

**Разрешение:** `time:vacation_types:edit` (только админ)

**Тело запроса:** такое же, как при создании (`7.3`).

### 7.5 Удалить тип отпуска

**DELETE** `/vacation-types/:id`

Удаляет тип отпуска. Возвращает `409 Conflict`, если тип уже используется
в существующих заявках на отпуск.

**Параметры пути:**

- `id` (string) - ID типа отпуска

**Разрешение:** `time:vacation_types:delete` (только админ)

## 8. Категории файлов (File Categories)

Дерево категорий для организации файлов (аналог папок на диске). У файла
может быть не более одной категории (`files.category_id`); категория может
иметь родителя (`parentId`), формируя произвольную вложенность.

**Системные категории.** Часть категорий задаётся кодом, а не админом —
у них проставлен `systemName` (миграция `010_file_category_system_name.sql`).
Такие категории нельзя удалить через API (`403`), и бэкенд сам раскладывает
по ним файлы при загрузке в зависимости от `entity_type`
(`internal/service/file.go`, карта `entityCategorySystemName`):

- `vacation` → категория `application` («Заявления», подкатегория внутри
  «Отпуска», `systemName = vacation`).
- `sick_leave` → категория `medical` («Больничные», категория верхнего
  уровня).

Если явный `category_id` при загрузке не передан и `entity_type` совпал с
одним из значений выше — файл автоматически попадёт в соответствующую
категорию. Уже загруженные до этой миграции файлы отпусков/больничных были
разложены по категориям задним числом (бэкфилл в той же миграции).

### 8.1 Получить дерево категорий

**GET** `/file-categories`

Возвращает категории в виде дерева — каждый узел содержит `children` с
вложенными подкатегориями.

**Разрешение:** Требуется только авторизация

**Пример ответа:**

```json
[
   {
      "id": "11111111-1111-1111-1111-111111111111",
      "name": "Отпуска",
      "systemName": "vacation",
      "parentId": null,
      "colorCode": "#2f80ed",
      "sortOrder": 1,
      "createdAt": "2026-08-10T07:36:12Z",
      "updatedAt": "2026-08-10T07:36:12Z",
      "children": [
         {
            "id": "22222222-2222-2222-2222-222222222222",
            "name": "Заявления",
            "systemName": "application",
            "parentId": "11111111-1111-1111-1111-111111111111",
            "colorCode": "#2f80ed",
            "sortOrder": 1,
            "createdAt": "2026-08-10T07:36:12Z",
            "updatedAt": "2026-08-10T07:36:12Z",
            "children": []
         }
      ]
   },
   {
      "id": "33333333-3333-3333-3333-333333333333",
      "name": "Больничные",
      "systemName": "medical",
      "parentId": null,
      "colorCode": "#eb5757",
      "sortOrder": 2,
      "createdAt": "2026-08-10T07:36:12Z",
      "updatedAt": "2026-08-10T07:36:12Z",
      "children": []
   }
]
```

`systemName` присутствует только у системных категорий (см. врезку выше) —
у обычных, созданных руками, это поле `null`.

### 8.2 Создать категорию

**POST** `/file-categories`

**Разрешение:** `time:file_categories:create` (только админ)

**Тело запроса:**

```json
{
   "name": "Трудовые",
   "parentId": "11111111-1111-1111-1111-111111111111",
   "colorCode": "#eb5757",
   "sortOrder": 1
}
```

`parentId` — необязателен (`null`/не передан для категории верхнего уровня).
Название должно быть уникальным среди дочерних категорий одного родителя.

### 8.3 Обновить категорию

**PUT** `/file-categories/:id`

Позволяет переименовать категорию, поменять цвет/порядок или переместить
её к другому родителю. Сервер проверяет, что новый родитель не является
самой категорией или её потомком (защита от цикла).

**Параметры пути:**

- `id` (string) - ID категории

**Разрешение:** `time:file_categories:edit` (только админ)

**Тело запроса:** такое же, как при создании (`8.2`).

### 8.4 Удалить категорию

**DELETE** `/file-categories/:id`

Удаляет категорию. Дочерние категории удаляются каскадом; файлы, лежавшие
в удалённой категории (или её потомках), становятся без категории
(`categoryId: null`), сами файлы не удаляются.

Системные категории (`systemName` не `null`) удалить нельзя — ответ `403`.

**Параметры пути:**

- `id` (string) - ID категории

**Разрешение:** `time:file_categories:delete` (только админ)

### 8.5 Файлы: привязка к категории

При загрузке файла через общий эндпоинт **POST** `/v1/files/upload`
(`multipart/form-data`) можно передать необязательное поле `category_id`.

Дополнительные эндпоинты для работы с категориями у уже загруженных файлов:

- **GET** `/v1/files/category/:categoryId` — список файлов в категории.
  Разрешение: `time:files:read` или `time:files.all:read`.
- **PUT** `/v1/files/:id/category` — переместить файл в другую категорию
  (или убрать из категории, если `categoryId` — пустая строка). Тело:
  `{ "categoryId": "..." }`. Разрешение: `time:files:edit` или
  `time:files.all:edit`.

### 8.6 Фильтр по году во всех списках файлов

Все три GET-эндпоинта, возвращающие список файлов, принимают необязательный
query-параметр `year` — фильтрует по году загрузки файла (`created_at`).
Без параметра возвращаются файлы за все годы, как и раньше.

- **GET** `/v1/files/entity/:entityType/:entityId?year=2026`
- **GET** `/v1/files/entity/:entityType?year=2026`
- **GET** `/v1/files/category/:categoryId?year=2026`

Нечисловое значение `year` — `400 Bad Request`.

## Коды ответов

- `200` - Успешный запрос
- `400` - Неверный запрос (валидация ошибок)
- `401` - Не авторизован
- `403` - Доступ запрещен (нет необходимых разрешений)
- `404` - Ресурс не найден
- `500` - Внутренняя ошибка сервера

## Формат ответов

Все ответы API возвращаются в следующем формате:

```json
{
  "success": true|false,
  "data": {...},
  "error": "Сообщение об ошибке (если success=false)"
}
```

## Обработка ошибок

При возникновении ошибки сервер возвращает соответствующий HTTP статус и сообщение об ошибке в теле ответа:

```json
{
   "success": false,
   "error": "Подробное сообщение об ошибке"
}
```

## Пагинация

В настоящее время API не поддерживает пагинацию для большинства endpoints. Все данные возвращаются целиком.

## Версионирование

API использует версионирование через префикс пути (`/v1`). При внесении критических изменений будет создана новая версия API.

---

_Документация обновлена: 2026_
