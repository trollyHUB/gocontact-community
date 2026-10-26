// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package network

import (
	"context"
	"net/http"
	"testing"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/repository/memory"
	"github.com/trollyHUB/gocontact-studio/internal/repository/sqlite"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

func TestLocalAPIServer_AndHTTPClient(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("ошибка открытия БД: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewContactRepo(db)
	graphRepo := sqlite.NewGraphRepo(db)
	topoCache := memory.NewTopologyCache()
	graphSvc := sna.NewGraphService(graphRepo, repo, topoCache, 2)
	_ = graphSvc.ReloadGraph(ctx, false)

	srv := NewLocalAPIServer(0, repo, graphSvc) // Выбираем свободный порт
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

	// Проверяем эндпоинты health, stats, graph
	resp, err := http.Get(srv.BaseURL() + "/api/v1/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Errorf("ошибка GET /api/v1/health: %v, status: %v", err, resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = http.Get(srv.BaseURL() + "/api/v1/stats")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Errorf("ошибка GET /api/v1/stats: %v, status: %v", err, resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = http.Get(srv.BaseURL() + "/api/v1/graph")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Errorf("ошибка GET /api/v1/graph: %v, status: %v", err, resp.StatusCode)
	}
	resp.Body.Close()
}
