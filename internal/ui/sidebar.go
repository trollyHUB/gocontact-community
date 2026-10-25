// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// buildSidebar создаёт левую навигационную панель групп/категорий и блок живой статистики БД.
func (u *AppUI) buildSidebar() fyne.CanvasObject {
	header := widget.NewLabelWithStyle("КАТЕГОРИИ И ГРУППЫ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	btnAll := widget.NewButtonWithIcon("Все контакты", theme.FolderOpenIcon(), func() {
		u.currentFilter.GroupID = 0
		u.groupList.UnselectAll()
		u.reloadData()
	})

	u.favOnlyCheck = widget.NewCheck("★ Только избранные", func(checked bool) {
		u.currentFilter.OnlyFavorite = checked
		u.reloadData()
	})

	u.groupList = widget.NewList(
		func() int {
			return len(u.groups)
		},
		func() fyne.CanvasObject {
			dot := canvas.NewCircle(ParseHexColor("#3B82F6"))
			dot.Resize(fyne.NewSize(12, 12))
			dotWrap := container.NewGridWrap(fyne.NewSize(12, 12), dot)
			nameLbl := widget.NewLabel("Категория")
			countLbl := widget.NewLabel("0")
			countLbl.TextStyle = fyne.TextStyle{Bold: true}
			return container.NewHBox(dotWrap, nameLbl, layout.NewSpacer(), countLbl)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(u.groups) {
				return
			}
			g := u.groups[id]
			box := item.(*fyne.Container)
			dotWrap := box.Objects[0].(*fyne.Container)
			dot := dotWrap.Objects[0].(*canvas.Circle)
			dot.FillColor = ParseHexColor(g.ColorHex)
			dot.Refresh()

			nameLbl := box.Objects[1].(*widget.Label)
			nameLbl.SetText(g.Name)

			countLbl := box.Objects[3].(*widget.Label)
			countLbl.SetText(fmt.Sprintf("%d", g.ContactCount))
		},
	)

	u.groupList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(u.groups) {
			u.currentFilter.GroupID = u.groups[id].ID
			u.reloadData()
		}
	}

	btnNewGroup := widget.NewButtonWithIcon("Новая категория", theme.ContentAddIcon(), func() {
		u.showNewGroupDialog()
	})

	u.statsSummaryLabel = widget.NewLabel("Загрузка статистики...")
	u.statsSummaryLabel.Wrapping = fyne.TextWrapWord

	topBlock := container.NewVBox(
		header,
		btnAll,
		u.favOnlyCheck,
		widget.NewSeparator(),
	)

	bottomBlock := container.NewVBox(
		widget.NewSeparator(),
		btnNewGroup,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("СТАТИСТИКА БАЗЫ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		u.statsSummaryLabel,
	)

	return container.NewPadded(container.NewBorder(topBlock, bottomBlock, nil, nil, u.groupList))
}
