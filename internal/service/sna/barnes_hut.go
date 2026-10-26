// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sna

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
)

// NodePos представляет сериализуемые координаты узла в снимке кадра.
type NodePos struct {
	ContactID int64   `json:"contact_id"`
	PosX      float64 `json:"pos_x"`
	PosY      float64 `json:"pos_y"`
	IsPinned  bool    `json:"is_pinned"`
}

// FrameSnapshot — неизменяемый снимок координат для отрисовки интерфейса на 60 FPS
// с атомарным чтением atomic.Pointer без захвата мьютексов физического движка.
type FrameSnapshot struct {
	Version   uint64
	Alpha     float64
	Positions map[int64]NodePos
	Timestamp time.Time
}

// SimCommandType задаёт тип команды управления симуляцией.
type SimCommandType int

const (
	CmdReheat SimCommandType = iota
	CmdPinNode
	CmdMoveNode
	CmdStop
)

// SimCommand инкапсулирует асинхронную команду от UI-потока.
type SimCommand struct {
	Type      SimCommandType
	ContactID int64
	PosX      float64
	PosY      float64
	IsPinned  bool
	Alpha     float64
}

// SimulationConfig содержит физические константы симуляции сил.
type SimulationConfig struct {
	Kr             float64       // Коэффициент отталкивания Кулона
	Ks             float64       // Коэффициент жесткости пружины Гука
	Lbase          float64       // Базовая длина связи
	Damping        float64       // Коэффициент демпфирования скорости
	Dt             float64       // Шаг интеграции по времени
	AlphaDecay     float64       // Коэффициент охлаждения температуры
	AlphaMin       float64       // Порог перехода в режим покоя (Idle)
	Theta          float64       // Критерий раскрытия Барнса — Хата
	CenterGravity  float64       // Коэффициент притяжения к центру
	TickerInterval time.Duration // Период тикера физического движка
}

// DefaultSimulationConfig возвращает сбалансированные физические параметры для 60 FPS.
func DefaultSimulationConfig() SimulationConfig {
	return SimulationConfig{
		Kr:             5500.0,
		Ks:             0.045,
		Lbase:          75.0,
		Damping:        0.86,
		Dt:             0.016,
		AlphaDecay:     0.982,
		AlphaMin:       0.005,
		Theta:          0.70,
		CenterGravity:  0.008,
		TickerInterval: 16 * time.Millisecond,
	}
}

// PhysicsSimulation — конкурентный 60 Hz движок симуляции сил на базе квадродерева Барнса — Хата.
type PhysicsSimulation struct {
	mu       sync.RWMutex
	nodes    map[int64]*graph.Node
	edges    []*graph.Relationship
	quadTree *graph.QuadTree

	cmdCh    chan SimCommand
	snapshot atomic.Pointer[FrameSnapshot]
	cfg      SimulationConfig

	alpha    float64
	version  uint64
	isIdle   bool
	onIdle   func(nodes []*graph.Node)
	stopOnce sync.Once
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewPhysicsSimulation создаёт и инициализирует физический симулятор.
func NewPhysicsSimulation(cfg SimulationConfig, onIdle func(nodes []*graph.Node)) *PhysicsSimulation {
	sim := &PhysicsSimulation{
		nodes:    make(map[int64]*graph.Node),
		quadTree: graph.NewQuadTree(512),
		cmdCh:    make(chan SimCommand, 32),
		cfg:      cfg,
		alpha:    1.0,
		onIdle:   onIdle,
		stopCh:   make(chan struct{}),
	}
	emptySnap := &FrameSnapshot{
		Version:   0,
		Alpha:     1.0,
		Positions: make(map[int64]NodePos),
		Timestamp: time.Now().UTC(),
	}
	sim.snapshot.Store(emptySnap)
	return sim
}

// UpdateTopology обновляет состав узлов и связей, пересчитывая массы и степени.
func (s *PhysicsSimulation) UpdateTopology(nodes []*graph.Node, edges []*graph.Relationship) {
	s.mu.Lock()
	defer s.mu.Unlock()

	degrees := make(map[int64]int, len(nodes))
	for _, e := range edges {
		if e == nil {
			continue
		}
		degrees[e.SourceID]++
		degrees[e.TargetID]++
	}

	s.nodes = make(map[int64]*graph.Node, len(nodes))
	for _, n := range nodes {
		if n == nil || n.ContactID <= 0 {
			continue
		}
		cp := *n
		deg := degrees[cp.ContactID]
		cp.Degree = deg
		cp.Mass = 1.0 + 0.25*float64(deg)
		s.nodes[cp.ContactID] = &cp
	}

	s.edges = make([]*graph.Relationship, len(edges))
	copy(s.edges, edges)

	s.alpha = 1.0
	s.isIdle = false
	s.version++
	s.publishSnapshotLocked()
}

// Start запускает фоновую горутину физического тикера 60 Гц.
func (s *PhysicsSimulation) Start(ctx context.Context) {
	s.wg.Add(1)
	go s.physicsLoop(ctx)
}

// Stop корректно останавливает физический цикл и ожидает завершения горутины.
func (s *PhysicsSimulation) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
		select {
		case s.cmdCh <- SimCommand{Type: CmdStop}:
		default:
		}
	})
	s.wg.Wait()
}

