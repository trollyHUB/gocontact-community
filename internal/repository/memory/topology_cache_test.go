// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package memory_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/repository/memory"
)

func TestTopologyCache_NodeOperations(t *testing.T) {
	cache := memory.NewTopologyCache()

	if cache.NodeCount() != 0 {
		t.Fatalf("ожидался пустой кэш узлов")
	}

	node1 := &graph.Node{
		ContactID:        1,
		FullName:         "Алексей Смирнов",
		Company:          "TechCorp",
		DunbarTier:       graph.DunbarSupportClique,
		BetweennessScore: 0.85,
	}

	cache.AddOrUpdateNode(node1)
	if cache.NodeCount() != 1 {
		t.Fatalf("ожидался 1 узел в кэше, получено: %d", cache.NodeCount())
	}

	// Чтение узла
	got, ok := cache.GetNode(1)
	if !ok || got.ContactID != 1 || got.FullName != "Алексей Смирнов" {
		t.Fatalf("некорректное чтение узла: %+v", got)
	}

	// Проверка глубокого копирования (мутация полученного не должна менять кэш)
	got.FullName = "Изменённое Имя"
	orig, _ := cache.GetNode(1)
	if orig.FullName != "Алексей Смирнов" {
		t.Fatalf("кэш был мутирован снаружи через возврат ссылки!")
	}

	// Удаление узла
	cache.RemoveNode(1)
	if cache.NodeCount() != 0 {
		t.Fatalf("узел не был удален")
	}
	if _, ok := cache.GetNode(1); ok {
		t.Fatalf("удалённый узел всё ещё возвращается")
	}
}

func TestTopologyCache_RelationshipOperations(t *testing.T) {
	cache := memory.NewTopologyCache()

	// Добавляем узлы
	cache.AddOrUpdateNode(&graph.Node{ContactID: 1, FullName: "A"})
	cache.AddOrUpdateNode(&graph.Node{ContactID: 2, FullName: "B"})
	cache.AddOrUpdateNode(&graph.Node{ContactID: 3, FullName: "C"})

	// Ненаправленная связь 1 <-> 2
	rel1 := &graph.Relationship{
		ID:         101,
		SourceID:   1,
		TargetID:   2,
		Type:       graph.RelFriend,
		Weight:     5,
		IsDirected: false,
	}
	cache.AddOrUpdateRelationship(rel1)

	// Направленная связь 1 -> 3
	rel2 := &graph.Relationship{
		ID:         102,
		SourceID:   1,
		TargetID:   3,
		Type:       graph.RelMentor,
		Weight:     4,
		IsDirected: true,
	}
	cache.AddOrUpdateRelationship(rel2)

	if cache.RelationshipCount() != 2 {
		t.Fatalf("ожидалось 2 уникальные связи, получено: %d", cache.RelationshipCount())
	}

	// Проверяем инцидентные связи для 1 (должны быть обе)
	adj1 := cache.GetAdjacentRelationships(1)
	if len(adj1) != 2 {
		t.Fatalf("ожидалось 2 инцидентные связи для узла 1, получено: %d", len(adj1))
	}

	// Проверяем смежность для 2 (должна быть ненаправленная 1 <-> 2)
	adj2 := cache.GetAdjacentRelationships(2)
	if len(adj2) != 1 || adj2[0].ID != 101 {
		t.Fatalf("узел 2 должен иметь связь 101, получено: %+v", adj2)
	}

	// Проверяем входящие для 3 (направленная от 1)
	in3 := cache.GetIncomingRelationships(3)
	if len(in3) != 1 || in3[0].ID != 102 {
		t.Fatalf("узел 3 должен иметь 1 входящую связь 102")
	}

	// Исходящих у 3 быть не должно
	out3 := cache.GetOutgoingRelationships(3)
	if len(out3) != 0 {
		t.Fatalf("у узла 3 не должно быть исходящих связей")
	}

	// Удаление связи по ID
	cache.RemoveRelationship(101)
	if cache.RelationshipCount() != 1 {
		t.Fatalf("ожидалась 1 оставшаяся связь, получено: %d", cache.RelationshipCount())
	}
	if len(cache.GetAdjacentRelationships(2)) != 0 {
		t.Fatalf("после удаления связи 101 узел 2 не должен иметь связей")
	}

	// Удаление узла 1 должно автоматически удалить и связь 102
	cache.RemoveNode(1)
	if len(cache.GetIncomingRelationships(3)) != 0 {
		t.Fatalf("после удаления узла 1 его инцидентная связь не очистилась из узла 3")
	}
}

