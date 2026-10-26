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

	// Секция «Связи и социальное окружение» (Спринт 2.4)
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
	title := widget.NewLabelWithStyle("🕸️ Связи и социальное окружение", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

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
				stars := strings.Repeat("★", r.Weight) + strings.Repeat("☆", 5-r.Weight)
				contextNote := ""
				if r.ContextNote != "" {
					contextNote = fmt.Sprintf(" — «%s»", r.ContextNote)
				}
				relsBox.Add(widget.NewLabel(fmt.Sprintf("• %s — %s (%s)%s", peerName, r.Type.Label(), stars, contextNote)))
			}
		} else {
			relsBox.Add(widget.NewLabel("Прямых связей с другими контактами пока не добавлено."))
		}
	} else {
		relsBox.Add(widget.NewLabel("Подсистема графа связей не активирована."))
	}

	btnAddRel := widget.NewButtonWithIcon("+ Добавить связь", theme.ContentAddIcon(), func() {
		u.showAddRelationshipDialog(c)
	})
	btnAddRel.Importance = widget.MediumImportance

	return widget.NewCard("", "", container.NewVBox(
		title,
		relsBox,
		container.NewHBox(btnAddRel),
	))
}

// showAddRelationshipDialog отображает модальное окно создания новой связи.
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
		dialog.ShowInformation("Внимание", "В адресной книге нет других контактов для создания связи.", u.mainWin)
		return
	}

	targetSelect := &widget.Select{
		Options:     targetOptions,
		Selected:    targetOptions[0],
		PlaceHolder: "Выберите целевой контакт",
	}
	targetSelect.ExtendBaseWidget(targetSelect)

	relTypeOptions := []string{
		"Познакомил (introduced_by)",
		"Партнёр (partner)",
		"Коллега (colleague)",
		"Наставник (mentor)",
		"Друг (friend)",
		"Семья (family)",
		"Руководитель (reports_to)",
		"Основатель (founder_of)",
	}
	relTypeSelect := &widget.Select{
		Options:     relTypeOptions,
		Selected:    relTypeOptions[0],
		PlaceHolder: "Тип связи",
	}
	relTypeSelect.ExtendBaseWidget(relTypeSelect)

	weightOptions := []string{
		"5 — Максимальное доверие",
		"4 — Высокое доверие",
		"3 — Обычное знакомство",
		"2 — Слабый контакт",
		"1 — Шапочное знакомство",
	}
	weightSelect := &widget.Select{
		Options:     weightOptions,
		Selected:    weightOptions[0],
		PlaceHolder: "Вес доверия",
	}
	weightSelect.ExtendBaseWidget(weightSelect)

	directedCheck := widget.NewCheck("Направленная связь", nil)
	directedCheck.SetChecked(true)

	contextEntry := widget.NewEntry()
	contextEntry.SetPlaceHolder("Например: Хакатон Алматы 2025, совместный проект")

	form := container.New(layout.NewFormLayout(),
		widget.NewLabel("От кого:"),
		widget.NewLabelWithStyle(source.FullName(), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),

		widget.NewLabel("Кому (Цель):"),
		targetSelect,

		widget.NewLabel("Тип связи:"),
		relTypeSelect,

		widget.NewLabel("Сила доверия:"),
		weightSelect,

		widget.NewLabel("Направленность:"),
		directedCheck,

		widget.NewLabel("Контекст знакомства:"),
		contextEntry,
	)

	dlg := dialog.NewCustomConfirm(
		"Добавить связь в социальный граф",
		"Сохранить связь",
		"Отмена",
		container.NewPadded(form),
		func(save bool) {
			if !save {
				return
			}
			targetID := extractIDFromOption(targetSelect.Selected)
			if targetID <= 0 {
				dialog.ShowError(fmt.Errorf("не выбран целевой контакт"), u.mainWin)
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
			u.setStatus("Связь успешно добавлена в социальный граф")
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
	return fmt.Sprintf("Контакт #%d", id)
}

func valueOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
