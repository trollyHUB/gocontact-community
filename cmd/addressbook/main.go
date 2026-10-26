// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/config"
	"github.com/trollyHUB/gocontact-studio/internal/infrastructure/network"
	"github.com/trollyHUB/gocontact-studio/internal/repository/memory"
	"github.com/trollyHUB/gocontact-studio/internal/repository/sqlite"
	"github.com/trollyHUB/gocontact-studio/internal/service"
	"github.com/trollyHUB/gocontact-studio/internal/service/crm"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
	"github.com/trollyHUB/gocontact-studio/internal/ui"
)

const defaultConfigPath = "configs/config.json"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("GoContact Studio v0.3.0 (Windows/amd64)")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Загрузка конфигурации приложения
	cfg, err := config.Load(defaultConfigPath)
	if err != nil {
		log.Printf("[WARN] используется конфигурация по умолчанию: %v", err)
	}

	// 2. Подключение к СУБД SQLite и выполнение миграций (включая 0002_social_graph_schema)
	db, err := sqlite.OpenAndMigrate(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("[FATAL] ошибка инициализации базы данных: %v", err)
	}
	defer db.Close()

	// Предзаполнение демонстрационными контактами при первом запуске
	if err := sqlite.SeedDemoContactsIfEmpty(ctx, db); err != nil {
		log.Printf("[WARN] ошибка предзаполнения демо-данных: %v", err)
	}

	// 3. Инициализация репозиториев, кэшей и сетевых адаптеров
	contactRepo := sqlite.NewContactRepo(db)
	groupRepo := sqlite.NewGroupRepo(db)
	graphRepo := sqlite.NewGraphRepo(db)
	crmRepo := sqlite.NewSQLiteCRMRepo(db)
	contactCache := memory.NewContactCache()
	topologyCache := memory.NewTopologyCache()
	apiClient := network.NewRandomUserClient(cfg.RandomUserAPIURL, time.Duration(cfg.NetworkTimeoutSec)*time.Second)

	// 4. Инициализация бизнес-сервисов и пулов воркеров
	contactService := service.NewContactService(
		contactRepo,
		groupRepo,
		contactCache,
		apiClient,
		cfg.WorkerPoolSize,
	)

	// Подсистема Социального графа (Social Network Analysis)
	graphService := sna.NewGraphService(
		graphRepo,
		contactRepo,
		topologyCache,
		cfg.WorkerPoolSize,
	)

	// Подсистема Personal CRM (Relationship Intelligence)
	crmService := crm.NewCRMService(
		crmRepo,
		contactRepo,
		graphRepo,
	)

	// Первичная синхронизация топологии и запуск физического движка
	if err := graphService.ReloadGraph(ctx, false); err != nil {
		log.Printf("[WARN] ошибка начальной загрузки графа: %v", err)
	}
	if err := graphService.RecalculateMetrics(ctx); err != nil {
		log.Printf("[WARN] ошибка расчёта начальных метрик графа: %v", err)
	}
	graphService.StartSimulation(ctx)
	defer graphService.StopSimulation()

	// 5. Запуск встроенного локального HTTP REST API сервера в фоновой горутине
	localAPIServer := network.NewLocalAPIServer(cfg.LocalAPIPort, contactRepo, graphService, crmService)
	if err := localAPIServer.Start(); err != nil {
		log.Printf("[WARN] не удалось запустить локальный REST API сервер: %v", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()
		_ = localAPIServer.Stop(shutdownCtx)
	}()

	// 6. Инициализация фонового планировщика автобэкапов на select + Ticker
	backupScheduler := service.NewBackupScheduler(contactService, cfg.BackupDir, cfg.AutoBackupMinutes)

	// 7. Инициализация и запуск графического интерфейса Fyne v2 (Контакты, Граф, Радар)
	appUI := ui.NewAppUI(cfg, defaultConfigPath, contactService, graphService, crmService, backupScheduler, localAPIServer)
	backupScheduler.Start(ctx, appUI.NotifyBackup)

	appUI.Run()
}
