// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package crm

import (
	"strings"
	"testing"
	"time"
)

func TestChannelType_WeightsAndLabels(t *testing.T) {
	channels := []ChannelType{
		ChannelMeeting,
		ChannelGiftIntro,
		ChannelCall,
		ChannelMessage,
		ChannelEmail,
	}

	for _, ch := range channels {
		if ch.Weight() <= 0 || ch.Weight() > 1.0 {
			t.Fatalf("некорректный вес канала %s: %f", ch, ch.Weight())
		}
		if ch.Label() == "" {
			t.Fatalf("пустой лейбл канала %s", ch)
		}
	}

	unknown := ChannelType("unknown")
	if unknown.Weight() != 0.50 {
		t.Fatalf("ожидался вес по умолчанию 0.50, получено %f", unknown.Weight())
	}
	if unknown.Label() != "Касание" {
		t.Fatalf("ожидался дефолтный лейбл, получено %s", unknown.Label())
	}
}

func TestHealthStatus_Labels(t *testing.T) {
	statuses := []HealthStatus{
		StatusThriving,
		StatusWarm,
		StatusCooling,
		StatusCold,
	}

	for _, st := range statuses {
		if st.Label() == "" {
			t.Fatalf("пустой лейбл статуса %s", st)
		}
	}
	if HealthStatus("unknown").Label() != "Не определено" {
		t.Fatalf("ожидался лейбл 'Не определено'")
	}
}

func TestInteraction_Validation(t *testing.T) {
	var nilInteraction *Interaction
	if err := nilInteraction.Validate(); err != ErrEmptyParticipants {
		t.Fatalf("ожидалась ошибка ErrEmptyParticipants для nil, получено: %v", err)
	}

	emptyParticipants := &Interaction{
		Channel: ChannelCall,
	}
	if err := emptyParticipants.Validate(); err != ErrEmptyParticipants {
		t.Fatalf("ожидалась ошибка ErrEmptyParticipants для пустых участников, получено: %v", err)
	}

	invalidChannel := &Interaction{
		Channel:        ChannelType("telepathy"),
		ParticipantIDs: []int64{1},
	}
	if err := invalidChannel.Validate(); err != ErrInvalidChannel {
		t.Fatalf("ожидалась ошибка ErrInvalidChannel, получено: %v", err)
	}

	valid := &Interaction{
		Channel:        ChannelMeeting,
		ParticipantIDs: []int64{1, 2},
		Sentiment:      5,  // должно нормализоваться
		DurationMin:    -5, // должно нормализоваться в 0
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("валидное взаимодействие вернуло ошибку: %v", err)
	}
	if valid.Sentiment != 0 {
		t.Fatalf("ожидался sentiment 0, получено: %d", valid.Sentiment)
	}
	if valid.DurationMin != 0 {
		t.Fatalf("ожидалась duration 0, получено: %d", valid.DurationMin)
	}
	if valid.OccurredAt.IsZero() {
		t.Fatalf("ожидалось автозаполнение даты occurred_at")
	}
}

func TestActionPromise_Validation(t *testing.T) {
	var nilPromise *ActionPromise
	if err := nilPromise.Validate(); err != ErrPromiseEmptyDesc {
		t.Fatalf("ожидалась ошибка ErrPromiseEmptyDesc, получено: %v", err)
	}

	emptyDesc := &ActionPromise{Description: "   "}
	if err := emptyDesc.Validate(); err != ErrPromiseEmptyDesc {
		t.Fatalf("ожидалась ошибка ErrPromiseEmptyDesc для пробелов, получено: %v", err)
	}

	valid := &ActionPromise{
		ContactID:   1,
		Description: "Отправить коммерческое предложение",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("ошибка валидации: %v", err)
	}
	if valid.Direction != "i_promised" {
		t.Fatalf("ожидался default direction 'i_promised'")
	}
	if valid.Status != "pending" {
		t.Fatalf("ожидался default status 'pending'")
	}
	if valid.DueAt.IsZero() {
		t.Fatalf("ожидался default due_at")
	}
}

func TestPreMeetingDossier_ToMarkdown(t *testing.T) {
	dossier := &PreMeetingDossier{
		ContactID:      1,
		FullName:       "Ерлан Тулегенов",
		RoleAndCompany: "Директор @ Astana Hub",
		Phone:          "+7 701 555-01-99",
		Email:          "erlan@astanahub.kz",
		Health: HealthSnapshot{
			ContactID:   1,
			DunbarTier:  DunbarSympathyGroup,
			HealthScore: 88,
			Status:      StatusThriving,
		},
		IntroducedByName:  "Алихан С.",
		IntroducedContext: "Хакатон Алматы",
		DirectConnections: []string{"Диана Рахимова", "Даурен Бектуров"},
		PendingPromises: []*ActionPromise{
			{
				Direction:   "i_promised",
				Description: "Выслать архитектуру синка",
				DueAt:       time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC),
			},
		},
		UpcomingEvents: []*LifeEvent{
			{
				Title:     "День рождения",
				EventDate: "14 мая",
				Notes:     "Любит фильтр-кофе",
			},
		},
		RecentTouches: []*Interaction{
			{
				Channel:     ChannelMeeting,
				Title:       "Обед в Del Papa",
				NotesMD:     "Обсудили гранты",
				DurationMin: 60,
				OccurredAt:  time.Date(2026, 10, 18, 14, 0, 0, 0, time.UTC),
			},
		},
		GeneratedAt: time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC),
	}

	md := dossier.ToMarkdown()
	if !strings.Contains(md, "Ерлан Тулегенов") {
		t.Fatalf("в Markdown нет имени контакта")
	}
	if !strings.Contains(md, "Astana Hub") {
		t.Fatalf("в Markdown нет компании")
	}
	if !strings.Contains(md, "Выслать архитектуру синка") {
		t.Fatalf("в Markdown нет обещания")
	}
	if !strings.Contains(md, "Обед в Del Papa") {
		t.Fatalf("в Markdown нет касания")
	}
}
