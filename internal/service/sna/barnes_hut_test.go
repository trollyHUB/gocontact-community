// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sna_test

import (
	"context"
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

func TestPhysicsSimulation_MassAndStep(t *testing.T) {
	cfg := sna.DefaultSimulationConfig()
	sim := sna.NewPhysicsSimulation(cfg, nil)

	nodes := []*graph.Node{
		{ContactID: 1, PosX: -30, PosY: 0},
		{ContactID: 2, PosX: 30, PosY: 0},
	}
	edges := []*graph.Relationship{
		{ID: 1, SourceID: 1, TargetID: 2, Weight: 5},
	}

	sim.UpdateTopology(nodes, edges)

	snap := sim.GetLatestSnapshot()
	if len(snap.Positions) != 2 {
		t.Fatalf("ожидалось 2 позиции в снапшоте, получено: %d", len(snap.Positions))
	}

	initialAlpha := snap.Alpha
	sim.Step()

	snapAfter := sim.GetLatestSnapshot()
	if snapAfter.Alpha >= initialAlpha {
		t.Fatalf("температура alpha должна была уменьшиться после шага")
	}
	if snapAfter.Version <= snap.Version {
		t.Fatalf("версия снапшота должна инкрементироваться")
	}
}

func TestPhysicsSimulation_SpringAttraction(t *testing.T) {
	cfg := sna.DefaultSimulationConfig()
	cfg.Kr = 0.0 // Отключаем отталкивание, чтобы изолированно проверить силу пружины
	cfg.CenterGravity = 0.0
	sim := sna.NewPhysicsSimulation(cfg, nil)

	// Размещаем узлы дальше идеальной длины L0 (для w=5 L0 = 75 * 0.6 = 45)
	// Расстояние между ними 100 > 45, значит пружина должна притягивать
	nodes := []*graph.Node{
		{ContactID: 1, PosX: -50, PosY: 0},
		{ContactID: 2, PosX: 50, PosY: 0},
	}
	edges := []*graph.Relationship{
		{ID: 1, SourceID: 1, TargetID: 2, Weight: 5},
	}

	sim.UpdateTopology(nodes, edges)
	sim.Step()

	snap := sim.GetLatestSnapshot()
	p1 := snap.Positions[1]
	p2 := snap.Positions[2]

	newDist := math.Abs(p2.PosX - p1.PosX)
	if newDist >= 100.0 {
		t.Fatalf("пружина Гука должна была притянуть узлы ближе (исходно 100), получено: %f", newDist)
	}
}

func TestPhysicsSimulation_IdleTransition(t *testing.T) {
	cfg := sna.DefaultSimulationConfig()
	cfg.AlphaMin = 0.5 // Повышаем порог для быстрого теста перехода в Idle
	cfg.AlphaDecay = 0.5
	cfg.TickerInterval = 5 * time.Millisecond

	var idleCalled atomic.Bool
	sim := sna.NewPhysicsSimulation(cfg, func(nodes []*graph.Node) {
		idleCalled.Store(true)
	})

	nodes := []*graph.Node{
		{ContactID: 1, PosX: 0, PosY: 0},
	}
	sim.UpdateTopology(nodes, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sim.Start(ctx)
	defer sim.Stop()

	// Ожидаем срабатывания колбэка onIdle
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if idleCalled.Load() {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if !idleCalled.Load() {
		t.Fatalf("симулятор не перешел в Idle при alpha < AlphaMin")
	}
}

func TestPhysicsSimulation_PinAndMoveCommands(t *testing.T) {
	cfg := sna.DefaultSimulationConfig()
	cfg.TickerInterval = 5 * time.Millisecond
	sim := sna.NewPhysicsSimulation(cfg, nil)

	nodes := []*graph.Node{
		{ContactID: 1, PosX: 10, PosY: 10},
	}
	sim.UpdateTopology(nodes, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sim.Start(ctx)
	defer sim.Stop()

	sim.PinNode(1, true, 50, 50)
	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		snap := sim.GetLatestSnapshot()
		p1 := snap.Positions[1]
		if p1.IsPinned && p1.PosX == 50 && p1.PosY == 50 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	snap := sim.GetLatestSnapshot()
	p1 := snap.Positions[1]
	if !p1.IsPinned || p1.PosX != 50 || p1.PosY != 50 {
		t.Fatalf("команда PinNode не обновила состояние узла: %+v", p1)
	}

	sim.MoveNode(1, 100, 200)
	deadline = time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		snap2 := sim.GetLatestSnapshot()
		p2 := snap2.Positions[1]
		if p2.PosX == 100 && p2.PosY == 200 {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	snap2 := sim.GetLatestSnapshot()
	p2 := snap2.Positions[1]
	if p2.PosX != 100 || p2.PosY != 200 {
		t.Fatalf("команда MoveNode не переместила узел: %+v", p2)
	}
}
