## 📋 Описание изменений

Кратко опишите, какую задачу или функциональность реализует данный Pull Request.

* **Связанный Issue:** closes #...
* **Раздел ТЗ:** `docs/01_SRS_SPECIFICATION.md` (FR-...)

---

## ✅ Чек-лист перед слиянием (Quality Gate)

- [ ] Соблюдены границы **Clean Architecture** (`internal/domain` имеет 0 внешних импортов, `internal/service` не зависит от UI).
- [ ] Добавлены табличные unit- / интеграционные тесты (`0 [no test files]`).
- [ ] Все тесты проходят локально: `go test -v -count=1 ./...`.
- [ ] Сборка Windows PE проверена с флагами `-ldflags="-s -w"`.
- [ ] Подписан CLA (в комментариях оставлена фраза `I have read the CLA Document and I hereby sign the CLA`).
- [ ] Обновлены соответствующие документы в `docs/` и `CHANGELOG.md`.
