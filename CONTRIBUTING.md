# 🤝 Руководство по участию в разработке (Contributing Guide)

Спасибо за интерес к развитию **GoContact Studio Community Edition**! Проект открыт для улучшений, баг-репортов и реализации функций из открытого плана **[Roadmap в README](./README.md#-перспективы-развития-дорожная-карта-community-edition)**.

---

## 🛠️ Требования к окружению

* **Go:** версия `1.22+` (рекомендуется `1.27+`)
* **ОС:** Windows 10/11, macOS или Linux (для компиляции графического интерфейса Fyne v2 требуется C-компилятор GCC / MinGW-w64 / Clang для OpenGL)
* **Git:** для контроля версий

---

## 🚀 Локальная сборка и проверка перед коммитом

1. **Клонирование репозитория:**
   ```powershell
   git clone https://github.com/trollyHUB/gocontact-community.git
   cd gocontact-community
   ```

2. **Активация встроенных Git-хуков (защита от утечек и валидация коммитов):**
   ```powershell
   powershell -File .\.githooks\setup-hooks.ps1
   # или на Linux/macOS:
   git config core.hooksPath .githooks
   ```

3. **Запуск всех тестов (Unit + Integration + Headless UI):**
   ```powershell
   go test -v -count=1 ./...
   ```

4. **Сборка исполняемого файла (Windows):**
   ```powershell
   go build -ldflags="-s -w" -o addressbook.exe ./cmd/addressbook
   ```

---

## 📐 Архитектурные правила проекта

При добавлении нового функционала соблюдайте границы слоёв **Clean Architecture**:

1. **`internal/domain`** — не должен импортировать внешние библиотеки, БД или UI-пакеты. Только чистые структуры, бизнес-правила, валидаторы и доменные ошибки.
2. **`internal/repository`** — интерфейсы хранилищ и их реализации (`sqlite`, `memory`). Все разделяемые структуры в памяти обязаны синхронизироваться через `sync.RWMutex`.
3. **`internal/service`** — бизнес-логика, воркер-пулы (`goroutines` + `chan`) и фоновые планировщики (`select` + `context.Context`).
4. **`internal/ui`** — презентационный слой (`Fyne v2`) и паттерн `MVP` (`ContactPresenter`). Длительные операции должны вызываться асинхронно, чтобы не блокировать поток рендеринга GUI.

---

## 📝 Стандарт именования коммитов (Conventional Commits)

Используйте понятные префиксы для коммитов:
* `feat:` — новая функциональность (например, `feat(graph): add interactive social graph widget`)
* `fix:` — исправление ошибки (например, `fix(ui): resolve windowed mode scrolling in detail panel`)
* `docs:` — обновление документации
* `test:` — добавление или обновление модульных/интеграционных тестов
* `refactor:` — рефакторинг кода без изменения внешнего поведения
* `chore:` — технические задачи, скрипты, зависимости

---

## ⚖️ Лицензия вкладов (CLA + DCO)

Проект распространяется по двойной лицензии (`AGPL-3.0-only OR LicenseRef-GoContact-Commercial`). Чтобы ваш Pull Request был принят в основную ветку:

1. **Подпишите [CLA](./CLA.md)**: в комментариях к созданному Pull Request оставьте фразу:
   ```text
   I have read the CLA Document and I hereby sign the CLA
   ```
2. **Используйте DCO подпись коммита**: коммиты должны создаваться с флагом `-s`:
   ```bash
   git commit -s -m "feat(module): description"
   ```
3. **SPDX-заголовок**: добавляйте двухстрочный заголовок в начало каждого нового файла исходного кода:
   ```go
   // SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
   // Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.
   ```
4. **Чистота зависимостей**: не добавляйте зависимости под лицензиями `GPL-only`, `SSPL`, `BSL` или без лицензии (см. [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)).
5. **Плагины**: разработка независимых Wasm-плагинов и MCP-серверов регулируется официальным исключением [PLUGIN_EXCEPTION.md](./PLUGIN_EXCEPTION.md).
