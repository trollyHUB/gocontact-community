// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	crmDomain "github.com/trollyHUB/gocontact-studio/internal/domain/crm"
	"github.com/trollyHUB/gocontact-studio/internal/service/crm"
)

// RadarView представляет экран дашборда «Keep-in-Touch Radar» и управления приоритетами внимания.
type RadarView struct {
	appUI      *AppUI
	crmService crm.CRMService

	contentBox    *fyne.Container
	scroll        *container.Scroll
	statsLabel    *widget.Label
	candidatesBox *fyne.Container
}

// NewRadarView создаёт новый экземпляр дашборда Keep-in-Touch Radar.
func NewRadarView(appUI *AppUI, crmService crm.CRMService) *RadarView {
	rv := &RadarView{
		appUI:         appUI,
		crmService:    crmService,
		candidatesBox: container.NewVBox(),
	}

	rv.buildUI()
	return rv
}

// Container возвращает графический объект экрана.
func (rv *RadarView) Container() fyne.CanvasObject {
	return rv.scroll
}

func (rv *RadarView) buildUI() {
	titleLbl := widget.NewLabelWithStyle("📡 Keep-in-Touch Radar (Радар отношений недели)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	subLbl := newWrappedValueLabel("Проактивная система ранжирования контактов по дефициту тепла, кругам Данбара и срокам открытых обещаний.")

	btnRefresh := widget.NewButtonWithIcon("Обновить радар", theme.ViewRefreshIcon(), func() {
		rv.RefreshData()
	})

	btnQuickLog := widget.NewButtonWithIcon("+ Записать касание (Ctrl+L)", theme.ContentAddIcon(), func() {
		rv.appUI.showTouchpointLoggerDialog(nil)
	})
	btnQuickLog.Importance = widget.HighImportance

	rv.statsLabel = widget.NewLabelWithStyle("Загрузка радара...", fyne.TextAlignTrailing, fyne.TextStyle{Italic: true})

	topBar := container.NewHBox(
		container.NewVBox(titleLbl, subLbl),
		layout.NewSpacer(),
		btnRefresh,
		btnQuickLog,
	)

	rv.contentBox = container.NewVBox(
		topBar,
		widget.NewSeparator(),
		rv.statsLabel,
		widget.NewSeparator(),
		rv.candidatesBox,
	)

	rv.scroll = container.NewVScroll(container.NewPadded(rv.contentBox))
	rv.RefreshData()
}

// RefreshData перезагружает список кандидатов и статистику радара.
func (rv *RadarView) RefreshData() {
	if rv.crmService == nil {
		return
	}

	ctx := context.Background()
	candidates, err := rv.crmService.GetRadarCandidates(ctx, 20)
	if err != nil {
		rv.statsLabel.SetText(fmt.Sprintf("Ошибка загрузки: %v", err))
		return
	}

	promises, _ := rv.crmService.ListPendingPromises(ctx)

	rv.statsLabel.SetText(fmt.Sprintf(
		"🔥 Требуют внимания: %d контактов | 📌 Открытых обещаний: %d",
		len(candidates), len(promises),
	))

	rv.candidatesBox.Objects = nil

	if len(candidates) == 0 {
		emptyCard := widget.NewCard(
			"🎉 Все важные связи в отличном тонусе!",
			"",
			widget.NewLabel("Нет остывающих контактов в ключевых кругах Данбара и просроченных обещаний на этой неделе."),
		)
		rv.candidatesBox.Add(emptyCard)
		rv.candidatesBox.Refresh()
		return
	}

	for i, cand := range candidates {
		rv.candidatesBox.Add(rv.buildCandidateCard(i+1, cand))
	}

	rv.candidatesBox.Refresh()
}

func (rv *RadarView) buildCandidateCard(rank int, cand *crmDomain.RadarCandidate) fyne.CanvasObject {
	badge := NewHealthRingBadge(cand.Health.HealthScore, cand.Health.Status)

	nameTitle := fmt.Sprintf("%d. %s", rank, cand.FullName)
	if cand.JobTitle != "" || cand.Company != "" {
		nameTitle += fmt.Sprintf(" • %s, %s", cand.JobTitle, cand.Company)
	}
	titleLbl := widget.NewLabelWithStyle(nameTitle, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	infoText := fmt.Sprintf("Орбита Данбара: %d | Последний контакт: %d дн. назад (целевой ритм: %d дн.)",
		cand.DunbarTier, cand.DaysSinceLastTouch, cand.TargetIntervalDays)
	subLbl := newWrappedValueLabel(infoText)

	reasonBox := container.NewVBox()
	if cand.PendingPromiseNote != "" {
		reasonBox.Add(widget.NewLabelWithStyle("⚠️ "+cand.PendingPromiseNote, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	}
	if cand.UpcomingEventNote != "" {
		reasonBox.Add(widget.NewLabelWithStyle("🎂 "+cand.UpcomingEventNote, fyne.TextAlignLeading, fyne.TextStyle{Italic: true}))
	}

	btnLog := widget.NewButtonWithIcon("⚡ Залогировать касание", theme.ContentAddIcon(), func() {
		contact, _ := rv.appUI.service.GetContactByID(context.Background(), cand.ContactID)
		rv.appUI.showTouchpointLoggerDialog(contact)
	})
	btnLog.Importance = widget.HighImportance

	btnDossier := widget.NewButtonWithIcon("📄 Досье", theme.DocumentIcon(), func() {
		rv.appUI.showPreMeetingDossierDialog(cand.ContactID)
	})

	btnSnooze := widget.NewButtonWithIcon("💤 Отложить на 7 дн.", theme.CancelIcon(), func() {
		_ = rv.crmService.SnoozeContact(context.Background(), cand.ContactID, 7*24*time.Hour)
		rv.RefreshData()
	})

	actions := container.NewHBox(btnLog, btnDossier, btnSnooze)

	cardContent := container.NewHBox(
		badge,
		container.NewVBox(
			titleLbl,
			subLbl,
			reasonBox,
			actions,
		),
	)

	return widget.NewCard("", "", cardContent)
}
