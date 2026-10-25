# Проектирование базы данных и хранилища данных
## Информационная система «Адресная книга» (GoContact Studio)

---

## 1. Архитектура хранения данных

Система использует **гибридную трехуровневую модель хранения данных**, что позволяет продемонстрировать все аспекты работы с данными в языке Go (недели 5, 7, 10, 12, 14):

1. **Реляционная СУБД (SQLite 3)** — основное персистентное хранилище (`addressbook.db`). Обеспечивает транзакционность (ACID), целостность внешних ключей, фильтрацию и сортировку на уровне SQL.
2. **Оперативный In-Memory кэш и индекс (`map` + `slice` + `sync.RWMutex`)** — хранит часто запрашиваемые контакты и быстрый индекс по ID и номеру телефона в оперативной памяти для мгновенного отклика GUI.
3. **Файловое хранилище (`JSON`, `CSV`, `vCard .vcf`, `config.json`)** — отвечает за конфигурацию приложения, импорт/экспорт данных и автоматические резервные копии (бэкапы).

---

## 2. Схема реляционной базы данных (SQLite)

### 2.1. ER-диаграмма (Сущность — Связь)

```
+-----------------------+          +----------------------------------+
|        groups         |          |             contacts             |
+-----------------------+          +----------------------------------+
| PK id (INTEGER)       | 1      * | PK id (INTEGER)                  |
|    name (TEXT UNIQUE) |<---------| FK group_id (INTEGER NULL)       |
|    color_hex (TEXT)   |          |    first_name (TEXT NOT NULL)    |
|    description (TEXT) |          |    last_name (TEXT NOT NULL)     |
|    created_at (TEXT)  |          |    middle_name (TEXT)            |
+-----------------------+          |    phone (TEXT NOT NULL UNIQUE)  |
                                   |    work_phone (TEXT)             |
                                   |    phone_type (TEXT)             |
                                   |    email (TEXT)                  |
                                   |    company (TEXT)                |
                                   |    job_title (TEXT)              |
                                   |    street (TEXT)                 |
                                   |    city (TEXT)                   |
                                   |    country (TEXT)                |
                                   |    postal_code (TEXT)            |
                                   |    birthday (TEXT)               |
                                   |    tags_json (TEXT)              |
                                   |    notes (TEXT)                  |
                                   |    is_favorite (INTEGER)         |
                                   |    avatar_url (TEXT)             |
                                   |    created_at (DATETIME)         |
                                   |    updated_at (DATETIME)         |
                                   +----------------------------------+
```

### 2.2. DDL-скрипт создания таблиц и индексов

```sql
-- Включение поддержки внешних ключей и режима WAL
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;

-- Таблица групп (категорий) контактов
CREATE TABLE IF NOT EXISTS groups (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    color_hex   TEXT NOT NULL DEFAULT '#3B82F6',
    description TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Предзаполнение базовых групп
INSERT OR IGNORE INTO groups (id, name, color_hex, description) VALUES
    (1, 'Семья',   '#EF4444', 'Близкие родственники и члены семьи'),
    (2, 'Работа',  '#3B82F6', 'Коллеги, партнеры и клиенты'),
    (3, 'Друзья',  '#10B981', 'Друзья и приятели'),
    (4, 'VIP',     '#F59E0B', 'Важные контакты'),
    (5, 'Прочее',  '#6B7280', 'Общие контакты');

-- Основная таблица контактов
CREATE TABLE IF NOT EXISTS contacts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id    INTEGER REFERENCES groups(id) ON DELETE SET NULL,
    first_name  TEXT NOT NULL,
    last_name   TEXT NOT NULL,
    middle_name TEXT NOT NULL DEFAULT '',
    phone       TEXT NOT NULL UNIQUE,
    work_phone  TEXT NOT NULL DEFAULT '',
    phone_type  TEXT NOT NULL DEFAULT 'mobile',
    email       TEXT NOT NULL DEFAULT '',
    company     TEXT NOT NULL DEFAULT '',
    job_title   TEXT NOT NULL DEFAULT '',
    street      TEXT NOT NULL DEFAULT '',
    city        TEXT NOT NULL DEFAULT '',
    country     TEXT NOT NULL DEFAULT '',
    postal_code TEXT NOT NULL DEFAULT '',
    birthday    TEXT NOT NULL DEFAULT '',
    tags_json   TEXT NOT NULL DEFAULT '[]',
    notes       TEXT NOT NULL DEFAULT '',
    is_favorite INTEGER NOT NULL DEFAULT 0,
    avatar_url  TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для ускорения поиска и фильтрации
CREATE INDEX IF NOT EXISTS idx_contacts_name ON contacts(last_name, first_name);
CREATE INDEX IF NOT EXISTS idx_contacts_phone ON contacts(phone);
CREATE INDEX IF NOT EXISTS idx_contacts_email ON contacts(email);
CREATE INDEX IF NOT EXISTS idx_contacts_group ON contacts(group_id);
CREATE INDEX IF NOT EXISTS idx_contacts_favorite ON contacts(is_favorite);
```

