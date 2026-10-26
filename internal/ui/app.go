// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/trollyHUB/gocontact-studio/assets"
	"github.com/trollyHUB/gocontact-studio/internal/config"
	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/infrastructure/network"
	"github.com/trollyHUB/gocontact-studio/internal/service"
	crmService "github.com/trollyHUB/gocontact-studio/internal/service/crm"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

// AppUI управляет главным окном и всеми графическими компонентами Fyne v2.
type AppUI struct {
	cfg             *config.Config
	configPath      string
	service         *service.ContactService
	graphService    sna.GraphService
	crmService      crmService.CRMService
	backupScheduler *service.BackupScheduler
	apiServer       *network.LocalAPIServer

	fyneApp   fyne.App
	mainWin   fyne.Window
	themeMode string

	// Навигация и вкладки
	tabs      *container.AppTabs
	graphView *GraphView
	radarView *RadarView

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

// NewAppUI создаёт и настраивает графическое приложение с поддержкой социального графа и Personal CRM.
func NewAppUI(
	cfg *config.Config,
	configPath string,
	svc *service.ContactService,
	graphSvc sna.GraphService,
	crmSvc crmService.CRMService,
	backupSched *service.BackupScheduler,
	apiServer *network.LocalAPIServer,
) *AppUI {
	fyneApp := app.NewWithID("kz.gocontact.studio")
	return NewAppUIWithApp(fyneApp, cfg, configPath, svc, graphSvc, crmSvc, backupSched, apiServer)
}

// NewAppUIWithApp инициализирует полный интерфейс на переданном экземпляре fyne.App (удобно для Headless-тестов).
func NewAppUIWithApp(
	fyneApp fyne.App,
	cfg *config.Config,
	configPath string,
	svc *service.ContactService,
	graphSvc sna.GraphService,
	crmSvc crmService.CRMService,
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
		graphService:    graphSvc,
		crmService:      crmSvc,
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

// SelectContactInMainTab переключает интерфейс на первую вкладку и выбирает указанный контакт.
func (u *AppUI) SelectContactInMainTab(c *domain.Contact) {
	if u.tabs != nil {
		u.tabs.SelectIndex(0)
	}
	u.selectedContact = c
	u.renderDetailPanel(c)
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

	// Вкладка 1: Главный трёхколоночный реестр контактов
	contactsSplit := container.NewHSplit(sidebar, masterDetailSplit)
	contactsSplit.Offset = 0.20

	// Вкладка 2: Интерактивная среда «Социальный граф» (Спринты 2.3 & 2.4)
	u.graphView = NewGraphView(u, u.service, u.graphService)

	// Вкладка 3: Дашборд «CRM Радар» (Спринт 3.3)
	u.radarView = NewRadarView(u, u.crmService)

	u.tabs = container.NewAppTabs(
		container.NewTabItem("📇 Контакты", contactsSplit),
		container.NewTabItem("🕸️ Социальный граф", u.graphView.Container()),
		container.NewTabItem("📡 CRM Радар", u.radarView.Container()),
	)
	u.tabs.OnSelected = func(tab *container.TabItem) {
		if tab.Text == "🕸️ Социальный граф" && u.graphView != nil {
			u.graphView.RefreshData()
		} else if tab.Text == "📡 CRM Радар" && u.radarView != nil {
			u.radarView.RefreshData()
		}
	}

	statusBar := u.buildStatusBar()

	root := container.NewBorder(
		topToolbar,
		statusBar,
		nil, nil,
		u.tabs,
	)

	u.mainWin.SetContent(root)
	u.setupKeyboardShortcuts()
}

func (u *AppUI) setupKeyboardShortcuts() {
	// Ctrl+1: Вкладка "Контакты", Ctrl+2: Вкладка "Социальный граф", Ctrl+3: Вкладка "CRM Радар"
	u.mainWin.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.Key1,
		Modifier: fyne.KeyModifierShortcutDefault,
	}, func(_ fyne.Shortcut) {
		if u.tabs != nil {
			u.tabs.SelectIndex(0)
		}
	})

	u.mainWin.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.Key2,
		Modifier: fyne.KeyModifierShortcutDefault,
	}, func(_ fyne.Shortcut) {
		if u.tabs != nil {
			u.tabs.SelectIndex(1)
		}
	})

	u.mainWin.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.Key3,
		Modifier: fyne.KeyModifierShortcutDefault,
	}, func(_ fyne.Shortcut) {
		if u.tabs != nil {
			u.tabs.SelectIndex(2)
		}
	})

	// Ctrl+L: Быстрое логирование касания
	u.mainWin.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyL,
		Modifier: fyne.KeyModifierShortcutDefault,
	}, func(_ fyne.Shortcut) {
		u.showTouchpointLoggerDialog(u.selectedContact)
	})

	// Ctrl+D: Досье перед встречей
	u.mainWin.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyD,
		Modifier: fyne.KeyModifierShortcutDefault,
	}, func(_ fyne.Shortcut) {
		if u.selectedContact != nil {
			u.showPreMeetingDossierDialog(u.selectedContact.ID)
		} else {
			dialog.ShowInformation("Подсказка", "Выберите контакт в списке слева, чтобы открыть досье (Ctrl+D)", u.mainWin)
		}
	})

	// Ctrl+F: Быстрый фокус на строке поиска
	u.mainWin.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyF,
		Modifier: fyne.KeyModifierShortcutDefault,
	}, func(_ fyne.Shortcut) {
		if u.searchEntry != nil {
			u.mainWin.Canvas().Focus(u.searchEntry)
		}
	})

	// Одиночные клавиши управления графом (D, Space, Escape, Home, 0)
	u.mainWin.Canvas().SetOnTypedKey(func(ev *fyne.KeyEvent) {
		switch ev.Name {
		case fyne.KeyEscape:
			if u.graphView != nil && u.graphView.canvasWidget != nil {
				u.graphView.canvasWidget.ClearPathHighlight()
			}
		case fyne.KeyD:
			if u.tabs != nil && u.tabs.SelectedIndex() == 1 && u.graphView != nil && u.graphView.canvasWidget != nil {
				if u.graphView.canvasWidget.mode == GraphModeForce {
					u.graphView.modeSelect.SetSelected("Орбиты Данбара")
				} else {
					u.graphView.modeSelect.SetSelected("Силовой граф")
				}
			}
		case fyne.KeySpace:
			if u.tabs != nil && u.tabs.SelectedIndex() == 1 && u.graphService != nil {
				u.graphService.ReheatSimulation(0.65)
			}
		case fyne.KeyHome, fyne.Key0:
			if u.tabs != nil && u.tabs.SelectedIndex() == 1 && u.graphView != nil && u.graphView.canvasWidget != nil {
				u.graphView.canvasWidget.ResetCamera()
			}
		}
	})
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

	toolsMenu := fyne.NewMenu("Сеть и Граф",
		fyne.NewMenuItem("Перейти к Социальному графу", func() {
			if u.tabs != nil {
				u.tabs.SelectIndex(1)
			}
		}),
		fyne.NewMenuItem("Пересчитать SNA метрики (Брандес/Лувен)", func() {
			if u.graphService != nil {
				go func() {
					_ = u.graphService.RecalculateMetrics(context.Background())
					fyne.Do(func() {
						if u.graphView != nil {
							u.graphView.RefreshData()
						}
						u.setStatus("Метрики социального графа успешно пересчитаны")
					})
				}()
			}
		}),
		fyne.NewMenuItemSeparator(),
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
		fyne.NewMenuItem("О программе", func() {
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
