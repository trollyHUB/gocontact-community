// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// buildContactListPanel создаёт центральную панель (Master): живой поиск, фильтр по городу, сортировка и список контактов.
func (u *AppUI) buildContactListPanel() fyne.CanvasObject {
	u.searchEntry = widget.NewEntry()
	u.searchEntry.SetPlaceHolder("Поиск по ФИО, телефону, email, компании или городу...")
	u.searchEntry.OnChanged = func(text string) {
		u.currentFilter.Query = text
		u.reloadData()
	}

	btnClearSearch := widget.NewButtonWithIcon("", theme.ContentClearIcon(), func() {
		u.searchEntry.SetText("")
	})

	searchRow := container.NewBorder(nil, nil, nil, btnClearSearch, u.searchEntry)

	sortOptions := []string{
		"По имени (А–Я)",
		"По имени (Я–А)",
		"По компании",
		"Сначала новые",
	}
	u.sortSelect = &widget.Select{
		Options:     sortOptions,
		Selected:    sortOptions[0],
		PlaceHolder: "Сортировка",
		OnChanged: func(selected string) {
			switch selected {
			case "По имени (Я–А)":
				u.currentFilter.SortBy = domain.SortByNameDesc
			case "По компании":
				u.currentFilter.SortBy = domain.SortByCompanyAsc
			case "Сначала новые":
				u.currentFilter.SortBy = domain.SortByCreatedDesc
			default:
				u.currentFilter.SortBy = domain.SortByNameAsc
			}
			u.reloadData()
		},
	}
	u.sortSelect.ExtendBaseWidget(u.sortSelect)

	cityOptions := []string{"Все города", "Алматы", "Астана", "Караганда", "Шымкент"}
	u.citySelect = &widget.Select{
		Options:     cityOptions,
		Selected:    cityOptions[0],
		PlaceHolder: "Город",
		OnChanged: func(selected string) {
			if selected == "" || selected == "Все города" {
				u.currentFilter.City = ""
			} else {
				u.currentFilter.City = selected
			}
			u.reloadData()
		},
	}
	u.citySelect.ExtendBaseWidget(u.citySelect)

	u.contactList = widget.NewList(
		func() int {
			return len(u.contacts)
		},
		func() fyne.CanvasObject {
			initialsBadge := widget.NewLabelWithStyle("АС", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
			nameLbl := widget.NewLabelWithStyle("Фамилия Имя", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			nameLbl.Truncation = fyne.TextTruncateEllipsis

			subLbl := widget.NewLabel("Телефон • Компания")
			subLbl.Truncation = fyne.TextTruncateEllipsis

			groupBadge := widget.NewLabel("[Группа]")

			textCol := container.NewVBox(
				container.NewBorder(nil, nil, nil, groupBadge, nameLbl),
				subLbl,
			)
			return container.NewBorder(nil, nil, container.NewPadded(initialsBadge), nil, textCol)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(u.contacts) {
				return
			}
			c := u.contacts[id]
			border := item.(*fyne.Container)

			textCol := border.Objects[0].(*fyne.Container)
			topRow := textCol.Objects[0].(*fyne.Container)
			nameLbl := topRow.Objects[0].(*widget.Label)
			groupBadge := topRow.Objects[1].(*widget.Label)
			subLbl := textCol.Objects[1].(*widget.Label)

			leftPad := border.Objects[1].(*fyne.Container)
			initialsLbl := leftPad.Objects[0].(*widget.Label)
			initialsLbl.SetText(c.Initials())

			starPrefix := ""
			if c.IsFavorite {
				starPrefix = "★ "
			}
			nameLbl.SetText(starPrefix + c.FullName())
			groupBadge.SetText(fmt.Sprintf("[%s]", c.GroupName))

			subInfo := c.Phone
			if c.Company != "" {
				subInfo += "  •  " + c.Company
			} else if c.Address.City != "" {
				subInfo += "  •  г. " + c.Address.City
			}
			subLbl.SetText(subInfo)
		},
	)

	u.contactList.OnSelected = func(id widget.ListItemID) {
		if id >= 0 && id < len(u.contacts) {
			u.selectedContact = u.contacts[id]
			u.renderDetailPanel(u.selectedContact)
		}
	}

	filterGrid := container.NewGridWithColumns(2, u.sortSelect, u.citySelect)
	topControls := container.NewVBox(
		searchRow,
		filterGrid,
		widget.NewSeparator(),
	)

	return container.NewPadded(container.NewBorder(topControls, nil, nil, nil, u.contactList))
}
