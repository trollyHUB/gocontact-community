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
)

// newWrappedValueLabel СЃРѕР·РґР°С‘С‚ С‚РµРєСЃС‚РѕРІСѓСЋ РјРµС‚РєСѓ СЃ Р°РІС‚РѕРјР°С‚РёС‡РµСЃРєРёРј РїРµСЂРµРЅРѕСЃРѕРј РґР»РёРЅРЅС‹С… СЃС‚СЂРѕРє РїРѕ С€РёСЂРёРЅРµ РѕРєРЅР°.
func newWrappedValueLabel(text string) *widget.Label {
	lbl := widget.NewLabel(text)
	lbl.Wrapping = fyne.TextWrapWord
	return lbl
}

// renderDetailPanel РѕС‚РѕР±СЂР°Р¶Р°РµС‚ РїСЂР°РІСѓСЋ РїР°РЅРµР»СЊ (Detail): РїРѕРґСЂРѕР±РЅСѓСЋ РєР°СЂС‚РѕС‡РєСѓ РІС‹Р±СЂР°РЅРЅРѕРіРѕ РєРѕРЅС‚Р°РєС‚Р° СЃ РІРµСЂС‚РёРєР°Р»СЊРЅРѕР№ РїСЂРѕРєСЂСѓС‚РєРѕР№.
func (u *AppUI) renderDetailPanel(c *domain.Contact) {
	if u.detailContainer == nil {
		return
	}
	u.detailContainer.Objects = nil

	if c == nil {
		emptyTitle := widget.NewLabelWithStyle("Р’С‹Р±РµСЂРёС‚Рµ РєРѕРЅС‚Р°РєС‚ РёР· СЃРїРёСЃРєР° СЃР»РµРІР°", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		emptySub := widget.NewLabelWithStyle("РёР»Рё РЅР°Р¶РјРёС‚Рµ В«Р”РѕР±Р°РІРёС‚СЊ РєРѕРЅС‚Р°РєС‚В», С‡С‚РѕР±С‹ СЃРѕР·РґР°С‚СЊ РЅРѕРІСѓСЋ РєР°СЂС‚РѕС‡РєСѓ", fyne.TextAlignCenter, fyne.TextStyle{})
		emptySub.Wrapping = fyne.TextWrapWord

		btnQuickCreate := widget.NewButtonWithIcon("РЎРѕР·РґР°С‚СЊ РєРѕРЅС‚Р°РєС‚", theme.ContentAddIcon(), func() {
			u.showContactFormDialog(nil)
		})
		btnQuickCreate.Importance = widget.HighImportance

		placeholder := container.NewCenter(container.NewVBox(
			emptyTitle,
			emptySub,
			container.NewCenter(btnQuickCreate),
		))
		u.detailContainer.Add(placeholder)
		u.detailContainer.Refresh()
		return
	}

	favIcon := "в† РР·Р±СЂР°РЅРЅРѕРµ"
	if c.IsFavorite {
		favIcon = "в… Р’ РёР·Р±СЂР°РЅРЅРѕРј"
	}

	titleLbl := widget.NewLabelWithStyle(c.FullName(), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	titleLbl.Wrapping = fyne.TextWrapWord

	subtitleLbl := newWrappedValueLabel(fmt.Sprintf("%s  |  РљР°С‚РµРіРѕСЂРёСЏ: %s", c.DisplayTitle(), c.GroupName))

	btnEdit := widget.NewButtonWithIcon("Р РµРґР°РєС‚РёСЂРѕРІР°С‚СЊ", theme.DocumentCreateIcon(), func() {
		u.showContactFormDialog(c)
	})
	btnEdit.Importance = widget.HighImportance

	btnToggleFav := widget.NewButton(favIcon, func() {
		updated, err := u.service.ToggleFavorite(context.Background(), c.ID)
		if err != nil {
			dialog.ShowError(err, u.mainWin)
			return
		}
		u.selectedContact = updated
		u.reloadData()
	})

	btnVCard := widget.NewButtonWithIcon("Р’РёР·РёС‚РєР° .vcf", theme.DownloadIcon(), func() {
		u.handleExportSingleVCard(c)
	})

	btnDelete := widget.NewButtonWithIcon("РЈРґР°Р»РёС‚СЊ РєРѕРЅС‚Р°РєС‚", theme.DeleteIcon(), func() {
		dialog.ShowConfirm(
			"РџРѕРґС‚РІРµСЂР¶РґРµРЅРёРµ СѓРґР°Р»РµРЅРёСЏ",
			fmt.Sprintf("Р’С‹ РґРµР№СЃС‚РІРёС‚РµР»СЊРЅРѕ С…РѕС‚РёС‚Рµ СѓРґР°Р»РёС‚СЊ РєРѕРЅС‚Р°РєС‚ В«%sВ» (%s)?", c.FullName(), c.Phone),
			func(confirm bool) {
				if !confirm {
					return
				}
				if err := u.service.DeleteContact(context.Background(), c.ID); err != nil {
					dialog.ShowError(err, u.mainWin)
					return
				}
				u.selectedContact = nil
				u.reloadData()
				u.setStatus("РљРѕРЅС‚Р°РєС‚ СѓРґР°Р»С‘РЅ")
			},
			u.mainWin,
		)
	})
	btnDelete.Importance = widget.DangerImportance

	actionBar := container.NewGridWithColumns(2, btnEdit, btnToggleFav, btnVCard, btnDelete)

	infoGrid := container.New(layout.NewFormLayout(),
		widget.NewLabelWithStyle("вЂў РћСЃРЅРѕРІРЅРѕР№ С‚РµР».:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(fmt.Sprintf("%s (%s)", c.Phone, c.PhoneType.Label())),

		widget.NewLabelWithStyle("вЂў Р Р°Р±РѕС‡РёР№ С‚РµР».:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(valueOrDash(c.WorkPhone)),

		widget.NewLabelWithStyle("вЂў Р­Р». РїРѕС‡С‚Р°:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(valueOrDash(c.Email)),

		widget.NewLabelWithStyle("вЂў РћСЂРіР°РЅРёР·Р°С†РёСЏ:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(valueOrDash(c.Company)),

		widget.NewLabelWithStyle("вЂў Р”РѕР»Р¶РЅРѕСЃС‚СЊ:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(valueOrDash(c.JobTitle)),

		widget.NewLabelWithStyle("вЂў РџРѕС‡С‚РѕРІС‹Р№ Р°РґСЂРµСЃ:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(c.Address.Format()),

		widget.NewLabelWithStyle("вЂў Р”РµРЅСЊ СЂРѕР¶РґРµРЅРёСЏ:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(valueOrDash(c.Birthday)),

		widget.NewLabelWithStyle("вЂў РўРµРіРё:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(c.FormattedTags()),

		widget.NewLabelWithStyle("вЂў РћР±РЅРѕРІР»РµРЅРѕ:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(c.UpdatedAt.Local().Format("02.01.2006 15:04:05")),
	)

	notesCard := widget.NewCard("Р—Р°РјРµС‚РєРё Рѕ РєРѕРЅС‚Р°РєС‚Рµ", "", newWrappedValueLabel(valueOrDash(c.Notes)))

	// РЎРµРєС†РёСЏ В«РЎРІСЏР·Рё Рё СЃРѕС†РёР°Р»СЊРЅРѕРµ РѕРєСЂСѓР¶РµРЅРёРµВ» (РЎРїСЂРёРЅС‚ 2.4)
	relationshipsSection := u.buildRelationshipsSection(c)

	detailBody := container.NewPadded(container.NewVBox(
		titleLbl,
		subtitleLbl,
		actionBar,
		widget.NewSeparator(),
		infoGrid,
		widget.NewSeparator(),
		notesCard,
		widget.NewSeparator(),
		relationshipsSection,
	))

	scroll := container.NewVScroll(detailBody)
	u.detailContainer.Add(scroll)
	u.detailContainer.Refresh()
}

func (u *AppUI) buildRelationshipsSection(c *domain.Contact) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("рџ•ёпёЏ РЎРІСЏР·Рё Рё СЃРѕС†РёР°Р»СЊРЅРѕРµ РѕРєСЂСѓР¶РµРЅРёРµ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	relsBox := container.NewVBox()

	if u.graphService != nil {
		rels, err := u.graphService.GetRelationshipsByContact(context.Background(), c.ID)
		if err == nil && len(rels) > 0 {
			for _, r := range rels {
				peerID := r.TargetID
				if peerID == c.ID {
					peerID = r.SourceID
				}
				peerName := u.getContactNameByID(peerID)
				stars := strings.Repeat("в…", r.Weight) + strings.Repeat("в†", 5-r.Weight)
				contextNote := ""
				if r.ContextNote != "" {
					contextNote = fmt.Sprintf(" вЂ” В«%sВ»", r.ContextNote)
				}
				relsBox.Add(widget.NewLabel(fmt.Sprintf("вЂў %s вЂ” %s (%s)%s", peerName, r.Type.Label(), stars, contextNote)))
			}
		} else {
			relsBox.Add(widget.NewLabel("РџСЂСЏРјС‹С… СЃРІСЏР·РµР№ СЃ РґСЂСѓРіРёРјРё РєРѕРЅС‚Р°РєС‚Р°РјРё РїРѕРєР° РЅРµ РґРѕР±Р°РІР»РµРЅРѕ."))
		}
	} else {
		relsBox.Add(widget.NewLabel("РџРѕРґСЃРёСЃС‚РµРјР° РіСЂР°С„Р° СЃРІСЏР·РµР№ РЅРµ Р°РєС‚РёРІРёСЂРѕРІР°РЅР°."))
	}

	btnAddRel := widget.NewButtonWithIcon("+ Р”РѕР±Р°РІРёС‚СЊ СЃРІСЏР·СЊ", theme.ContentAddIcon(), func() {
		u.showAddRelationshipDialog(c)
	})
	btnAddRel.Importance = widget.MediumImportance

	return widget.NewCard("", "", container.NewVBox(
		title,
		relsBox,
		container.NewHBox(btnAddRel),
	))
}

// showAddRelationshipDialog РѕС‚РѕР±СЂР°Р¶Р°РµС‚ РјРѕРґР°Р»СЊРЅРѕРµ РѕРєРЅРѕ СЃРѕР·РґР°РЅРёСЏ РЅРѕРІРѕР№ СЃРІСЏР·Рё.
func (u *AppUI) showAddRelationshipDialog(source *domain.Contact) {
	if source == nil {
		return
	}

	allContacts, err := u.service.ListContacts(context.Background(), domain.ContactFilter{})
	if err != nil {
		dialog.ShowError(err, u.mainWin)
		return
	}

	var targetOptions []string
	for _, ct := range allContacts {
		if ct.ID != source.ID {
			targetOptions = append(targetOptions, fmt.Sprintf("#%d: %s (%s)", ct.ID, ct.FullName(), valueOrDash(ct.Company)))
		}
	}

	if len(targetOptions) == 0 {
		dialog.ShowInformation("Р’РЅРёРјР°РЅРёРµ", "Р’ Р°РґСЂРµСЃРЅРѕР№ РєРЅРёРіРµ РЅРµС‚ РґСЂСѓРіРёС… РєРѕРЅС‚Р°РєС‚РѕРІ РґР»СЏ СЃРѕР·РґР°РЅРёСЏ СЃРІСЏР·Рё.", u.mainWin)
		return
	}

	targetSelect := &widget.Select{
		Options:     targetOptions,
		Selected:    targetOptions[0],
		PlaceHolder: "Р’С‹Р±РµСЂРёС‚Рµ С†РµР»РµРІРѕР№ РєРѕРЅС‚Р°РєС‚",
	}
	targetSelect.ExtendBaseWidget(targetSelect)

	relTypeOptions := []string{
		"РџРѕР·РЅР°РєРѕРјРёР» (introduced_by)",
		"РџР°СЂС‚РЅС‘СЂ (partner)",
		"РљРѕР»Р»РµРіР° (colleague)",
		"РќР°СЃС‚Р°РІРЅРёРє (mentor)",
		"Р”СЂСѓРі (friend)",
		"РЎРµРјСЊСЏ (family)",
		"Р СѓРєРѕРІРѕРґРёС‚РµР»СЊ (reports_to)",
		"РћСЃРЅРѕРІР°С‚РµР»СЊ (founder_of)",
	}
	relTypeSelect := &widget.Select{
		Options:     relTypeOptions,
		Selected:    relTypeOptions[0],
		PlaceHolder: "РўРёРї СЃРІСЏР·Рё",
	}
	relTypeSelect.ExtendBaseWidget(relTypeSelect)

	weightOptions := []string{
		"5 вЂ” РњР°РєСЃРёРјР°Р»СЊРЅРѕРµ РґРѕРІРµСЂРёРµ",
		"4 вЂ” Р’С‹СЃРѕРєРѕРµ РґРѕРІРµСЂРёРµ",
		"3 вЂ” РћР±С‹С‡РЅРѕРµ Р·РЅР°РєРѕРјСЃС‚РІРѕ",
		"2 вЂ” РЎР»Р°Р±С‹Р№ РєРѕРЅС‚Р°РєС‚",
		"1 вЂ” РЁР°РїРѕС‡РЅРѕРµ Р·РЅР°РєРѕРјСЃС‚РІРѕ",
	}
	weightSelect := &widget.Select{
		Options:     weightOptions,
		Selected:    weightOptions[0],
		PlaceHolder: "Р’РµСЃ РґРѕРІРµСЂРёСЏ",
	}
	weightSelect.ExtendBaseWidget(weightSelect)

	directedCheck := widget.NewCheck("РќР°РїСЂР°РІР»РµРЅРЅР°СЏ СЃРІСЏР·СЊ", nil)
	directedCheck.SetChecked(true)

	contextEntry := widget.NewEntry()
	contextEntry.SetPlaceHolder("РќР°РїСЂРёРјРµСЂ: РҐР°РєР°С‚РѕРЅ РђР»РјР°С‚С‹ 2025, СЃРѕРІРјРµСЃС‚РЅС‹Р№ РїСЂРѕРµРєС‚")

	form := container.New(layout.NewFormLayout(),
		widget.NewLabel("РћС‚ РєРѕРіРѕ:"),
		widget.NewLabelWithStyle(source.FullName(), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),

		widget.NewLabel("РљРѕРјСѓ (Р¦РµР»СЊ):"),
		targetSelect,

		widget.NewLabel("РўРёРї СЃРІСЏР·Рё:"),
		relTypeSelect,

		widget.NewLabel("РЎРёР»Р° РґРѕРІРµСЂРёСЏ:"),
		weightSelect,

		widget.NewLabel("РќР°РїСЂР°РІР»РµРЅРЅРѕСЃС‚СЊ:"),
		directedCheck,

		widget.NewLabel("РљРѕРЅС‚РµРєСЃС‚ Р·РЅР°РєРѕРјСЃС‚РІР°:"),
		contextEntry,
	)

	dlg := dialog.NewCustomConfirm(
		"Р”РѕР±Р°РІРёС‚СЊ СЃРІСЏР·СЊ РІ СЃРѕС†РёР°Р»СЊРЅС‹Р№ РіСЂР°С„",
		"РЎРѕС…СЂР°РЅРёС‚СЊ СЃРІСЏР·СЊ",
		"РћС‚РјРµРЅР°",
		container.NewPadded(form),
		func(save bool) {
			if !save {
				return
			}
			targetID := extractIDFromOption(targetSelect.Selected)
			if targetID <= 0 {
				dialog.ShowError(fmt.Errorf("РЅРµ РІС‹Р±СЂР°РЅ С†РµР»РµРІРѕР№ РєРѕРЅС‚Р°РєС‚"), u.mainWin)
				return
			}

			relType := graph.RelIntroducedBy
			switch {
			case strings.Contains(relTypeSelect.Selected, "partner"):
				relType = graph.RelPartner
			case strings.Contains(relTypeSelect.Selected, "colleague"):
				relType = graph.RelColleague
			case strings.Contains(relTypeSelect.Selected, "mentor"):
				relType = graph.RelMentor
			case strings.Contains(relTypeSelect.Selected, "friend"):
				relType = graph.RelFriend
			case strings.Contains(relTypeSelect.Selected, "family"):
				relType = graph.RelFamily
			case strings.Contains(relTypeSelect.Selected, "reports_to"):
				relType = graph.RelReportsTo
			case strings.Contains(relTypeSelect.Selected, "founder_of"):
				relType = graph.RelFounderOf
			}

			weight := 5
			if len(weightSelect.Selected) > 0 {
				if w, err := strconv.Atoi(string(weightSelect.Selected[0])); err == nil {
					weight = w
				}
			}

			newRel := &graph.Relationship{
				SourceID:    source.ID,
				TargetID:    targetID,
				Type:        relType,
				Weight:      weight,
				IsDirected:  directedCheck.Checked,
				ContextNote: strings.TrimSpace(contextEntry.Text),
			}

			ctx := context.Background()
			if _, err := u.graphService.CreateRelationship(ctx, newRel); err != nil {
				dialog.ShowError(err, u.mainWin)
				return
			}

			_ = u.graphService.ReloadGraph(ctx, false)
			u.renderDetailPanel(source)
			if u.graphView != nil {
				u.graphView.RefreshData()
			}
			u.setStatus("РЎРІСЏР·СЊ СѓСЃРїРµС€РЅРѕ РґРѕР±Р°РІР»РµРЅР° РІ СЃРѕС†РёР°Р»СЊРЅС‹Р№ РіСЂР°С„")
		},
		u.mainWin,
	)

	dlg.Resize(fyne.NewSize(620, 420))
	dlg.Show()
}

func (u *AppUI) getContactNameByID(id int64) string {
	for _, c := range u.contacts {
		if c.ID == id {
			return c.FullName()
		}
	}
	return fmt.Sprintf("РљРѕРЅС‚Р°РєС‚ #%d", id)
}

func valueOrDash(s string) string {
	if s == "" {
		return "вЂ”"
	}
	return s
}
