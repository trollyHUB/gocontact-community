// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/service"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

// GraphView объединяет интерактивный холст SocialGraphCanvas, панель управления слоями
// и боковой инспектор поиска пути интро (Dijkstra) и характеристик узлов.
type GraphView struct {
	appUI        *AppUI
	contactSvc   *service.ContactService
	graphSvc     sna.GraphService
	canvasWidget *SocialGraphCanvas
	rootSplit    *container.Split

	// Виджеты верхней панели
	modeSelect   *widget.RadioGroup
	implicitChk  *widget.Check
	dunbarSelect *widget.Select
	statsLabel   *widget.Label

	// Правая панель (Инспектор)
	inspectorContainer *container.Scroll
	sourceSelect       *widget.Select
	targetSelect       *widget.Select
	introResultBox     *fyne.Container
	selectedNodeBox    *fyne.Container

	contactsCache []*domain.Contact
}

// NewGraphView создаёт новую рабочую область «Социальный граф».
func NewGraphView(appUI *AppUI, contactSvc *service.ContactService, graphSvc sna.GraphService) *GraphView {
	gv := &GraphView{
		appUI:        appUI,
		contactSvc:   contactSvc,
		graphSvc:     graphSvc,
		canvasWidget: NewSocialGraphCanvas(graphSvc),
	}

	gv.buildUI()
	gv.bindEvents()
	return gv
}

// Container возвращает корневой графический контейнер Fyne v2.
func (gv *GraphView) Container() fyne.CanvasObject {
	return gv.rootSplit
}

func (gv *GraphView) buildUI() {
	topToolbar := gv.buildTopToolbar()
	sidebar := gv.buildRightInspector()

	// Основной сплит: Холст графа (72%) слева + Боковой инспектор (28%) справа
	gv.rootSplit = container.NewHSplit(
		container.NewBorder(topToolbar, nil, nil, nil, gv.canvasWidget),
		sidebar,
	)
	gv.rootSplit.Offset = 0.72
}

