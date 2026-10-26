// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package graph

import (
	"math"
	"testing"
)

func TestQuadTree_BoundingBox(t *testing.T) {
	bb := BoundingBox{MinX: -100, MinY: -50, MaxX: 100, MaxY: 150}
	if bb.Width() != 200 {
		t.Fatalf("ожидалась ширина 200, получено: %f", bb.Width())
	}
	if bb.Height() != 200 {
		t.Fatalf("ожидалась высота 200, получено: %f", bb.Height())
	}
	if !bb.Contains(0, 0) || !bb.Contains(-100, -50) || !bb.Contains(100, 150) {
		t.Fatalf("точка внутри границ должна возвращать true")
	}
	if bb.Contains(-101, 0) || bb.Contains(0, 151) {
		t.Fatalf("точка вне границ должна возвращать false")
	}
}

func TestQuadTree_InsertAndStructure(t *testing.T) {
	tree := NewQuadTree(64)
	tree.ResetWithBounds(-100, -100, 100, 100)

	n1 := &Node{ContactID: 1, PosX: -50, PosY: -50, Mass: 2.0}
	n2 := &Node{ContactID: 2, PosX: 50, PosY: 50, Mass: 3.0}

	tree.Insert(n1)
	root := tree.Root()
	if root == nil || !root.IsLeaf || root.Body != n1 || root.TotalMass != 2.0 {
		t.Fatalf("некорректная вставка первого узла в корень: %+v", root)
	}

	tree.Insert(n2)
	if root.IsLeaf {
		t.Fatalf("корень должен был разделиться на поддеревья")
	}
	if root.TotalMass != 5.0 {
		t.Fatalf("ожидалась суммарная масса 5.0, получено: %f", root.TotalMass)
	}
	// Проверка центра масс: (-50*2 + 50*3) / 5 = 50 / 5 = 10
	if math.Abs(root.CenterX-10.0) > 1e-6 || math.Abs(root.CenterY-10.0) > 1e-6 {
		t.Fatalf("неверный расчёт центра масс: CenterX=%f, CenterY=%f", root.CenterX, root.CenterY)
	}
}

func TestQuadTree_IdenticalPositions(t *testing.T) {
	tree := NewQuadTree(128)
	tree.ResetWithBounds(-100, -100, 100, 100)

	n1 := &Node{ContactID: 1, PosX: 10, PosY: 10, Mass: 2.0}
	n2 := &Node{ContactID: 2, PosX: 10, PosY: 10, Mass: 2.0}

	tree.Insert(n1)
	tree.Insert(n2)

	root := tree.Root()
	if root.TotalMass != 4.0 {
		t.Fatalf("ожидалась общая масса 4.0, получено: %f", root.TotalMass)
	}
}

func TestQuadTree_RepulsionAccuracy(t *testing.T) {
	tree := NewQuadTree(64)
	tree.ResetWithBounds(-500, -500, 500, 500)

	n1 := &Node{ContactID: 1, PosX: 0, PosY: 0, Mass: 1.0}
	n2 := &Node{ContactID: 2, PosX: 100, PosY: 0, Mass: 2.0}

	tree.Insert(n2)

	// Аналитическое отталкивание n1 от n2 (n1 в x=0, n2 в x=100)
	// dx = 0 - 100 = -100, dy = 0. dist = 100. force = (kr * m1 * m2) / dist^2
	kr := 10000.0
	fx, fy := tree.ComputeRepulsion(n1, 0.7, kr)

	expectedForce := (kr * 1.0 * 2.0) / (100.0*100.0 + 1e-4)
	expectedFx := (-100.0 / math.Sqrt(100.0*100.0+1e-4)) * expectedForce

	if math.Abs(fy) > 1e-6 {
		t.Fatalf("fy должна быть равна 0, получено: %f", fy)
	}
	if math.Abs(fx-expectedFx) > 0.01 {
		t.Fatalf("ожидалась сила fx=%f, получено: %f", expectedFx, fx)
	}
}

func TestQuadTree_EmptyAndSelfRepulsion(t *testing.T) {
	tree := NewQuadTree(16)
	tree.ResetWithBounds(-100, -100, 100, 100)

	n1 := &Node{ContactID: 1, PosX: 0, PosY: 0, Mass: 1.0}

	fx, fy := tree.ComputeRepulsion(n1, 0.7, 5000.0)
	if fx != 0 || fy != 0 {
		t.Fatalf("пустое дерево не должно создавать силу")
	}

	tree.Insert(n1)
	// Отталкивание тела от самого себя обязано быть строго 0
	fx, fy = tree.ComputeRepulsion(n1, 0.7, 5000.0)
	if fx != 0 || fy != 0 {
		t.Fatalf("тело не должно отталкиваться от самого себя")
	}

	// Nil дерево
	var nilTree *QuadTree
	fx, fy = nilTree.ComputeRepulsion(n1, 0.7, 5000.0)
	if fx != 0 || fy != 0 {
		t.Fatalf("nil дерево должно возвращать 0, 0")
	}
}

func TestQuadTree_ZeroAllocPool(t *testing.T) {
	nodes := make([]*Node, 100)
	for i := 0; i < 100; i++ {
		nodes[i] = &Node{
			ContactID: int64(i + 1),
			PosX:      float64((i % 10) * 20),
			PosY:      float64((i / 10) * 20),
			Mass:      1.25,
		}
	}

	tree := NewQuadTree(512)
	bounds := BoundingBox{MinX: -100, MinY: -100, MaxX: 300, MaxY: 300}

	// Прогрев пула
	tree.Reset(bounds)
	for _, n := range nodes {
		tree.Insert(n)
	}

	// Замер аллокаций при перестроении квадродерева
	allocs := testing.AllocsPerRun(10, func() {
		tree.Reset(bounds)
		for _, n := range nodes {
			tree.Insert(n)
		}
	})

	if allocs > 0 {
		t.Fatalf("ожидалось 0 аллокаций/кадр благодаря flat pool, получено: %f", allocs)
	}
}
