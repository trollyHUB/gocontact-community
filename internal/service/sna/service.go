// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sna

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/repository"
	"github.com/trollyHUB/gocontact-studio/internal/repository/memory"
)

// GraphService определяет главный контракт сервисного слоя управления социальным графом,
// фоновой физической симуляцией Barnes-Hut и алгоритмической аналитикой SNA.
type GraphService interface {
	ReloadGraph(ctx context.Context, includeImplicit bool) error
	RecalculateMetrics(ctx context.Context) error
	FindWarmIntroPath(ctx context.Context, sourceID, targetID int64) (*graph.IntroPath, error)

	StartSimulation(ctx context.Context)
	StopSimulation()
	ReheatSimulation(alpha float64)
	PinNode(contactID int64, isPinned bool, posX, posY float64)
	MoveNode(contactID int64, posX, posY float64)
	GetLatestSnapshot() *FrameSnapshot

	CreateRelationship(ctx context.Context, rel *graph.Relationship) (int64, error)
	UpdateRelationship(ctx context.Context, rel *graph.Relationship) error
	DeleteRelationship(ctx context.Context, id int64) error
	GetRelationshipsByContact(ctx context.Context, contactID int64) ([]*graph.Relationship, error)
	GetAllRelationships(ctx context.Context) ([]*graph.Relationship, error)
	GetEgoSubgraph(ctx context.Context, egoID int64, maxHops int) ([]*graph.EgoNode, error)

	GetNode(contactID int64) (*graph.Node, bool)
	GetTopologySnapshot() (map[int64]*graph.Node, []*graph.Relationship, uint64)
}

type graphService struct {
	mu          sync.RWMutex
	graphRepo   repository.GraphRepository
	contactRepo repository.ContactRepository
	cache       *memory.TopologyCache
	sim         *PhysicsSimulation
	workerCount int
}

// NewGraphService создаёт и настраивает новый экземпляр подсистемы социального графа.
func NewGraphService(
	graphRepo repository.GraphRepository,
	contactRepo repository.ContactRepository,
	cache *memory.TopologyCache,
	workerCount int,
) GraphService {
	if cache == nil {
		cache = memory.NewTopologyCache()
	}
	if workerCount <= 0 {
		workerCount = 4
	}

	s := &graphService{
		graphRepo:   graphRepo,
		contactRepo: contactRepo,
		cache:       cache,
		workerCount: workerCount,
	}

	// Физический симулятор с колбэком сохранения координат в SQLite при наступлении покоя
	cfg := DefaultSimulationConfig()
	s.sim = NewPhysicsSimulation(cfg, func(nodes []*graph.Node) {
		s.flushLayoutsBatch(nodes)
	})

	return s
}

// ReloadGraph перезагружает всю топологию контактов, связей и координат из репозиториев в кэш и физику.
func (s *graphService) ReloadGraph(ctx context.Context, includeImplicit bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	contacts, err := s.contactRepo.List(ctx, domain.ContactFilter{})
	if err != nil {
		return fmt.Errorf("ошибка загрузки контактов: %w", err)
	}

	explicitRels, err := s.graphRepo.GetAllRelationships(ctx)
	if err != nil {
		return fmt.Errorf("ошибка загрузки связей: %w", err)
	}

	var allRels []*graph.Relationship
	allRels = append(allRels, explicitRels...)

	if includeImplicit {
		implicitRels, err := s.graphRepo.GetImplicitRelationships(ctx)
		if err == nil && len(implicitRels) > 0 {
			allRels = append(allRels, implicitRels...)
		}
	}

	layouts, _ := s.graphRepo.GetAllLayouts(ctx)
	if layouts == nil {
		layouts = make(map[int64]*graph.GraphLayout)
	}

	// Классификация орбит Данбара
	dunbarTiers := ClassifyDunbarTiers(contacts, allRels)

	nodes := make([]*graph.Node, 0, len(contacts))
	for _, c := range contacts {
		if c == nil || c.ID <= 0 {
			continue
		}
		node := &graph.Node{
			ContactID:      c.ID,
			FullName:       c.FullName(),
			Company:        c.Company,
			AvatarInitials: c.Initials(),
			DunbarTier:     dunbarTiers[c.ID],
		}

		if lay, ok := layouts[c.ID]; ok {
			node.PosX = lay.PosX
			node.PosY = lay.PosY
			node.IsPinned = lay.IsPinned
			node.CommunityID = lay.CommunityID
			node.BetweennessScore = lay.BetweennessScore
		} else {
			// Начальное случайное распределение в круге радиусом 250px
			angle := rand.Float64() * 2 * 3.1415926535
			radius := 50.0 + rand.Float64()*200.0
			node.PosX = radius * 0.707 * angle
			node.PosY = radius * 0.707 * (1 - angle)
		}

		nodes = append(nodes, node)
	}

	// Синхронизация L1 In-Memory кэша
	s.cache.Reset(nodes, allRels)

	// Синхронизация симулятора физики
	s.sim.UpdateTopology(nodes, allRels)

	return nil
}

