// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"context"
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"

	"github.com/trollyHUB/gocontact-studio/internal/config"
	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// presenter.go реализует слой Presenter (MVP), связывающий события UI с бизнес-сервисом ContactService.

// reloadData перезагружает группы, контакты и статистику из сервисного слоя и обновляет виджеты.
func (u *AppUI) reloadData() {
	ctx := context.Background()

	groups, err := u.service.ListGroups(ctx)
	if err == nil {
		u.groups = groups
	}

	contacts, err := u.service.ListContacts(ctx, u.currentFilter)
	if err == nil {
		u.contacts = contacts
	}

	stats, err := u.service.GetStats(ctx)
	if err == nil && u.statsSummaryLabel != nil {
		u.statsSummaryLabel.SetText(fmt.Sprintf(
			"Всего: %d  |  ★ Избранные: %d\nС Email: %d  |  С компанией: %d",
			stats.TotalContacts, stats.FavoriteContacts, stats.WithEmailCount, stats.WithCompanyCount,
		))
	}

	if u.groupList != nil {
		u.groupList.Refresh()
	}
	if u.contactList != nil {
		u.contactList.Refresh()
	}

	// Обновляем карточку выбранного контакта
	if u.selectedContact != nil {
		var found *domain.Contact
		for _, item := range u.contacts {
			if item.ID == u.selectedContact.ID {
				found = item
				break
			}
		}
		u.selectedContact = found
	} else if len(u.contacts) > 0 {
		u.selectedContact = u.contacts[0]
	}
	u.renderDetailPanel(u.selectedContact)
}

// resetFilters сбрасывает все поисковые фильтры, категории и сортировку в исходное состояние.
func (u *AppUI) resetFilters() {
	u.currentFilter = domain.ContactFilter{SortBy: domain.SortByNameAsc}
	u.searchEntry.SetText("")
	u.favOnlyCheck.SetChecked(false)
	if u.citySelect != nil {
		u.citySelect.Selected = "Все города"
		u.citySelect.Refresh()
	}
	u.groupList.UnselectAll()
	u.reloadData()
	u.setStatus("Фильтры сброшены")
}

// toggleTheme переключает тему оформления между тёмной и светлой и сохраняет выбор в config.json.
func (u *AppUI) toggleTheme() {
	if u.themeMode == "dark" {
		u.themeMode = "light"
	} else {
		u.themeMode = "dark"
	}
	u.fyneApp.Settings().SetTheme(NewStudioTheme(u.themeMode))
	u.cfg.Theme = u.themeMode
	_ = config.Save(u.configPath, u.cfg)
	u.setStatus(fmt.Sprintf("Тема оформления переключена: %s", u.themeMode))
}

// setStatus обновляет текст в нижней строке состояния.
func (u *AppUI) setStatus(msg string) {
	if u.statusLabel != nil {
		u.statusLabel.SetText(fmt.Sprintf("Статус: %s  |  Отображено: %d", msg, len(u.contacts)))
	}
}

// NotifyBackup потокобезопасно обновляет строку состояния при завершении фонового автобэкапа.
func (u *AppUI) NotifyBackup(path string, count int, err error) {
	fyne.Do(func() {
		if err != nil {
			u.setStatus("Ошибка автобэкапа: " + err.Error())
			return
		}
		u.setStatus(fmt.Sprintf("Автобэкап выполнен (%d контактов) -> %s", count, filepath.Base(path)))
	})
}
