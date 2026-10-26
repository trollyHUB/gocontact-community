// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sna

import (
	"sort"

	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
)

// DetectCommunities производит автоматическую кластеризацию узлов графа по алгоритму Лувена,
// максимизируя модулярность Ньюмана — Гирвана Q, и присваивает компактные CommunityID ∈ [0..K-1].
func DetectCommunities(
	nodes []*graph.Node,
	edges []*graph.Relationship,
) map[int64]int {
	result := make(map[int64]int, len(nodes))
	if len(nodes) == 0 {
		return result
	}

	// 1. Построение взвешенной матрицы смежности
	adj := make(map[int64]map[int64]float64, len(nodes))
	k := make(map[int64]float64, len(nodes)) // Взвешенная степень узла
	totalWeight := 0.0

	for _, n := range nodes {
		if n == nil || n.ContactID <= 0 {
			continue
		}
		adj[n.ContactID] = make(map[int64]float64)
	}

	for _, e := range edges {
		if e == nil {
			continue
		}
		w := float64(e.Weight)
		if w < 1 {
			w = 1
		}
		adj[e.SourceID][e.TargetID] += w
		k[e.SourceID] += w
		if !e.IsDirected {
			adj[e.TargetID][e.SourceID] += w
			k[e.TargetID] += w
			totalWeight += w
		} else {
			totalWeight += w * 0.5
		}
	}

	m2 := 2.0 * totalWeight // 2m
	if m2 <= 0 {
		// Нет связей: каждый узел в своём отдельном сообществе
		for i, n := range nodes {
			if n != nil {
				result[n.ContactID] = i
				n.CommunityID = i
			}
		}
		return result
	}

	// 2. Начальное состояние: каждый узел в своем уникальном сообществе
	comm := make(map[int64]int64, len(nodes))
	sTot := make(map[int64]float64, len(nodes)) // Сумма степеней узлов сообщества

	for _, n := range nodes {
		if n == nil || n.ContactID <= 0 {
			continue
		}
		id := n.ContactID
		comm[id] = id
		sTot[id] = k[id]
	}

	// 3. Фаза итеративной оптимизации прироста модулярности ΔQ
	improved := true
	passes := 0
	const maxPasses = 15

	for improved && passes < maxPasses {
		improved = false
		passes++

		for _, n := range nodes {
			if n == nil || n.ContactID <= 0 {
				continue
			}
			u := n.ContactID
			cOrig := comm[u]
			ku := k[u]

			// Вычисляем связи u с его текущим сообществом
			kInOrig := 0.0
			for v, w := range adj[u] {
				if comm[v] == cOrig && v != u {
					kInOrig += w
				}
			}

			// Временно удаляем u из его сообщества
			sTot[cOrig] -= ku

			// Находим соседние сообщества и считаем связь u с каждым из них
			commWeight := make(map[int64]float64)
			for v, w := range adj[u] {
				if v != u {
					commWeight[comm[v]] += w
				}
			}

			bestComm := cOrig
			bestGain := 0.0

			for cCandidate, kInCand := range commWeight {
				// Прирост модулярности ΔQ = k_{i,in} - (Σ_tot * k_i) / (2m)
				gain := kInCand - (sTot[cCandidate]*ku)/m2
				// Вычитаем потерю от выхода из оригинального сообщества
				loss := kInOrig - (sTot[cOrig]*ku)/m2
				diff := gain - loss

				if diff > bestGain {
					bestGain = diff
					bestComm = cCandidate
				}
			}

			// Перемещаем u в сообщество с максимальным приростом
			comm[u] = bestComm
			sTot[bestComm] += ku

			if bestComm != cOrig {
				improved = true
			}
		}
	}

	// 4. Компактификация идентификаторов сообществ в диапазон [0..K-1]
	uniqueComms := make(map[int64]bool)
	for _, c := range comm {
		uniqueComms[c] = true
	}

	sortedComms := make([]int64, 0, len(uniqueComms))
	for c := range uniqueComms {
		sortedComms = append(sortedComms, c)
	}
	sort.Slice(sortedComms, func(i, j int) bool { return sortedComms[i] < sortedComms[j] })

	compactMap := make(map[int64]int, len(sortedComms))
	for idx, c := range sortedComms {
		compactMap[c] = idx
	}

	for _, n := range nodes {
		if n == nil || n.ContactID <= 0 {
			continue
		}
		cID := compactMap[comm[n.ContactID]]
		result[n.ContactID] = cID
		n.CommunityID = cID
	}

	return result
}
