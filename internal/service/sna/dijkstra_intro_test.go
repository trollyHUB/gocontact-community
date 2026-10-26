// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sna_test

import (
	"errors"
	"math"
	"testing"

	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

func TestEdgeResistanceFormula(t *testing.T) {
	// w = 5, RelMentor (gamma = 0.80)
	// (6-5)/5 = 0.2. 0.2^1.5 = 0.0894427. 0.0894427 * 0.8 + 0.35 = 0.42155
	r5 := sna.EdgeResistance(5, graph.RelMentor)
	expectedR5 := math.Pow(0.2, 1.5)*0.80 + 0.35
	if math.Abs(r5-expectedR5) > 1e-5 {
		t.Fatalf("ожидалось сопротивление %f, получено: %f", expectedR5, r5)
	}

	// w = 1, RelImplicitColleague (gamma = 1.65)
	// (6-1)/5 = 1.0. 1.0 * 1.65 + 0.35 = 2.00
	r1 := sna.EdgeResistance(1, graph.RelImplicitColleague)
	if math.Abs(r1-2.00) > 1e-5 {
		t.Fatalf("ожидалось сопротивление 2.00, получено: %f", r1)
	}
}

func TestCompoundTrustFormula(t *testing.T) {
	// Одиночное ребро веса 5
	ct1 := sna.CompoundTrust([]int{5})
	if math.Abs(ct1-1.0) > 1e-5 {
		t.Fatalf("для веса 5 доверие должно быть 1.0, получено: %f", ct1)
	}

	// 2 шага с весами 5 и 4:
	// 0.90^(2-1) * (5/5) * (4/5) = 0.90 * 1.0 * 0.8 = 0.72
	ct2 := sna.CompoundTrust([]int{5, 4})
	if math.Abs(ct2-0.72) > 1e-5 {
		t.Fatalf("ожидался compound trust 0.72, получено: %f", ct2)
	}
}

func TestFindIntroPath_DirectAndMultiHop(t *testing.T) {
	nodes := map[int64]*graph.Node{
		1: {ContactID: 1, FullName: "Алиса"},
		2: {ContactID: 2, FullName: "Боб"},
		3: {ContactID: 3, FullName: "Чарли"},
	}

	edges := []*graph.Relationship{
		{ID: 1, SourceID: 1, TargetID: 2, Type: graph.RelFriend, Weight: 5, IsDirected: false},
		{ID: 2, SourceID: 2, TargetID: 3, Type: graph.RelMentor, Weight: 4, IsDirected: false},
	}

	path, err := sna.FindIntroPath(1, 3, nodes, edges)
	if err != nil {
		t.Fatalf("неожиданная ошибка поиска пути: %v", err)
	}
	if len(path.Hops) != 2 {
		t.Fatalf("ожидалось 2 шага, получено: %d", len(path.Hops))
	}
	if path.Hops[0].FromContactID != 1 || path.Hops[0].ToContactID != 2 {
		t.Fatalf("неверный первый шаг: %+v", path.Hops[0])
	}
	if path.Hops[1].FromContactID != 2 || path.Hops[1].ToContactID != 3 {
		t.Fatalf("неверный второй шаг: %+v", path.Hops[1])
	}
	if math.Abs(path.CompoundTrust-0.72) > 1e-4 {
		t.Fatalf("ожидался CompoundTrust 0.72, получено: %f", path.CompoundTrust)
	}
}

func TestFindIntroPath_ChoosesHigherTrustOverShorterLowTrust(t *testing.T) {
	// Узел 1 связан с 3 напрямую слабым неявным ребром (вес 1, RelImplicitColleague -> R = 2.00)
	// И через надежного посредника 2 (1->2 вес 5 RelIntroducedBy R=0.42, 2->3 вес 5 RelMentor R=0.42 -> R_sum = 0.84)
	nodes := map[int64]*graph.Node{
		1: {ContactID: 1},
		2: {ContactID: 2},
		3: {ContactID: 3},
	}

	edges := []*graph.Relationship{
		{ID: 1, SourceID: 1, TargetID: 3, Type: graph.RelImplicitColleague, Weight: 1, IsDirected: false},
		{ID: 2, SourceID: 1, TargetID: 2, Type: graph.RelIntroducedBy, Weight: 5, IsDirected: false},
		{ID: 3, SourceID: 2, TargetID: 3, Type: graph.RelMentor, Weight: 5, IsDirected: false},
	}

	path, err := sna.FindIntroPath(1, 3, nodes, edges)
	if err != nil {
		t.Fatalf("ошибка: %v", err)
	}
	// Алгоритм обязан предпочесть 2 шага через узла 2, так как сопротивление 0.84 < 2.00
	if len(path.Hops) != 2 || path.Hops[0].ToContactID != 2 {
		t.Fatalf("алгоритм должен был выбрать путь наименьшего сопротивления через посредника 2")
	}
}

func TestFindIntroPath_EdgeCases(t *testing.T) {
	nodes := map[int64]*graph.Node{
		1: {ContactID: 1},
		2: {ContactID: 2},
		3: {ContactID: 3},
	}

	// Направленное ребро 1 -> 2 (обратный путь 2 -> 1 невозможен)
	edges := []*graph.Relationship{
		{ID: 1, SourceID: 1, TargetID: 2, Type: graph.RelReportsTo, Weight: 4, IsDirected: true},
	}

	// 1 -> 2 должно найти
	if _, err := sna.FindIntroPath(1, 2, nodes, edges); err != nil {
		t.Fatalf("путь 1->2 должен существовать: %v", err)
	}

	// 2 -> 1 должно вернуть ErrIntroPathNotFound
	_, err := sna.FindIntroPath(2, 1, nodes, edges)
	if !errors.Is(err, graph.ErrIntroPathNotFound) {
		t.Fatalf("ожидалась ошибка ErrIntroPathNotFound, получено: %v", err)
	}

	// Несуществующий узел
	_, err = sna.FindIntroPath(1, 999, nodes, edges)
	if !errors.Is(err, graph.ErrNodeNotFound) {
		t.Fatalf("ожидалась ошибка ErrNodeNotFound, получено: %v", err)
	}

	// Сам к себе
	selfPath, err := sna.FindIntroPath(1, 1, nodes, edges)
	if err != nil || len(selfPath.Hops) != 0 || selfPath.CompoundTrust != 1.0 {
		t.Fatalf("путь к себе должен быть пустым с доверием 1.0")
	}
}
