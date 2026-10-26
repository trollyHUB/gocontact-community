// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	crm "github.com/trollyHUB/gocontact-studio/internal/domain/crm"
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

	btnLogTouch := widget.NewButtonWithIcon("⚡ Записать касание", theme.ContentAddIcon(), func() {
		u.showTouchpointLoggerDialog(c)
	})
	btnLogTouch.Importance = widget.HighImportance

	btnDossier := widget.NewButtonWithIcon("📄 Досье перед встречей", theme.DocumentIcon(), func() {
		u.showPreMeetingDossierDialog(c.ID)
	})

	crmBar := container.NewGridWithColumns(2, btnLogTouch, btnDossier)
	actionBar := container.NewGridWithColumns(2, btnEdit, btnToggleFav, btnVCard, btnDelete)

	var headerBox fyne.CanvasObject
	if u.crmService != nil {
		healthSnap, err := u.crmService.CalculateContactHealth(context.Background(), c.ID)
		if err == nil {
			badge := NewHealthRingBadge(healthSnap.HealthScore, healthSnap.Status)
			headerBox = container.NewHBox(badge, container.NewVBox(titleLbl, subtitleLbl))
		} else {
			headerBox = container.NewVBox(titleLbl, subtitleLbl)
		}
	} else {
		headerBox = container.NewVBox(titleLbl, subtitleLbl)
	}

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

	// Секции Personal CRM (Спринт 3.3)
	timelineSection := u.buildTimelineSection(c)
	promisesSection := u.buildPromisesSection(c)
	lifeEventsSection := u.buildLifeEventsSection(c)

	detailBody := container.NewPadded(container.NewVBox(
		headerBox,
		crmBar,
		actionBar,
		widget.NewSeparator(),
		timelineSection,
		widget.NewSeparator(),
		promisesSection,
		widget.NewSeparator(),
		lifeEventsSection,
		widget.NewSeparator(),
		relationshipsSection,
		widget.NewSeparator(),
		infoGrid,
		widget.NewSeparator(),
		notesCard,
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

func (u *AppUI) buildTimelineSection(c *domain.Contact) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("🕒 Хронология взаимодействий (Timeline)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	body := container.NewVBox()

	if u.crmService != nil {
		touches, err := u.crmService.GetContactTimeline(context.Background(), c.ID, 5)
		if err == nil && len(touches) > 0 {
			for _, t := range touches {
				dateStr := t.OccurredAt.Local().Format("02.01.2006 15:04")
				dur := ""
				if t.DurationMin > 0 {
					dur = fmt.Sprintf(" (%d мин)", t.DurationMin)
				}
				lbl := widget.NewLabel(fmt.Sprintf("%s%s — %s: %s", t.Channel.Label(), dur, dateStr, t.Title))
				lbl.Wrapping = fyne.TextWrapWord
				body.Add(lbl)
				if t.NotesMD != "" {
					sub := widget.NewLabel(fmt.Sprintf("   > %s", t.NotesMD))
					sub.Wrapping = fyne.TextWrapWord
					body.Add(sub)
				}
			}
		} else {
			body.Add(widget.NewLabel("История взаимодействий пока пуста."))
		}
	} else {
		body.Add(widget.NewLabel("Сервис Personal CRM не подключен."))
	}

	btnLog := widget.NewButtonWithIcon("+ Записать касание", theme.ContentAddIcon(), func() {
		u.showTouchpointLoggerDialog(c)
	})
	btnLog.Importance = widget.MediumImportance

	return widget.NewCard("", "", container.NewVBox(
		title,
		body,
		container.NewHBox(btnLog),
	))
}

func (u *AppUI) buildPromisesSection(c *domain.Contact) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("🎯 Взаимные обещания и обязательства", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	body := container.NewVBox()

	if u.crmService != nil {
		promises, err := u.crmService.ListPromisesByContact(context.Background(), c.ID)
		if err == nil && len(promises) > 0 {
			for _, p := range promises {
				who := "Вы пообещали"
				if p.Direction == "they_promised" {
					who = "Вам пообещали"
				}
				statusIcon := "⏳"
				if p.Status == "completed" {
					statusIcon = "✅"
				}
				dueStr := p.DueAt.Format("02.01.2006")
				pText := fmt.Sprintf("%s [%s]: %s (Срок: %s)", statusIcon, who, p.Description, dueStr)

				pID := p.ID
				pStatus := p.Status
				pRow := container.NewHBox(
					newWrappedValueLabel(pText),
				)
				if pStatus == "pending" {
					btnDone := widget.NewButton("Выполнено", func() {
						_ = u.crmService.UpdatePromiseStatus(context.Background(), pID, "completed")
						u.renderDetailPanel(c)
					})
					pRow.Add(btnDone)
				}
				body.Add(pRow)
			}
		} else {
			body.Add(widget.NewLabel("Открытых обещаний нет."))
		}
	}

	return widget.NewCard("", "", container.NewVBox(title, body))
}

func (u *AppUI) buildLifeEventsSection(c *domain.Contact) fyne.CanvasObject {
	title := widget.NewLabelWithStyle("🌱 Памятные даты и важный контекст", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	body := container.NewVBox()

	if u.crmService != nil {
		events, err := u.crmService.ListLifeEvents(context.Background(), c.ID)
		if err == nil && len(events) > 0 {
			for _, ev := range events {
				body.Add(widget.NewLabel(fmt.Sprintf("• %s: %s (%s)", ev.Title, ev.EventDate, ev.Notes)))
			}
		} else {
			body.Add(widget.NewLabel("Памятных дат пока не сохранено."))
		}
	}

	btnAddDate := widget.NewButtonWithIcon("+ Добавить дату", theme.ContentAddIcon(), func() {
		u.showAddLifeEventDialog(c)
	})
	btnAddDate.Importance = widget.LowImportance

	return widget.NewCard("", "", container.NewVBox(
		title,
		body,
		container.NewHBox(btnAddDate),
	))
}

func (u *AppUI) showAddLifeEventDialog(c *domain.Contact) {
	if u.crmService == nil || c == nil {
		return
	}

	titleEntry := widget.NewEntry()
	titleEntry.SetPlaceHolder("Например: День рождения, Годовщина...")

	dateEntry := widget.NewEntry()
	dateEntry.SetText(time.Now().Format("2006-01-02"))

	notesEntry := widget.NewEntry()
	notesEntry.SetPlaceHolder("Предпочтения, хобби, идеи подарков...")

	form := container.NewVBox(
		widget.NewLabel("Событие / Название:"),
		titleEntry,
		widget.NewLabel("Дата (ГГГГ-ММ-ДД):"),
		dateEntry,
		widget.NewLabel("Заметки и предпочтения:"),
		notesEntry,
	)

	d := dialog.NewCustomConfirm(
		"Добавить памятную дату",
		"Сохранить",
		"Отмена",
		form,
		func(confirm bool) {
			if !confirm || strings.TrimSpace(titleEntry.Text) == "" {
				return
			}
			_, _ = u.crmService.CreateLifeEvent(context.Background(), &crm.LifeEvent{
				ContactID:        c.ID,
				EventType:        "custom",
				Title:            strings.TrimSpace(titleEntry.Text),
				EventDate:        strings.TrimSpace(dateEntry.Text),
				IsRecurring:      true,
				RemindDaysBefore: 7,
				Notes:            strings.TrimSpace(notesEntry.Text),
			})
			u.renderDetailPanel(c)
		},
		u.mainWin,
	)
	d.Resize(fyne.NewSize(420, 300))
	d.Show()
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
