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
	"fyne.io/fyne/v2/widget"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/domain/crm"
)

// showTouchpointLoggerDialog открывает модальное окно быстрого логирования касания.
func (u *AppUI) showTouchpointLoggerDialog(preselected *domain.Contact) {
	if u.crmService == nil {
		dialog.ShowInformation("Внимание", "Сервис Personal CRM не активирован", u.mainWin)
		return
	}

	allContacts, err := u.service.ListContacts(context.Background(), domain.ContactFilter{})
	if err != nil || len(allContacts) == 0 {
		dialog.ShowInformation("Внимание", "В адресной книге нет контактов для записи взаимодействия.", u.mainWin)
		return
	}

	var contactOptions []string
	selectedIndex := 0
	for i, c := range allContacts {
		label := fmt.Sprintf("#%d: %s (%s)", c.ID, c.FullName(), valueOrDash(c.Company))
		contactOptions = append(contactOptions, label)
		if preselected != nil && c.ID == preselected.ID {
			selectedIndex = i
		}
	}

	contactSelect := &widget.Select{
		Options:  contactOptions,
		Selected: contactOptions[selectedIndex],
	}
	contactSelect.ExtendBaseWidget(contactSelect)

	// Селектор канала
	channelOptions := []string{
		"🤝 Личная встреча",
		"📞 Звонок",
		"💬 Сообщение",
		"✉️ Эл. почта",
		"🎁 Подарок / Интро",
	}
	channelRadio := widget.NewRadioGroup(channelOptions, nil)
	channelRadio.Horizontal = true
	channelRadio.SetSelected(channelOptions[0])

	titleEntry := widget.NewEntry()
	titleEntry.SetPlaceHolder("Например: Обед в кафе, Обсуждение планов на Q1...")

	durationEntry := widget.NewEntry()
	durationEntry.SetText("30")

	notesEntry := widget.NewMultiLineEntry()
	notesEntry.SetPlaceHolder("Краткие итоги общения, договоренности, контекст...")
	notesEntry.Wrapping = fyne.TextWrapWord
	notesEntry.SetMinRowsVisible(4)

	// Секция обещания
	promiseDescEntry := widget.NewEntry()
	promiseDescEntry.SetPlaceHolder("Что пообещали сделать...")

	promiseDirection := widget.NewRadioGroup([]string{"Я пообещал", "Мне пообещали"}, nil)
	promiseDirection.Horizontal = true
	promiseDirection.SetSelected("Я пообещал")

	promiseDueEntry := widget.NewEntry()
	promiseDueEntry.SetText(time.Now().Add(7 * 24 * time.Hour).Format("02.01.2006"))

	promiseBox := container.NewVBox(
		widget.NewLabelWithStyle("Обязательство (Action Promise):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		promiseDirection,
		promiseDescEntry,
		container.NewHBox(widget.NewLabel("Срок исполнения:"), promiseDueEntry),
	)
	promiseBox.Hide()

	promiseCheck := widget.NewCheck("Зафиксировать обещание по итогам встречи", func(checked bool) {
		if checked {
			promiseBox.Show()
		} else {
			promiseBox.Hide()
		}
	})

	formContent := container.NewVBox(
		widget.NewLabelWithStyle("Контакт (участник):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		contactSelect,
		widget.NewLabelWithStyle("Канал коммуникации:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		channelRadio,
		widget.NewLabelWithStyle("Заголовок встречи / звонка:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		titleEntry,
		container.NewHBox(
			widget.NewLabelWithStyle("Длительность (мин):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			durationEntry,
		),
		widget.NewLabelWithStyle("Заметки (поддерживается Markdown):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		notesEntry,
		widget.NewSeparator(),
		promiseCheck,
		promiseBox,
	)

	d := dialog.NewCustomConfirm(
		"⚡ Записать взаимодействие (1-Click Logger)",
		"💾 Сохранить",
		"Отмена",
		container.NewPadded(container.NewVScroll(formContent)),
		func(confirm bool) {
			if !confirm {
				return
			}

			targetID := extractIDFromOption(contactSelect.Selected)
			if targetID <= 0 {
				dialog.ShowError(fmt.Errorf("не удалось определить участника"), u.mainWin)
				return
			}

			var ch crm.ChannelType
			switch channelRadio.Selected {
			case "🤝 Личная встреча":
				ch = crm.ChannelMeeting
			case "📞 Звонок":
				ch = crm.ChannelCall
			case "💬 Сообщение":
				ch = crm.ChannelMessage
			case "✉️ Эл. почта":
				ch = crm.ChannelEmail
			case "🎁 Подарок / Интро":
				ch = crm.ChannelGiftIntro
			default:
				ch = crm.ChannelMeeting
			}

			dur, _ := strconv.Atoi(durationEntry.Text)
			if dur < 0 {
				dur = 0
			}

			title := strings.TrimSpace(titleEntry.Text)
			if title == "" {
				title = ch.Label()
			}

			touch := &crm.Interaction{
				Channel:        ch,
				Title:          title,
				NotesMD:        strings.TrimSpace(notesEntry.Text),
				DurationMin:    dur,
				OccurredAt:     time.Now().UTC(),
				ParticipantIDs: []int64{targetID},
			}

			var promise *crm.ActionPromise
			if promiseCheck.Checked && strings.TrimSpace(promiseDescEntry.Text) != "" {
				dir := "i_promised"
				if promiseDirection.Selected == "Мне пообещали" {
					dir = "they_promised"
				}
				due, pErr := time.Parse("02.01.2006", strings.TrimSpace(promiseDueEntry.Text))
				if pErr != nil {
					due = time.Now().Add(7 * 24 * time.Hour)
				}
				promise = &crm.ActionPromise{
					ContactID:   targetID,
					Direction:   dir,
					Description: strings.TrimSpace(promiseDescEntry.Text),
					DueAt:       due,
					Status:      "pending",
				}
			}

			_, err := u.crmService.LogInteraction(context.Background(), touch, promise)
			if err != nil {
				dialog.ShowError(err, u.mainWin)
				return
			}

			u.setStatus("Касание успешно сохранено")
			if u.radarView != nil {
				u.radarView.RefreshData()
			}
			if u.selectedContact != nil && u.selectedContact.ID == targetID {
				u.renderDetailPanel(u.selectedContact)
			}
		},
		u.mainWin,
	)

	d.Resize(fyne.NewSize(580, 520))
	d.Show()
}
