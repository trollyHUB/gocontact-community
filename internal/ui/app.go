// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"context"

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
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

// AppUI СѓРїСЂР°РІР»СЏРµС‚ РіР»Р°РІРЅС‹Рј РѕРєРЅРѕРј Рё РІСЃРµРјРё РіСЂР°С„РёС‡РµСЃРєРёРјРё РєРѕРјРїРѕРЅРµРЅС‚Р°РјРё Fyne v2.
type AppUI struct {
	cfg             *config.Config
	configPath      string
	service         *service.ContactService
	graphService    sna.GraphService
	backupScheduler *service.BackupScheduler
	apiServer       *network.LocalAPIServer

	fyneApp   fyne.App
	mainWin   fyne.Window
	themeMode string

	// РќР°РІРёРіР°С†РёСЏ Рё РІРєР»Р°РґРєРё
	tabs      *container.AppTabs
	graphView *GraphView

	// РЎРѕСЃС‚РѕСЏРЅРёРµ С„РёР»СЊС‚СЂРѕРІ Рё РґР°РЅРЅС‹С…
	currentFilter   domain.ContactFilter
	contacts        []*domain.Contact
	groups          []*domain.Group
	selectedContact *domain.Contact

	// Р’РёРґР¶РµС‚С‹ РёРЅС‚РµСЂС„РµР№СЃР°
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

// NewAppUI СЃРѕР·РґР°С‘С‚ Рё РЅР°СЃС‚СЂР°РёРІР°РµС‚ РіСЂР°С„РёС‡РµСЃРєРѕРµ РїСЂРёР»РѕР¶РµРЅРёРµ СЃ РїРѕРґРґРµСЂР¶РєРѕР№ СЃРѕС†РёР°Р»СЊРЅРѕРіРѕ РіСЂР°С„Р°.
func NewAppUI(
	cfg *config.Config,
	configPath string,
	svc *service.ContactService,
	graphSvc sna.GraphService,
	backupSched *service.BackupScheduler,
	apiServer *network.LocalAPIServer,
) *AppUI {
	fyneApp := app.NewWithID("kz.gocontact.studio")
	return NewAppUIWithApp(fyneApp, cfg, configPath, svc, graphSvc, backupSched, apiServer)
}

// NewAppUIWithApp РёРЅРёС†РёР°Р»РёР·РёСЂСѓРµС‚ РїРѕР»РЅС‹Р№ РёРЅС‚РµСЂС„РµР№СЃ РЅР° РїРµСЂРµРґР°РЅРЅРѕРј СЌРєР·РµРјРїР»СЏСЂРµ fyne.App (СѓРґРѕР±РЅРѕ РґР»СЏ Headless-С‚РµСЃС‚РѕРІ).
func NewAppUIWithApp(
	fyneApp fyne.App,
	cfg *config.Config,
	configPath string,
	svc *service.ContactService,
	graphSvc sna.GraphService,
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

// Run Р·Р°РїСѓСЃРєР°РµС‚ РіР»Р°РІРЅС‹Р№ С†РёРєР» СЃРѕР±С‹С‚РёР№ РіСЂР°С„РёС‡РµСЃРєРѕРіРѕ РѕРєРЅР°.
func (u *AppUI) Run() {
	u.mainWin.ShowAndRun()
}

// SelectContactInMainTab РїРµСЂРµРєР»СЋС‡Р°РµС‚ РёРЅС‚РµСЂС„РµР№СЃ РЅР° РїРµСЂРІСѓСЋ РІРєР»Р°РґРєСѓ Рё РІС‹Р±РёСЂР°РµС‚ СѓРєР°Р·Р°РЅРЅС‹Р№ РєРѕРЅС‚Р°РєС‚.
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

	// РџСЂР°РІР°СЏ С‡Р°СЃС‚СЊ: РЎРїРёСЃРѕРє РєРѕРЅС‚Р°РєС‚РѕРІ (Master) + РљР°СЂС‚РѕС‡РєР° РєРѕРЅС‚Р°РєС‚Р° (Detail)
	masterDetailSplit := container.NewHSplit(centerPanel, u.detailContainer)
	masterDetailSplit.Offset = 0.39

	// Р’РєР»Р°РґРєР° 1: Р“Р»Р°РІРЅС‹Р№ С‚СЂС‘С…РєРѕР»РѕРЅРѕС‡РЅС‹Р№ СЂРµРµСЃС‚СЂ РєРѕРЅС‚Р°РєС‚РѕРІ
	contactsSplit := container.NewHSplit(sidebar, masterDetailSplit)
	contactsSplit.Offset = 0.20

	// Р’РєР»Р°РґРєР° 2: РРЅС‚РµСЂР°РєС‚РёРІРЅР°СЏ СЃСЂРµРґР° В«РЎРѕС†РёР°Р»СЊРЅС‹Р№ РіСЂР°С„В» (РЎРїСЂРёРЅС‚С‹ 2.3 & 2.4)
	u.graphView = NewGraphView(u, u.service, u.graphService)

	u.tabs = container.NewAppTabs(
		container.NewTabItem("рџ“‡ РљРѕРЅС‚Р°РєС‚С‹", contactsSplit),
		container.NewTabItem("рџ•ёпёЏ РЎРѕС†РёР°Р»СЊРЅС‹Р№ РіСЂР°С„", u.graphView.Container()),
	)
	u.tabs.OnSelected = func(tab *container.TabItem) {
		if tab.Text == "рџ•ёпёЏ РЎРѕС†РёР°Р»СЊРЅС‹Р№ РіСЂР°С„" && u.graphView != nil {
			u.graphView.RefreshData()
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
}

func (u *AppUI) buildMainMenu() *fyne.MainMenu {
	fileMenu := fyne.NewMenu("Р¤Р°Р№Р»",
		fyne.NewMenuItem("+ РќРѕРІС‹Р№ РєРѕРЅС‚Р°РєС‚...", func() {
			u.showContactFormDialog(nil)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("РРјРїРѕСЂС‚ РёР· С„Р°Р№Р»Р° (CSV / JSON / VCF)...", func() {
			u.handleImportFileDialog()
		}),
		fyne.NewMenuItem("Р­РєСЃРїРѕСЂС‚ РІ JSON...", func() {
			u.handleExportDialog(".json")
		}),
		fyne.NewMenuItem("Р­РєСЃРїРѕСЂС‚ РІ CSV (Excel)...", func() {
			u.handleExportDialog(".csv")
		}),
		fyne.NewMenuItem("Р­РєСЃРїРѕСЂС‚ РІ vCard (.vcf)...", func() {
			u.handleExportDialog(".vcf")
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("РЎРѕР·РґР°С‚СЊ СЂРµР·РµСЂРІРЅСѓСЋ РєРѕРїРёСЋ СЃРµР№С‡Р°СЃ", func() {
			u.backupScheduler.TriggerNow()
		}),
	)

	contactsMenu := fyne.NewMenu("РљРѕРЅС‚Р°РєС‚С‹",
		fyne.NewMenuItem("РџРѕРєР°Р·Р°С‚СЊ РІСЃРµ РєРѕРЅС‚Р°РєС‚С‹", func() {
			u.resetFilters()
		}),
		fyne.NewMenuItem("в… РўРѕР»СЊРєРѕ РёР·Р±СЂР°РЅРЅС‹Рµ", func() {
			u.favOnlyCheck.SetChecked(true)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("РЎРѕР·РґР°С‚СЊ РЅРѕРІСѓСЋ РєР°С‚РµРіРѕСЂРёСЋ...", func() {
			u.showNewGroupDialog()
		}),
		fyne.NewMenuItem("РџРµСЂРµРёРјРµРЅРѕРІР°С‚СЊ РІС‹Р±СЂР°РЅРЅСѓСЋ РєР°С‚РµРіРѕСЂРёСЋ...", func() {
			u.showEditSelectedGroupDialog()
		}),
		fyne.NewMenuItem("РЈРґР°Р»РёС‚СЊ РІС‹Р±СЂР°РЅРЅСѓСЋ РєР°С‚РµРіРѕСЂРёСЋ", func() {
			u.handleDeleteSelectedGroup()
		}),
	)

	toolsMenu := fyne.NewMenu("РЎРµС‚СЊ Рё Р“СЂР°С„",
		fyne.NewMenuItem("РџРµСЂРµР№С‚Рё Рє РЎРѕС†РёР°Р»СЊРЅРѕРјСѓ РіСЂР°С„Сѓ", func() {
			if u.tabs != nil {
				u.tabs.SelectIndex(1)
			}
		}),
		fyne.NewMenuItem("РџРµСЂРµСЃС‡РёС‚Р°С‚СЊ SNA РјРµС‚СЂРёРєРё (Р‘СЂР°РЅРґРµСЃ/Р›СѓРІРµРЅ)", func() {
			if u.graphService != nil {
				go func() {
					_ = u.graphService.RecalculateMetrics(context.Background())
					fyne.Do(func() {
						if u.graphView != nil {
							u.graphView.RefreshData()
						}
						u.setStatus("РњРµС‚СЂРёРєРё СЃРѕС†РёР°Р»СЊРЅРѕРіРѕ РіСЂР°С„Р° СѓСЃРїРµС€РЅРѕ РїРµСЂРµСЃС‡РёС‚Р°РЅС‹")
					})
				}()
			}
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Р—Р°РіСЂСѓР·РёС‚СЊ 5 РєРѕРЅС‚Р°РєС‚РѕРІ РёР· REST API (Worker Pool)", func() {
			u.handleFetchFromAPI(5)
		}),
		fyne.NewMenuItem("РњРЅРѕРіРѕРїРѕС‚РѕС‡РЅС‹Р№ Р°СѓРґРёС‚ Р±Р°Р·С‹ (Goroutines)", func() {
			u.showAuditReportDialog()
		}),
		fyne.NewMenuItem("РўРµСЃС‚РёСЂРѕРІР°С‚СЊ Р»РѕРєР°Р»СЊРЅС‹Р№ REST API СЃРµСЂРІРµСЂ", func() {
			u.showLocalAPITesterDialog()
		}),
	)

	viewMenu := fyne.NewMenu("Р’РёРґ Рё РЎРїСЂР°РІРєР°",
		fyne.NewMenuItem("РџРµСЂРµРєР»СЋС‡РёС‚СЊ С‚РµРјСѓ (РўС‘РјРЅР°СЏ / РЎРІРµС‚Р»Р°СЏ)", func() {
			u.toggleTheme()
		}),
		fyne.NewMenuItem("Рћ РїСЂРѕРіСЂР°РјРјРµ", func() {
			u.showAboutDialog()
		}),
	)

	return fyne.NewMainMenu(fileMenu, contactsMenu, toolsMenu, viewMenu)
}

func (u *AppUI) buildTopBar() fyne.CanvasObject {
	btnAdd := widget.NewButtonWithIcon("+ Р”РѕР±Р°РІРёС‚СЊ", theme.ContentAddIcon(), func() {
		u.showContactFormDialog(nil)
	})
	btnAdd.Importance = widget.HighImportance

	btnImport := widget.NewButtonWithIcon("РРјРїРѕСЂС‚", theme.UploadIcon(), func() {
		u.handleImportFileDialog()
	})

	btnExportJSON := widget.NewButtonWithIcon("Р­РєСЃРїРѕСЂС‚ JSON", theme.DownloadIcon(), func() {
		u.handleExportDialog(".json")
	})

	btnExportCSV := widget.NewButtonWithIcon("Р­РєСЃРїРѕСЂС‚ CSV", theme.DocumentSaveIcon(), func() {
		u.handleExportDialog(".csv")
	})

	btnFetchAPI := widget.NewButtonWithIcon("REST API", theme.SearchReplaceIcon(), func() {
		u.handleFetchFromAPI(5)
	})

	btnAudit := widget.NewButtonWithIcon("РђСѓРґРёС‚ Р±Р°Р·С‹", theme.InfoIcon(), func() {
		u.showAuditReportDialog()
	})

	btnTheme := widget.NewButtonWithIcon("РўРµРјР°", theme.ColorPaletteIcon(), func() {
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
