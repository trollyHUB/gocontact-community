// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// showPreMeetingDossierDialog открывает модальное окно одностраничного досье перед встречей.
func (u *AppUI) showPreMeetingDossierDialog(contactID int64) {
	if u.crmService == nil {
		dialog.ShowInformation("Внимание", "Сервис Personal CRM не активирован", u.mainWin)
		return
	}

	dossier, err := u.crmService.GeneratePreMeetingDossier(context.Background(), contactID)
	if err != nil {
		dialog.ShowError(err, u.mainWin)
		return
	}

	// 1. Верхняя карточка контакта и здоровья
	healthBadge := NewHealthRingBadge(dossier.Health.HealthScore, dossier.Health.Status)
	nameLbl := widget.NewLabelWithStyle(dossier.FullName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	roleLbl := newWrappedValueLabel(dossier.RoleAndCompany)

	statusDesc := fmt.Sprintf("Теплота: %s (%d/100) • Орбита Данбара: %d",
		dossier.Health.Status.Label(), dossier.Health.HealthScore, dossier.Health.DunbarTier)
	healthLbl := widget.NewLabel(statusDesc)

	headerInfo := container.NewVBox(nameLbl, roleLbl, healthLbl)
	if dossier.IntroducedByName != "" {
		introText := fmt.Sprintf("🤝 Кто познакомил: %s", dossier.IntroducedByName)
		if dossier.IntroducedContext != "" {
			introText += fmt.Sprintf(" («%s»)", dossier.IntroducedContext)
		}
		headerInfo.Add(widget.NewLabel(introText))
	}
	if len(dossier.DirectConnections) > 0 {
		headerInfo.Add(widget.NewLabel(fmt.Sprintf("🔗 Общие знакомые (%d): %s",
			len(dossier.DirectConnections), strings.Join(dossier.DirectConnections, ", "))))
	}

	topCard := widget.NewCard("", "", container.NewHBox(
		healthBadge,
		headerInfo,
	))

	// 2. Секция открытых обещаний
	promisesBox := container.NewVBox()
	promisesBox.Add(widget.NewLabelWithStyle("⚠️ Открытые обещания перед встречей:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	if len(dossier.PendingPromises) == 0 {
		promisesBox.Add(widget.NewLabel("Нет невыполненных обещаний."))
	} else {
		for _, p := range dossier.PendingPromises {
			who := "Вы пообещали"
			if p.Direction == "they_promised" {
				who = "Вам пообещали"
			}
			dueStr := p.DueAt.Format("02.01.2006")
			pLbl := widget.NewLabel(fmt.Sprintf("• [%s]: %s (Дедлайн: %s)", who, p.Description, dueStr))
			pLbl.Wrapping = fyne.TextWrapWord
			promisesBox.Add(pLbl)
		}
	}

	// 3. Личный контекст и важные даты
	contextBox := container.NewVBox()
	contextBox.Add(widget.NewLabelWithStyle("👨‍👩‍👦 Личный контекст и важные даты (О чём спросить):", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	if len(dossier.UpcomingEvents) == 0 && dossier.KeyNotes == "" {
		contextBox.Add(widget.NewLabel("Личных заметок пока нет."))
	} else {
		for _, ev := range dossier.UpcomingEvents {
			evLbl := widget.NewLabel(fmt.Sprintf("• %s: %s (%s)", ev.Title, ev.EventDate, ev.Notes))
			evLbl.Wrapping = fyne.TextWrapWord
			contextBox.Add(evLbl)
		}
		if dossier.KeyNotes != "" {
			notesLbl := widget.NewLabel(fmt.Sprintf("• Заметки: %s", dossier.KeyNotes))
			notesLbl.Wrapping = fyne.TextWrapWord
			contextBox.Add(notesLbl)
		}
	}

	// 4. Сводка последних 3 касаний
	touchesBox := container.NewVBox()
	touchesBox.Add(widget.NewLabelWithStyle("⏪ Краткая сводка последних касаний:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	if len(dossier.RecentTouches) == 0 {
		touchesBox.Add(widget.NewLabel("Касаний в базе пока не зафиксировано."))
	} else {
		for i, touch := range dossier.RecentTouches {
			dateStr := touch.OccurredAt.Format("02.01.2006")
			dur := ""
			if touch.DurationMin > 0 {
				dur = fmt.Sprintf(" (%d мин)", touch.DurationMin)
			}
			tLbl := widget.NewLabel(fmt.Sprintf("%d. [%s%s, %s] %s", i+1, touch.Channel.Label(), dur, dateStr, touch.Title))
			tLbl.Wrapping = fyne.TextWrapWord
			touchesBox.Add(tLbl)
			if touch.NotesMD != "" {
				noteSub := widget.NewLabel(fmt.Sprintf("   > %s", touch.NotesMD))
				noteSub.Wrapping = fyne.TextWrapWord
				touchesBox.Add(noteSub)
			}
		}
	}

	body := container.NewVBox(
		topCard,
		widget.NewSeparator(),
		promisesBox,
		widget.NewSeparator(),
		contextBox,
		widget.NewSeparator(),
		touchesBox,
	)

	scroll := container.NewVScroll(container.NewPadded(body))

	btnCopyMD := widget.NewButtonWithIcon("📋 Скопировать Markdown", theme.ContentCopyIcon(), func() {
		u.mainWin.Clipboard().SetContent(dossier.ToMarkdown())
		u.setStatus("Досье скопировано в буфер обмена")
	})
	btnCopyMD.Importance = widget.HighImportance

	btnLog := widget.NewButtonWithIcon("⚡ Записать касание", theme.ContentAddIcon(), func() {
		contact, _ := u.service.GetContactByID(context.Background(), contactID)
		u.showTouchpointLoggerDialog(contact)
	})

	actionBar := container.NewHBox(btnCopyMD, btnLog)

	content := container.NewBorder(nil, actionBar, nil, nil, scroll)

	d := dialog.NewCustom(
		fmt.Sprintf("📄 Досье перед встречей: %s", dossier.FullName),
		"Закрыть",
		content,
		u.mainWin,
	)
	d.Resize(fyne.NewSize(740, 600))
	d.Show()
}
