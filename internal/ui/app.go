// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/trollyHUB/gocontact-studio/assets"
	"github.com/trollyHUB/gocontact-studio/internal/config"
	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/infrastructure/network"
	"github.com/trollyHUB/gocontact-studio/internal/service"
)

// AppUI управляет главным окном и всеми графическими компонентами Fyne v2 (Неделя 9).
type AppUI struct {
	cfg             *config.Config
	configPath      string
	service         *service.ContactService
	backupScheduler *service.BackupScheduler
	apiServer       *network.LocalAPIServer

	fyneApp   fyne.App
	mainWin   fyne.Window
	themeMode string

	// Состояние фильтров и данных
	currentFilter   domain.ContactFilter
	contacts        []*domain.Contact
	groups          []*domain.Group
	selectedContact *domain.Contact

	// Виджеты интерфейса
	searchEntry       *widget.Entry
	sortSelect        *widget.Select
	citySelect        *widget.Select
	favOnlyCheck      *widget.Check
	groupList         *widget.List
	contactList       *widget.List
	detailContainer   *fyne.Container
	statsSummaryLabel *widget.Label
	statusLabel       *widget.Label
	progressBar       *widget.ProgressBar
}

// NewAppUI создаёт и настраивает графическое приложение.
func NewAppUI(
	cfg *config.Config,
	configPath string,
	svc *service.ContactService,
	backupSched *service.BackupScheduler,
	apiServer *network.LocalAPIServer,
) *AppUI {
	fyneApp := app.NewWithID("kz.gocontact.studio")
	return NewAppUIWithApp(fyneApp, cfg, configPath, svc, backupSched, apiServer)
}

// NewAppUIWithApp инициализирует полный интерфейс на переданном экземпляре fyne.App (удобно для Headless-тестов).
func NewAppUIWithApp(
	fyneApp fyne.App,
	cfg *config.Config,
	configPath string,
	svc *service.ContactService,
	backupSched *service.BackupScheduler,
	apiServer *network.LocalAPIServer,
) *AppUI {
	themeMode := cfg.Theme
	if themeMode != "light" {
		themeMode = "dark"
	}
	fyneApp.Settings().SetTheme(NewStudioTheme(themeMode))

	win := fyneApp.NewWindow(cfg.AppName)
	if len(assets.AppIconPNG) > 0 {
		win.SetIcon(fyne.NewStaticResource("icon.png", assets.AppIconPNG))
	}
	win.Resize(fyne.NewSize(float32(cfg.WindowWidth), float32(cfg.WindowHeight)))
	win.CenterOnScreen()

	ui := &AppUI{
		cfg:             cfg,
		configPath:      configPath,
		service:         svc,
		backupScheduler: backupSched,
		apiServer:       apiServer,
		fyneApp:         fyneApp,
		mainWin:         win,
		themeMode:       themeMode,
		currentFilter: domain.ContactFilter{
			SortBy: domain.SortByNameAsc,
		},
	}

	ui.buildUI()
	ui.reloadData()
	return ui
}

// Run запускает главный цикл событий графического окна.
func (u *AppUI) Run() {
	u.mainWin.ShowAndRun()
}

func (u *AppUI) buildUI() {
	u.detailContainer = container.NewStack()
	u.mainWin.SetMainMenu(u.buildMainMenu())

	topToolbar := u.buildTopBar()
	sidebar := u.buildSidebar()
	centerPanel := u.buildContactListPanel()
	u.renderDetailPanel(nil)

	// Правая часть: Список контактов (Master) + Карточка контакта (Detail)
	masterDetailSplit := container.NewHSplit(centerPanel, u.detailContainer)
	masterDetailSplit.Offset = 0.39

	// Главный трёхколоночный макет: Боковая панель категорий + Master-Detail
	mainSplit := container.NewHSplit(sidebar, masterDetailSplit)
	mainSplit.Offset = 0.20

	statusBar := u.buildStatusBar()

	root := container.NewBorder(
		topToolbar,
		statusBar,
		nil, nil,
		mainSplit,
	)

	u.mainWin.SetContent(root)
}

