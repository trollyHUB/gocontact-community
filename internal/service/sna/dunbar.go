// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sna

import (
	"math"
	"sort"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
)

// CalculateClosenessIndex вычисляет индекс социальной близости контакта:
// I(u) = 0.55 * max_{e ∈ Inc(u)} w(e) + 0.30 * ln(1 + deg(u)) + 0.15 * 𝕀(IsFavorite(u)).
func CalculateClosenessIndex(maxWeight int, degree int, isFavorite bool) float64 {
	favVal := 0.0
	if isFavorite {
		favVal = 1.0
	}
	mw := float64(maxWeight)
	if mw < 0 {
		mw = 0
	}
	deg := float64(degree)
	if deg < 0 {
		deg = 0
	}

	return 0.55*mw + 0.30*math.Log(1.0+deg) + 0.15*favVal
}

type contactScored struct {
	contactID int64
	score     float64
}

// ClassifyDunbarTiers классифицирует контакты по концентрическим кругам Данбара.
// Учитывает dunbar_override (если задан > 0), а остальные контакты распределяет по убыванию I(u)
// по квотам: топ 5 -> Tier 5, след. 10 -> Tier 15, след. 35 -> Tier 50, след. 100 -> Tier 150, остаток -> Tier 500.
func ClassifyDunbarTiers(
	contacts []*domain.Contact,
	edges []*graph.Relationship,
) map[int64]graph.DunbarTier {
	result := make(map[int64]graph.DunbarTier, len(contacts))
	if len(contacts) == 0 {
		return result
	}

	// Расчёт максимального веса и степеней из инцидентных рёбер
	maxWeights := make(map[int64]int, len(contacts))
	degrees := make(map[int64]int, len(contacts))

	for _, e := range edges {
		if e == nil {
			continue
		}
		degrees[e.SourceID]++
		degrees[e.TargetID]++

		if e.Weight > maxWeights[e.SourceID] {
			maxWeights[e.SourceID] = e.Weight
		}
		if e.Weight > maxWeights[e.TargetID] {
			maxWeights[e.TargetID] = e.Weight
		}
	}

	var toRank []contactScored

	for _, c := range contacts {
		if c == nil || c.ID <= 0 {
			continue
		}

		// Ручное переопределение Данбара имеет наивысший приоритет
		if c.DunbarOverride > 0 && graph.DunbarTier(c.DunbarOverride).IsValid() {
			result[c.ID] = graph.DunbarTier(c.DunbarOverride)
			continue
		}

		deg := degrees[c.ID]
		mw := maxWeights[c.ID]
		score := CalculateClosenessIndex(mw, deg, c.IsFavorite)

		toRank = append(toRank, contactScored{
			contactID: c.ID,
			score:     score,
		})
	}

	// Сортировка по убыванию индекса близости I(u)
	sort.Slice(toRank, func(i, j int) bool {
		if math.Abs(toRank[i].score-toRank[j].score) > 1e-6 {
			return toRank[i].score > toRank[j].score
		}
		return toRank[i].contactID < toRank[j].contactID
	})

	// Распределение по квотам концентрических орбит Данбара:
	// Первые 5 -> 5 (Support Clique)
	// Следующие 10 (поз. 6..15) -> 15 (Sympathy Group)
	// Следующие 35 (поз. 16..50) -> 50 (Affinity Group)
	// Следующие 100 (поз. 51..150) -> 150 (Active Network)
	// Остальные -> 500 (Acquaintances)
	for i, item := range toRank {
		rank := i + 1
		switch {
		case rank <= 5:
			result[item.contactID] = graph.DunbarSupportClique
		case rank <= 15:
			result[item.contactID] = graph.DunbarSympathyGroup
		case rank <= 50:
			result[item.contactID] = graph.DunbarAffinityGroup
		case rank <= 150:
			result[item.contactID] = graph.DunbarActiveNetwork
		default:
			result[item.contactID] = graph.DunbarAcquaintances
		}
	}

	return result
}
