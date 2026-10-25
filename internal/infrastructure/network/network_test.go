// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package network

import (
	"context"
	"testing"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/repository/sqlite"
)

func TestLocalAPIServer_AndHTTPClient(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("ошибка открытия БД: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewContactRepo(db)
	srv := NewLocalAPIServer(0, repo) // Выбираем свободный порт
	if err := srv.Start(); err != nil {
		t.Fatalf("не удалось запустить локальный REST API сервер: %v", err)
	}
	defer srv.Stop(ctx)

	newContact := &domain.Contact{
		GroupID:   2,
		FirstName: "Темирлан",
		LastName:  "Сериков",
		Phone:     "87017776655",
		Email:     "temirlan@api.kz",
		Company:   "REST API Corp",
	}

	created, err := CreateContactViaHTTP(ctx, srv.BaseURL(), newContact)
	if err != nil {
		t.Fatalf("ошибка создания контакта через HTTP POST: %v", err)
	}
	if created.ID <= 0 || created.Phone != "+7 (701) 777-66-55" {
		t.Errorf("неожиданный ответ от REST API: %+v", created)
	}

	avatarURL := GenerateAvatarURL(created)
	if avatarURL == "" {
		t.Error("ожидался сгенерированный URL аватара по Email")
	}
}