// Reheat мгновенно подогревает симуляцию (например, при захвате узла курсором).
func (s *PhysicsSimulation) Reheat(alpha float64) {
	select {
	case s.cmdCh <- SimCommand{Type: CmdReheat, Alpha: alpha}:
	default:
	}
}

// PinNode закрепляет или открепляет узел на холсте.
func (s *PhysicsSimulation) PinNode(contactID int64, isPinned bool, posX, posY float64) {
	select {
	case s.cmdCh <- SimCommand{
		Type:      CmdPinNode,
		ContactID: contactID,
		IsPinned:  isPinned,
		PosX:      posX,
		PosY:      posY,
	}:
	default:
	}
}

// MoveNode перемещает узел в процессе перетаскивания (DragNode).
func (s *PhysicsSimulation) MoveNode(contactID int64, posX, posY float64) {
	select {
	case s.cmdCh <- SimCommand{
		Type:      CmdMoveNode,
		ContactID: contactID,
		PosX:      posX,
		PosY:      posY,
	}:
	default:
	}
}

// GetLatestSnapshot возвращает последний снимок позиций для Fyne-виджета без блокировок.
func (s *PhysicsSimulation) GetLatestSnapshot() *FrameSnapshot {
	return s.snapshot.Load()
}

// Step выполняет единичный шаг симуляции физики (публичный для детерминированного тестирования).
func (s *PhysicsSimulation) Step() {
	s.drainCommands()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stepInternal()
}

func (s *PhysicsSimulation) physicsLoop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(s.cfg.TickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case cmd := <-s.cmdCh:
			if cmd.Type == CmdStop {
				return
			}
			s.handleCommand(cmd)
		case <-ticker.C:
			// Сбрасываем накопившиеся команды перед тиком
			s.drainCommands()

			s.mu.Lock()
			if s.isIdle || len(s.nodes) == 0 {
				s.mu.Unlock()
				continue
			}

			s.stepInternal()

			// Проверка перехода в Idle (0% CPU в покое)
			if s.alpha < s.cfg.AlphaMin {
				s.isIdle = true
				nodesList := make([]*graph.Node, 0, len(s.nodes))
				for _, n := range s.nodes {
					cp := *n
					nodesList = append(nodesList, &cp)
				}
				s.mu.Unlock()

				if s.onIdle != nil {
					s.onIdle(nodesList)
				}
				continue
			}
			s.mu.Unlock()
		}
	}
}

func (s *PhysicsSimulation) drainCommands() {
	for {
		select {
		case cmd := <-s.cmdCh:
			if cmd.Type == CmdStop {
				return
			}
			s.handleCommand(cmd)
		default:
			return
		}
	}
}

func (s *PhysicsSimulation) handleCommand(cmd SimCommand) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch cmd.Type {
	case CmdReheat:
		if cmd.Alpha > s.alpha {
			s.alpha = cmd.Alpha
		}
		s.isIdle = false
	case CmdPinNode:
		if n, ok := s.nodes[cmd.ContactID]; ok {
			n.IsPinned = cmd.IsPinned
			n.PosX = cmd.PosX
			n.PosY = cmd.PosY
			n.VelX = 0
			n.VelY = 0
		}
		s.alpha = math.Max(s.alpha, 0.35)
		s.isIdle = false
		s.publishSnapshotLocked()
	case CmdMoveNode:
		if n, ok := s.nodes[cmd.ContactID]; ok {
			n.PosX = cmd.PosX
			n.PosY = cmd.PosY
			n.VelX = 0
			n.VelY = 0
		}
		s.alpha = math.Max(s.alpha, 0.35)
		s.isIdle = false
		s.publishSnapshotLocked()
	}
}

