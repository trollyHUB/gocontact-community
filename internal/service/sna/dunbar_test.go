// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sna_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

func TestCalculateClosenessIndex(t *testing.T) {
	// maxWeight = 5, deg = 10, isFavorite = true
	// I = 0.55*5 + 0.30*ln(11) + 0.15*1.0 = 2.75 + 0.30*2.397895 + 0.15 = 2.75 + 0.719368 + 0.15 = 3.619368
	score := sna.CalculateClosenessIndex(5, 10, true)
	expected := 0.55*5.0 + 0.30*math.Log(11.0) + 0.15
	if math.Abs(score-expected) > 1e-5 {
		t.Fatalf("ожидался индекс %f, получено: %f", expected, score)
	}
}

func TestDunbar_ManualOverrides(t *testing.T) {
	contacts := []*domain.Contact{
		{ID: 1, DunbarOverride: 5},
		{ID: 2, DunbarOverride: 150},
		{ID: 3, DunbarOverride: 0},
	}

	tiers := sna.ClassifyDunbarTiers(contacts, nil)

	if tiers[1] != graph.DunbarSupportClique {
		t.Fatalf("для ID=1 ожидался оверрайд Tier 5, получено: %v", tiers[1])
	}
	if tiers[2] != graph.DunbarActiveNetwork {
		t.Fatalf("для ID=2 ожидался оверрайд Tier 150, получено: %v", tiers[2])
	}
	if tiers[3] != graph.DunbarSupportClique {
		// ID 3 один без оверрайда, попадает в топ-1 квоты -> Tier 5
		t.Fatalf("для ID=3 ожидался Tier 5, получено: %v", tiers[3])
	}
}

func TestDunbar_QuotaDistribution(t *testing.T) {
	// Создаем 160 контактов с возрастающим числом связей
	contacts := make([]*domain.Contact, 160)
	var edges []*graph.Relationship

	edgeID := int64(1)
	for i := 0; i < 160; i++ {
		cid := int64(i + 1)
		contacts[i] = &domain.Contact{
			ID:         cid,
			FirstName:  fmt.Sprintf("User_%d", cid),
			IsFavorite: i < 5,
		}
		// У контактов с меньшим индексом создаем больше связей и выше вес
		deg := 160 - i
		for d := 0; d < deg; d++ {
			edges = append(edges, &graph.Relationship{
				ID:       edgeID,
				SourceID: cid,
				TargetID: int64(1000 + d),
				Weight:   5,
			})
			edgeID++
		}
	}

	tiers := sna.ClassifyDunbarTiers(contacts, edges)

	tierCounts := make(map[graph.DunbarTier]int)
	for _, tier := range tiers {
		tierCounts[tier]++
	}

	// Проверяем заполнение квот:
	// Топ 5 -> 5
	// След 10 -> 15
	// След 35 -> 50
	// След 100 -> 150
	// Остаток 10 -> 500
	if tierCounts[graph.DunbarSupportClique] != 5 {
		t.Fatalf("ожидалось 5 в SupportClique, получено: %d", tierCounts[graph.DunbarSupportClique])
	}
	if tierCounts[graph.DunbarSympathyGroup] != 10 {
		t.Fatalf("ожидалось 10 в SympathyGroup, получено: %d", tierCounts[graph.DunbarSympathyGroup])
	}
	if tierCounts[graph.DunbarAffinityGroup] != 35 {
		t.Fatalf("ожидалось 35 в AffinityGroup, получено: %d", tierCounts[graph.DunbarAffinityGroup])
	}
	if tierCounts[graph.DunbarActiveNetwork] != 100 {
		t.Fatalf("ожидалось 100 в ActiveNetwork, получено: %d", tierCounts[graph.DunbarActiveNetwork])
	}
	if tierCounts[graph.DunbarAcquaintances] != 10 {
		t.Fatalf("ожидалось 10 в Acquaintances, получено: %d", tierCounts[graph.DunbarAcquaintances])
	}
}

func TestDunbar_Empty(t *testing.T) {
	tiers := sna.ClassifyDunbarTiers(nil, nil)
	if len(tiers) != 0 {
		t.Fatalf("для пустого списка ожидался пустой результат")
	}
}
