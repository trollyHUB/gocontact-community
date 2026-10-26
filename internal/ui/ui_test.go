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
		t.Error("РѕР¶РёРґР°Р»СЃСЏ РЅРµРїСѓСЃС‚РѕР№ РѕСЃРЅРѕРІРЅРѕР№ С†РІРµС‚ С‚РµРјС‹")
	}
	if lightTheme.Size(theme.SizeNamePadding) != 6 {
		t.Errorf("РѕР¶РёРґР°Р»СЃСЏ РѕС‚СЃС‚СѓРї 6, РїРѕР»СѓС‡РµРЅРѕ %v", lightTheme.Size(theme.SizeNamePadding))
	}

	red := ParseHexColor("#EF4444")
	if red.R != 239 {
		t.Errorf("РЅРµРІРµСЂРЅС‹Р№ СЂР°Р·Р±РѕСЂ HEX-С†РІРµС‚Р° #EF4444: %+v", red)
	}

	if valueOrDash("") != "вЂ”" || valueOrDash("РђР»РјР°С‚С‹") != "РђР»РјР°С‚С‹" {
		t.Error("РѕС€РёР±РєР° С„СѓРЅРєС†РёРё valueOrDash")
	}
}

func TestFullAppUI_WithSocialGraphTabsAndScreenshots(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("РѕС€РёР±РєР° РѕС‚РєСЂС‹С‚РёСЏ in-memory Р‘Р”: %v", err)
	}
	defer db.Close()

	if err := sqlite.SeedDemoContactsIfEmpty(ctx, db); err != nil {
		t.Fatalf("РѕС€РёР±РєР° СЃРёРґРёСЂРѕРІР°РЅРёСЏ РґРµРјРѕ-РєРѕРЅС‚Р°РєС‚РѕРІ: %v", err)
	}

	contactRepo := sqlite.NewContactRepo(db)
	groupRepo := sqlite.NewGroupRepo(db)
	graphRepo := sqlite.NewGraphRepo(db)
	cache := memory.NewContactCache()
	topCache := memory.NewTopologyCache()
	apiClient := network.NewRandomUserClient("", 2*time.Second)

	svc := service.NewContactService(contactRepo, groupRepo, cache, apiClient, 2)
	graphSvc := sna.NewGraphService(graphRepo, contactRepo, topCache, 2)

	// РЎРѕР·РґР°С‘Рј С‚РµСЃС‚РѕРІС‹Рµ СЃРІСЏР·Рё РјРµР¶РґСѓ РєРѕРЅС‚Р°РєС‚Р°РјРё 1, 2, 3
	_, _ = graphSvc.CreateRelationship(ctx, &graph.Relationship{
		SourceID:    1,
		TargetID:    2,
		Type:        graph.RelIntroducedBy,
		Weight:      5,
		IsDirected:  true,
		ContextNote: "РҐР°РєР°С‚РѕРЅ РђР»РјР°С‚С‹ 2025",
	})
	_, _ = graphSvc.CreateRelationship(ctx, &graph.Relationship{
		SourceID:    2,
		TargetID:    3,
		Type:        graph.RelPartner,
		Weight:      4,
		IsDirected:  false,
		ContextNote: "РЎРѕРІРјРµСЃС‚РЅС‹Р№ Р°РєСЃРµР»РµСЂР°С‚РѕСЂ",
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

	// РџСЂРѕРІРµСЂСЏРµРј РЅР°Р»РёС‡РёРµ РґРІСѓС… РІРєР»Р°РґРѕРє: РљРѕРЅС‚Р°РєС‚С‹ Рё РЎРѕС†РёР°Р»СЊРЅС‹Р№ РіСЂР°С„
	if len(appUI.tabs.Items) != 2 {
		t.Fatalf("РѕР¶РёРґР°Р»РѕСЃСЊ 2 РІРєР»Р°РґРєРё РїСЂРёР»РѕР¶РµРЅРёСЏ, РїРѕР»СѓС‡РµРЅРѕ %d", len(appUI.tabs.Items))
	}

	// РџРµСЂРµРєР»СЋС‡Р°РµРјСЃСЏ РЅР° РІРєР»Р°РґРєСѓ В«РЎРѕС†РёР°Р»СЊРЅС‹Р№ РіСЂР°С„В»
	appUI.tabs.SelectIndex(1)
	appUI.graphView.RefreshData()

	// РџСЂРѕРІРµСЂСЏРµРј Р·Р°С…РІР°С‚ СЃРєСЂРёРЅС€РѕС‚РѕРІ
	outDir := os.Getenv("GOCONTACT_SCREENSHOT_DIR")
	if outDir == "" {
		outDir = filepath.Join("..", "..", "docs", "screenshots", "v0.2.0")
	}

	// 1. РЎРєСЂРёРЅС€РѕС‚ СЂРµР¶РёРјР° Force Graph
	saveCanvasPNG(t, appUI.mainWin, filepath.Join(outDir, "ui_social_graph_force.png"))

	// 2. РЎРєСЂРёРЅС€РѕС‚ СЂРµР¶РёРјР° Dunbar Orbits
	appUI.graphView.modeSelect.SetSelected("РћСЂР±РёС‚С‹ Р”Р°РЅР±Р°СЂР°")
	saveCanvasPNG(t, appUI.mainWin, filepath.Join(outDir, "ui_social_graph_dunbar.png"))

	// 3. РЎРєСЂРёРЅС€РѕС‚ РїРѕРёСЃРєР° С†РµРїРѕС‡РєРё РёРЅС‚СЂРѕ (Dijkstra)
	appUI.graphView.modeSelect.SetSelected("РЎРёР»РѕРІРѕР№ РіСЂР°С„")
	if len(appUI.graphView.sourceSelect.Options) >= 2 {
		appUI.graphView.sourceSelect.SetSelected(appUI.graphView.sourceSelect.Options[0])
		appUI.graphView.targetSelect.SetSelected(appUI.graphView.targetSelect.Options[1])
		appUI.graphView.handleFindWarmIntro()
		saveCanvasPNG(t, appUI.mainWin, filepath.Join(outDir, "ui_warm_intro_path.png"))
	}

	// Р’РѕР·РІСЂР°С‰Р°РµРјСЃСЏ РЅР° РІРєР»Р°РґРєСѓ РљРѕРЅС‚Р°РєС‚С‹ Рё РїСЂРѕРІРµСЂСЏРµРј РєР°СЂС‚РѕС‡РєСѓ СЃРІСЏР·РµР№
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

	// РџСЂРѕРІРµСЂСЏРµРј Р·СѓРј Рё РїР°РЅРѕСЂР°РјРёСЂРѕРІР°РЅРёРµ
	canvas.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, 2)})
	if canvas.zoom <= 1.0 {
		t.Errorf("РѕР¶РёРґР°Р»РѕСЃСЊ СѓРІРµР»РёС‡РµРЅРёРµ Р·СѓРјР°, С‚РµРєСѓС‰РёР№ Р·СѓРј: %f", canvas.zoom)
	}

	canvas.ResetCamera()
	if canvas.zoom != 1.0 || canvas.panX != 0 || canvas.panY != 0 {
		t.Errorf("РѕС€РёР±РєР° СЃР±СЂРѕСЃР° РєР°РјРµСЂС‹: zoom=%f, panX=%f, panY=%f", canvas.zoom, canvas.panX, canvas.panY)
	}

	// РџСЂРѕРІРµСЂСЏРµРј С„РёР»СЊС‚СЂР°С†РёСЋ РѕСЂР±РёС‚
	canvas.SetDunbarFilter(graph.DunbarSupportClique)
	if canvas.dunbarFilter != graph.DunbarSupportClique {
		t.Errorf("РѕР¶РёРґР°Р»СЃСЏ С„РёР»СЊС‚СЂ DunbarSupportClique, РїРѕР»СѓС‡РµРЅРѕ %v", canvas.dunbarFilter)
	}

	// РџСЂРѕРІРµСЂСЏРµРј РєР»РёРє РїРѕ РїСѓСЃС‚РѕРјСѓ С„РѕРЅСѓ
	canvas.Tapped(&fyne.PointEvent{Position: fyne.NewPos(10, 10)})
	if canvas.selectedNodeID != 0 {
		t.Errorf("РѕР¶РёРґР°Р»СЃСЏ СЃР±СЂРѕСЃ РІС‹Р±СЂР°РЅРЅРѕРіРѕ СѓР·Р»Р°, РїРѕР»СѓС‡РµРЅРѕ %d", canvas.selectedNodeID)
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
