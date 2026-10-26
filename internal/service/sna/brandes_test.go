// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sna_test

import (
	"testing"

	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

func TestBrandes_StarGraph(t *testing.T) {
	// Граф «Звезда»: центр (1), лучи (2, 3, 4, 5).
	// Все пути между лучами проходят через 1.
	// Центральность узла 1 должна быть строго 1.0, лучей — 0.0.
	nodes := []*graph.Node{
		{ContactID: 1},
		{ContactID: 2},
		{ContactID: 3},
		{ContactID: 4},
		{ContactID: 5},
	}
	edges := []*graph.Relationship{
		{ID: 1, SourceID: 1, TargetID: 2, IsDirected: false},
		{ID: 2, SourceID: 1, TargetID: 3, IsDirected: false},
		{ID: 3, SourceID: 1, TargetID: 4, IsDirected: false},
		{ID: 4, SourceID: 1, TargetID: 5, IsDirected: false},
	}

	scores := sna.ComputeBetweennessCentrality(nodes, edges)

	if scores[1] != 1.0 {
		t.Fatalf("центральный узел звезды обязан иметь Betweenness = 1.0, получено: %f", scores[1])
	}
	for id := int64(2); id <= 5; id++ {
		if scores[id] != 0.0 {
			t.Fatalf("периферийный узел %d звезды должен иметь Betweenness = 0.0, получено: %f", id, scores[id])
		}
	}
}

func TestBrandes_LineGraph(t *testing.T) {
	// Линейный граф из 3 узлов: 1 - 2 - 3
	// Узел 2 соединяет 1 и 3, его Betweenness = 1.0, у 1 и 3 = 0.0
	nodes := []*graph.Node{
		{ContactID: 1},
		{ContactID: 2},
		{ContactID: 3},
	}
	edges := []*graph.Relationship{
		{ID: 1, SourceID: 1, TargetID: 2, IsDirected: false},
		{ID: 2, SourceID: 2, TargetID: 3, IsDirected: false},
	}

	scores := sna.ComputeBetweennessCentrality(nodes, edges)

	if scores[2] != 1.0 {
		t.Fatalf("узел-мост 2 должен иметь Betweenness = 1.0, получено: %f", scores[2])
	}
	if scores[1] != 0.0 || scores[3] != 0.0 {
		t.Fatalf("концевые узлы должны иметь Betweenness = 0.0")
	}
}

func TestBrandes_CompleteGraph(t *testing.T) {
	// Полный граф K4: между любой парой есть прямое ребро, никто не является посредником
	nodes := []*graph.Node{
		{ContactID: 1},
		{ContactID: 2},
		{ContactID: 3},
		{ContactID: 4},
	}
	var edges []*graph.Relationship
	id := int64(1)
	for i := int64(1); i <= 4; i++ {
		for j := i + 1; j <= 4; j++ {
			edges = append(edges, &graph.Relationship{
				ID:         id,
				SourceID:   i,
				TargetID:   j,
				IsDirected: false,
			})
			id++
		}
	}

	scores := sna.ComputeBetweennessCentrality(nodes, edges)
	for _, n := range nodes {
		if scores[n.ContactID] != 0.0 {
			t.Fatalf("в полном графе посредничество должно быть 0.0, у %d получено: %f", n.ContactID, scores[n.ContactID])
		}
	}
}

func TestBrandes_EmptyAndSmall(t *testing.T) {
	scores := sna.ComputeBetweennessCentrality(nil, nil)
	if len(scores) != 0 {
		t.Fatalf("для пустого графа ожидался пустой результат")
	}

	twoNodes := []*graph.Node{{ContactID: 1}, {ContactID: 2}}
	scores2 := sna.ComputeBetweennessCentrality(twoNodes, nil)
	if scores2[1] != 0.0 || scores2[2] != 0.0 {
		t.Fatalf("для n <= 2 скоры должны быть 0.0")
	}
}
