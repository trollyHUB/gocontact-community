// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sna_test

import (
	"testing"

	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

func TestLouvain_TwoCliquesWithWeakBridge(t *testing.T) {
	// Две плотные клики по 3 узла, соединенные одним слабым мостом 3 <-> 4:
	// Клика 1: 1, 2, 3 (веса 5)
	// Клика 2: 4, 5, 6 (веса 5)
	// Мост: 3 <-> 4 (вес 1)
	nodes := []*graph.Node{
		{ContactID: 1}, {ContactID: 2}, {ContactID: 3},
		{ContactID: 4}, {ContactID: 5}, {ContactID: 6},
	}

	edges := []*graph.Relationship{
		// Клика 1
		{ID: 1, SourceID: 1, TargetID: 2, Weight: 5, IsDirected: false},
		{ID: 2, SourceID: 2, TargetID: 3, Weight: 5, IsDirected: false},
		{ID: 3, SourceID: 1, TargetID: 3, Weight: 5, IsDirected: false},
		// Клика 2
		{ID: 4, SourceID: 4, TargetID: 5, Weight: 5, IsDirected: false},
		{ID: 5, SourceID: 5, TargetID: 6, Weight: 5, IsDirected: false},
		{ID: 6, SourceID: 4, TargetID: 6, Weight: 5, IsDirected: false},
		// Слабый межкластерный мост
		{ID: 7, SourceID: 3, TargetID: 4, Weight: 1, IsDirected: false},
	}

	comms := sna.DetectCommunities(nodes, edges)

	// Узлы 1, 2, 3 обязаны принадлежать одному кластеру
	c1 := comms[1]
	if comms[2] != c1 || comms[3] != c1 {
		t.Fatalf("узлы клики 1 должны иметь одинаковый CommunityID, получено: %d, %d, %d", c1, comms[2], comms[3])
	}

	// Узлы 4, 5, 6 обязаны принадлежать второму кластеру
	c2 := comms[4]
	if comms[5] != c2 || comms[6] != c2 {
		t.Fatalf("узлы клики 2 должны иметь одинаковый CommunityID, получено: %d, %d, %d", c2, comms[5], comms[6])
	}

	if c1 == c2 {
		t.Fatalf("клики должны быть разбиты на 2 разных сообщества, но получили общий c=%d", c1)
	}
}

func TestLouvain_DisconnectedComponents(t *testing.T) {
	nodes := []*graph.Node{
		{ContactID: 1}, {ContactID: 2},
		{ContactID: 3}, {ContactID: 4},
	}
	edges := []*graph.Relationship{
		{ID: 1, SourceID: 1, TargetID: 2, Weight: 4, IsDirected: false},
		{ID: 2, SourceID: 3, TargetID: 4, Weight: 4, IsDirected: false},
	}

	comms := sna.DetectCommunities(nodes, edges)

	if comms[1] != comms[2] {
		t.Fatalf("1 и 2 должны быть в одном сообществе")
	}
	if comms[3] != comms[4] {
		t.Fatalf("3 и 4 должны быть в одном сообществе")
	}
	if comms[1] == comms[3] {
		t.Fatalf("несвязанные компоненты должны иметь разные сообщества")
	}
}

func TestLouvain_EmptyAndSingle(t *testing.T) {
	res := sna.DetectCommunities(nil, nil)
	if len(res) != 0 {
		t.Fatalf("для nil графа ожидался пустой результат")
	}

	single := []*graph.Node{{ContactID: 42}}
	resSingle := sna.DetectCommunities(single, nil)
	if resSingle[42] != 0 {
		t.Fatalf("единственный узел должен получить community 0, получено: %d", resSingle[42])
	}
}
