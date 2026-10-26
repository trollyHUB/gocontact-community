// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package memory

import (
	"sync"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
)

// TopologyCache — высокопроизводительный потокобезопасный L1 In-Memory кэш графа
// для расчёта физики Barnes-Hut (60 FPS) и мгновенных SNA-запросов (Dijkstra, Louvain, Brandes).
type TopologyCache struct {
	mu          sync.RWMutex
	version     uint64
	lastUpdated time.Time
	nodes       map[int64]*graph.Node
	adjOut      map[int64][]*graph.Relationship // Исходящие и ненаправленные рёбра
	adjIn       map[int64][]*graph.Relationship // Входящие и ненаправленные рёбра
}

// NewTopologyCache создаёт новый пустой экземпляр кэша топологии.
func NewTopologyCache() *TopologyCache {
	return &TopologyCache{
		nodes:       make(map[int64]*graph.Node),
		adjOut:      make(map[int64][]*graph.Relationship),
		adjIn:       make(map[int64][]*graph.Relationship),
		lastUpdated: time.Now().UTC(),
	}
}

// Reset атомарно перезаполняет весь кэш набором узлов и рёбер, инкрементируя версию.
func (c *TopologyCache) Reset(nodes []*graph.Node, edges []*graph.Relationship) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nodes = make(map[int64]*graph.Node, len(nodes))
	c.adjOut = make(map[int64][]*graph.Relationship)
	c.adjIn = make(map[int64][]*graph.Relationship)

	for _, n := range nodes {
		if n == nil || n.ContactID <= 0 {
			continue
		}
		c.nodes[n.ContactID] = cloneNode(n)
	}

	for _, e := range edges {
		if e == nil || e.Validate() != nil {
			continue
		}
		c.insertEdgeInternal(cloneRelationship(e))
	}

	c.version++
	c.lastUpdated = time.Now().UTC()
}

// AddOrUpdateNode добавляет или обновляет узел графа в кэше.
func (c *TopologyCache) AddOrUpdateNode(node *graph.Node) {
	if node == nil || node.ContactID <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nodes[node.ContactID] = cloneNode(node)
	c.version++
	c.lastUpdated = time.Now().UTC()
}

// GetNode возвращает глубокую копию узла по ContactID.
func (c *TopologyCache) GetNode(contactID int64) (*graph.Node, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	node, ok := c.nodes[contactID]
	if !ok {
		return nil, false
	}
	return cloneNode(node), true
}

// RemoveNode удаляет узел и все связанные с ним рёбра.
func (c *TopologyCache) RemoveNode(contactID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.nodes[contactID]; !exists {
		return
	}

	delete(c.nodes, contactID)

	// Удаляем исходящие из этого узла
	delete(c.adjOut, contactID)
	// Удаляем входящие в этот узел
	delete(c.adjIn, contactID)

	// Очищаем упоминания узла в списках остальных узлов
	for u, edges := range c.adjOut {
		filtered := edges[:0]
		for _, e := range edges {
			if e.SourceID != contactID && e.TargetID != contactID {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) > 0 {
			c.adjOut[u] = filtered
		} else {
			delete(c.adjOut, u)
		}
	}

	for u, edges := range c.adjIn {
		filtered := edges[:0]
		for _, e := range edges {
			if e.SourceID != contactID && e.TargetID != contactID {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) > 0 {
			c.adjIn[u] = filtered
		} else {
			delete(c.adjIn, u)
		}
	}

	c.version++
	c.lastUpdated = time.Now().UTC()
}

// AddOrUpdateRelationship добавляет или обновляет связь в списках смежности.
func (c *TopologyCache) AddOrUpdateRelationship(rel *graph.Relationship) {
	if rel == nil || rel.Validate() != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	// Удаляем старую версию связи с таким же ID или парой узлов, если есть
	c.removeEdgeInternal(rel.ID, rel.SourceID, rel.TargetID, rel.Type)

	c.insertEdgeInternal(cloneRelationship(rel))
	c.version++
	c.lastUpdated = time.Now().UTC()
}

// RemoveRelationship удаляет связь по её ID.
func (c *TopologyCache) RemoveRelationship(relID int64) {
	if relID <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.removeEdgeByIDInternal(relID)
	c.version++
	c.lastUpdated = time.Now().UTC()
}

// GetAdjacentRelationships возвращает все инцидентные связи контакта (исходящие и входящие без дублей).
func (c *TopologyCache) GetAdjacentRelationships(contactID int64) []*graph.Relationship {
	c.mu.RLock()
	defer c.mu.RUnlock()

	seen := make(map[int64]bool)
	var result []*graph.Relationship

	// Исходящие
	for _, e := range c.adjOut[contactID] {
		if !seen[e.ID] {
			seen[e.ID] = true
			result = append(result, cloneRelationship(e))
		}
	}

	// Входящие
	for _, e := range c.adjIn[contactID] {
		if !seen[e.ID] {
			seen[e.ID] = true
			result = append(result, cloneRelationship(e))
		}
	}

	return result
}

// GetOutgoingRelationships возвращает исходящие (и двунаправленные) рёбра для узла.
func (c *TopologyCache) GetOutgoingRelationships(contactID int64) []*graph.Relationship {
	c.mu.RLock()
	defer c.mu.RUnlock()

	edges := c.adjOut[contactID]
	result := make([]*graph.Relationship, len(edges))
	for i, e := range edges {
		result[i] = cloneRelationship(e)
	}
	return result
}

// GetIncomingRelationships возвращает входящие (и двунаправленные) рёбра для узла.
func (c *TopologyCache) GetIncomingRelationships(contactID int64) []*graph.Relationship {
	c.mu.RLock()
	defer c.mu.RUnlock()

	edges := c.adjIn[contactID]
	result := make([]*graph.Relationship, len(edges))
	for i, e := range edges {
		result[i] = cloneRelationship(e)
	}
	return result
}

// Snapshot делает мгновенный снимок топологии графа для 60 FPS физического движка.
// Блокировка чтения mu.RLock удерживается минимальное время (<15 мкс).
func (c *TopologyCache) Snapshot() (map[int64]*graph.Node, []*graph.Relationship, uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodesCopy := make(map[int64]*graph.Node, len(c.nodes))
	for k, v := range c.nodes {
		nodesCopy[k] = cloneNode(v)
	}

	seen := make(map[int64]bool)
	var edgesCopy []*graph.Relationship

	for _, edges := range c.adjOut {
		for _, e := range edges {
			if !seen[e.ID] {
				seen[e.ID] = true
				edgesCopy = append(edgesCopy, cloneRelationship(e))
			}
		}
	}
	for _, edges := range c.adjIn {
		for _, e := range edges {
			if !seen[e.ID] {
				seen[e.ID] = true
				edgesCopy = append(edgesCopy, cloneRelationship(e))
			}
		}
	}

	return nodesCopy, edgesCopy, c.version
}

// Version возвращает текущую ревизию топологии.
func (c *TopologyCache) Version() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.version
}

// LastUpdated возвращает время последней модификации топологии.
func (c *TopologyCache) LastUpdated() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUpdated
}

