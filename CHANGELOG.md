# 📦 Журнал изменений (Changelog) — GoContact Studio

Все значимые изменения проекта **GoContact Studio** («Адресная книга» на Go) документируются в этом файле.
Формат основан на стандарте [Keep a Changelog](https://keepachangelog.com/ru/1.1.0/), а проект придерживается [Семантического версионирования (SemVer)](https://semver.org/lang/ru/).

---

## [Unreleased] — На пути к v1.0.0 («Personal CRM & Social Graph Edition»)
- Граф социальных и деловых связей (**Interactive Force-Directed Social Graph**) на базе `canvas` Fyne v2;
- Движок **Personal CRM (PRM)** с журналом взаимодействий, таймлайном встреч и индексом теплоты контакта (*Relationship Health Score*);
- Двусторонняя синхронизация по протоколу **CardDAV / WebDAV (RFC 6352)** и децентрализованный **P2P Local-First Sync (mDNS + CRDT)**;
- Поддержка стандарта **vCard 4.0 (RFC 6350)** с полным двусторонним импортом/экспортом и полем `RELATED`;
- Нечёткая дедупликация контактов (**Fuzzy Deduplication**: расстояние Левенштейна, Jaro-Winkler, нормализация E.164);
- Сквозное шифрование локальной базы и бэкапов (**SQLCipher / AES-256-GCM + Argon2id**) и встроенный **MCP-сервер** для локальных ИИ-ассистентов.

---

## [0.1.0] — 2026-10-22 (Первый публичный релиз — Foundation Baseline)

### ✨ Добавлено (Added)
- **Ядро предметной области (`internal/domain`)**:
  - Структуры `Contact`, `Address`, `AuditInfo`, `Group`, перечисления `PhoneType`, теги и бизнес-методы (`FullName()`, `DisplayPhone()`, `HasTag()`, `Clone()`);
  - Строгая валидация и автоматическая нормализация телефонных номеров СНГ/Казахстана (`+7 (XXX) XXX-XX-XX`) и международных стандартов;
  - Кастомная иерархия ошибок (`ValidationError` с методами `Error()` и `Unwrap()`, сигнальные ошибки `ErrContactNotFound`, `ErrDuplicatePhone`, `ErrInvalidEmail`).
- **Реляционное хранилище SQLite и In-Memory кэш (`internal/repository`)**:
  - Чистый Go-драйвер `modernc.org/sqlite` (без внешних зависимостей SQLite DLL) с режимом `PRAGMA journal_mode=WAL` и `foreign_keys=ON`;
  - Автоматические миграции схемы БД (`groups`, `contacts`, индексы поиска) и начальное заполнение демо-данными (`SeedDefaultContactsIfEmpty`);
  - Потокобезопасный In-Memory кэш (`memory.ContactCache`) на базе `map[int64]*domain.Contact` и `sync.RWMutex` с защитой от гонок данных.
- **Файловый ввод-вывод и сериализация (`internal/infrastructure/fileio`)**:
  - Экспорт и импорт полных дампов в формате **JSON** (`encoding/json`);
  - Экспорт и импорт таблиц **CSV** (`encoding/csv`, UTF-8), совместимых с Excel и Google Contacts;
  - Генерация электронных визиток в международном формате **vCard 3.0 (`.vcf`, RFC 2426)**;
  - Готовые демонстрационные наборы данных в `assets/` (`sample_contacts.csv`, `sample_contacts.json`, `sample_contacts.vcf`) и `exports/`.
- **Сетевой слой и REST API (`internal/infrastructure/network`)**:
  - HTTP-клиент интеграции с внешним REST API (`randomuser.me`) с контекстными таймаутами и автоматическим офлайн-резервом (fallback);
  - Встроенный локальный HTTP REST API сервер (`127.0.0.1:8089`) с эндпоинтами `/api/v1/health`, `/api/v1/contacts`, `/api/v1/stats`;
  - Сетевой загрузчик графических аватаров по HTTP (`AvatarClient`).
- **Многопоточность и конкурентные сервисы (`internal/service`)**:
  - Паттерн **Worker Pool** (`goroutines`, буферизированные каналы `jobs`/`results`, `sync.WaitGroup`) для пакетного импорта контактов с потоковой передачей прогресса в UI;
  - Параллельный многопоточный **Аудит базы контактов** для поиска неполных карточек и проверки полноты бизнес-полей;
  - Фоновый планировщик автоматического резервного копирования (`BackupScheduler`) на базе `time.Ticker`, мультиплексора `select` и `context.Context`.
- **Графический интерфейс пользователя (`internal/ui`, Fyne v2)**:
  - Трёхколоночный адаптивный макет **Master-Detail** (Левая панель навигации и аналитики, Средняя панель живого поиска и списка карточек, Правая панель детального просмотра);
  - Кастомная тема оформления `StudioTheme` с мгновенным переключением **Тёмной (Dark)** и **Светлой (Light)** палитры;
  - Адаптивная 3-колоночная модальная форма создания/редактирования контакта (`780×560`), табличный отчёт аудита (`widget.NewTable`), индикаторы прогресса и строка состояния;
  - Встраивание ресурсов иконки приложения в единый `.exe` файл через директиву `//go:embed`.
- **Тестирование и документация (`docs/`)**:
  - 100% прохождение табличных unit-тестов и интеграционных тестов во всех 8 пакетах (`config`, `domain`, `fileio`, `network`, `memory`, `sqlite`, `service`, `ui`);
  - Полный комплект промышленной технической документации в папке `docs/` (ТЗ, Clean Architecture, SQLite WAL, UI/UX и правовой пакет двойного лицензирования).
