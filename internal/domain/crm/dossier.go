// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package crm

import (
	"fmt"
	"strings"
	"time"
)

// RadarCandidate представляет кандидата для Keep-in-Touch Radar дашборда.
type RadarCandidate struct {
	ContactID          int64          `json:"contact_id"`
	FullName           string         `json:"full_name"`
	JobTitle           string         `json:"job_title"`
	Company            string         `json:"company"`
	DunbarTier         DunbarTier     `json:"dunbar_tier"`
	Health             HealthSnapshot `json:"health"`
	DaysSinceLastTouch int            `json:"days_since_last_touch"`
	TargetIntervalDays int            `json:"target_interval_days"`
	LastTouchSummary   string         `json:"last_touch_summary"`
	PendingPromiseNote string         `json:"pending_promise_note,omitempty"`
	UpcomingEventNote  string         `json:"upcoming_event_note,omitempty"`
	UrgencyScore       float64        `json:"urgency_score"`
}

// PreMeetingDossier — собранный одностраничный бриф перед звонком или встречей.
type PreMeetingDossier struct {
	ContactID         int64            `json:"contact_id"`
	FullName          string           `json:"full_name"`
	RoleAndCompany    string           `json:"role_and_company"`
	Email             string           `json:"email"`
	Phone             string           `json:"phone"`
	Health            HealthSnapshot   `json:"health"`
	IntroducedByName  string           `json:"introduced_by_name,omitempty"`
	IntroducedContext string           `json:"introduced_context,omitempty"`
	DirectConnections []string         `json:"direct_connections"`
	RecentTouches     []*Interaction   `json:"recent_touches"`
	PendingPromises   []*ActionPromise `json:"pending_promises"`
	UpcomingEvents    []*LifeEvent     `json:"upcoming_events"`
	KeyNotes          string           `json:"key_notes"`
	GeneratedAt       time.Time        `json:"generated_at"`
}

// ToMarkdown форматирует досье в лаконичный 1-страничный Markdown-документ.
func (d *PreMeetingDossier) ToMarkdown() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# 📄 Досье перед встречей: %s\n\n", d.FullName))
	sb.WriteString(fmt.Sprintf("**Должность / Компания:** %s\n", d.RoleAndCompany))
	if d.Phone != "" {
		sb.WriteString(fmt.Sprintf("**Телефон:** %s | **Email:** %s\n", d.Phone, d.Email))
	}
	sb.WriteString(fmt.Sprintf("**Теплота связи:** %d/100 (%s) • Орбита Данбара: %d\n",
		d.Health.HealthScore, d.Health.Status.Label(), d.Health.DunbarTier))

	if d.IntroducedByName != "" {
		ctx := ""
		if d.IntroducedContext != "" {
			ctx = fmt.Sprintf(" («%s»)", d.IntroducedContext)
		}
		sb.WriteString(fmt.Sprintf("**Кто познакомил:** %s%s\n", d.IntroducedByName, ctx))
	}

	if len(d.DirectConnections) > 0 {
		sb.WriteString(fmt.Sprintf("**Общие знакомые / Связи (%d):** %s\n",
			len(d.DirectConnections), strings.Join(d.DirectConnections, ", ")))
	}
	sb.WriteString("\n---\n\n")

	// Обещания
	sb.WriteString("### ⚠️ Открытые обещания и обязательства:\n")
	if len(d.PendingPromises) == 0 {
		sb.WriteString("_Нет открытых обещаний._\n\n")
	} else {
		for _, p := range d.PendingPromises {
			who := "Вы пообещали"
			if p.Direction == "they_promised" {
				who = "Вам пообещали"
			}
			due := p.DueAt.Format("02.01.2006")
			sb.WriteString(fmt.Sprintf("• [%s]: %s (Срок: %s)\n", who, p.Description, due))
		}
		sb.WriteString("\n")
	}

	// Вехи жизни
	sb.WriteString("### 👨‍👩‍👦 Личный контекст и важные даты:\n")
	if len(d.UpcomingEvents) == 0 && d.KeyNotes == "" {
		sb.WriteString("_Личных заметок пока не добавлено._\n\n")
	} else {
		for _, e := range d.UpcomingEvents {
			sb.WriteString(fmt.Sprintf("• %s: %s (%s)\n", e.Title, e.EventDate, e.Notes))
		}
		if d.KeyNotes != "" {
			sb.WriteString(fmt.Sprintf("• Заметки: %s\n", d.KeyNotes))
		}
		sb.WriteString("\n")
	}

	// Последние касания
	sb.WriteString("### ⏪ Хроника последних касаний:\n")
	if len(d.RecentTouches) == 0 {
		sb.WriteString("_Взаимодействий в базе пока не зафиксировано._\n\n")
	} else {
		for i, touch := range d.RecentTouches {
			dateStr := touch.OccurredAt.Format("02.01.2006")
			dur := ""
			if touch.DurationMin > 0 {
				dur = fmt.Sprintf(", %d мин", touch.DurationMin)
			}
			title := touch.Title
			if title == "" {
				title = touch.Channel.Label()
			}
			sb.WriteString(fmt.Sprintf("%d. **[%s%s]** %s — %s\n", i+1, touch.Channel.Label(), dur, dateStr, title))
			if touch.NotesMD != "" {
				sb.WriteString(fmt.Sprintf("   > %s\n", strings.ReplaceAll(touch.NotesMD, "\n", " ")))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("_Сгенерировано в GoContact Studio: %s_\n", d.GeneratedAt.Format("02.01.2006 15:04")))
	return sb.String()
}
