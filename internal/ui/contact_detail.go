// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// newWrappedValueLabel создаёт текстовую метку с автоматическим переносом длинных строк по ширине окна.
func newWrappedValueLabel(text string) *widget.Label {
	lbl := widget.NewLabel(text)
	lbl.Wrapping = fyne.TextWrapWord
	return lbl
}

// renderDetailPanel отображает правую панель (Detail): подробную карточку выбранного контакта с вертикальной прокруткой.
func (u *AppUI) renderDetailPanel(c *domain.Contact) {
	if u.detailContainer == nil {
		return
	}
	u.detailContainer.Objects = nil

	if c == nil {
		emptyTitle := widget.NewLabelWithStyle("Выберите контакт из списка слева", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		emptySub := widget.NewLabelWithStyle("или нажмите «Добавить контакт», чтобы создать новую карточку", fyne.TextAlignCenter, fyne.TextStyle{})
		emptySub.Wrapping = fyne.TextWrapWord

		btnQuickCreate := widget.NewButtonWithIcon("Создать контакт", theme.ContentAddIcon(), func() {
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

	favIcon := "☆ Избранное"
	if c.IsFavorite {
		favIcon = "★ В избранном"
	}

	titleLbl := widget.NewLabelWithStyle(c.FullName(), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	titleLbl.Wrapping = fyne.TextWrapWord

	subtitleLbl := newWrappedValueLabel(fmt.Sprintf("%s  |  Категория: %s", c.DisplayTitle(), c.GroupName))

	btnEdit := widget.NewButtonWithIcon("Редактировать", theme.DocumentCreateIcon(), func() {
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

	btnVCard := widget.NewButtonWithIcon("Визитка .vcf", theme.DownloadIcon(), func() {
		u.handleExportSingleVCard(c)
	})

	btnDelete := widget.NewButtonWithIcon("Удалить контакт", theme.DeleteIcon(), func() {
		dialog.ShowConfirm(
			"Подтверждение удаления",
			fmt.Sprintf("Вы действительно хотите удалить контакт «%s» (%s)?", c.FullName(), c.Phone),
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
				u.setStatus("Контакт удалён")
			},
			u.mainWin,
		)
	})
	btnDelete.Importance = widget.DangerImportance

	// Двухколоночная сетка 2x2: минимальная ширина панели уменьшена вдвое, поэтому в оконном режиме ничего не обрезается справа
	actionBar := container.NewGridWithColumns(2, btnEdit, btnToggleFav, btnVCard, btnDelete)

	infoGrid := container.New(layout.NewFormLayout(),
		widget.NewLabelWithStyle("• Основной тел.:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(fmt.Sprintf("%s (%s)", c.Phone, c.PhoneType.Label())),

		widget.NewLabelWithStyle("• Рабочий тел.:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(valueOrDash(c.WorkPhone)),

		widget.NewLabelWithStyle("• Эл. почта:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(valueOrDash(c.Email)),

		widget.NewLabelWithStyle("• Организация:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(valueOrDash(c.Company)),

		widget.NewLabelWithStyle("• Должность:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(valueOrDash(c.JobTitle)),

		widget.NewLabelWithStyle("• Почтовый адрес:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(c.Address.Format()),

		widget.NewLabelWithStyle("• День рождения:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(valueOrDash(c.Birthday)),

		widget.NewLabelWithStyle("• Теги:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(c.FormattedTags()),

		widget.NewLabelWithStyle("• Обновлено:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newWrappedValueLabel(c.UpdatedAt.Local().Format("02.01.2006 15:04:05")),
	)

	notesCard := widget.NewCard("Заметки о контакте", "", newWrappedValueLabel(valueOrDash(c.Notes)))

	detailBody := container.NewPadded(container.NewVBox(
		titleLbl,
		subtitleLbl,
		actionBar,
		widget.NewSeparator(),
		infoGrid,
		widget.NewSeparator(),
		notesCard,
	))

	scroll := container.NewVScroll(detailBody)
	u.detailContainer.Add(scroll)
	u.detailContainer.Refresh()
}

func valueOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