func (s *PhysicsSimulation) stepInternal() {
	if len(s.nodes) == 0 {
		return
	}

	// 1. Построение границ и квадродерева Барнса — Хата
	var minX, minY, maxX, maxY float64
	first := true
	for _, n := range s.nodes {
		if first {
			minX, maxX = n.PosX, n.PosX
			minY, maxY = n.PosY, n.PosY
			first = false
			continue
		}
		if n.PosX < minX {
			minX = n.PosX
		}
		if n.PosX > maxX {
			maxX = n.PosX
		}
		if n.PosY < minY {
			minY = n.PosY
		}
		if n.PosY > maxY {
			maxY = n.PosY
		}
	}

	const padding = 60.0
	s.quadTree.ResetWithBounds(minX-padding, minY-padding, maxX+padding, maxY+padding)
	for _, n := range s.nodes {
		s.quadTree.Insert(n)
	}

	// 2. Сброс сил и расчёт отталкивания Кулона + гравитации к центру
	for _, n := range s.nodes {
		fx, fy := s.quadTree.ComputeRepulsion(n, s.cfg.Theta, s.cfg.Kr)
		// Центростремительная сила
		fx -= s.cfg.CenterGravity * n.PosX
		fy -= s.cfg.CenterGravity * n.PosY
		n.ForceX = fx
		n.ForceY = fy
	}

	// 3. Расчёт силы притяжения Гука вдоль рёбер
	for _, e := range s.edges {
		if e == nil {
			continue
		}
		n1, ok1 := s.nodes[e.SourceID]
		n2, ok2 := s.nodes[e.TargetID]
		if !ok1 || !ok2 {
			continue
		}

		dx := n2.PosX - n1.PosX
		dy := n2.PosY - n1.PosY
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 1e-4 {
			dist = 1e-4
			dx = 0.01
		}

		w := float64(e.Weight)
		if w < 1 {
			w = 1
		} else if w > 5 {
			w = 5
		}
		l0 := s.cfg.Lbase * (1.6 - 0.2*w)
		fSpring := s.cfg.Ks * (dist - l0)

		fSpringX := (dx / dist) * fSpring
		fSpringY := (dy / dist) * fSpring

		n1.ForceX += fSpringX
		n1.ForceY += fSpringY
		n2.ForceX -= fSpringX
		n2.ForceY -= fSpringY
	}

	// 4. Интеграция скорости и координат с демпфированием
	const maxVel = 80.0
	for _, n := range s.nodes {
		if n.IsPinned {
			n.VelX = 0
			n.VelY = 0
			continue
		}

		ax := n.ForceX / n.Mass
		ay := n.ForceY / n.Mass

		n.VelX = (n.VelX + ax*s.cfg.Dt) * s.cfg.Damping
		n.VelY = (n.VelY + ay*s.cfg.Dt) * s.cfg.Damping

		speed := math.Sqrt(n.VelX*n.VelX + n.VelY*n.VelY)
		if speed > maxVel {
			ratio := maxVel / speed
			n.VelX *= ratio
			n.VelY *= ratio
		}

		n.PosX += n.VelX * s.cfg.Dt * s.alpha
		n.PosY += n.VelY * s.cfg.Dt * s.alpha
	}

	// 5. Охлаждение температуры симуляции
	s.alpha *= s.cfg.AlphaDecay
	s.version++
	s.publishSnapshotLocked()
}

func (s *PhysicsSimulation) publishSnapshotLocked() {
	positions := make(map[int64]NodePos, len(s.nodes))
	for id, n := range s.nodes {
		positions[id] = NodePos{
			ContactID: id,
			PosX:      n.PosX,
			PosY:      n.PosY,
			IsPinned:  n.IsPinned,
		}
	}
	snap := &FrameSnapshot{
		Version:   s.version,
		Alpha:     s.alpha,
		Positions: positions,
		Timestamp: time.Now().UTC(),
	}
	s.snapshot.Store(snap)
}
