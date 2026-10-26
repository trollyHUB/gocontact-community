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

// GraphView РѕР±СЉРµРґРёРЅСЏРµС‚ РёРЅС‚РµСЂР°РєС‚РёРІРЅС‹Р№ С…РѕР»СЃС‚ SocialGraphCanvas, РїР°РЅРµР»СЊ СѓРїСЂР°РІР»РµРЅРёСЏ СЃР»РѕСЏРјРё
// Рё Р±РѕРєРѕРІРѕР№ РёРЅСЃРїРµРєС‚РѕСЂ РїРѕРёСЃРєР° РїСѓС‚Рё РёРЅС‚СЂРѕ (Dijkstra) Рё С…Р°СЂР°РєС‚РµСЂРёСЃС‚РёРє СѓР·Р»РѕРІ.
type GraphView struct {
	appUI        *AppUI
	contactSvc   *service.ContactService
	graphSvc     sna.GraphService
	canvasWidget *SocialGraphCanvas
	rootSplit    *container.Split

	// Р’РёРґР¶РµС‚С‹ РІРµСЂС…РЅРµР№ РїР°РЅРµР»Рё
	modeSelect   *widget.RadioGroup
	implicitChk  *widget.Check
	dunbarSelect *widget.Select
	statsLabel   *widget.Label

	// РџСЂР°РІР°СЏ РїР°РЅРµР»СЊ (РРЅСЃРїРµРєС‚РѕСЂ)
	inspectorContainer *container.Scroll
	sourceSelect       *widget.Select
	targetSelect       *widget.Select
	introResultBox     *fyne.Container
	selectedNodeBox    *fyne.Container

	contactsCache []*domain.Contact
}

// NewGraphView СЃРѕР·РґР°С‘С‚ РЅРѕРІСѓСЋ СЂР°Р±РѕС‡СѓСЋ РѕР±Р»Р°СЃС‚СЊ В«РЎРѕС†РёР°Р»СЊРЅС‹Р№ РіСЂР°С„В».
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

// Container РІРѕР·РІСЂР°С‰Р°РµС‚ РєРѕСЂРЅРµРІРѕР№ РіСЂР°С„РёС‡РµСЃРєРёР№ РєРѕРЅС‚РµР№РЅРµСЂ Fyne v2.
func (gv *GraphView) Container() fyne.CanvasObject {
	return gv.rootSplit
}

func (gv *GraphView) buildUI() {
	topToolbar := gv.buildTopToolbar()
	sidebar := gv.buildRightInspector()

	// РћСЃРЅРѕРІРЅРѕР№ СЃРїР»РёС‚: РҐРѕР»СЃС‚ РіСЂР°С„Р° (72%) СЃР»РµРІР° + Р‘РѕРєРѕРІРѕР№ РёРЅСЃРїРµРєС‚РѕСЂ (28%) СЃРїСЂР°РІР°
	gv.rootSplit = container.NewHSplit(
		container.NewBorder(topToolbar, nil, nil, nil, gv.canvasWidget),
		sidebar,
	)
	gv.rootSplit.Offset = 0.72
}