// RecalculateMetrics запускает расчёт алгоритмов SNA (Brandes, Louvain, Dunbar) и сохраняет в БД.
func (s *graphService) RecalculateMetrics(ctx context.Context) error {
	nodesMap, edges, _ := s.cache.Snapshot()
	if len(nodesMap) == 0 {
		return nil
	}

	nodes := make([]*graph.Node, 0, len(nodesMap))
	for _, n := range nodesMap {
		nodes = append(nodes, n)
	}

	// 1. Центральность Брандеса O(|V| * |E|)
	ComputeBetweennessCentrality(nodes, edges)

	// 2. Кластеризация Лувена (Modularity Q)
	DetectCommunities(nodes, edges)

	// 3. Обновление узлов в TopologyCache
	for _, n := range nodes {
		s.cache.AddOrUpdateNode(n)
	}

	// 4. Пакетный сброс лейаутов и метрик в базу данных SQLite
	layouts := make([]*graph.GraphLayout, len(nodes))
	for i, n := range nodes {
		layouts[i] = &graph.GraphLayout{
			ContactID:        n.ContactID,
			PosX:             n.PosX,
			PosY:             n.PosY,
			IsPinned:         n.IsPinned,
			CommunityID:      n.CommunityID,
			BetweennessScore: n.BetweennessScore,
		}
	}

	if err := s.graphRepo.SaveLayoutBatch(ctx, layouts); err != nil {
		return fmt.Errorf("ошибка пакетной записи метрик в БД: %w", err)
	}

	return nil
}

// FindWarmIntroPath находит путь максимального доверия между двумя контактами.
func (s *graphService) FindWarmIntroPath(ctx context.Context, sourceID, targetID int64) (*graph.IntroPath, error) {
	nodesMap, edges, _ := s.cache.Snapshot()
	return FindIntroPath(sourceID, targetID, nodesMap, edges)
}

func (s *graphService) StartSimulation(ctx context.Context) {
	s.sim.Start(ctx)
}

func (s *graphService) StopSimulation() {
	s.sim.Stop()
}

func (s *graphService) ReheatSimulation(alpha float64) {
	s.sim.Reheat(alpha)
}

func (s *graphService) PinNode(contactID int64, isPinned bool, posX, posY float64) {
	s.sim.PinNode(contactID, isPinned, posX, posY)
	if node, ok := s.cache.GetNode(contactID); ok {
		node.IsPinned = isPinned
		node.PosX = posX
		node.PosY = posY
		s.cache.AddOrUpdateNode(node)
	}
}

func (s *graphService) MoveNode(contactID int64, posX, posY float64) {
	s.sim.MoveNode(contactID, posX, posY)
}

func (s *graphService) GetLatestSnapshot() *FrameSnapshot {
	return s.sim.GetLatestSnapshot()
}

func (s *graphService) CreateRelationship(ctx context.Context, rel *graph.Relationship) (int64, error) {
	id, err := s.graphRepo.CreateRelationship(ctx, rel)
	if err != nil {
		return 0, err
	}
	rel.ID = id
	s.cache.AddOrUpdateRelationship(rel)
	s.sim.Reheat(0.40)
	return id, nil
}

func (s *graphService) UpdateRelationship(ctx context.Context, rel *graph.Relationship) error {
	if err := s.graphRepo.UpdateRelationship(ctx, rel); err != nil {
		return err
	}
	s.cache.AddOrUpdateRelationship(rel)
	s.sim.Reheat(0.30)
	return nil
}

func (s *graphService) DeleteRelationship(ctx context.Context, id int64) error {
	if err := s.graphRepo.DeleteRelationship(ctx, id); err != nil {
		return err
	}
	s.cache.RemoveRelationship(id)
	s.sim.Reheat(0.30)
	return nil
}

func (s *graphService) GetRelationshipsByContact(ctx context.Context, contactID int64) ([]*graph.Relationship, error) {
	return s.graphRepo.GetRelationshipsByContact(ctx, contactID)
}

func (s *graphService) GetAllRelationships(ctx context.Context) ([]*graph.Relationship, error) {
	return s.graphRepo.GetAllRelationships(ctx)
}

func (s *graphService) GetEgoSubgraph(ctx context.Context, egoID int64, maxHops int) ([]*graph.EgoNode, error) {
	return s.graphRepo.QueryEgoSubgraph(ctx, egoID, maxHops)
}

func (s *graphService) GetNode(contactID int64) (*graph.Node, bool) {
	return s.cache.GetNode(contactID)
}

func (s *graphService) GetTopologySnapshot() (map[int64]*graph.Node, []*graph.Relationship, uint64) {
	return s.cache.Snapshot()
}

func (s *graphService) flushLayoutsBatch(nodes []*graph.Node) {
	if len(nodes) == 0 {
		return
	}
	layouts := make([]*graph.GraphLayout, len(nodes))
	for i, n := range nodes {
		layouts[i] = &graph.GraphLayout{
			ContactID:        n.ContactID,
			PosX:             n.PosX,
			PosY:             n.PosY,
			IsPinned:         n.IsPinned,
			CommunityID:      n.CommunityID,
			BetweennessScore: n.BetweennessScore,
		}
	}
	_ = s.graphRepo.SaveLayoutBatch(context.Background(), layouts)
}