func (gv *GraphView) buildTopToolbar() fyne.CanvasObject {
	gv.modeSelect = widget.NewRadioGroup([]string{"Силовой граф", "Орбиты Данбара"}, func(selected string) {
		if selected == "Орбиты Данбара" {
			gv.canvasWidget.SetMode(GraphModeDunbar)
		} else {
			gv.canvasWidget.SetMode(GraphModeForce)
		}
	})
	gv.modeSelect.Horizontal = true
	gv.modeSelect.SetSelected("Силовой граф")

	gv.implicitChk = widget.NewCheck("Неявные связи", func(checked bool) {
		gv.canvasWidget.SetShowImplicit(checked)
	})

	dunbarOptions := []string{"Все орбиты", "Орбита 5 (Ядро)", "Орбита 15 (Симпатия)", "Орбита 50 (Доверие)", "Орбита 150 (Сеть)"}
	gv.dunbarSelect = &widget.Select{
		Options:     dunbarOptions,
		Selected:    dunbarOptions[0],
		PlaceHolder: "Фильтр круга",
		OnChanged: func(val string) {
			switch {
			case strings.Contains(val, "5 "):
				gv.canvasWidget.SetDunbarFilter(graph.DunbarSupportClique)
			case strings.Contains(val, "15 "):
				gv.canvasWidget.SetDunbarFilter(graph.DunbarSympathyGroup)
			case strings.Contains(val, "50 "):
				gv.canvasWidget.SetDunbarFilter(graph.DunbarAffinityGroup)
			case strings.Contains(val, "150 "):
				gv.canvasWidget.SetDunbarFilter(graph.DunbarActiveNetwork)
			default:
				gv.canvasWidget.SetDunbarFilter(graph.DunbarNone)
			}
		},
	}
	gv.dunbarSelect.ExtendBaseWidget(gv.dunbarSelect)

	btnResetCam := widget.NewButtonWithIcon("Сброс камеры", theme.ViewFullScreenIcon(), func() {
		gv.canvasWidget.ResetCamera()
	})

	btnFindIntro := widget.NewButtonWithIcon("Найти интро", theme.SearchIcon(), func() {
		gv.inspectorContainer.ScrollToTop()
	})
	btnFindIntro.Importance = widget.HighImportance

	gv.statsLabel = widget.NewLabelWithStyle("Узлов: 0 | Рёбер: 0", fyne.TextAlignTrailing, fyne.TextStyle{Italic: true})

	toolbarBox := container.NewHBox(
		widget.NewLabelWithStyle("Режим:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		gv.modeSelect,
		widget.NewSeparator(),
		gv.implicitChk,
		widget.NewSeparator(),
		gv.dunbarSelect,
		widget.NewSeparator(),
		btnResetCam,
		btnFindIntro,
		layout.NewSpacer(),
		gv.statsLabel,
	)

	return container.NewVBox(
		container.NewPadded(toolbarBox),
		widget.NewSeparator(),
	)
}

func (gv *GraphView) buildRightInspector() fyne.CanvasObject {
	// Карточка поиска тёплого пути интро
	introCard := gv.buildWarmIntroCard()

	// Карточка выбранного узла
	gv.selectedNodeBox = container.NewVBox()
	gv.renderEmptyNodeInspector()

	inspectorContent := container.NewVBox(
		introCard,
		widget.NewSeparator(),
		gv.selectedNodeBox,
	)

	gv.inspectorContainer = container.NewVScroll(container.NewPadded(inspectorContent))
	return gv.inspectorContainer
}

func (gv *GraphView) buildWarmIntroCard() fyne.CanvasObject {
	titleLbl := widget.NewLabelWithStyle("🤝 Поиск пути интро (Дейкстра)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	gv.sourceSelect = &widget.Select{
		Options:     []string{"Загрузка..."},
		PlaceHolder: "Источник (от кого)",
	}
	gv.sourceSelect.ExtendBaseWidget(gv.sourceSelect)

	gv.targetSelect = &widget.Select{
		Options:     []string{"Загрузка..."},
		PlaceHolder: "Цель (с кем познакомиться)",
	}
	gv.targetSelect.ExtendBaseWidget(gv.targetSelect)

	gv.introResultBox = container.NewVBox()

	btnSearch := widget.NewButtonWithIcon("⚡ Найти тёплый путь (<1 мс)", theme.MediaPlayIcon(), func() {
		gv.handleFindWarmIntro()
	})
	btnSearch.Importance = widget.HighImportance

	btnClear := widget.NewButtonWithIcon("Сбросить путь", theme.CancelIcon(), func() {
		gv.canvasWidget.ClearPathHighlight()
		gv.introResultBox.Objects = nil
		gv.introResultBox.Refresh()
	})

	actions := container.NewGridWithColumns(2, btnSearch, btnClear)

	return widget.NewCard("", "", container.NewVBox(
		titleLbl,
		widget.NewLabel("От кого (Источник):"),
		gv.sourceSelect,
		widget.NewLabel("Кому (Цель знакомства):"),
		gv.targetSelect,
		actions,
		gv.introResultBox,
	))
}

func (gv *GraphView) handleFindWarmIntro() {
	if gv.sourceSelect.Selected == "" || gv.targetSelect.Selected == "" {
		dialog.ShowInformation("Внимание", "Пожалуйста, выберите исходный контакт и контакт цели", gv.appUI.mainWin)
		return
	}

	srcID := extractIDFromOption(gv.sourceSelect.Selected)
	dstID := extractIDFromOption(gv.targetSelect.Selected)
	if srcID <= 0 || dstID <= 0 {
		dialog.ShowError(fmt.Errorf("не удалось определить ID контактов"), gv.appUI.mainWin)
		return
	}
	if srcID == dstID {
		dialog.ShowInformation("Внимание", "Источник и цель должны быть разными людьми", gv.appUI.mainWin)
		return
	}

	path, err := gv.graphSvc.FindWarmIntroPath(context.Background(), srcID, dstID)
	if err != nil {
		gv.canvasWidget.ClearPathHighlight()
		gv.introResultBox.Objects = []fyne.CanvasObject{
			widget.NewLabelWithStyle("❌ Путь знакомств не найден", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			newWrappedValueLabel("Контакты находятся в непересекающихся изолированных кластерах сети."),
		}
		gv.introResultBox.Refresh()
		return
	}

	// Подсвечиваем цепочку на холсте изумрудным цветом
	gv.canvasWidget.HighlightIntroPath(path)

	// Рендерим карточку цепочки
	trustPercent := int(path.CompoundTrust * 100)
	statusBadge := widget.NewLabelWithStyle(
		fmt.Sprintf("✅ Найдена цепочка: %d шага • Доверие: %d%%", len(path.Hops), trustPercent),
		fyne.TextAlignLeading, fyne.TextStyle{Bold: true},
	)

	hopList := container.NewVBox()
	for i, hop := range path.Hops {
		fromName := gv.getContactNameByID(hop.FromContactID)
		toName := gv.getContactNameByID(hop.ToContactID)
		note := hop.ContextNote
		if note == "" {
			note = "Прямой контакт"
		}

		card := widget.NewCard(
			fmt.Sprintf("Шаг %d: %s ➔ %s", i+1, fromName, toName),
			fmt.Sprintf("%s (вес доверия: %d/5)", hop.ViaEdgeType.Label(), hop.EdgeWeight),
			widget.NewLabel(fmt.Sprintf("«%s»", note)),
		)
		hopList.Add(card)
	}

	btnCopyBrief := widget.NewButtonWithIcon("📋 Скопировать бриф интро", theme.ContentCopyIcon(), func() {
		var brief strings.Builder
		brief.WriteString(fmt.Sprintf("Привет! Прошу тёплого интро через нашу сеть (Доверие: %d%%):\n", trustPercent))
		for i, hop := range path.Hops {
			fromName := gv.getContactNameByID(hop.FromContactID)
			toName := gv.getContactNameByID(hop.ToContactID)
			brief.WriteString(fmt.Sprintf("%d. %s -> %s [%s]\n", i+1, fromName, toName, hop.ViaEdgeType.Label()))
		}
		gv.appUI.mainWin.Clipboard().SetContent(brief.String())
		gv.appUI.setStatus("Бриф цепочки знакомства скопирован в буфер обмена")
	})

	gv.introResultBox.Objects = []fyne.CanvasObject{
		widget.NewSeparator(),
		statusBadge,
		hopList,
		btnCopyBrief,
	}
	gv.introResultBox.Refresh()
}

func (gv *GraphView) renderEmptyNodeInspector() {
	gv.selectedNodeBox.Objects = []fyne.CanvasObject{
		widget.NewLabelWithStyle("👤 Инспектор узла сети", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel("Кликните на любой узел на холсте графа для просмотра связей и метрик Данбара"),
	}
	gv.selectedNodeBox.Refresh()
}

func (gv *GraphView) renderSelectedNodeInspector(nodeID int64) {
	node, ok := gv.graphSvc.GetNode(nodeID)
	if !ok || node == nil {
		gv.renderEmptyNodeInspector()
		return
	}

	titleLbl := widget.NewLabelWithStyle(node.FullName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	subLbl := newWrappedValueLabel(fmt.Sprintf("%s | Сообщество #%d", valueOrDash(node.Company), node.CommunityID))

	brokerRole := "Участник сети"
	if node.BetweennessScore > 0.30 {
		brokerRole = "★ Супер-коннектор (Брокер сети)"
	}

	metricsGrid := container.New(layout.NewFormLayout(),
		widget.NewLabelWithStyle("Круг Данбара:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(node.DunbarTier.Label()),

		widget.NewLabelWithStyle("Роль в графе:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(brokerRole),

		widget.NewLabelWithStyle("Центральность:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("%.3f", node.BetweennessScore)),

		widget.NewLabelWithStyle("Степень связности:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("%d связей", node.Degree)),
	)

	// Список прямых связей
	rels, _ := gv.graphSvc.GetRelationshipsByContact(context.Background(), nodeID)
	relsBox := container.NewVBox()
	if len(rels) == 0 {
		relsBox.Add(widget.NewLabel("Прямых подтверждённых связей пока нет"))
	} else {
		for _, r := range rels {
			peerID := r.TargetID
			if peerID == nodeID {
				peerID = r.SourceID
			}
			peerName := gv.getContactNameByID(peerID)
			starRating := strings.Repeat("★", r.Weight) + strings.Repeat("☆", 5-r.Weight)
			relsBox.Add(widget.NewLabel(fmt.Sprintf("• %s — %s (%s)", peerName, r.Type.Label(), starRating)))
		}
	}

	btnAddRel := widget.NewButtonWithIcon("+ Добавить связь", theme.ContentAddIcon(), func() {
		contact, _ := gv.contactSvc.GetContactByID(context.Background(), nodeID)
		if contact != nil {
			gv.appUI.showAddRelationshipDialog(contact)
		}
	})

	btnOpenCard := widget.NewButtonWithIcon("📂 Открыть карточку", theme.FolderOpenIcon(), func() {
		contact, _ := gv.contactSvc.GetContactByID(context.Background(), nodeID)
		if contact != nil {
			gv.appUI.SelectContactInMainTab(contact)
		}
	})

	actions := container.NewGridWithColumns(2, btnAddRel, btnOpenCard)

	gv.selectedNodeBox.Objects = []fyne.CanvasObject{
		titleLbl,
		subLbl,
		metricsGrid,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Прямые связи:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		relsBox,
		actions,
	}
	gv.selectedNodeBox.Refresh()
}

func (gv *GraphView) bindEvents() {
	gv.canvasWidget.OnNodeSelected = func(nodeID int64) {
		gv.renderSelectedNodeInspector(nodeID)
	}

	gv.canvasWidget.OnCanvasTapped = func() {
		gv.renderEmptyNodeInspector()
	}
}

// RefreshData синхронизирует списки контактов для выпадающих меню интро и обновляет статус-бар.
func (gv *GraphView) RefreshData() {
	contacts, err := gv.contactSvc.ListContacts(context.Background(), domain.ContactFilter{})
	if err != nil {
		return
	}
	gv.contactsCache = contacts

	options := make([]string, 0, len(contacts))
	for _, c := range contacts {
		options = append(options, fmt.Sprintf("#%d: %s (%s)", c.ID, c.FullName(), valueOrDash(c.Company)))
	}

	gv.sourceSelect.Options = options
	gv.targetSelect.Options = options
	if len(options) >= 2 {
		if gv.sourceSelect.Selected == "" {
			gv.sourceSelect.SetSelected(options[0])
		}
		if gv.targetSelect.Selected == "" {
			gv.targetSelect.SetSelected(options[1])
		}
	}

	nodesMap, edges, _ := gv.graphSvc.GetTopologySnapshot()
	gv.statsLabel.SetText(fmt.Sprintf("Узлов: %d | Рёбер: %d", len(nodesMap), len(edges)))
	gv.canvasWidget.Refresh()
}

func (gv *GraphView) getContactNameByID(id int64) string {
	for _, c := range gv.contactsCache {
		if c.ID == id {
			return c.FullName()
		}
	}
	if n, ok := gv.graphSvc.GetNode(id); ok {
		return n.FullName
	}
	return fmt.Sprintf("Контакт #%d", id)
}

func extractIDFromOption(opt string) int64 {
	if !strings.HasPrefix(opt, "#") {
		return 0
	}
	parts := strings.SplitN(opt, ":", 2)
	idStr := strings.TrimPrefix(parts[0], "#")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	return id
}
