// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package crm

import (
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain/crm"
)

// CalculateUrgencyScore рассчитывает индекс срочности внимания для Keep-in-Touch Radar.
// Чем выше UrgencyScore, тем приоритетнее контакт в списке рекомендаций на неделю.
func CalculateUrgencyScore(snap *crm.HealthSnapshot, hasPendingPromise bool, hasUpcomingEvent bool, now time.Time) float64 {
	if snap == nil {
		return 0.0
	}

	// Если контакт отложен (Snooze), скрываем его из приоритетного радара
	if !snap.SnoozedUntil.IsZero() && now.Before(snap.SnoozedUntil) {
		return -1.0
	}

	var dunbarWeight float64
	switch snap.DunbarTier {
	case crm.DunbarSupportClique: // 5
		dunbarWeight = 5.0
	case crm.DunbarSympathyGroup: // 15
		dunbarWeight = 3.5
	case crm.DunbarAffinityGroup: // 50
		dunbarWeight = 2.0
	case crm.DunbarActiveNetwork: // 150
		dunbarWeight = 1.0
	case crm.DunbarAcquaintance: // 500
		dunbarWeight = 0.5
	default:
		dunbarWeight = 1.5
	}

	// Дефицит энергии отношений: [0.0 (Thriving) .. 1.0 (Cold)]
	deficit := 1.0 - (float64(snap.HealthScore) / 100.0)
	if deficit < 0 {
		deficit = 0
	}

	urgency := deficit * dunbarWeight

	// Бонус за наличие открытого обещания
	if hasPendingPromise {
		urgency += 2.5
	}

	// Бонус за приближающееся важное событие (например, День рождения в течение 7 дней)
	if hasUpcomingEvent {
		urgency += 2.0
	}

	return urgency
}
