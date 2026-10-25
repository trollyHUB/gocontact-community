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
	"github.com/trollyHUB/gocontact-studio/internal/ui"
)

const defaultConfigPath = "configs/config.json"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("GoContact Studio v1.0.0 (Windows/amd64)")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Загрузка конфигурации приложения (Неделя 14)
	cfg, err := config.Load(defaultConfigPath)
	if err != nil {
		log.Printf("[WARN] используется конфигурация по умолчанию: %v", err)
	}

	// 2. Подключение к СУБД SQLite и выполнение миграций (Неделя 12)
	db, err := sqlite.OpenAndMigrate(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("[FATAL] ошибка инициализации базы данных: %v", err)
	}
	defer db.Close()

	// Предзаполнение демонстрационными контактами при первом запуске
	if err := sqlite.SeedDemoContactsIfEmpty(ctx, db); err != nil {
		log.Printf("[WARN] ошибка предзаполнения демо-данных: %v", err)
	}

	// 3. Инициализация репозиториев, кэша и сетевых адаптеров (Недели 5, 6, 11)
	contactRepo := sqlite.NewContactRepo(db)
	groupRepo := sqlite.NewGroupRepo(db)
	contactCache := memory.NewContactCache()
	apiClient := network.NewRandomUserClient(cfg.RandomUserAPIURL, time.Duration(cfg.NetworkTimeoutSec)*time.Second)

	// 4. Инициализация бизнес-сервисов и конкурентного пула воркеров (Недели 6, 10)
	contactService := service.NewContactService(
		contactRepo,
		groupRepo,
		contactCache,
		apiClient,
		cfg.WorkerPoolSize,
	)

	// 5. Запуск встроенного локального HTTP REST API сервера в фоновой горутине (Неделя 11)
	localAPIServer := network.NewLocalAPIServer(cfg.LocalAPIPort, contactRepo)
	if err := localAPIServer.Start(); err != nil {
		log.Printf("[WARN] не удалось запустить локальный REST API сервер: %v", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()
		_ = localAPIServer.Stop(shutdownCtx)
	}()

	// 6. Инициализация фонового планировщика автобэкапов на select + Ticker (Неделя 10)
	backupScheduler := service.NewBackupScheduler(contactService, cfg.BackupDir, cfg.AutoBackupMinutes)

	// 7. Инициализация и запуск графического интерфейса Fyne v2 (Неделя 9)
	appUI := ui.NewAppUI(cfg, defaultConfigPath, contactService, backupScheduler, localAPIServer)
	backupScheduler.Start(ctx, appUI.NotifyBackup)

	appUI.Run()
}
