// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sna

import (
	"container/heap"
	"math"

	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
)

// RelModifier возвращает коэффициент γ(τ) типа связи для формулы сопротивления интро.
func RelModifier(t graph.RelationshipType) float64 {
	switch t {
	case graph.RelIntroducedBy, graph.RelMentor, graph.RelFounderOf, graph.RelFamily:
		return 0.80
	case graph.RelFriend, graph.RelPartner, graph.RelColleague:
		return 1.00
	case graph.RelImplicitColleague:
		return 1.65
	default:
		return 1.00
	}
}

// EdgeResistance вычисляет сопротивление интро по формуле:
// R(e) = ((6 - w(e)) / 5.0)^1.5 * γ(τ(e)) + δ_hop, где δ_hop = 0.35.
func EdgeResistance(weight int, relType graph.RelationshipType) float64 {
	w := float64(weight)
	if w < 1 {
		w = 1
	} else if w > 5 {
		w = 5
	}
	base := (6.0 - w) / 5.0
	gamma := RelModifier(relType)
	const deltaHop = 0.35
	return math.Pow(base, 1.5)*gamma + deltaHop
}

// CompoundTrust вычисляет совокупный индекс доверия цепочки знакомств P:
// CompoundTrust(P) = 0.90^(k-1) * ∏ (w(e_i) / 5.0).
func CompoundTrust(weights []int) float64 {
	k := len(weights)
	if k == 0 {
		return 1.0
	}
	trust := math.Pow(0.90, float64(k-1))
	for _, w := range weights {
		val := float64(w)
		if val < 1 {
			val = 1
		} else if val > 5 {
			val = 5
		}
		trust *= val / 5.0
	}
	return trust
}

type dijkstraItem struct {
	contactID int64
	cost      float64
	index     int
}

type dijkstraPQ []*dijkstraItem

func (pq dijkstraPQ) Len() int           { return len(pq) }
func (pq dijkstraPQ) Less(i, j int) bool { return pq[i].cost < pq[j].cost }
func (pq dijkstraPQ) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *dijkstraPQ) Push(x any) {
	n := len(*pq)
	item := x.(*dijkstraItem)
	item.index = n
	*pq = append(*pq, item)
}
func (pq *dijkstraPQ) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

type edgeNeighbor struct {
	targetID    int64
	relType     graph.RelationshipType
	weight      int
	contextNote string
	resistance  float64
}

type hopRecord struct {
	fromID      int64
	relType     graph.RelationshipType
	weight      int
	contextNote string
	resistance  float64
}

// FindIntroPath выполняет поиск пути максимального доверия (минимального сопротивления интро)
// с использованием модифицированного алгоритма Дейкстры на бинарной куче.
func FindIntroPath(
	sourceID, targetID int64,
	nodes map[int64]*graph.Node,
	edges []*graph.Relationship,
) (*graph.IntroPath, error) {
	if sourceID == targetID {
		return &graph.IntroPath{
			SourceID:        sourceID,
			TargetID:        targetID,
			Hops:            nil,
			TotalResistance: 0.0,
			CompoundTrust:   1.0,
		}, nil
	}

	if _, ok := nodes[sourceID]; !ok {
		return nil, graph.ErrNodeNotFound
	}
	if _, ok := nodes[targetID]; !ok {
		return nil, graph.ErrNodeNotFound
	}

	adj := make(map[int64][]edgeNeighbor)
	for _, e := range edges {
		if e == nil || e.Validate() != nil {
			continue
		}
		res := EdgeResistance(e.Weight, e.Type)
		adj[e.SourceID] = append(adj[e.SourceID], edgeNeighbor{
			targetID:    e.TargetID,
			relType:     e.Type,
			weight:      e.Weight,
			contextNote: e.ContextNote,
			resistance:  res,
		})
		if !e.IsDirected {
			adj[e.TargetID] = append(adj[e.TargetID], edgeNeighbor{
				targetID:    e.SourceID,
				relType:     e.Type,
				weight:      e.Weight,
				contextNote: e.ContextNote,
				resistance:  res,
			})
		}
	}

	dist := make(map[int64]float64)
	pred := make(map[int64]hopRecord)
	visited := make(map[int64]bool)

	pq := &dijkstraPQ{}
	heap.Init(pq)

	dist[sourceID] = 0.0
	heap.Push(pq, &dijkstraItem{contactID: sourceID, cost: 0.0})

	found := false

	for pq.Len() > 0 {
		top := heap.Pop(pq).(*dijkstraItem)
		u := top.contactID

		if visited[u] {
			continue
		}
		visited[u] = true

		if u == targetID {
			found = true
			break
		}

		for _, edge := range adj[u] {
			v := edge.targetID
			if visited[v] {
				continue
			}

			newCost := dist[u] + edge.resistance
			currentDist, exists := dist[v]
			if !exists || newCost < currentDist {
				dist[v] = newCost
				pred[v] = hopRecord{
					fromID:      u,
					relType:     edge.relType,
					weight:      edge.weight,
					contextNote: edge.contextNote,
					resistance:  edge.resistance,
				}
				heap.Push(pq, &dijkstraItem{contactID: v, cost: newCost})
			}
		}
	}

	if !found {
		return nil, graph.ErrIntroPathNotFound
	}

	var hops []graph.IntroHop
	var weights []int
	curr := targetID

	for curr != sourceID {
		record, ok := pred[curr]
		if !ok {
			return nil, graph.ErrIntroPathNotFound
		}
		hop := graph.IntroHop{
			FromContactID: record.fromID,
			ToContactID:   curr,
			ViaEdgeType:   record.relType,
			EdgeWeight:    record.weight,
			ContextNote:   record.contextNote,
		}
		hops = append(hops, hop)
		weights = append(weights, record.weight)
		curr = record.fromID
	}

	// Разворачиваем hops в хронологический порядок от источника к цели
	for i, j := 0, len(hops)-1; i < j; i, j = i+1, j-1 {
		hops[i], hops[j] = hops[j], hops[i]
	}
	for i, j := 0, len(weights)-1; i < j; i, j = i+1, j-1 {
		weights[i], weights[j] = weights[j], weights[i]
	}

	return &graph.IntroPath{
		SourceID:        sourceID,
		TargetID:        targetID,
		Hops:            hops,
		TotalResistance: dist[targetID],
		CompoundTrust:   CompoundTrust(weights),
	}, nil
}
