// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package crm

import (
	"math"

	"github.com/trollyHUB/gocontact-studio/internal/domain/crm"
)

// DunbarHalfLifeDays возвращает период полураспада энергии отношений в днях (T_half).
func DunbarHalfLifeDays(tier crm.DunbarTier) float64 {
	switch tier {
	case crm.DunbarSupportClique: // Круг 5: семья, ближайшие соратники
		return 7.0
	case crm.DunbarSympathyGroup: // Круг 15: близкие друзья, ключевые партнёры
		return 21.0
	case crm.DunbarAffinityGroup: // Круг 50: активный деловой и дружеский круг
		return 60.0
	case crm.DunbarActiveNetwork: // Круг 150: стабильная сеть контактов
		return 120.0
	case crm.DunbarAcquaintance: // Круг 500: знакомые
		return 240.0
	default:
		return 60.0
	}
}

// CalculateDecayedEnergy рассчитывает затухание энергии отношений по экспоненциальной модели:
// E(t) = E0 * exp(-lambda * delta_t), где lambda = ln(2) / T_half.
func CalculateDecayedEnergy(initialEnergy float64, daysElapsed float64, tier crm.DunbarTier) float64 {
	if daysElapsed <= 0 {
		return math.Min(1.0, math.Max(0.0, initialEnergy))
	}
	tHalf := DunbarHalfLifeDays(tier)
	lambda := math.Ln2 / tHalf
	decayed := initialEnergy * math.Exp(-lambda*daysElapsed)
	if decayed < 0.0 {
		return 0.0
	}
	if decayed > 1.0 {
		return 1.0
	}
	return decayed
}

// ApplyInteractionEnergy добавляет порцию энергии от совершённого взаимодействия:
// E_new = min(1.0, E_current + W_channel).
func ApplyInteractionEnergy(currentEnergy float64, channel crm.ChannelType) float64 {
	newEnergy := currentEnergy + channel.Weight()
	if newEnergy > 1.0 {
		return 1.0
	}
	if newEnergy < 0.0 {
		return 0.0
	}
	return newEnergy
}

// EnergyToScore конвертирует непрерывную энергию [0.0..1.0] в дискретный HealthScore [0..100] и статус.
func EnergyToScore(energy float64) (int, crm.HealthStatus) {
	score := int(math.Round(energy * 100))
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	var status crm.HealthStatus
	switch {
	case score >= 80:
		status = crm.StatusThriving
	case score >= 50:
		status = crm.StatusWarm
	case score >= 25:
		status = crm.StatusCooling
	default:
		status = crm.StatusCold
	}

	return score, status
}
