// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package service

import (
	"context"
	"testing"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/infrastructure/network"
	"github.com/trollyHUB/gocontact-studio/internal/repository/memory"
	"github.com/trollyHUB/gocontact-studio/internal/repository/sqlite"
)

func TestContactService_ConcurrentBatchImportAndAudit(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("ошибка открытия in-memory БД: %v", err)
	}
	defer db.Close()

	contactRepo := sqlite.NewContactRepo(db)
	groupRepo := sqlite.NewGroupRepo(db)
	cache := memory.NewContactCache()
	apiClient := network.NewRandomUserClient("", 2*time.Second)

	svc := NewContactService(contactRepo, groupRepo, cache, apiClient, 4)

	batch := []*domain.Contact{
		{FirstName: "Алия", LastName: "Касымова", Phone: "87011112233", Email: "aliya@test.kz", Address: domain.Address{City: "Алматы"}},
		{FirstName: "Ерлан", LastName: "Нуров", Phone: "87022223344", Email: ""},                                                       // без email и адреса
		{FirstName: "Дубликат", LastName: "Номера", Phone: "+7 (701) 111-22-33", Email: "dup@test.kz"},                                 // дубликат телефона
		{FirstName: "", LastName: "Невалидный", Phone: "123", Email: "bad-email"},                                                      // невалидная запись
	}

	progressChan := make(chan float64, 10)
	summary := svc.ImportBatchConcurrent(ctx, batch, progressChan)

	if summary.TotalProcessed != 4 {
		t.Errorf("ожидалось TotalProcessed=4, получено %d", summary.TotalProcessed)
	}
	if summary.SuccessCount != 2 {
		t.Errorf("ожидалось SuccessCount=2, получено %d (ошибки: %v)", summary.SuccessCount, summary.ErrorMessages)
	}
	if summary.DuplicateCount != 1 {
		t.Errorf("ожидалось DuplicateCount=1, получено %d", summary.DuplicateCount)
	}
	if summary.InvalidCount != 1 {
		t.Errorf("ожидалось InvalidCount=1, получено %d", summary.InvalidCount)
	}

	// Проверяем параллельный аудит адресной книги
	audit, err := svc.AuditContactsConcurrent(ctx)
	if err != nil {
		t.Fatalf("ошибка AuditContactsConcurrent: %v", err)
	}
	if audit.TotalChecked != 2 || audit.MissingEmailCount != 1 {
		t.Errorf("неожиданный отчёт аудита: %+v", audit)
	}
}