func (u *AppUI) buildMainMenu() *fyne.MainMenu {
	fileMenu := fyne.NewMenu("Файл",
		fyne.NewMenuItem("+ Новый контакт...", func() {
			u.showContactFormDialog(nil)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Импорт из файла (CSV / JSON / VCF)...", func() {
			u.handleImportFileDialog()
		}),
		fyne.NewMenuItem("Экспорт в JSON...", func() {
			u.handleExportDialog(".json")
		}),
		fyne.NewMenuItem("Экспорт в CSV (Excel)...", func() {
			u.handleExportDialog(".csv")
		}),
		fyne.NewMenuItem("Экспорт в vCard (.vcf)...", func() {
			u.handleExportDialog(".vcf")
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Создать резервную копию сейчас", func() {
			u.backupScheduler.TriggerNow()
		}),
	)

	contactsMenu := fyne.NewMenu("Контакты",
		fyne.NewMenuItem("Показать все контакты", func() {
			u.resetFilters()
		}),
		fyne.NewMenuItem("★ Только избранные", func() {
			u.favOnlyCheck.SetChecked(true)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Создать новую категорию...", func() {
			u.showNewGroupDialog()
		}),
		fyne.NewMenuItem("Переименовать выбранную категорию...", func() {
			u.showEditSelectedGroupDialog()
		}),
		fyne.NewMenuItem("Удалить выбранную категорию", func() {
			u.handleDeleteSelectedGroup()
		}),
	)

	toolsMenu := fyne.NewMenu("Сеть и Горутины",
		fyne.NewMenuItem("Загрузить 5 контактов из REST API (Worker Pool)", func() {
			u.handleFetchFromAPI(5)
		}),
		fyne.NewMenuItem("Многопоточный аудит базы (Goroutines)", func() {
			u.showAuditReportDialog()
		}),
		fyne.NewMenuItem("Тестировать локальный REST API сервер", func() {
			u.showLocalAPITesterDialog()
		}),
	)

	viewMenu := fyne.NewMenu("Вид и Справка",
		fyne.NewMenuItem("Переключить тему (Тёмная / Светлая)", func() {
			u.toggleTheme()
		}),
		fyne.NewMenuItem("О программе (Матрица 15 недель)", func() {
			u.showAboutDialog()
		}),
	)

	return fyne.NewMainMenu(fileMenu, contactsMenu, toolsMenu, viewMenu)
}

func (u *AppUI) buildTopBar() fyne.CanvasObject {
	btnAdd := widget.NewButtonWithIcon("+ Добавить", theme.ContentAddIcon(), func() {
		u.showContactFormDialog(nil)
	})
	btnAdd.Importance = widget.HighImportance

	btnImport := widget.NewButtonWithIcon("Импорт", theme.UploadIcon(), func() {
		u.handleImportFileDialog()
	})

	btnExportJSON := widget.NewButtonWithIcon("Экспорт JSON", theme.DownloadIcon(), func() {
		u.handleExportDialog(".json")
	})

	btnExportCSV := widget.NewButtonWithIcon("Экспорт CSV", theme.DocumentSaveIcon(), func() {
		u.handleExportDialog(".csv")
	})

	btnFetchAPI := widget.NewButtonWithIcon("REST API", theme.SearchReplaceIcon(), func() {
		u.handleFetchFromAPI(5)
	})

	btnAudit := widget.NewButtonWithIcon("Аудит базы", theme.InfoIcon(), func() {
		u.showAuditReportDialog()
	})

	btnTheme := widget.NewButtonWithIcon("Тема", theme.ColorPaletteIcon(), func() {
		u.toggleTheme()
	})

	buttonRow := container.NewHBox(
		btnAdd,
		widget.NewSeparator(),
		btnImport,
		btnExportJSON,
		btnExportCSV,
		widget.NewSeparator(),
		btnFetchAPI,
		btnAudit,
		layout.NewSpacer(),
		btnTheme,
	)

	return container.NewVBox(
		container.NewPadded(buttonRow),
		widget.NewSeparator(),
	)
}
