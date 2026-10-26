// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sna_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

// BenchmarkBarnesHutTick_1000Nodes проверяет шаг симуляции физики Barnes-Hut
// на графе из 1 000 узлов и 3 500 рёбер (SLA <= 3.5 мс).
func BenchmarkBarnesHutTick_1000Nodes(b *testing.B) {
	const nodeCount = 1000
	const edgeCount = 3500

	r := rand.New(rand.NewSource(42))
	nodes := make([]*graph.Node, nodeCount)
	for i := 0; i < nodeCount; i++ {
		nodes[i] = &graph.Node{
			ContactID: int64(i + 1),
			FullName:  fmt.Sprintf("Узел_%d", i+1),
			PosX:      r.Float64()*1000 - 500,
			PosY:      r.Float64()*1000 - 500,
		}
	}

	edges := make([]*graph.Relationship, edgeCount)
	for i := 0; i < edgeCount; i++ {
		src := int64(r.Intn(nodeCount) + 1)
		dst := int64(r.Intn(nodeCount) + 1)
		if src == dst {
			dst = (dst % int64(nodeCount)) + 1
		}
		edges[i] = &graph.Relationship{
			ID:       int64(i + 1),
			SourceID: src,
			TargetID: dst,
			Type:     graph.RelColleague,
			Weight:   r.Intn(5) + 1,
		}
	}

	sim := sna.NewPhysicsSimulation(sna.DefaultSimulationConfig(), nil)
	sim.UpdateTopology(nodes, edges)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sim.Step()
	}
}

// BenchmarkDijkstraWarmIntro_5000Nodes замеряет поиск кратчайшего пути интро
// на графе из 5 000 узлов и 15 000 рёбер (SLA <= 1.2 мс).
func BenchmarkDijkstraWarmIntro_5000Nodes(b *testing.B) {
	const nodeCount = 5000
	const edgeCount = 15000

	r := rand.New(rand.NewSource(1337))
	nodes := make(map[int64]*graph.Node, nodeCount)
	for i := 0; i < nodeCount; i++ {
		id := int64(i + 1)
		nodes[id] = &graph.Node{
			ContactID: id,
			FullName:  fmt.Sprintf("Контакт_%d", id),
		}
	}

	// Гарантируем связный хребет между 1 и 5000
	edges := make([]*graph.Relationship, 0, edgeCount)
	for i := 1; i < 5000; i += 100 {
		next := i + 100
		if next > 5000 {
			next = 5000
		}
		edges = append(edges, &graph.Relationship{
			ID:       int64(len(edges) + 1),
			SourceID: int64(i),
			TargetID: int64(next),
			Type:     graph.RelIntroducedBy,
			Weight:   5,
		})
	}

	// Добавляем случайные рёбра до 15 000
	for len(edges) < edgeCount {
		src := int64(r.Intn(nodeCount) + 1)
		dst := int64(r.Intn(nodeCount) + 1)
		if src == dst {
			continue
		}
		edges = append(edges, &graph.Relationship{
			ID:       int64(len(edges) + 1),
			SourceID: src,
			TargetID: dst,
			Type:     graph.RelColleague,
			Weight:   r.Intn(5) + 1,
		})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = sna.FindIntroPath(1, 5000, nodes, edges)
	}
}

// BenchmarkLouvainCommunity_2000Nodes замеряет скорость выделения сообществ
// на графе из 2 000 узлов и 6 000 рёбер (SLA <= 18 мс).
func BenchmarkLouvainCommunity_2000Nodes(b *testing.B) {
	const nodeCount = 2000
	const edgeCount = 6000

	r := rand.New(rand.NewSource(2026))
	nodes := make([]*graph.Node, nodeCount)
	for i := 0; i < nodeCount; i++ {
		id := int64(i + 1)
		nodes[i] = &graph.Node{ContactID: id}
	}

	edges := make([]*graph.Relationship, edgeCount)
	for i := 0; i < edgeCount; i++ {
		src := int64(r.Intn(nodeCount) + 1)
		dst := int64(r.Intn(nodeCount) + 1)
		if src == dst {
			dst = (dst % int64(nodeCount)) + 1
		}
		edges[i] = &graph.Relationship{
			ID:       int64(i + 1),
			SourceID: src,
			TargetID: dst,
			Weight:   r.Intn(5) + 1,
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = sna.DetectCommunities(nodes, edges)
	}
}

// BenchmarkBrandesCentrality_1000Nodes замеряет расчёт посреднической центральности
// на графе из 1 000 узлов и 3 000 рёбер (SLA <= 40 мс).
func BenchmarkBrandesCentrality_1000Nodes(b *testing.B) {
	const nodeCount = 1000
	const edgeCount = 3000

	r := rand.New(rand.NewSource(777))
	nodes := make([]*graph.Node, nodeCount)
	for i := 0; i < nodeCount; i++ {
		id := int64(i + 1)
		nodes[i] = &graph.Node{ContactID: id}
	}

	edges := make([]*graph.Relationship, edgeCount)
	for i := 0; i < edgeCount; i++ {
		src := int64(r.Intn(nodeCount) + 1)
		dst := int64(r.Intn(nodeCount) + 1)
		if src == dst {
			dst = (dst % int64(nodeCount)) + 1
		}
		edges[i] = &graph.Relationship{
			ID:       int64(i + 1),
			SourceID: src,
			TargetID: dst,
			Weight:   r.Intn(5) + 1,
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = sna.ComputeBetweennessCentrality(nodes, edges)
	}
}
