// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/trollyHUB/gocontact-studio/internal/config"
	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/infrastructure/network"
	"github.com/trollyHUB/gocontact-studio/internal/repository/memory"
	"github.com/trollyHUB/gocontact-studio/internal/repository/sqlite"
	"github.com/trollyHUB/gocontact-studio/internal/service"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

func TestStudioTheme_AndHelpers(t *testing.T) {
	darkTheme := NewStudioTheme("dark")
	lightTheme := NewStudioTheme("light")

	if darkTheme.Color(theme.ColorNamePrimary, theme.VariantDark) == nil {
		t.Error("ожидался непустой основной цвет темы")
	}
	if lightTheme.Size(theme.SizeNamePadding) != 6 {
		t.Errorf("ожидался отступ 6, получено %v", lightTheme.Size(theme.SizeNamePadding))
	}

	red := ParseHexColor("#EF4444")
	if red.R != 239 {
		t.Errorf("неверный разбор HEX-цвета #EF4444: %+v", red)
	}

	if valueOrDash("") != "—" || valueOrDash("Алматы") != "Алматы" {
		t.Error("ошибка функции valueOrDash")
	}
}

func TestFullAppUI_WithSocialGraphTabsAndScreenshots(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("ошибка открытия in-memory БД: %v", err)
	}
	defer db.Close()

	if err := sqlite.SeedDemoContactsIfEmpty(ctx, db); err != nil {
		t.Fatalf("ошибка сидирования демо-контактов: %v", err)
	}

	contactRepo := sqlite.NewContactRepo(db)
	groupRepo := sqlite.NewGroupRepo(db)
	graphRepo := sqlite.NewGraphRepo(db)
	cache := memory.NewContactCache()
	topCache := memory.NewTopologyCache()
	apiClient := network.NewRandomUserClient("", 2*time.Second)

	svc := service.NewContactService(contactRepo, groupRepo, cache, apiClient, 2)
	graphSvc := sna.NewGraphService(graphRepo, contactRepo, topCache, 2)

	// Создаём тестовые связи между контактами 1, 2, 3
	_, _ = graphSvc.CreateRelationship(ctx, &graph.Relationship{
		SourceID:    1,
		TargetID:    2,
		Type:        graph.RelIntroducedBy,
		Weight:      5,
		IsDirected:  true,
		ContextNote: "Хакатон Алматы 2025",
	})
	_, _ = graphSvc.CreateRelationship(ctx, &graph.Relationship{
		SourceID:    2,
		TargetID:    3,
		Type:        graph.RelPartner,
		Weight:      4,
		IsDirected:  false,
		ContextNote: "Совместный акселератор",
	})
	_ = graphSvc.ReloadGraph(ctx, false)
	_ = graphSvc.RecalculateMetrics(ctx)

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	cfg := config.DefaultConfig()
	backupSched := service.NewBackupScheduler(svc, filepath.Join(tmpDir, "backups"), 10)
	localAPI := network.NewLocalAPIServer(0, contactRepo, graphSvc)

	headlessApp := test.NewApp()
	defer headlessApp.Quit()

	appUI := NewAppUIWithApp(headlessApp, cfg, cfgPath, svc, graphSvc, backupSched, localAPI)
	appUI.mainWin.Resize(fyne.NewSize(1240, 780))

	// Проверяем наличие двух вкладок: Контакты и Социальный граф
	if len(appUI.tabs.Items) != 2 {
		t.Fatalf("ожидалось 2 вкладки приложения, получено %d", len(appUI.tabs.Items))
	}

	// Переключаемся на вкладку «Социальный граф»
	appUI.tabs.SelectIndex(1)
	appUI.graphView.RefreshData()

	// Проверяем захват скриншотов
	outDir := os.Getenv("GOCONTACT_SCREENSHOT_DIR")
	if outDir == "" {
		if os.Getenv("CAPTURE_SCREENSHOTS") == "1" {
			outDir = filepath.Join("..", "..", "docs", "screenshots", "v0.2.0")
		} else {
			outDir = t.TempDir()
		}
	}

	// 1. Скриншот режима Force Graph
	saveCanvasPNG(t, appUI.mainWin, filepath.Join(outDir, "ui_social_graph_force.png"))

	// 2. Скриншот режима Dunbar Orbits
	appUI.graphView.modeSelect.SetSelected("Орбиты Данбара")
	saveCanvasPNG(t, appUI.mainWin, filepath.Join(outDir, "ui_social_graph_dunbar.png"))

	// 3. Скриншот поиска цепочки интро (Dijkstra)
	appUI.graphView.modeSelect.SetSelected("Силовой граф")
	if len(appUI.graphView.sourceSelect.Options) >= 2 {
		appUI.graphView.sourceSelect.SetSelected(appUI.graphView.sourceSelect.Options[0])
		appUI.graphView.targetSelect.SetSelected(appUI.graphView.targetSelect.Options[1])
		appUI.graphView.handleFindWarmIntro()
		saveCanvasPNG(t, appUI.mainWin, filepath.Join(outDir, "ui_warm_intro_path.png"))
	}

	// Возвращаемся на вкладку Контакты и проверяем карточку связей
	appUI.tabs.SelectIndex(0)
	if len(appUI.contacts) > 0 {
		appUI.renderDetailPanel(appUI.contacts[0])
	}
}

func TestSocialGraphCanvas_Interactions(t *testing.T) {
	topCache := memory.NewTopologyCache()
	graphSvc := sna.NewGraphService(nil, nil, topCache, 1)

	canvas := NewSocialGraphCanvas(graphSvc)
	canvas.Resize(fyne.NewSize(800, 600))

	// Проверяем зум и панорамирование
	canvas.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, 2)})
	if canvas.zoom <= 1.0 {
		t.Errorf("ожидалось увеличение зума, текущий зум: %f", canvas.zoom)
	}

	canvas.ResetCamera()
	if canvas.zoom != 1.0 || canvas.panX != 0 || canvas.panY != 0 {
		t.Errorf("ошибка сброса камеры: zoom=%f, panX=%f, panY=%f", canvas.zoom, canvas.panX, canvas.panY)
	}

	// Проверяем фильтрацию орбит
	canvas.SetDunbarFilter(graph.DunbarSupportClique)
	if canvas.dunbarFilter != graph.DunbarSupportClique {
		t.Errorf("ожидался фильтр DunbarSupportClique, получено %v", canvas.dunbarFilter)
	}

	// Проверяем клик по пустому фону
	canvas.Tapped(&fyne.PointEvent{Position: fyne.NewPos(10, 10)})
	if canvas.selectedNodeID != 0 {
		t.Errorf("ожидался сброс выбранного узла, получено %d", canvas.selectedNodeID)
	}
}

func saveCanvasPNG(t *testing.T, win fyne.Window, path string) {
	t.Helper()
	img := win.Canvas().Capture()
	if img == nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	_ = png.Encode(f, img)
}
