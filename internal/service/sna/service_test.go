// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sna_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/repository/memory"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

// MockContactRepo — mock репозиторий контактов
type MockContactRepo struct {
	mu       sync.Mutex
	contacts map[int64]*domain.Contact
}

func NewMockContactRepo() *MockContactRepo {
	return &MockContactRepo{contacts: make(map[int64]*domain.Contact)}
}

func (m *MockContactRepo) Create(ctx context.Context, c *domain.Contact) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.contacts[c.ID] = c
	return nil
}

func (m *MockContactRepo) GetByID(ctx context.Context, id int64) (*domain.Contact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.contacts[id]
	if !ok {
		return nil, domain.ErrContactNotFound
	}
	return c, nil
}

func (m *MockContactRepo) GetByPhone(ctx context.Context, phone string) (*domain.Contact, error) {
	return nil, nil
}
func (m *MockContactRepo) Update(ctx context.Context, c *domain.Contact) error { return nil }
func (m *MockContactRepo) Delete(ctx context.Context, id int64) error          { return nil }
func (m *MockContactRepo) List(ctx context.Context, filter domain.ContactFilter) ([]*domain.Contact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]*domain.Contact, 0, len(m.contacts))
	for _, c := range m.contacts {
		res = append(res, c)
	}
	return res, nil
}
func (m *MockContactRepo) Stats(ctx context.Context) (*domain.BookStats, error) { return nil, nil }

// MockGraphRepo — mock репозиторий графа
type MockGraphRepo struct {
	mu            sync.Mutex
	rels          map[int64]*graph.Relationship
	layouts       map[int64]*graph.GraphLayout
	idCounter     int64
	saveBatchCall int
}

func NewMockGraphRepo() *MockGraphRepo {
	return &MockGraphRepo{
		rels:    make(map[int64]*graph.Relationship),
		layouts: make(map[int64]*graph.GraphLayout),
	}
}

func (m *MockGraphRepo) CreateRelationship(ctx context.Context, rel *graph.Relationship) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idCounter++
	rel.ID = m.idCounter
	cp := *rel
	m.rels[rel.ID] = &cp
	return rel.ID, nil
}

func (m *MockGraphRepo) GetRelationship(ctx context.Context, id int64) (*graph.Relationship, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rels[id]
	if !ok {
		return nil, graph.ErrRelationshipNotFound
	}
	return r, nil
}

func (m *MockGraphRepo) UpdateRelationship(ctx context.Context, rel *graph.Relationship) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *rel
	m.rels[rel.ID] = &cp
	return nil
}

func (m *MockGraphRepo) DeleteRelationship(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rels, id)
	return nil
}

func (m *MockGraphRepo) GetRelationshipsByContact(ctx context.Context, cid int64) ([]*graph.Relationship, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []*graph.Relationship
	for _, r := range m.rels {
		if r.SourceID == cid || r.TargetID == cid {
			res = append(res, r)
		}
	}
	return res, nil
}

func (m *MockGraphRepo) GetAllRelationships(ctx context.Context) ([]*graph.Relationship, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var res []*graph.Relationship
	for _, r := range m.rels {
		res = append(res, r)
	}
	return res, nil
}

func (m *MockGraphRepo) SaveLayout(ctx context.Context, l *graph.GraphLayout) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *l
	m.layouts[l.ContactID] = &cp
	return nil
}

func (m *MockGraphRepo) SaveLayoutBatch(ctx context.Context, layouts []*graph.GraphLayout) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saveBatchCall++
	for _, l := range layouts {
		cp := *l
		m.layouts[l.ContactID] = &cp
	}
	return nil
}

func (m *MockGraphRepo) GetLayout(ctx context.Context, cid int64) (*graph.GraphLayout, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l, ok := m.layouts[cid]
	if !ok {
		return nil, graph.ErrNodeNotFound
	}
	return l, nil
}

func (m *MockGraphRepo) GetAllLayouts(ctx context.Context) (map[int64]*graph.GraphLayout, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make(map[int64]*graph.GraphLayout)
	for k, v := range m.layouts {
		cp := *v
		res[k] = &cp
	}
	return res, nil
}

func (m *MockGraphRepo) QueryEgoSubgraph(ctx context.Context, egoID int64, maxHops int) ([]*graph.EgoNode, error) {
	return nil, nil
}

func (m *MockGraphRepo) GetImplicitRelationships(ctx context.Context) ([]*graph.Relationship, error) {
	return nil, nil
}

