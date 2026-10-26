// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package graph

import (
	"math"
)

// BoundingBox определяет двумерную прямоугольную область пространства для квадродерева.
type BoundingBox struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

// Width возвращает ширину ограничивающего прямоугольника.
func (b BoundingBox) Width() float64 {
	return b.MaxX - b.MinX
}

// Height возвращает высоту ограничивающего прямоугольника.
func (b BoundingBox) Height() float64 {
	return b.MaxY - b.MinY
}

// Contains проверяет, находится ли точка внутри прямоугольника (включая границы).
func (b BoundingBox) Contains(x, y float64) bool {
	return x >= b.MinX && x <= b.MaxX && y >= b.MinY && y <= b.MaxY
}

// QuadNode представляет узел пространственного квадродерева Барнса — Хата.
// Святая зона Clean Architecture: 0 внешних зависимостей, только стандартная библиотека math.
type QuadNode struct {
	MinX      float64
	MinY      float64
	MaxX      float64
	MaxY      float64
	Width     float64
	CenterX   float64
	CenterY   float64
	TotalMass float64
	Body      *Node
	IsLeaf    bool

	NW *QuadNode
	NE *QuadNode
	SW *QuadNode
	SE *QuadNode

	Children [4]*QuadNode
}

// ComputeRepulsion рекурсивно вычисляет силу отталкивания узла n от квадранта q
// в строгом соответствии с формулой Барнса — Хата из спецификации 04_BACKEND_DOMAIN_AND_ALGORITHMS.md.
func (q *QuadNode) ComputeRepulsion(n *Node, theta, kr float64) (fx, fy float64) {
	if q == nil || q.TotalMass == 0 || (q.IsLeaf && q.Body == n) {
		return 0, 0
	}
	dx := n.PosX - q.CenterX
	dy := n.PosY - q.CenterY
	distSq := dx*dx + dy*dy + 1e-4
	dist := math.Sqrt(distSq)

	if q.IsLeaf || (q.Width/dist) < theta {
		force := (kr * n.Mass * q.TotalMass) / distSq
		return (dx / dist) * force, (dy / dist) * force
	}
	for i := 0; i < 4; i++ {
		if q.Children[i] != nil {
			cfx, cfy := q.Children[i].ComputeRepulsion(n, theta, kr)
			fx += cfx
			fy += cfy
		}
	}
	return fx, fy
}

func (q *QuadNode) getQuadrant(x, y float64) int {
	midX := (q.MinX + q.MaxX) * 0.5
	midY := (q.MinY + q.MaxY) * 0.5
	if y >= midY {
		if x <= midX {
			return 0 // NW
		}
		return 1 // NE
	}
	if x <= midX {
		return 2 // SW
	}
	return 3 // SE
}

// QuadTree представляет 2D-квадродерево с поддержкой плоского пула памяти
// для достижения 0 аллокаций на кадр симуляции (0 allocs/frame).
type QuadTree struct {
	root    *QuadNode
	pool    []QuadNode
	poolIdx int
}

// NewQuadTree инициализирует квадродерево с предвыделенным плоским пулом узлов.
func NewQuadTree(capacity int) *QuadTree {
	if capacity <= 0 {
		capacity = 256
	}
	return &QuadTree{
		pool: make([]QuadNode, capacity),
	}
}

// Reset сбрасывает индекс пула и инициализирует корень дерева заданными границами.
func (t *QuadTree) Reset(bounds BoundingBox) {
	t.poolIdx = 0
	if bounds.MaxX <= bounds.MinX {
		bounds.MaxX = bounds.MinX + 1.0
	}
	if bounds.MaxY <= bounds.MinY {
		bounds.MaxY = bounds.MinY + 1.0
	}
	t.root = t.allocNode(bounds.MinX, bounds.MinY, bounds.MaxX, bounds.MaxY)
}

