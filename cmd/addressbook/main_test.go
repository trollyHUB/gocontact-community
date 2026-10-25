// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/config"
	"github.com/trollyHUB/gocontact-studio/internal/infrastructure/network"
	"github.com/trollyHUB/gocontact-studio/internal/repository/memory"
	"github.com/trollyHUB/gocontact-studio/internal/repository/sqlite"
	"github.com/trollyHUB/gocontact-studio/internal/service"
)

func TestMainBootstrapComponents(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("ошибка загрузки конфигурации: %v", err)
	}

	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("ошибка открытия in-memory SQLite: %v", err)
	}
	defer db.Close()

	if err := sqlite.SeedDemoContactsIfEmpty(ctx, db); err != nil {
		t.Fatalf("ошибка предзаполнения демо-контактов: %v", err)
	}

	contactRepo := sqlite.NewContactRepo(db)
	groupRepo := sqlite.NewGroupRepo(db)
	cache := memory.NewContactCache()
	apiClient := network.NewRandomUserClient(cfg.RandomUserAPIURL, 2*time.Second)

	svc := service.NewContactService(contactRepo, groupRepo, cache, apiClient, cfg.WorkerPoolSize)
	stats, err := svc.GetStats(ctx)
	if err != nil {
		t.Fatalf("ошибка получения статистики: %v", err)
	}
	if stats.TotalContacts == 0 {
		t.Errorf("ожидались предзаполненные контакты после старта, получено 0")
	}
}