func TestGraphService_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	cRepo := NewMockContactRepo()
	gRepo := NewMockGraphRepo()
	cache := memory.NewTopologyCache()

	// Наполняем тестовыми контактами
	c1 := &domain.Contact{ID: 1, FirstName: "Алиса", LastName: "Иванова", Company: "TechCorp"}
	c2 := &domain.Contact{ID: 2, FirstName: "Боб", LastName: "Петров", Company: "TechCorp"}
	c3 := &domain.Contact{ID: 3, FirstName: "Чарли", LastName: "Сидоров", Company: "FinanceLLC"}
	_ = cRepo.Create(ctx, c1)
	_ = cRepo.Create(ctx, c2)
	_ = cRepo.Create(ctx, c3)

	svc := sna.NewGraphService(gRepo, cRepo, cache, 4)

	// Запуск и перезагрузка графа
	if err := svc.ReloadGraph(ctx, false); err != nil {
		t.Fatalf("ошибка ReloadGraph: %v", err)
	}

	snapNodes, _, _ := svc.GetTopologySnapshot()
	if len(snapNodes) != 3 {
		t.Fatalf("ожидалось 3 узла в кэше, получено: %d", len(snapNodes))
	}

	// Создание связи 1 <-> 2
	rel1 := &graph.Relationship{
		SourceID:   1,
		TargetID:   2,
		Type:       graph.RelColleague,
		Weight:     5,
		IsDirected: false,
	}
	id1, err := svc.CreateRelationship(ctx, rel1)
	if err != nil || id1 <= 0 {
		t.Fatalf("ошибка создания связи 1: %v", err)
	}

	// Создание связи 2 <-> 3
	rel2 := &graph.Relationship{
		SourceID:   2,
		TargetID:   3,
		Type:       graph.RelFriend,
		Weight:     4,
		IsDirected: false,
	}
	id2, err := svc.CreateRelationship(ctx, rel2)
	if err != nil || id2 <= 0 {
		t.Fatalf("ошибка создания связи 2: %v", err)
	}

	// Запуск пересчёта метрик SNA (Brandes, Louvain, Dunbar)
	if err := svc.RecalculateMetrics(ctx); err != nil {
		t.Fatalf("ошибка RecalculateMetrics: %v", err)
	}

	// Узел 2 соединяет 1 и 3, его BetweennessScore обязан быть > 0 (равен 1.0)
	n2, ok := svc.GetNode(2)
	if !ok || n2.BetweennessScore <= 0 {
		t.Fatalf("узел-мост 2 должен получить высокий BetweennessScore, получено: %f", n2.BetweennessScore)
	}

	// Поиск тёплого пути интро между 1 и 3
	introPath, err := svc.FindWarmIntroPath(ctx, 1, 3)
	if err != nil {
		t.Fatalf("ошибка поиска интро: %v", err)
	}
	if len(introPath.Hops) != 2 || introPath.CompoundTrust <= 0 {
		t.Fatalf("некорректный путь интро: %+v", introPath)
	}

	// Проверка работы физического симулятора и закрепления узла
	svc.StartSimulation(ctx)
	defer svc.StopSimulation()

	svc.PinNode(1, true, 120, 150)
	time.Sleep(20 * time.Millisecond)

	frameSnap := svc.GetLatestSnapshot()
	if frameSnap == nil || len(frameSnap.Positions) == 0 {
		t.Fatalf("снапшот кадра не должен быть пустым")
	}

	p1 := frameSnap.Positions[1]
	if !p1.IsPinned || p1.PosX != 120 || p1.PosY != 150 {
		t.Fatalf("PinNode не обновил координаты узла 1")
	}

	// Удаление связи
	if err := svc.DeleteRelationship(ctx, id1); err != nil {
		t.Fatalf("ошибка удаления связи: %v", err)
	}
}

func TestGraphService_ConcurrencyAndRace(t *testing.T) {
	ctx := context.Background()
	cRepo := NewMockContactRepo()
	gRepo := NewMockGraphRepo()
	cache := memory.NewTopologyCache()

	const userCount = 10
	for i := 1; i <= userCount; i++ {
		_ = cRepo.Create(ctx, &domain.Contact{ID: int64(i), FirstName: "User"})
	}

	svc := sna.NewGraphService(gRepo, cRepo, cache, 4)
	_ = svc.ReloadGraph(ctx, false)

	svc.StartSimulation(ctx)
	defer svc.StopSimulation()

	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// 5 горутин-читателей
	for r := 0; r < 5; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					_ = svc.GetLatestSnapshot()
					_, _, _ = svc.GetTopologySnapshot()
					_, _ = svc.GetNode(1)
					time.Sleep(1 * time.Millisecond)
				}
			}
		}()
	}

	// 3 горутины-писателя
	for w := 0; w < 3; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				svc.ReheatSimulation(0.35)
				svc.MoveNode(1, float64(i*5), float64(i*5))
				_ = svc.RecalculateMetrics(ctx)
				time.Sleep(2 * time.Millisecond)
			}
		}(w)
	}

	time.Sleep(60 * time.Millisecond)
	close(stopCh)
	wg.Wait()
}
