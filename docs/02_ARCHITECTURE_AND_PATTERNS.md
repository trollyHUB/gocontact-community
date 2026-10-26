# Архитектура и паттерны проектирования (SAD — Software Architecture Document)
## Информационная система «Адресная книга» (GoContact Studio)

---

## 1. Архитектурная парадигма: Clean Architecture + MVP

Для обеспечения максимальной чистоты кода, тестируемости и строгого разделения ответственности (Separation of Concerns) проект строится по принципам **Чистой Архитектуры (Clean Architecture Роберта Мартина / Hexagonal Architecture)** в сочетании с паттерном **MVP (Model–View–Presenter)** для графического слоя.

### Главное правило зависимостей (The Dependency Rule):
> Зависимости исходного кода направлены **строго внутрь** — от внешних деталей реализации (GUI, База данных, Сеть, Файлы) к ядру бизнес-логики (`Domain`). Ядро не знает, в какой СУБД хранятся контакты и какая библиотека рисует кнопки на экране.

```
+-----------------------------------------------------------------------+
|                        EXTERNAL & UI LAYER                            |
|   [ Fyne GUI (Views) ]   [ SQLite DB ]   [ CSV/JSON Files ]  [ API ]  |
+-----------------------------------------------------------------------+
               |                 |                  |             |
               v                 v                  v             v
+-----------------------------------------------------------------------+
|                        INTERFACE ADAPTERS                             |
|   [ UI Presenters ]      [ SQLite Repo ]   [ File Exporter ] [ HTTP ] |
+-----------------------------------------------------------------------+
                                 |
                                 v
+-----------------------------------------------------------------------+
|                     APPLICATION / USE CASES                           |
|          [ ContactService ]   [ ImportExportService ]                 |
|          [ BackupWorker ]     [ Repository Interfaces ]               |
+-----------------------------------------------------------------------+
                                 |
                                 v
+-----------------------------------------------------------------------+
|                        DOMAIN LAYER (CORE)                            |
|     [ Contact, Group, Address ]   [ Custom Errors ]   [ Validator ]   |
+-----------------------------------------------------------------------+
```

---

## 2. Стандартная файловая структура проекта (Standard Go Project Layout)

Структура проекта организована по мировому стандарту сообщества Go (`golang-standards/project-layout`):

