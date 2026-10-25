# 📇 GoContact Studio Community Edition

[![CI](https://github.com/trollyHUB/gocontact-community/actions/workflows/ci.yml/badge.svg)](https://github.com/trollyHUB/gocontact-community/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/trollyHUB/gocontact-community?color=brightgreen)](https://github.com/trollyHUB/gocontact-community/releases/latest)
[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat&logo=go)](https://golang.org)
[![GUI Framework](https://img.shields.io/badge/GUI-Fyne%20v2.8-blue)](https://fyne.io)
[![Database](https://img.shields.io/badge/Database-SQLite%20(Pure%20Go)-lightgrey?logo=sqlite)](https://modernc.org/sqlite)
[![Architecture](https://img.shields.io/badge/Architecture-Clean%20%2B%20MVP-orange)](#-архитектура-clean-architecture--mvp)
[![Concurrency](https://img.shields.io/badge/Concurrency-Worker%20Pool%20%7C%20Select-brightgreen)](#-конкурентная-модель)
[![License](https://img.shields.io/badge/License-AGPL--3.0--only-blue.svg)](./LICENSE)

**GoContact Studio Community Edition** — открытая настольная информационная система «Адресная книга» / Local-First Personal CRM, разработанная на языке **Go (1.27+)** с аппаратным графическим интерфейсом на базе **Fyne v2** и встраиваемой реляционной СУБД **SQLite (WAL режим)**.

Проект построен по строгим стандартам **Clean Architecture (Чистая архитектура)** и **MVP (Model-View-Presenter)** с нулевыми сторонними зависимостями в доменном слое.

---

## ✨ Ключевые возможности (Релиз v0.1.0)

* 🗂️ **Полнофункциональный CRUD контактов и групп:** Создание, чтение, обновление, мягкое и безвозвратное удаление с авто-инкрементными первичными ключами.
* 🔎 **Параметризованный многокритериальный поиск:** Фильтрация по подстроке имени, email, номеру телефона, компании, должности, диапазону дат создания и тегам.
* 🛡️ **Валидация и нормализация номеров:** Поддержка форматов РК/РФ (`+7 (XXX) XXX-XX-XX`) и международного стандарта E.164.
* ⚡ **Многопоточный импорт (Worker Pool):** Пакетная параллельная загрузка сотен контактов с настраиваемым пулом горутин и контекстной отменой (`context.WithCancel`).
* 💾 **Резервное копирование по расписанию:** Фоновый планировщик (`select` + `time.Ticker`) с ротацией снимков базы данных.
* 🔄 **Импорт и экспорт данных:** Поддержка промышленных форматов **JSON**, **CSV** и визитных карточек **vCard 3.0 (.vcf)**.
* 🌐 **Встроенный локальный REST API сервер:** HTTP-сервер на порту `:8089` (`/api/v1/health`, `/api/v1/contacts`, `/api/v1/stats`).
* 🎨 **Двухрежимная адаптивная тема:** Поддержка Dark и Light тем оформления с динамическим переключением.

---

## 📂 Готовые файлы для демонстрации Импорта и Экспорта

В каталоге [`assets/`](./assets/) подготовлены эталонные наборы тестовых данных:
* [`assets/sample_contacts.json`](./assets/sample_contacts.json) — 15 контактов со сложной структурой;
* [`assets/sample_contacts.csv`](./assets/sample_contacts.csv) — 15 контактов в табличном формате с кавычками и запятыми;
* [`assets/sample_contacts.vcf`](./assets/sample_contacts.vcf) — 15 контактов по спецификации RFC 2426 (vCard 3.0);
* [`assets/import_stress_50.json`](./assets/import_stress_50.json) — 50 контактов для стресс-тестирования Worker Pool.

---

## 🚀 Быстрый старт, сборка и тестирование

### Требования
* Go `1.27+`
* Наличие C-компилятора (GCC / MinGW / TDM-GCC) для сборки Fyne OpenGL виджетов.

### Запуск и сборка на Windows
```powershell
# Запуск в режиме разработки
go run ./cmd/addressbook

# Сборка исполняемого файла с обязательными флагами Windows PE
go build -ldflags="-s -w" -o addressbook.exe ./cmd/addressbook

# Запуск полного набора модульных и интеграционных тестов
go test -v -count=1 ./...
```

---

## 🗺️ Перспективы развития: Дорожная карта Community Edition

Развитие открытой платформы **GoContact Studio Community Edition** осуществляется поэтапно по стандартам SemVer от базовой версии `v0.1.0` до полнофункциональной платформы `v1.0.0`. Документация и спецификации каждого этапа публикуются в открытом репозитории по мере выхода соответствующих релизов:

* ✅ **v0.1.0 — Foundation Baseline (Выпущен):** чистая архитектура, Fyne v2 MVP, SQLite WAL, JSON/CSV/vCard 3.0, Worker Pool, встроенный REST API;
* 🕸️ **v0.2.0 — Social Graph & Network Intelligence:** интерактивный граф связей 60 FPS (*Barnes-Hut*), орбиты Данбара, кластеризация сообществ Лувена и поиск тёплого интро (Дейкстра);
* 🤝 **v0.3.0 — Personal CRM (PRM):** таймлайн встреч и касаний, экспоненциальная модель затухания связи (*Health Score*), вехи жизни и бриф перед звонком;
* 🧠 **v0.4.0 — Entity Resolution & Smart Deduplication:** неизменяемый `Entity UUIDv7`, фонетический поиск дублей RU/KZ/EN (*Fellegi-Sunter*), мастер слияния с `Undo`, **vCard 4.0** + QR-визитки;
* 🔄 **v0.5.0 — CardDAV Bridge & P2P Local-First Sync:** синхронизация с iOS/Android по **CardDAV (RFC 6352/6578)** и локальная P2P-синхронизация **mDNS + PAKE + mTLS** на базе полевых **HLC-CRDT**;
* ⚡ **v0.6.0 — Automation, Event Bus & Smart Workflows:** шина событий Go `EventBus`, AST-запросы умных папок, парсер подписей из буфера обмена `Ctrl+Shift+V` и подписанные `HMAC` Webhooks;
* 🗺️ **v0.7.0 — Custom Schema (EAV/JSONB), Geo-BI & PDF:** конструктор произвольных полей, офлайн гео-карта `SQLite R-Tree` для командировок, нативные BI-графики и экспорт PDF-досье;
* 🔐 **v0.8.0 — Zero-Knowledge Security & Local AI (MCP):** конвертное шифрование **AES-256-GCM + Argon2id**, хеш-цепочка аудита, встроенный **MCP-сервер** и локальный семантический RAG через *Ollama*;
* 🧩 **v0.9.0 — Wasm Plugins, Command Palette & CLI/TUI:** песочница плагинов **WebAssembly (`wazero`)**, мгновенная палитра `Ctrl+K`, терминальный TUI-режим и утилита самодиагностики `doctor`;
* 🏆 **v1.0.0 — Enterprise GA & Cross-Platform:** локализация **Русский / Қазақша / English**, полнотекстовый поиск **SQLite FTS5** на 100 000+ контактов (`< 5 мс`) и установщики для Windows, macOS и Linux.

---

## 📚 Промышленная техническая документация (`docs/`)

1. 📄 **[01_SRS_SPECIFICATION.md](./docs/01_SRS_SPECIFICATION.md)** — Спецификация требований к ПО (SRS), функциональные и нефункциональные требования.
2. 🏗️ **[02_ARCHITECTURE_AND_PATTERNS.md](./docs/02_ARCHITECTURE_AND_PATTERNS.md)** — Архитектура (Clean Architecture + MVP), структура пакетов и паттерны проектирования.
3. 🗄️ **[03_DATABASE_AND_STORAGE.md](./docs/03_DATABASE_AND_STORAGE.md)** — ER-диаграмма БД, схема таблиц и индексов SQLite WAL, форматы JSON, CSV, vCard.
4. 🎨 **[04_UI_UX_AND_CONCURRENCY.md](./docs/04_UI_UX_AND_CONCURRENCY.md)** — Проектирование GUI, адаптивная тема и конкурентная модель (`Worker Pool`, `select`, `RWMutex`).
5. ⚖️ **[docs/legal/](./docs/legal/)** — Правовое руководство и условия двойного лицензирования.

Стандарты открытого ПО и правовая защита в корне репозитория:
* 📜 **[CHANGELOG.md](./CHANGELOG.md)** — История релизов по стандарту SemVer.
* 🤝 **[CONTRIBUTING.md](./CONTRIBUTING.md)** & **[CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md)** — Правила участия и кодекс поведения.
* 🔐 **[SECURITY.md](./SECURITY.md)** — Политика информационной безопасности.
* ⚖️ **[LICENSE](./LICENSE)** — GNU AGPL v3 (`AGPL-3.0-only`).
* 💼 **[LICENSE-COMMERCIAL.md](./LICENSE-COMMERCIAL.md)** — Условия коммерческой лицензии и [шаблон договора](./docs/legal/COMMERCIAL_LICENSE_TEMPLATE.md).
* ™️ **[TRADEMARK.md](./TRADEMARK.md)** — Политика защиты бренда и товарных знаков.
* 🔌 **[PLUGIN_EXCEPTION.md](./PLUGIN_EXCEPTION.md)** — Официальное исключение AGPL ст. 7 для независимых Wasm/MCP плагинов.
* ✍️ **[CLA.md](./CLA.md)** — Соглашение о передаче прав на вклад (Contributor License Agreement).
* 📦 **[THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)** — Сторонние зависимости и отчёт [`docs/legal/third_party_licenses.csv`](./docs/legal/third_party_licenses.csv).
* 📁 **[licenses/](./licenses/)** — Полные тексты лицензий (AGPL-3.0, BSD-3, MIT, Apache-2.0).
* 📘 **[docs/legal/LICENSING_GUIDE.md](./docs/legal/LICENSING_GUIDE.md)** — Полное руководство по защите интеллектуальной собственности.

---

## ⚖️ Лицензия и защита проекта

GoContact Studio Community Edition распространяется по **двойной лицензии**: `AGPL-3.0-only OR LicenseRef-GoContact-Commercial`.

* **Бесплатно по [GNU AGPL v3](./LICENSE)** — для личного использования, команд, self-hosted и open-source проектов. Если вы изменяете код и распространяете его или предоставляете доступ по сети (ст. 13), вы обязаны открыть исходный код под AGPL v3.
* **[Коммерческая лицензия](./LICENSE-COMMERCIAL.md)** — для закрытых продуктов, SaaS, white-label и OEM без раскрытия исходного кода.
* **[Исключение для плагинов](./PLUGIN_EXCEPTION.md)** — разработчики Wasm-плагинов и MCP-серверов могут свободно выбирать любую лицензию (включая закрытую) для своих модулей.
* **[TRADEMARK.md](./TRADEMARK.md)** — название «GoContact Studio» и логотип не лицензируются вместе с кодом; форки обязаны переименовываться.
* **[NOTICE](./NOTICE)** — правообладатель, дополнительные условия по ст. 7 AGPL (атрибуция), история лицензии.

Copyright © 2026 Tolegen Manassov ([trollyHUB](https://github.com/trollyHUB)).
