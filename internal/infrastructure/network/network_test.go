// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package network

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	crmDomain "github.com/trollyHUB/gocontact-studio/internal/domain/crm"
	"github.com/trollyHUB/gocontact-studio/internal/repository/memory"
	"github.com/trollyHUB/gocontact-studio/internal/repository/sqlite"
	crmService "github.com/trollyHUB/gocontact-studio/internal/service/crm"
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
	crmRepo := sqlite.NewCRMRepo(db)
	topoCache := memory.NewTopologyCache()
	graphSvc := sna.NewGraphService(graphRepo, repo, topoCache, 2)
	_ = graphSvc.ReloadGraph(ctx, false)

	crmSvc := crmService.NewCRMService(crmRepo, repo, graphRepo)

	srv := NewLocalAPIServer(0, repo, graphSvc, crmSvc) // Выбираем свободный порт
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

	// Проверяем новые эндпоинты CRM v0.3.0
	// 1. POST /api/v1/crm/interactions
	interReq := CRMInteractionRequest{
		Interaction: &crmDomain.Interaction{
			Channel:        crmDomain.ChannelMeeting,
			Title:          "Встреча в кафе 'Nedelka'",
			ParticipantIDs: []int64{created.ID},
			Sentiment:      1,
		},
		Promise: &crmDomain.ActionPromise{
			ContactID:   created.ID,
			Description: "Отправить коммерческое предложение",
			DueAt:       time.Now().Add(24 * time.Hour),
		},
	}
	reqBody, _ := json.Marshal(interReq)
	postResp, err := http.Post(srv.BaseURL()+"/api/v1/crm/interactions", "application/json", bytes.NewReader(reqBody))
	if err != nil || postResp.StatusCode != http.StatusCreated {
		t.Fatalf("ошибка POST /api/v1/crm/interactions: %v, status: %v", err, postResp.StatusCode)
	}
	postResp.Body.Close()

	// 2. GET /api/v1/crm/interactions?contact_id=...
	getInterResp, err := http.Get(srv.BaseURL() + "/api/v1/crm/interactions?contact_id=" + strconv.FormatInt(created.ID, 10))
	if err != nil || getInterResp.StatusCode != http.StatusOK {
		t.Errorf("ошибка GET /api/v1/crm/interactions: %v, status: %v", err, getInterResp.StatusCode)
	}
	getInterResp.Body.Close()

	// 3. GET /api/v1/crm/radar
	radarResp, err := http.Get(srv.BaseURL() + "/api/v1/crm/radar?limit=5")
	if err != nil || radarResp.StatusCode != http.StatusOK {
		t.Errorf("ошибка GET /api/v1/crm/radar: %v, status: %v", err, radarResp.StatusCode)
	}
	radarResp.Body.Close()

	// 4. GET /api/v1/crm/dossier
	dossierResp, err := http.Get(srv.BaseURL() + "/api/v1/crm/dossier?contact_id=" + strconv.FormatInt(created.ID, 10))
	if err != nil || dossierResp.StatusCode != http.StatusOK {
		t.Errorf("ошибка GET /api/v1/crm/dossier: %v, status: %v", err, dossierResp.StatusCode)
	}
	dossierResp.Body.Close()
}