// NodeCount возвращает количество узлов в кэше.
func (c *TopologyCache) NodeCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.nodes)
}

// RelationshipCount возвращает количество уникальных связей в кэше.
func (c *TopologyCache) RelationshipCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	seen := make(map[int64]bool)
	for _, edges := range c.adjOut {
		for _, e := range edges {
			seen[e.ID] = true
		}
	}
	for _, edges := range c.adjIn {
		for _, e := range edges {
			seen[e.ID] = true
		}
	}
	return len(seen)
}

// Вспомогательные внутренние методы (вызываются под mu.Lock):

func (c *TopologyCache) insertEdgeInternal(rel *graph.Relationship) {
	c.adjOut[rel.SourceID] = append(c.adjOut[rel.SourceID], rel)
	c.adjIn[rel.TargetID] = append(c.adjIn[rel.TargetID], rel)

	if !rel.IsDirected {
		c.adjOut[rel.TargetID] = append(c.adjOut[rel.TargetID], rel)
		c.adjIn[rel.SourceID] = append(c.adjIn[rel.SourceID], rel)
	}
}

func (c *TopologyCache) removeEdgeInternal(id, sourceID, targetID int64, relType graph.RelationshipType) {
	if id > 0 {
		c.removeEdgeByIDInternal(id)
		return
	}

	removeMatching := func(list []*graph.Relationship) []*graph.Relationship {
		filtered := list[:0]
		for _, e := range list {
			if (e.SourceID == sourceID && e.TargetID == targetID && e.Type == relType) ||
				(!e.IsDirected && e.SourceID == targetID && e.TargetID == sourceID && e.Type == relType) {
				continue
			}
			filtered = append(filtered, e)
		}
		return filtered
	}

	c.adjOut[sourceID] = removeMatching(c.adjOut[sourceID])
	c.adjIn[targetID] = removeMatching(c.adjIn[targetID])
	c.adjOut[targetID] = removeMatching(c.adjOut[targetID])
	c.adjIn[sourceID] = removeMatching(c.adjIn[sourceID])
}

func (c *TopologyCache) removeEdgeByIDInternal(relID int64) {
	removeByID := func(list []*graph.Relationship) []*graph.Relationship {
		filtered := list[:0]
		for _, e := range list {
			if e.ID != relID {
				filtered = append(filtered, e)
			}
		}
		return filtered
	}

	for k, list := range c.adjOut {
		c.adjOut[k] = removeByID(list)
	}
	for k, list := range c.adjIn {
		c.adjIn[k] = removeByID(list)
	}
}

func cloneNode(n *graph.Node) *graph.Node {
	if n == nil {
		return nil
	}
	cp := *n
	return &cp
}

func cloneRelationship(r *graph.Relationship) *graph.Relationship {
	if r == nil {
		return nil
	}
	cp := *r
	return &cp
}