```text
GoLandProject/
├── cmd/
│   └── addressbook/
│       └── main.go                  # Точка входа: инициализация конфига, БД, сервисов и запуск GUI
├── configs/
│   └── config.json                  # Конфигурационный файл приложения (путь к БД, тема, интервал бэкапа)
├── assets/
│   ├── embed.go                     # Встраивание ресурсов приложения через директиву //go:embed
│   ├── icon.png                     # Иконка приложения (встраивается через go:embed)
│   └── sample_contacts.csv          # Тестовый набор контактов для демонстрации импорта
├── docs/                            # Полный комплект технической документации
│   ├── 01_SRS_SPECIFICATION.md
│   ├── 02_ARCHITECTURE_AND_PATTERNS.md
│   ├── 03_DATABASE_AND_STORAGE.md
│   ├── 04_UI_UX_AND_CONCURRENCY.md
│   ├── 05_ROADMAP_AND_STAGES.md
│   └── 06_DEFENSE_GUIDE.md
├── internal/                        # Приватный код приложения (изолирован от внешнего импорта)
│   ├── config/
│   │   ├── config.go                # Загрузка, парсинг и сохранение JSON-конфигурации
│   │   └── config_test.go           # Unit-тесты конфигурационного пакета
│   ├── domain/                      # Слой 1: Ядро предметной области (0 внешних зависимостей)
│   │   ├── contact.go               # Структуры Contact, Address, AuditInfo, PhoneType, методы
│   │   ├── group.go                 # Структура Group и константы категорий
│   │   ├── errors.go                # Кастомные типы ошибок (ValidationError, ErrNotFound и др.)
│   │   ├── validator.go             # Валидация полей контакта (регулярные выражения, правила)
│   │   └── validator_test.go        # Unit-тесты валидатора и доменной модели
│   ├── repository/                  # Слой 2: Хранилище данных (Реализация интерфейсов)
│   │   ├── interfaces.go            # Интерфейсы ContactRepository и GroupRepository
│   │   ├── memory/                  # In-Memory кэш и быстрый индекс на map + sync.RWMutex
│   │   │   ├── cache.go
│   │   │   └── cache_test.go        # Конкурентные тесты потокобезопасного кэша
│   │   └── sqlite/                  # Реляционное хранилище SQLite
│   │       ├── db.go                # Подключение к SQLite, PRAGMA-настройки, миграции схемы
│   │       ├── contact_repo.go      # SQL-операции INSERT, SELECT, UPDATE, DELETE, фильтрация
│   │       ├── group_repo.go        # SQL-операции над группами контактов
│   │       └── contact_repo_test.go # Интеграционные тесты БД на in-memory SQLite (:memory:)
│   ├── infrastructure/              # Слой 3: Внешние адаптеры (Файлы и Сеть)
│   │   ├── fileio/
│   │   │   ├── interfaces.go        # Интерфейсы Exporter и Importer (Паттерн Strategy)
│   │   │   ├── json_storage.go      # Экспорт/импорт в формат JSON
│   │   │   ├── csv_storage.go       # Экспорт/импорт в формат CSV
│   │   │   ├── vcard_storage.go     # Генерация карточек .vcf (vCard 3.0)
│   │   │   └── fileio_test.go       # Тесты сериализации/десериализации
│   │   └── network/
│   │       ├── randomuser_api.go    # HTTP-клиент для загрузки контактов из REST API
│   │       ├── local_api.go         # Встроенный локальный HTTP REST API сервер
│   │       ├── avatar_client.go     # Генерация и проверка сетевых аватаров (Gravatar / UI-Avatars)
│   │       └── network_test.go      # Интеграционные тесты HTTP-клиента и REST-сервера
│   ├── service/                     # Слой 4: Бизнес-логика и конкурентные воркеры (Use Cases)
│   │   ├── contact_service.go       # Основной сервис управления контактами и группами
│   │   ├── import_worker.go         # Конкурентный Worker Pool (goroutines + channels) и аудитор
│   │   ├── backup_scheduler.go      # Фоновый планировщик автобэкапов (select + time.Ticker)
│   │   └── contact_service_test.go  # Unit-тесты бизнес-логики и горутин
│   └── ui/                          # Слой 5: Графический интерфейс пользователя (Fyne v2)
│       ├── app.go                   # Инициализация Fyne App, главного окна и меню
│       ├── presenter.go             # Связующее звено (Presenter) между UI и сервисами
│       ├── theme.go                 # Настройка цветовой палитры (Светлая/Темная тема)
│       ├── sidebar.go               # Левая панель: навигация по группам и статистика
│       ├── contact_list.go          # Центральная панель: строка поиска + список контактов
│       ├── contact_detail.go        # Правая панель: карточка просмотра контакта
│       ├── status_bar.go            # Нижняя строка состояния + ProgressBar фоновых задач
│       ├── contact_form.go          # Модальное окно создания/редактирования контакта и групп
│       ├── import_export.go         # Диалоги выбора файла, экспорта и прогресса импорта
│       ├── stats_dialog.go          # Окно аналитики, таблицы аудита и тестера REST API
│       └── ui_test.go               # Headless UI-тесты и проверка рендера окон
├── go.mod                           # Определение Go-модуля и зависимостей
└── README.md                        # Общее описание и инструкция по сборке
```

---

## 3. Применяемые паттерны проектирования (Design Patterns)

Использование классических и идиоматичных для Go паттернов проектирования является огромным плюсом на защите проекта:

### 3.1. Паттерн «Репозиторий» (Repository Pattern)
Изолирует бизнес-логику от SQL-запросов. Сервис работает с интерфейсом `ContactRepository`:
```go
type ContactRepository interface {
    Create(ctx context.Context, contact *domain.Contact) error
    GetByID(ctx context.Context, id int64) (*domain.Contact, error)
    Update(ctx context.Context, contact *domain.Contact) error
    Delete(ctx context.Context, id int64) error
    List(ctx context.Context, filter domain.ContactFilter) ([]*domain.Contact, error)
    Stats(ctx context.Context) (*domain.BookStats, error)
}
```
Благодаря этому интерфейсу мы можем подменять SQLite на In-Memory хранилище в тестах за одну строчку кода.

