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
	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/infrastructure/network"
	"github.com/trollyHUB/gocontact-studio/internal/repository/memory"
	"github.com/trollyHUB/gocontact-studio/internal/repository/sqlite"
	"github.com/trollyHUB/gocontact-studio/internal/service"
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

func TestFullAppUI_StartupSearchAndActions(t *testing.T) {
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
	cache := memory.NewContactCache()
	apiClient := network.NewRandomUserClient("", 2*time.Second)
	svc := service.NewContactService(contactRepo, groupRepo, cache, apiClient, 2)

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.json")
	cfg := config.DefaultConfig()
	backupSched := service.NewBackupScheduler(svc, filepath.Join(tmpDir, "backups"), 10)
	localAPI := network.NewLocalAPIServer(0, contactRepo)

	headlessApp := test.NewApp()
	defer headlessApp.Quit()

	appUI := NewAppUIWithApp(headlessApp, cfg, cfgPath, svc, backupSched, localAPI)
	appUI.mainWin.Resize(fyne.NewSize(1240, 780))

	outDir := os.Getenv("GOCONTACT_SCREENSHOT_DIR")
	if outDir != "" {
		saveCanvasPNG(t, appUI.mainWin, filepath.Join(outDir, "ui_main_dark.png"))
	}

	if len(appUI.contacts) != 6 {
		t.Fatalf("ожидалось 6 предзаполненных контактов при старте окна, получено %d", len(appUI.contacts))
	}
	if len(appUI.groups) < 5 {
		t.Fatalf("ожидалось не менее 5 групп, получено %d", len(appUI.groups))
	}

	// Проверяем живой поиск через строку ввода
	test.Type(appUI.searchEntry, "Смирнов")
	if len(appUI.contacts) != 1 {
		t.Errorf("после поиска 'Смирнов' ожидался 1 контакт, получено %d", len(appUI.contacts))
	}

	// Сброс фильтров
	appUI.resetFilters()
	if len(appUI.contacts) != 6 {
		t.Errorf("после сброса фильтров ожидалось 6 контактов, получено %d", len(appUI.contacts))
	}

	// Переключение темы оформления
	appUI.toggleTheme()
	if appUI.themeMode != "light" {
		t.Errorf("ожидалось переключение на тему 'light', получено %q", appUI.themeMode)
	}
	if outDir != "" {
		saveCanvasPNG(t, appUI.mainWin, filepath.Join(outDir, "ui_main_light.png"))
	}

	// Возвращаем тёмную тему
	appUI.toggleTheme()

	// Проверяем диалоговые окна, сортировку и фильтрацию по группам
	appUI.sortSelect.SetSelected("По компании")
	if appUI.currentFilter.SortBy != domain.SortByCompanyAsc {
		t.Errorf("ожидалась сортировка SortByCompanyAsc, получено %v", appUI.currentFilter.SortBy)
	}

	appUI.favOnlyCheck.SetChecked(true)
	if !appUI.currentFilter.OnlyFavorite || len(appUI.contacts) != 4 {
		t.Errorf("ожидалось 4 избранных контакта, получено %d", len(appUI.contacts))
	}
	appUI.resetFilters()

	report, err := svc.AuditContactsConcurrent(ctx)
	if err != nil {
		t.Fatalf("ошибка аудита: %v", err)
	}
	appUI.renderAuditModal(report)
	if outDir != "" {
		saveCanvasPNG(t, appUI.mainWin, filepath.Join(outDir, "ui_audit_modal.png"))
	}

	// Проверяем открытие прокручиваемой двухколоночной формы редактирования
	if len(appUI.contacts) > 0 {
		appUI.showContactFormDialog(appUI.contacts[0])
		if outDir != "" {
			saveCanvasPNG(t, appUI.mainWin, filepath.Join(outDir, "ui_contact_form.png"))
		}
	}
	appUI.showContactFormDialog(nil)
	appUI.showNewGroupDialog()
	appUI.showAboutDialog()
}

func saveCanvasPNG(t *testing.T, win fyne.Window, path string) {
	t.Helper()
	img := win.Canvas().Capture()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("ошибка создания папки для скриншота: %v", err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("ошибка создания файла скриншота: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("ошибка сохранения PNG: %v", err)
	}
}