func (gv *GraphView) buildTopToolbar() fyne.CanvasObject {
	gv.modeSelect = widget.NewRadioGroup([]string{"РЎРёР»РѕРІРѕР№ РіСЂР°С„", "РћСЂР±РёС‚С‹ Р”Р°РЅР±Р°СЂР°"}, func(selected string) {
		if selected == "РћСЂР±РёС‚С‹ Р”Р°РЅР±Р°СЂР°" {
			gv.canvasWidget.SetMode(GraphModeDunbar)
		} else {
			gv.canvasWidget.SetMode(GraphModeForce)
		}
	})
	gv.modeSelect.Horizontal = true
	gv.modeSelect.SetSelected("РЎРёР»РѕРІРѕР№ РіСЂР°С„")

	gv.implicitChk = widget.NewCheck("РќРµСЏРІРЅС‹Рµ СЃРІСЏР·Рё", func(checked bool) {
		gv.canvasWidget.SetShowImplicit(checked)
	})

	dunbarOptions := []string{"Р’СЃРµ РѕСЂР±РёС‚С‹", "РћСЂР±РёС‚Р° 5 (РЇРґСЂРѕ)", "РћСЂР±РёС‚Р° 15 (РЎРёРјРїР°С‚РёСЏ)", "РћСЂР±РёС‚Р° 50 (Р”РѕРІРµСЂРёРµ)", "РћСЂР±РёС‚Р° 150 (РЎРµС‚СЊ)"}
	gv.dunbarSelect = &widget.Select{
		Options:     dunbarOptions,
		Selected:    dunbarOptions[0],
		PlaceHolder: "Р¤РёР»СЊС‚СЂ РєСЂСѓРіР°",
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

	btnResetCam := widget.NewButtonWithIcon("РЎР±СЂРѕСЃ РєР°РјРµСЂС‹", theme.ViewFullScreenIcon(), func() {
		gv.canvasWidget.ResetCamera()
	})

	btnFindIntro := widget.NewButtonWithIcon("РќР°Р№С‚Рё РёРЅС‚СЂРѕ", theme.SearchIcon(), func() {
		gv.inspectorContainer.ScrollToTop()
	})
	btnFindIntro.Importance = widget.HighImportance

	gv.statsLabel = widget.NewLabelWithStyle("РЈР·Р»РѕРІ: 0 | Р С‘Р±РµСЂ: 0", fyne.TextAlignTrailing, fyne.TextStyle{Italic: true})

	toolbarBox := container.NewHBox(
		widget.NewLabelWithStyle("Р РµР¶РёРј:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
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
	// РљР°СЂС‚РѕС‡РєР° РїРѕРёСЃРєР° С‚С‘РїР»РѕРіРѕ РїСѓС‚Рё РёРЅС‚СЂРѕ
	introCard := gv.buildWarmIntroCard()

	// РљР°СЂС‚РѕС‡РєР° РІС‹Р±СЂР°РЅРЅРѕРіРѕ СѓР·Р»Р°
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
	titleLbl := widget.NewLabelWithStyle("рџ¤ќ РџРѕРёСЃРє РїСѓС‚Рё РёРЅС‚СЂРѕ (Р”РµР№РєСЃС‚СЂР°)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	gv.sourceSelect = &widget.Select{
		Options:     []string{"Р—Р°РіСЂСѓР·РєР°..."},
		PlaceHolder: "РСЃС‚РѕС‡РЅРёРє (РѕС‚ РєРѕРіРѕ)",
	}
	gv.sourceSelect.ExtendBaseWidget(gv.sourceSelect)

	gv.targetSelect = &widget.Select{
		Options:     []string{"Р—Р°РіСЂСѓР·РєР°..."},
		PlaceHolder: "Р¦РµР»СЊ (СЃ РєРµРј РїРѕР·РЅР°РєРѕРјРёС‚СЊСЃСЏ)",
	}
	gv.targetSelect.ExtendBaseWidget(gv.targetSelect)

	gv.introResultBox = container.NewVBox()

	btnSearch := widget.NewButtonWithIcon("вљЎ РќР°Р№С‚Рё С‚С‘РїР»С‹Р№ РїСѓС‚СЊ (<1 РјСЃ)", theme.MediaPlayIcon(), func() {
		gv.handleFindWarmIntro()
	})
	btnSearch.Importance = widget.HighImportance

	btnClear := widget.NewButtonWithIcon("РЎР±СЂРѕСЃРёС‚СЊ РїСѓС‚СЊ", theme.CancelIcon(), func() {
		gv.canvasWidget.ClearPathHighlight()
		gv.introResultBox.Objects = nil
		gv.introResultBox.Refresh()
	})

	actions := container.NewGridWithColumns(2, btnSearch, btnClear)

	return widget.NewCard("", "", container.NewVBox(
		titleLbl,
		widget.NewLabel("РћС‚ РєРѕРіРѕ (РСЃС‚РѕС‡РЅРёРє):"),
		gv.sourceSelect,
		widget.NewLabel("РљРѕРјСѓ (Р¦РµР»СЊ Р·РЅР°РєРѕРјСЃС‚РІР°):"),
		gv.targetSelect,
		actions,
		gv.introResultBox,
	))
}

func (gv *GraphView) handleFindWarmIntro() {
	if gv.sourceSelect.Selected == "" || gv.targetSelect.Selected == "" {
		dialog.ShowInformation("Р’РЅРёРјР°РЅРёРµ", "РџРѕР¶Р°Р»СѓР№СЃС‚Р°, РІС‹Р±РµСЂРёС‚Рµ РёСЃС…РѕРґРЅС‹Р№ РєРѕРЅС‚Р°РєС‚ Рё РєРѕРЅС‚Р°РєС‚ С†РµР»Рё", gv.appUI.mainWin)
		return
	}

	srcID := extractIDFromOption(gv.sourceSelect.Selected)
	dstID := extractIDFromOption(gv.targetSelect.Selected)
	if srcID <= 0 || dstID <= 0 {
		dialog.ShowError(fmt.Errorf("РЅРµ СѓРґР°Р»РѕСЃСЊ РѕРїСЂРµРґРµР»РёС‚СЊ ID РєРѕРЅС‚Р°РєС‚РѕРІ"), gv.appUI.mainWin)
		return
	}
	if srcID == dstID {
		dialog.ShowInformation("Р’РЅРёРјР°РЅРёРµ", "РСЃС‚РѕС‡РЅРёРє Рё С†РµР»СЊ РґРѕР»Р¶РЅС‹ Р±С‹С‚СЊ СЂР°Р·РЅС‹РјРё Р»СЋРґСЊРјРё", gv.appUI.mainWin)
		return
	}

	path, err := gv.graphSvc.FindWarmIntroPath(context.Background(), srcID, dstID)
	if err != nil {
		gv.canvasWidget.ClearPathHighlight()
		gv.introResultBox.Objects = []fyne.CanvasObject{
			widget.NewLabelWithStyle("вќЊ РџСѓС‚СЊ Р·РЅР°РєРѕРјСЃС‚РІ РЅРµ РЅР°Р№РґРµРЅ", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			newWrappedValueLabel("РљРѕРЅС‚Р°РєС‚С‹ РЅР°С…РѕРґСЏС‚СЃСЏ РІ РЅРµРїРµСЂРµСЃРµРєР°СЋС‰РёС…СЃСЏ РёР·РѕР»РёСЂРѕРІР°РЅРЅС‹С… РєР»Р°СЃС‚РµСЂР°С… СЃРµС‚Рё."),
		}
		gv.introResultBox.Refresh()
		return
	}

	// РџРѕРґСЃРІРµС‡РёРІР°РµРј С†РµРїРѕС‡РєСѓ РЅР° С…РѕР»СЃС‚Рµ РёР·СѓРјСЂСѓРґРЅС‹Рј С†РІРµС‚РѕРј
	gv.canvasWidget.HighlightIntroPath(path)

	// Р РµРЅРґРµСЂРёРј РєР°СЂС‚РѕС‡РєСѓ С†РµРїРѕС‡РєРё
	trustPercent := int(path.CompoundTrust * 100)
	statusBadge := widget.NewLabelWithStyle(
		fmt.Sprintf("вњ… РќР°Р№РґРµРЅР° С†РµРїРѕС‡РєР°: %d С€Р°РіР° вЂў Р”РѕРІРµСЂРёРµ: %d%%", len(path.Hops), trustPercent),
		fyne.TextAlignLeading, fyne.TextStyle{Bold: true},
	)

	hopList := container.NewVBox()
	for i, hop := range path.Hops {
		fromName := gv.getContactNameByID(hop.FromContactID)
		toName := gv.getContactNameByID(hop.ToContactID)
		note := hop.ContextNote
		if note == "" {
			note = "РџСЂСЏРјРѕР№ РєРѕРЅС‚Р°РєС‚"
		}

		card := widget.NewCard(
			fmt.Sprintf("РЁР°Рі %d: %s вћ” %s", i+1, fromName, toName),
			fmt.Sprintf("%s (РІРµСЃ РґРѕРІРµСЂРёСЏ: %d/5)", hop.ViaEdgeType.Label(), hop.EdgeWeight),
			widget.NewLabel(fmt.Sprintf("В«%sВ»", note)),
		)
		hopList.Add(card)
	}

	btnCopyBrief := widget.NewButtonWithIcon("рџ“‹ РЎРєРѕРїРёСЂРѕРІР°С‚СЊ Р±СЂРёС„ РёРЅС‚СЂРѕ", theme.ContentCopyIcon(), func() {
		var brief strings.Builder
		brief.WriteString(fmt.Sprintf("РџСЂРёРІРµС‚! РџСЂРѕС€Сѓ С‚С‘РїР»РѕРіРѕ РёРЅС‚СЂРѕ С‡РµСЂРµР· РЅР°С€Сѓ СЃРµС‚СЊ (Р”РѕРІРµСЂРёРµ: %d%%):\n", trustPercent))
		for i, hop := range path.Hops {
			fromName := gv.getContactNameByID(hop.FromContactID)
			toName := gv.getContactNameByID(hop.ToContactID)
			brief.WriteString(fmt.Sprintf("%d. %s -> %s [%s]\n", i+1, fromName, toName, hop.ViaEdgeType.Label()))
		}
		gv.appUI.mainWin.Clipboard().SetContent(brief.String())
		gv.appUI.setStatus("Р‘СЂРёС„ С†РµРїРѕС‡РєРё Р·РЅР°РєРѕРјСЃС‚РІР° СЃРєРѕРїРёСЂРѕРІР°РЅ РІ Р±СѓС„РµСЂ РѕР±РјРµРЅР°")
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
		widget.NewLabelWithStyle("рџ‘¤ РРЅСЃРїРµРєС‚РѕСЂ СѓР·Р»Р° СЃРµС‚Рё", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel("РљР»РёРєРЅРёС‚Рµ РЅР° Р»СЋР±РѕР№ СѓР·РµР» РЅР° С…РѕР»СЃС‚Рµ РіСЂР°С„Р° РґР»СЏ РїСЂРѕСЃРјРѕС‚СЂР° СЃРІСЏР·РµР№ Рё РјРµС‚СЂРёРє Р”Р°РЅР±Р°СЂР°"),
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
	subLbl := newWrappedValueLabel(fmt.Sprintf("%s | РЎРѕРѕР±С‰РµСЃС‚РІРѕ #%d", valueOrDash(node.Company), node.CommunityID))

	brokerRole := "РЈС‡Р°СЃС‚РЅРёРє СЃРµС‚Рё"
	if node.BetweennessScore > 0.30 {
		brokerRole = "в… РЎСѓРїРµСЂ-РєРѕРЅРЅРµРєС‚РѕСЂ (Р‘СЂРѕРєРµСЂ СЃРµС‚Рё)"
	}

	metricsGrid := container.New(layout.NewFormLayout(),
		widget.NewLabelWithStyle("РљСЂСѓРі Р”Р°РЅР±Р°СЂР°:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(node.DunbarTier.Label()),

		widget.NewLabelWithStyle("Р РѕР»СЊ РІ РіСЂР°С„Рµ:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(brokerRole),

		widget.NewLabelWithStyle("Р¦РµРЅС‚СЂР°Р»СЊРЅРѕСЃС‚СЊ:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("%.3f", node.BetweennessScore)),

		widget.NewLabelWithStyle("РЎС‚РµРїРµРЅСЊ СЃРІСЏР·РЅРѕСЃС‚Рё:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("%d СЃРІСЏР·РµР№", node.Degree)),
	)

	// РЎРїРёСЃРѕРє РїСЂСЏРјС‹С… СЃРІСЏР·РµР№
	rels, _ := gv.graphSvc.GetRelationshipsByContact(context.Background(), nodeID)
	relsBox := container.NewVBox()
	if len(rels) == 0 {
		relsBox.Add(widget.NewLabel("РџСЂСЏРјС‹С… РїРѕРґС‚РІРµСЂР¶РґС‘РЅРЅС‹С… СЃРІСЏР·РµР№ РїРѕРєР° РЅРµС‚"))
	} else {
		for _, r := range rels {
			peerID := r.TargetID
			if peerID == nodeID {
				peerID = r.SourceID
			}
			peerName := gv.getContactNameByID(peerID)
			starRating := strings.Repeat("в…", r.Weight) + strings.Repeat("в†", 5-r.Weight)
			relsBox.Add(widget.NewLabel(fmt.Sprintf("вЂў %s вЂ” %s (%s)", peerName, r.Type.Label(), starRating)))
		}
	}

	btnAddRel := widget.NewButtonWithIcon("+ Р”РѕР±Р°РІРёС‚СЊ СЃРІСЏР·СЊ", theme.ContentAddIcon(), func() {
		contact, _ := gv.contactSvc.GetContactByID(context.Background(), nodeID)
		if contact != nil {
			gv.appUI.showAddRelationshipDialog(contact)
		}
	})

	btnOpenCard := widget.NewButtonWithIcon("рџ“‚ РћС‚РєСЂС‹С‚СЊ РєР°СЂС‚РѕС‡РєСѓ", theme.FolderOpenIcon(), func() {
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
		widget.NewLabelWithStyle("РџСЂСЏРјС‹Рµ СЃРІСЏР·Рё:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
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

// RefreshData СЃРёРЅС…СЂРѕРЅРёР·РёСЂСѓРµС‚ СЃРїРёСЃРєРё РєРѕРЅС‚Р°РєС‚РѕРІ РґР»СЏ РІС‹РїР°РґР°СЋС‰РёС… РјРµРЅСЋ РёРЅС‚СЂРѕ Рё РѕР±РЅРѕРІР»СЏРµС‚ СЃС‚Р°С‚СѓСЃ-Р±Р°СЂ.
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
	gv.statsLabel.SetText(fmt.Sprintf("РЈР·Р»РѕРІ: %d | Р С‘Р±РµСЂ: %d", len(nodesMap), len(edges)))
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
	return fmt.Sprintf("РљРѕРЅС‚Р°РєС‚ #%d", id)
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