func TestTopologyCache_ResetAndSnapshot(t *testing.T) {
	cache := memory.NewTopologyCache()

	nodes := []*graph.Node{
		{ContactID: 10, FullName: "X"},
		{ContactID: 20, FullName: "Y"},
	}
	edges := []*graph.Relationship{
		{ID: 1, SourceID: 10, TargetID: 20, Type: graph.RelColleague, Weight: 3},
	}

	initialVer := cache.Version()
	cache.Reset(nodes, edges)

	if cache.Version() <= initialVer {
		t.Fatalf("версия кэша должна инкрементироваться при Reset")
	}
	if cache.NodeCount() != 2 || cache.RelationshipCount() != 1 {
		t.Fatalf("некорректное состояние после Reset")
	}

	snapNodes, snapEdges, ver := cache.Snapshot()
	if len(snapNodes) != 2 || len(snapEdges) != 1 || ver != cache.Version() {
		t.Fatalf("Snapshot вернул некорректные данные: nodes=%d, edges=%d, ver=%d",
			len(snapNodes), len(snapEdges), ver)
	}

	if cache.LastUpdated().IsZero() {
		t.Fatalf("время lastUpdated не должно быть нулевым")
	}
}

func TestTopologyCache_ConcurrencyAndRace(t *testing.T) {
	cache := memory.NewTopologyCache()

	// Инициализируем начальный граф
	const initNodeCount = 20
	for i := 1; i <= initNodeCount; i++ {
		cache.AddOrUpdateNode(&graph.Node{
			ContactID: int64(i),
			FullName:  fmt.Sprintf("User_%d", i),
		})
	}
	for i := 1; i < initNodeCount; i++ {
		cache.AddOrUpdateRelationship(&graph.Relationship{
			ID:       int64(i),
			SourceID: int64(i),
			TargetID: int64(i + 1),
			Type:     graph.RelColleague,
			Weight:   3,
		})
	}

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// 10 горутин-читателей (симулируют 60 FPS рендеринг и фоновые алгоритмы)
	for r := 0; r < 10; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					// Быстрые операции чтения под RLock
					_ = cache.Version()
					_ = cache.NodeCount()
					_ = cache.RelationshipCount()
					_, _, _ = cache.Snapshot()
					randID := int64(rand.Intn(initNodeCount) + 1)
					_, _ = cache.GetNode(randID)
					_ = cache.GetAdjacentRelationships(randID)
					_ = cache.GetOutgoingRelationships(randID)
					_ = cache.GetIncomingRelationships(randID)
					time.Sleep(100 * time.Microsecond)
				}
			}
		}(r)
	}

	// 5 горутин-писателей (симулируют обновления из SQLite и пользовательские действия)
	for w := 0; w < 5; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			relIDCounter := int64(1000 + writerID*100)
			for i := 0; i < 50; i++ {
				u1 := int64(rand.Intn(initNodeCount) + 1)
				u2 := int64(rand.Intn(initNodeCount) + 1)
				if u1 == u2 {
					u2 = (u1 % initNodeCount) + 1
				}

				// Добавление связи
				relIDCounter++
				cache.AddOrUpdateRelationship(&graph.Relationship{
					ID:         relIDCounter,
					SourceID:   u1,
					TargetID:   u2,
					Type:       graph.RelFriend,
					Weight:     rand.Intn(5) + 1,
					IsDirected: rand.Intn(2) == 1,
				})

				// Добавление / обновление узла
				cache.AddOrUpdateNode(&graph.Node{
					ContactID: u1,
					FullName:  fmt.Sprintf("User_%d_rev%d", u1, i),
					PosX:      rand.Float64() * 500,
					PosY:      rand.Float64() * 500,
				})

				if i%10 == 0 {
					cache.RemoveRelationship(relIDCounter)
				}
				time.Sleep(300 * time.Microsecond)
			}
		}(w)
	}

	// Даём горутинам активно поработать
	time.Sleep(100 * time.Millisecond)
	close(stopCh)
	wg.Wait()

	if cache.NodeCount() == 0 {
		t.Fatalf("кэш не должен быть пустым после конкурентного теста")
	}
}
