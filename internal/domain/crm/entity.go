// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package crm

import (
	"strings"
	"time"
)

// DunbarTier представляет числовую категорию орбиты Данбара (5, 15, 50, 150, 500).
type DunbarTier int

const (
	DunbarSupportClique DunbarTier = 5
	DunbarSympathyGroup DunbarTier = 15
	DunbarAffinityGroup DunbarTier = 50
	DunbarActiveNetwork DunbarTier = 150
	DunbarAcquaintance  DunbarTier = 500
)

// ChannelType определяет канал коммуникации и его энергетический вес.
type ChannelType string

const (
	ChannelMeeting   ChannelType = "meeting"    // Личная встреча / обед (W = 1.00)
	ChannelGiftIntro ChannelType = "gift_intro" // Подарок, интро или значимая услуга (W = 0.95)
	ChannelCall      ChannelType = "call"       // Телефонный или видеозвонок (W = 0.80)
	ChannelMessage   ChannelType = "message"    // Личное сообщение в мессенджере (W = 0.45)
	ChannelEmail     ChannelType = "email"      // Короткое письмо / формальное касание (W = 0.30)
)

// Weight возвращает базовый энергетический вес канала касания.
func (c ChannelType) Weight() float64 {
	switch c {
	case ChannelMeeting:
		return 1.00
	case ChannelGiftIntro:
		return 0.95
	case ChannelCall:
		return 0.80
	case ChannelMessage:
		return 0.45
	case ChannelEmail:
		return 0.30
	default:
		return 0.50
	}
}

// Label возвращает понятное русскоязычное название и пиктограмму канала.
func (c ChannelType) Label() string {
	switch c {
	case ChannelMeeting:
		return "🤝 Личная встреча"
	case ChannelGiftIntro:
		return "🎁 Подарок / Интро"
	case ChannelCall:
		return "📞 Звонок"
	case ChannelMessage:
		return "💬 Сообщение"
	case ChannelEmail:
		return "✉️ Эл. почта"
	default:
		return "Касание"
	}
}

// HealthStatus классифицирует состояние связи по индексу HealthScore [0..100].
type HealthStatus string

const (
	StatusThriving HealthStatus = "thriving" // 80..100: Активная, процветающая связь
	StatusWarm     HealthStatus = "warm"     // 50..79:  Нормальный рабочий тонус
	StatusCooling  HealthStatus = "cooling"  // 25..49:  Связь остывает (попадает на радар)
	StatusCold     HealthStatus = "cold"     // 0..24:   Холодная / забытая связь
)

// Label возвращает человекочитаемый статус с цветовой индикацией.
func (s HealthStatus) Label() string {
	switch s {
	case StatusThriving:
		return "🟢 Активная (Thriving)"
	case StatusWarm:
		return "🟡 В тонусе (Warm)"
	case StatusCooling:
		return "🟠 Остывает (Cooling)"
	case StatusCold:
		return "🔴 Холодная (Cold)"
	default:
		return "Не определено"
	}
}

// Interaction описывает факт общения с одним или несколькими контактами.
type Interaction struct {
	ID             int64       `json:"id"`
	Channel        ChannelType `json:"channel"`
	Title          string      `json:"title"`
	NotesMD        string      `json:"notes_md"`
	Location       string      `json:"location"`
	OccurredAt     time.Time   `json:"occurred_at"`
	DurationMin    int         `json:"duration_min"`
	Sentiment      int         `json:"sentiment"` // -1 (негативный), 0 (нейтральный), +1 (позитивный)
	ParticipantIDs []int64     `json:"participant_ids"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

// Validate проверяет корректность полей взаимодействия.
func (i *Interaction) Validate() error {
	if i == nil {
		return ErrEmptyParticipants
	}
	if len(i.ParticipantIDs) == 0 {
		return ErrEmptyParticipants
	}
	switch i.Channel {
	case ChannelMeeting, ChannelGiftIntro, ChannelCall, ChannelMessage, ChannelEmail:
	default:
		return ErrInvalidChannel
	}
	if i.Sentiment < -1 || i.Sentiment > 1 {
		i.Sentiment = 0
	}
	if i.DurationMin < 0 {
		i.DurationMin = 0
	}
	if i.OccurredAt.IsZero() {
		i.OccurredAt = time.Now().UTC()
	}
	return nil
}

// ActionPromise представляет взаимное обязательство (обещание).
type ActionPromise struct {
	ID                  int64     `json:"id"`
	ContactID           int64     `json:"contact_id"`
	SourceInteractionID int64     `json:"source_interaction_id,omitempty"`
	Direction           string    `json:"direction"` // "i_promised" | "they_promised"
	Description         string    `json:"description"`
	DueAt               time.Time `json:"due_at"`
	Status              string    `json:"status"` // "pending" | "completed" | "cancelled"
	CompletedAt         string    `json:"completed_at,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

// Validate проверяет валидность обещания.
func (p *ActionPromise) Validate() error {
	if p == nil || strings.TrimSpace(p.Description) == "" {
		return ErrPromiseEmptyDesc
	}
	if p.Direction != "i_promised" && p.Direction != "they_promised" {
		p.Direction = "i_promised"
	}
	if p.Status != "pending" && p.Status != "completed" && p.Status != "cancelled" {
		p.Status = "pending"
	}
	if p.DueAt.IsZero() {
		p.DueAt = time.Now().Add(7 * 24 * time.Hour)
	}
	return nil
}

// LifeEvent описывает важную дату или веху в жизни контакта.
type LifeEvent struct {
	ID               int64     `json:"id"`
	ContactID        int64     `json:"contact_id"`
	EventType        string    `json:"event_type"` // "birthday", "anniversary", "child_birth", "promotion", "relocation", "gift_idea", "custom"
	Title            string    `json:"title"`
	EventDate        string    `json:"event_date"` // YYYY-MM-DD или --MM-DD
	IsRecurring      bool      `json:"is_recurring"`
	RemindDaysBefore int       `json:"remind_days_before"`
	Notes            string    `json:"notes"`
	CreatedAt        time.Time `json:"created_at"`
}

// HealthSnapshot содержит вычисленные метрики теплоты связи контакта.
type HealthSnapshot struct {
	ContactID          int64        `json:"contact_id"`
	DunbarTier         DunbarTier   `json:"dunbar_tier"`
	TargetIntervalDays int          `json:"target_interval_days"`
	RawEnergy          float64      `json:"raw_energy"`
	HealthScore        int          `json:"health_score"` // [0..100]
	Status             HealthStatus `json:"health_status"`
	LastInteractionAt  time.Time    `json:"last_interaction_at"`
	SnoozedUntil       time.Time    `json:"snoozed_until,omitempty"`
	UrgencyScore       float64      `json:"urgency_score"`
	UpdatedAt          time.Time    `json:"updated_at"`
}
