// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sna

import (
	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
)

// ComputeBetweennessCentrality вычисляет центральность по посредничеству для каждого узла
// по алгоритму Ульрика Брандеса за O(|V| * |E|) и нормализует значение в диапазон [0.0 .. 1.0].
func ComputeBetweennessCentrality(
	nodes []*graph.Node,
	edges []*graph.Relationship,
) map[int64]float64 {
	result := make(map[int64]float64, len(nodes))
	n := len(nodes)
	if n <= 2 {
		for _, node := range nodes {
			if node != nil {
				result[node.ContactID] = 0.0
				node.BetweennessScore = 0.0
			}
		}
		return result
	}

	// Построение списков смежности
	adj := make(map[int64][]int64, n)
	for _, node := range nodes {
		if node != nil && node.ContactID > 0 {
			adj[node.ContactID] = nil
			result[node.ContactID] = 0.0
		}
	}

	for _, e := range edges {
		if e == nil {
			continue
		}
		adj[e.SourceID] = append(adj[e.SourceID], e.TargetID)
		if !e.IsDirected {
			adj[e.TargetID] = append(adj[e.TargetID], e.SourceID)
		}
	}

	// Алгоритм Брандеса: обход из каждого узла s ∈ V
	for _, sNode := range nodes {
		if sNode == nil || sNode.ContactID <= 0 {
			continue
		}
		s := sNode.ContactID

		// Стек порядка обратного обхода
		stack := make([]int64, 0, n)

		// Список предшественников на кратчайших путях
		pred := make(map[int64][]int64, n)

		// Количество кратчайших путей из s в v
		sigma := make(map[int64]float64, n)
		sigma[s] = 1.0

		// Расстояние от s до v
		dist := make(map[int64]int, n)
		for _, node := range nodes {
			if node != nil {
				dist[node.ContactID] = -1
			}
		}
		dist[s] = 0

		// Очередь BFS
		queue := make([]int64, 0, n)
		queue = append(queue, s)

		for len(queue) > 0 {
			v := queue[0]
			queue = queue[1:]
			stack = append(stack, v)

			for _, w := range adj[v] {
				// w найден впервые?
				if dist[w] < 0 {
					dist[w] = dist[v] + 1
					queue = append(queue, w)
				}
				// Кратчайший путь в w через v?
				if dist[w] == dist[v]+1 {
					sigma[w] += sigma[v]
					pred[w] = append(pred[w], v)
				}
			}
		}

		// Накопление зависимостей дельта
		delta := make(map[int64]float64, n)

		for len(stack) > 0 {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			for _, v := range pred[w] {
				coeff := (sigma[v] / sigma[w]) * (1.0 + delta[w])
				delta[v] += coeff
			}

			if w != s {
				result[w] += delta[w]
			}
		}
	}

	// Для ненаправленных графов каждое ребро учитывается дважды (s->t и t->s)
	// Нормализационный делитель для графа: (|V|-1)(|V|-2)/2
	maxPossible := float64((n - 1) * (n - 2))
	if maxPossible < 1.0 {
		maxPossible = 1.0
	}

	for _, node := range nodes {
		if node == nil || node.ContactID <= 0 {
			continue
		}
		raw := result[node.ContactID]
		// Делим на 2 для ненаправленных путей
		rawScore := raw * 0.5
		normScore := (2.0 * rawScore) / maxPossible
		if normScore > 1.0 {
			normScore = 1.0
		}
		result[node.ContactID] = normScore
		node.BetweennessScore = normScore
	}

	return result
}