### 3.2. Паттерн «Стратегия» (Strategy Pattern) для Экспорта и Импорта
Единый интерфейс для разных форматов файлов (`JSON`, `CSV`, `vCard`):
```go
type Exporter interface {
    Export(w io.Writer, contacts []*domain.Contact) error
    FormatName() string
    FileExtension() string
}

type Importer interface {
    Import(r io.Reader) ([]*domain.Contact, error)
}
```
Когда пользователь выбирает в меню «Экспорт в CSV» или «Экспорт в JSON», `ContactService` просто вызывает нужную стратегию, не меняя свой код (принцип **Open/Closed Principle** из SOLID).

### 3.3. Паттерн «Композиция вместо наследования» (Composition over Inheritance)
В Go нет классов и `extends`. Вместо этого мы используем **встраивание структур (Struct Embedding)**:
```go
type AuditInfo struct {
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Address struct {
    Street     string `json:"street"`
    City       string `json:"city"`
    Country    string `json:"country"`
    PostalCode string `json:"postal_code"`
}

type Contact struct {
    ID        int64   `json:"id"`
    FirstName string  `json:"first_name"`
    LastName  string  `json:"last_name"`
    Address   Address `json:"address"` // Вложенная структура
    AuditInfo         // Встроенная структура (методы AuditInfo доступны напрямую у Contact)
}
```

### 3.4. Конкурентный паттерн «Worker Pool» + «Fan-Out / Fan-In»
Для выполнения требований 10-й недели (горутины, каналы, `select`, `sync.WaitGroup`):
* При массовом импорте или проверке контактов запускается пул из $N$ горутин-воркеров.
* Задачи подаются в канал `jobs <-chan RawContact`.
* Результаты и ошибки собираются в канал `results chan<- ProcessResult`.
* Канал прогресса `progress chan<- float64` плавно обновляет `ProgressBar` в графическом интерфейсе.

### 3.5. Паттерн «Внедрение зависимостей» (Dependency Injection)
Ни один пакет не использует глобальные переменные для БД или конфигурации. Все зависимости создаются в `cmd/addressbook/main.go` и передаются явно через конструкторы (`NewSQLiteRepo(db)`, `NewContactService(repo, cache, apiClient)`, `NewMainWindow(service)`).

---

## 4. Обоснование выбора технологического стека

| Компонент | Выбранный инструмент | Почему это лучшее решение (Best Practice) |
| :--- | :--- | :--- |
| **Язык** | **Go 1.27** | Строгая статическая типизация, высокая скорость компиляции, встроенная конкурентность. |
| **GUI Фреймворк** | **Fyne v2 (`fyne.io/fyne/v2`)** | Самый популярный нативный GUI-фреймворк на чистом Go (26k+ звезд на GitHub). В отличие от веб-оберток (Wails/Electron), весь код интерфейса пишется **на самом Go**, что идеально соответствует требованиям учебного курса по Go. Имеет встроенные таблицы, списки, диалоги, иконки и переключение тем. |
| **СУБД** | **SQLite (`modernc.org/sqlite`)** | Встраиваемая реляционная БД. Драйвер `modernc.org/sqlite` написан на **чистом Go (CGO-free)** для работы с БД и полностью совместим со стандартным пакетом `database/sql`. База хранится в одном файле `addressbook.db`. |
| **Форматы обмена** | Стандартные пакеты `encoding/json`, `encoding/csv` | Не требуют сторонних библиотек, демонстрируют глубокое владение стандартной библиотекой Go (`stdlib`). |
| **Сеть** | Стандартный `net/http` + `context` | Идиоматичная работа с REST API, настройка `http.Client{Timeout: 10 * time.Second}`, защита от зависаний через `context.WithTimeout`. |
| **Тестирование** | Стандартный пакет `testing` + `fyne.io/fyne/v2/test` | Позволяет запускать тесты одной командой `go test ./...` в любой среде без настройки внешних тест-раннеров. |