// ResetWithBounds инициализирует корень дерева с явными числовыми границами.
func (t *QuadTree) ResetWithBounds(minX, minY, maxX, maxY float64) {
	t.Reset(BoundingBox{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY})
}

// Root возвращает корневой узел квадродерева.
func (t *QuadTree) Root() *QuadNode {
	return t.root
}

// Insert добавляет узел графа в квадродерево с агрегацией массы и центра масс.
func (t *QuadTree) Insert(node *Node) {
	if t.root == nil || node == nil || node.Mass <= 0 {
		return
	}
	t.insert(t.root, node, 0)
}

// ComputeRepulsion вычисляет суммарную силу отталкивания узла от всего дерева.
func (t *QuadTree) ComputeRepulsion(n *Node, theta, kr float64) (fx, fy float64) {
	if t == nil || t.root == nil {
		return 0, 0
	}
	return t.root.ComputeRepulsion(n, theta, kr)
}

func (t *QuadTree) allocNode(minX, minY, maxX, maxY float64) *QuadNode {
	if t.poolIdx >= len(t.pool) {
		t.pool = append(t.pool, QuadNode{})
	}
	n := &t.pool[t.poolIdx]
	t.poolIdx++

	width := maxX - minX
	if h := maxY - minY; h > width {
		width = h
	}

	*n = QuadNode{
		MinX:      minX,
		MinY:      minY,
		MaxX:      maxX,
		MaxY:      maxY,
		Width:     width,
		CenterX:   0,
		CenterY:   0,
		TotalMass: 0,
		Body:      nil,
		IsLeaf:    true,
	}
	return n
}

func (t *QuadTree) subdivide(q *QuadNode) {
	midX := (q.MinX + q.MaxX) * 0.5
	midY := (q.MinY + q.MaxY) * 0.5

	q.NW = t.allocNode(q.MinX, midY, midX, q.MaxY)
	q.NE = t.allocNode(midX, midY, q.MaxX, q.MaxY)
	q.SW = t.allocNode(q.MinX, q.MinY, midX, midY)
	q.SE = t.allocNode(midX, q.MinY, q.MaxX, midY)

	q.Children[0] = q.NW
	q.Children[1] = q.NE
	q.Children[2] = q.SW
	q.Children[3] = q.SE
	q.IsLeaf = false
}

func (t *QuadTree) insert(q *QuadNode, node *Node, depth int) {
	if q.IsLeaf && q.Body == nil {
		q.Body = node
		q.TotalMass = node.Mass
		q.CenterX = node.PosX
		q.CenterY = node.PosY
		return
	}

	if depth >= 32 || q.Width < 1e-6 {
		newMass := q.TotalMass + node.Mass
		q.CenterX = (q.CenterX*q.TotalMass + node.PosX*node.Mass) / newMass
		q.CenterY = (q.CenterY*q.TotalMass + node.PosY*node.Mass) / newMass
		q.TotalMass = newMass
		return
	}

	if q.IsLeaf {
		oldBody := q.Body
		q.Body = nil
		t.subdivide(q)

		idxOld := q.getQuadrant(oldBody.PosX, oldBody.PosY)
		t.insert(q.Children[idxOld], oldBody, depth+1)

		idxNew := q.getQuadrant(node.PosX, node.PosY)
		t.insert(q.Children[idxNew], node, depth+1)

		newMass := q.TotalMass + node.Mass
		q.CenterX = (q.CenterX*q.TotalMass + node.PosX*node.Mass) / newMass
		q.CenterY = (q.CenterY*q.TotalMass + node.PosY*node.Mass) / newMass
		q.TotalMass = newMass
		return
	}

	newMass := q.TotalMass + node.Mass
	q.CenterX = (q.CenterX*q.TotalMass + node.PosX*node.Mass) / newMass
	q.CenterY = (q.CenterY*q.TotalMass + node.PosY*node.Mass) / newMass
	q.TotalMass = newMass

	idx := q.getQuadrant(node.PosX, node.PosY)
	t.insert(q.Children[idx], node, depth+1)
}
