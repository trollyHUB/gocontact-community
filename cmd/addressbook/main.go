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
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
	"github.com/trollyHUB/gocontact-studio/internal/ui"
)

const defaultConfigPath = "configs/config.json"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("GoContact Studio v0.2.0 (Windows/amd64)")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Р—Р°РіСЂСѓР·РєР° РєРѕРЅС„РёРіСѓСЂР°С†РёРё РїСЂРёР»РѕР¶РµРЅРёСЏ
	cfg, err := config.Load(defaultConfigPath)
	if err != nil {
		log.Printf("[WARN] РёСЃРїРѕР»СЊР·СѓРµС‚СЃСЏ РєРѕРЅС„РёРіСѓСЂР°С†РёСЏ РїРѕ СѓРјРѕР»С‡Р°РЅРёСЋ: %v", err)
	}

	// 2. РџРѕРґРєР»СЋС‡РµРЅРёРµ Рє РЎРЈР‘Р” SQLite Рё РІС‹РїРѕР»РЅРµРЅРёРµ РјРёРіСЂР°С†РёР№ (РІРєР»СЋС‡Р°СЏ 0002_social_graph_schema)
	db, err := sqlite.OpenAndMigrate(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("[FATAL] РѕС€РёР±РєР° РёРЅРёС†РёР°Р»РёР·Р°С†РёРё Р±Р°Р·С‹ РґР°РЅРЅС‹С…: %v", err)
	}
	defer db.Close()

	// РџСЂРµРґР·Р°РїРѕР»РЅРµРЅРёРµ РґРµРјРѕРЅСЃС‚СЂР°С†РёРѕРЅРЅС‹РјРё РєРѕРЅС‚Р°РєС‚Р°РјРё РїСЂРё РїРµСЂРІРѕРј Р·Р°РїСѓСЃРєРµ
	if err := sqlite.SeedDemoContactsIfEmpty(ctx, db); err != nil {
		log.Printf("[WARN] РѕС€РёР±РєР° РїСЂРµРґР·Р°РїРѕР»РЅРµРЅРёСЏ РґРµРјРѕ-РґР°РЅРЅС‹С…: %v", err)
	}

	// 3. РРЅРёС†РёР°Р»РёР·Р°С†РёСЏ СЂРµРїРѕР·РёС‚РѕСЂРёРµРІ, РєСЌС€РµР№ Рё СЃРµС‚РµРІС‹С… Р°РґР°РїС‚РµСЂРѕРІ
	contactRepo := sqlite.NewContactRepo(db)
	groupRepo := sqlite.NewGroupRepo(db)
	graphRepo := sqlite.NewGraphRepo(db)
	contactCache := memory.NewContactCache()
	topologyCache := memory.NewTopologyCache()
	apiClient := network.NewRandomUserClient(cfg.RandomUserAPIURL, time.Duration(cfg.NetworkTimeoutSec)*time.Second)

	// 4. РРЅРёС†РёР°Р»РёР·Р°С†РёСЏ Р±РёР·РЅРµСЃ-СЃРµСЂРІРёСЃРѕРІ Рё РїСѓР»РѕРІ РІРѕСЂРєРµСЂРѕРІ
	contactService := service.NewContactService(
		contactRepo,
		groupRepo,
		contactCache,
		apiClient,
		cfg.WorkerPoolSize,
	)

	// РџРѕРґСЃРёСЃС‚РµРјР° РЎРѕС†РёР°Р»СЊРЅРѕРіРѕ РіСЂР°С„Р° (Social Network Analysis)
	graphService := sna.NewGraphService(
		graphRepo,
		contactRepo,
		topologyCache,
		cfg.WorkerPoolSize,
	)

	// РџРµСЂРІРёС‡РЅР°СЏ СЃРёРЅС…СЂРѕРЅРёР·Р°С†РёСЏ С‚РѕРїРѕР»РѕРіРёРё Рё Р·Р°РїСѓСЃРє С„РёР·РёС‡РµСЃРєРѕРіРѕ РґРІРёР¶РєР°
	if err := graphService.ReloadGraph(ctx, false); err != nil {
		log.Printf("[WARN] РѕС€РёР±РєР° РЅР°С‡Р°Р»СЊРЅРѕР№ Р·Р°РіСЂСѓР·РєРё РіСЂР°С„Р°: %v", err)
	}
	if err := graphService.RecalculateMetrics(ctx); err != nil {
		log.Printf("[WARN] РѕС€РёР±РєР° СЂР°СЃС‡С‘С‚Р° РЅР°С‡Р°Р»СЊРЅС‹С… РјРµС‚СЂРёРє РіСЂР°С„Р°: %v", err)
	}
	graphService.StartSimulation(ctx)
	defer graphService.StopSimulation()

	// 5. Р—Р°РїСѓСЃРє РІСЃС‚СЂРѕРµРЅРЅРѕРіРѕ Р»РѕРєР°Р»СЊРЅРѕРіРѕ HTTP REST API СЃРµСЂРІРµСЂР° РІ С„РѕРЅРѕРІРѕР№ РіРѕСЂСѓС‚РёРЅРµ
	localAPIServer := network.NewLocalAPIServer(cfg.LocalAPIPort, contactRepo, graphService)
	if err := localAPIServer.Start(); err != nil {
		log.Printf("[WARN] РЅРµ СѓРґР°Р»РѕСЃСЊ Р·Р°РїСѓСЃС‚РёС‚СЊ Р»РѕРєР°Р»СЊРЅС‹Р№ REST API СЃРµСЂРІРµСЂ: %v", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutdownCancel()
		_ = localAPIServer.Stop(shutdownCtx)
	}()

	// 6. РРЅРёС†РёР°Р»РёР·Р°С†РёСЏ С„РѕРЅРѕРІРѕРіРѕ РїР»Р°РЅРёСЂРѕРІС‰РёРєР° Р°РІС‚РѕР±СЌРєР°РїРѕРІ РЅР° select + Ticker
	backupScheduler := service.NewBackupScheduler(contactService, cfg.BackupDir, cfg.AutoBackupMinutes)

	// 7. РРЅРёС†РёР°Р»РёР·Р°С†РёСЏ Рё Р·Р°РїСѓСЃРє РіСЂР°С„РёС‡РµСЃРєРѕРіРѕ РёРЅС‚РµСЂС„РµР№СЃР° Fyne v2 СЃ РІРєР»Р°РґРєРѕР№ СЃРѕС†РёР°Р»СЊРЅРѕРіРѕ РіСЂР°С„Р°
	appUI := ui.NewAppUI(cfg, defaultConfigPath, contactService, graphService, backupScheduler, localAPIServer)
	backupScheduler.Start(ctx, appUI.NotifyBackup)

	appUI.Run()
}