---

## 3. Примеры SQL-запросов (Неделя 12)

В проекте используются все основные виды SQL-запросов:

1. **INSERT (Добавление контакта):**
   ```sql
   INSERT INTO contacts (
       group_id, first_name, last_name, middle_name, phone, work_phone,
       phone_type, email, company, job_title, street, city, country,
       postal_code, birthday, tags_json, notes, is_favorite, avatar_url,
       created_at, updated_at
   ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
   ```
2. **SELECT + JOIN + Динамическая фильтрация (Поиск и фильтрация):**
   ```sql
   SELECT c.*, COALESCE(g.name, 'Без группы') AS group_name
   FROM contacts c
   LEFT JOIN groups g ON c.group_id = g.id
   WHERE (
       ? = '' OR
       c.first_name LIKE '%' || ? || '%' OR
       c.last_name  LIKE '%' || ? || '%' OR
       c.phone      LIKE '%' || ? || '%' OR
       c.email      LIKE '%' || ? || '%' OR
       c.company    LIKE '%' || ? || '%' OR
       c.city       LIKE '%' || ? || '%'
   )
   AND (? = 0 OR c.group_id = ?)
   AND (? = 0 OR c.is_favorite = 1)
   ORDER BY c.is_favorite DESC, c.last_name ASC, c.first_name ASC;
   ```
3. **UPDATE (Обновление контакта):**
   ```sql
   UPDATE contacts
   SET group_id = ?, first_name = ?, last_name = ?, phone = ?, email = ?,
       company = ?, city = ?, is_favorite = ?, updated_at = CURRENT_TIMESTAMP
   WHERE id = ?;
   ```
4. **DELETE (Удаление контакта):**
   ```sql
   DELETE FROM contacts WHERE id = ?;
   ```
5. **Агрегатная аналитика (`GROUP BY`, `COUNT`):**
   ```sql
   SELECT COALESCE(g.name, 'Без группы') AS group_name, COUNT(c.id) AS total
   FROM groups g
   LEFT JOIN contacts c ON c.group_id = g.id
   GROUP BY g.id, g.name;
   ```

---

## 4. Спецификация форматов файлов (Неделя 7 и 14)

### 4.1. Формат экспорта/бэкапа JSON (`contacts_export.json`)
Демонстрирует работу с пакетом `encoding/json`, тегами структур и вложенной сериализацией:
```json
{
  "version": "1.0",
  "exported_at": "2026-10-22T16:30:00Z",
  "total_count": 1,
  "contacts": [
    {
      "id": 1,
      "group_id": 2,
      "group_name": "Работа",
      "first_name": "Алексей",
      "last_name": "Смирнов",
      "middle_name": "Игоревич",
      "phone": "+7 (701) 555-01-99",
      "phone_type": "mobile",
      "email": "a.smirnov@techcorp.kz",
      "company": "TechCorp",
      "job_title": "Lead Go Developer",
      "address": {
        "street": "пр. Достык, 120",
        "city": "Алматы",
        "country": "Казахстан",
        "postal_code": "050000"
      },
      "tags": ["golang", "backend", "mentor"],
      "is_favorite": true,
      "notes": "Познакомились на Go-митапе",
      "created_at": "2026-10-22T12:00:00Z",
      "updated_at": "2026-10-22T12:00:00Z"
    }
  ]
}
```

### 4.2. Формат табличного обмена CSV (`contacts.csv`)
Демонстрирует работу с `encoding/csv` (построчное чтение и запись с заголовком):
```csv
first_name,last_name,phone,email,company,job_title,city,country,group_name,is_favorite,notes
Алексей,Смирнов,+77015550199,a.smirnov@techcorp.kz,TechCorp,Lead Go Developer,Алматы,Казахстан,Работа,true,Познакомились на Go-митапе
Мария,Иванова,+77071234567,m.ivanova@design.io,DesignStudio,UI/UX Designer,Астана,Казахстан,Друзья,false,Дизайнер интерфейсов
```

### 4.3. Конфигурационный файл (`configs/config.json` — Неделя 14)
```json
{
  "app_name": "GoContact Studio — Адресная книга",
  "version": "1.0.0",
  "database_path": "data/addressbook.db",
  "backup_dir": "data/backups",
  "auto_backup_minutes": 10,
  "theme": "dark",
  "window_width": 1200,
  "window_height": 760,
  "network_timeout_sec": 10,
  "random_user_api_url": "https://randomuser.me/api/"
}
```
