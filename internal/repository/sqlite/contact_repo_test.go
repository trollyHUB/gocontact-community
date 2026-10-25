// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

func TestContactRepo_CRUDAndFilter(t *testing.T) {
	ctx := context.Background()
	db, err := OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("не удалось открыть in-memory SQLite: %v", err)
	}
	defer db.Close()

	repo := NewContactRepo(db)

	// 1. Создание контакта (INSERT)
	contact := &domain.Contact{
		GroupID:    2,
		FirstName:  "Арман",
		LastName:   "Султанов",
		Phone:      "87019991122",
		Email:      "arman@kolesa.kz",
		Company:    "Kolesa Group",
		JobTitle:   "Senior Go Engineer",
		Address:    domain.Address{City: "Алматы", Country: "Казахстан"},
		Tags:       []string{"golang", "highload"},
		IsFavorite: true,
	}

	if err := repo.Create(ctx, contact); err != nil {
		t.Fatalf("ошибка Create: %v", err)
	}
	if contact.ID <= 0 {
		t.Fatalf("ожидался присвоенный ID > 0, получено %d", contact.ID)
	}

	// Проверка защиты от дубликата номера телефона
	dup := &domain.Contact{
		FirstName: "Дубликат",
		LastName:  "Тестов",
		Phone:     "+7 (701) 999-11-22",
	}
	if err := repo.Create(ctx, dup); !errors.Is(err, domain.ErrDuplicatePhone) {
		t.Errorf("ожидалась ошибка ErrDuplicatePhone, получено: %v", err)
	}

	// 2. Чтение по ID (SELECT)
	fetched, err := repo.GetByID(ctx, contact.ID)
	if err != nil {
		t.Fatalf("ошибка GetByID: %v", err)
	}
	if fetched.FullName() != "Султанов Арман" || fetched.GroupName != "Работа" {
		t.Errorf("неверные данные после чтения: %+v", fetched)
	}

	// 3. Обновление (UPDATE)
	fetched.JobTitle = "Staff Go Architect"
	fetched.Address.City = "Астана"
	if err := repo.Update(ctx, fetched); err != nil {
		t.Fatalf("ошибка Update: %v", err)
	}

	// 4. Поиск и фильтрация по кириллической подстроке (SELECT + Filter)
	list, err := repo.List(ctx, domain.ContactFilter{
		Query:        "астана",
		GroupID:      2,
		OnlyFavorite: true,
	})
	if err != nil {
		t.Fatalf("ошибка List: %v", err)
	}
	if len(list) != 1 || list[0].JobTitle != "Staff Go Architect" {
		t.Fatalf("ожидался 1 обновлённый контакт, получено: %d", len(list))
	}

	// 5. Статистика
	stats, err := repo.Stats(ctx)
	if err != nil {
		t.Fatalf("ошибка Stats: %v", err)
	}
	if stats.TotalContacts != 1 || stats.FavoriteContacts != 1 {
		t.Errorf("некорректная статистика: %+v", stats)
	}

	// 6. Удаление (DELETE)
	if err := repo.Delete(ctx, contact.ID); err != nil {
		t.Fatalf("ошибка Delete: %v", err)
	}
	if _, err := repo.GetByID(ctx, contact.ID); !errors.Is(err, domain.ErrContactNotFound) {
		t.Errorf("ожидалась ошибка ErrContactNotFound после удаления, получено: %v", err)
	}
}
