// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"fmt"
	"image/color"
	"math"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

// GraphMode определяет режим визуализации социального графа.
type GraphMode int

const (
	GraphModeForce  GraphMode = iota // Силовая укладка Barnes-Hut
	GraphModeDunbar                  // Концентрические орбиты Данбара
)

// Токены палитры темы StudioTheme для графа
var (
	colorCanvasBgDark   = color.NRGBA{R: 15, G: 20, B: 28, A: 255}    // #0F141C
	colorCanvasBgLight  = color.NRGBA{R: 248, G: 250, B: 252, A: 255} // #F8FAFC
	colorOrbitRingDark  = color.NRGBA{R: 30, G: 41, B: 59, A: 180}    // #1E293B
	colorOrbitRingLight = color.NRGBA{R: 226, G: 232, B: 240, A: 220} // #E2E8F0
	colorEdgeDefault    = color.NRGBA{R: 71, G: 85, B: 105, A: 160}   // #475569
	colorEdgeImplicit   = color.NRGBA{R: 51, G: 65, B: 85, A: 120}    // #334155
	colorEdgeIntroPath  = color.NRGBA{R: 16, G: 185, B: 129, A: 255}  // #10B981 (Изумрудный)
	colorNodeBroker     = color.NRGBA{R: 245, G: 158, B: 11, A: 255}  // #F59E0B (Золото)
	colorNodeSelected   = color.NRGBA{R: 56, G: 189, B: 248, A: 255}  // #38BDF8 (Небесный)

	communityColors = []color.NRGBA{
		{R: 99, G: 102, B: 241, A: 255}, // #6366F1 Indigo
		{R: 236, G: 72, B: 153, A: 255}, // #EC4899 Pink
		{R: 20, G: 184, B: 166, A: 255}, // #14B8A6 Teal
		{R: 249, G: 115, B: 22, A: 255}, // #F97316 Orange
		{R: 139, G: 92, B: 246, A: 255}, // #8B5CF6 Purple
		{R: 6, G: 182, B: 212, A: 255},  // #06B6D4 Cyan
	}
)

// SocialGraphCanvas — кастомный интерактивный виджет визуализации социального графа на Fyne v2 Canvas.
// Поддерживает Pan, Zoom (0.2x..3.0x), Drag узлов, Tooltip при наведении, выбор узлов и подсветку пути интро.
type SocialGraphCanvas struct {
	widget.BaseWidget
	mu sync.RWMutex

	graphService sna.GraphService
	mode         GraphMode

	// Камера и аффинные трансформации
	panX, panY float32
	zoom       float32 // Диапазон [0.20 .. 3.00]

	// Интерактивное состояние
	selectedNodeID  int64
	hoveredNodeID   int64
	draggedNodeID   int64
	isDraggingWorld bool
	lastDragPos     fyne.Position

	// Фильтры и подсветка
	dunbarFilter     graph.DunbarTier
	showImplicit     bool
	highlightedNodes map[int64]bool
	highlightedEdges map[string]bool

	// Колбэки
	OnNodeSelected func(nodeID int64)
	OnCanvasTapped func()
	OnHoverChanged func(node *graph.Node)
}

// NewSocialGraphCanvas создаёт и инициализирует виджет холста графа.
func NewSocialGraphCanvas(svc sna.GraphService) *SocialGraphCanvas {
	c := &SocialGraphCanvas{
		graphService:     svc,
		mode:             GraphModeForce,
		zoom:             1.0,
		panX:             0.0,
		panY:             0.0,
		highlightedNodes: make(map[int64]bool),
		highlightedEdges: make(map[string]bool),
	}
	c.ExtendBaseWidget(c)
	return c
}

// SetMode переключает режим отображения: силовой граф или орбиты Данбара.
func (c *SocialGraphCanvas) SetMode(m GraphMode) {
	c.mu.Lock()
	c.mode = m
	c.mu.Unlock()
	c.Refresh()
}

// SetDunbarFilter фильтрует отображение узлов по кругу Данбара (0 = все).
func (c *SocialGraphCanvas) SetDunbarFilter(tier graph.DunbarTier) {
	c.mu.Lock()
	c.dunbarFilter = tier
	c.mu.Unlock()
	c.Refresh()
}

// SetShowImplicit включает или отключает отображение неявных связей по компании.
func (c *SocialGraphCanvas) SetShowImplicit(show bool) {
	c.mu.Lock()
	c.showImplicit = show
	c.mu.Unlock()
	c.Refresh()
}

// ResetCamera центрирует камеру и устанавливает 100% масштаб.
func (c *SocialGraphCanvas) ResetCamera() {
	c.mu.Lock()
	c.panX = 0
	c.panY = 0
	c.zoom = 1.0
	c.mu.Unlock()
	c.Refresh()
}

// HighlightIntroPath подсвечивает цепочку тёплого интро на холсте изумрудным цветом.
func (c *SocialGraphCanvas) HighlightIntroPath(path *graph.IntroPath) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.highlightedNodes = make(map[int64]bool)
	c.highlightedEdges = make(map[string]bool)

	if path == nil || len(path.Hops) == 0 {
		c.Refresh()
		return
	}

	c.highlightedNodes[path.SourceID] = true
	c.highlightedNodes[path.TargetID] = true

	for _, hop := range path.Hops {
		c.highlightedNodes[hop.FromContactID] = true
		c.highlightedNodes[hop.ToContactID] = true
		key1 := fmt.Sprintf("%d-%d", hop.FromContactID, hop.ToContactID)
		key2 := fmt.Sprintf("%d-%d", hop.ToContactID, hop.FromContactID)
		c.highlightedEdges[key1] = true
		c.highlightedEdges[key2] = true
	}

	c.Refresh()
}

// ClearPathHighlight очищает подсветку цепочки интро.
func (c *SocialGraphCanvas) ClearPathHighlight() {
	c.mu.Lock()
	c.highlightedNodes = make(map[int64]bool)
	c.highlightedEdges = make(map[string]bool)
	c.mu.Unlock()
	c.Refresh()
}

// SelectNode программно выбирает узел на холсте.
func (c *SocialGraphCanvas) SelectNode(nodeID int64) {
	c.mu.Lock()
	c.selectedNodeID = nodeID
	c.mu.Unlock()
	c.Refresh()
}

// -----------------------------------------------------------------------------
// Реализация интерактивных интерфейсов Fyne v2
// -----------------------------------------------------------------------------

// MouseIn обрабатывает вход курсора в границы холста.
func (c *SocialGraphCanvas) MouseIn(e *desktop.MouseEvent) {}

// MouseMoved обрабатывает перемещение курсора для подсветки узла (Hover).
func (c *SocialGraphCanvas) MouseMoved(e *desktop.MouseEvent) {
	node := c.findNodeAt(e.Position, c.Size())
	c.mu.Lock()
	var newHoverID int64
	if node != nil {
		newHoverID = node.ContactID
	}
	if newHoverID != c.hoveredNodeID {
		c.hoveredNodeID = newHoverID
		c.mu.Unlock()
		if c.OnHoverChanged != nil {
			c.OnHoverChanged(node)
		}
		c.Refresh()
		return
	}
	c.mu.Unlock()
}

// MouseOut сбрасывает состояние Hover при уходе курсора с холста.
func (c *SocialGraphCanvas) MouseOut() {
	c.mu.Lock()
	if c.hoveredNodeID != 0 {
		c.hoveredNodeID = 0
		c.mu.Unlock()
		if c.OnHoverChanged != nil {
			c.OnHoverChanged(nil)
		}
		c.Refresh()
		return
	}
	c.mu.Unlock()
}

// Dragged обрабатывает перетаскивание узла или панорамирование холста (Pan).
func (c *SocialGraphCanvas) Dragged(e *fyne.DragEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.draggedNodeID == 0 && !c.isDraggingWorld {
		// Начало перетаскивания: проверяем узел под курсором
		node := c.findNodeAt(e.Position, c.Size())
		if node != nil {
			c.draggedNodeID = node.ContactID
		} else {
			c.isDraggingWorld = true
		}
	}

	if c.draggedNodeID > 0 {
		// Перетаскиваем узел: конвертируем в мировые координаты
		wx, wy := c.screenToWorld(e.Position, c.Size())
		if c.graphService != nil {
			c.graphService.MoveNode(c.draggedNodeID, wx, wy)
		}
		c.Refresh()
		return
	}

	if c.isDraggingWorld {
		// Панорамируем холст
		c.panX += e.Dragged.DX / c.zoom
		c.panY += e.Dragged.DY / c.zoom
		c.Refresh()
	}
}

// DragEnd фиксирует узел или завершает панорамирование.
func (c *SocialGraphCanvas) DragEnd() {
	c.mu.Lock()
	draggedID := c.draggedNodeID
	c.draggedNodeID = 0
	c.isDraggingWorld = false
	c.mu.Unlock()

	if draggedID > 0 && c.graphService != nil {
		// Подогреваем симуляцию для плавной релаксации
		c.graphService.ReheatSimulation(0.35)
	}
	c.Refresh()
}

// Scrolled масштабирует холст относительно курсора (Zoom 0.20x .. 3.00x).
func (c *SocialGraphCanvas) Scrolled(e *fyne.ScrollEvent) {
	c.mu.Lock()
	delta := e.Scrolled.DY * 0.05
	newZoom := c.zoom + delta
	if newZoom < 0.20 {
		newZoom = 0.20
	} else if newZoom > 3.00 {
		newZoom = 3.00
	}
	c.zoom = newZoom
	c.mu.Unlock()

	c.Refresh()
}

// Tapped обрабатывает одиночный клик для выбора узла или сброса выделения.
func (c *SocialGraphCanvas) Tapped(e *fyne.PointEvent) {
	node := c.findNodeAt(e.Position, c.Size())
	c.mu.Lock()
	if node != nil {
		c.selectedNodeID = node.ContactID
		c.mu.Unlock()
		if c.OnNodeSelected != nil {
			c.OnNodeSelected(node.ContactID)
		}
	} else {
		c.selectedNodeID = 0
		c.mu.Unlock()
		if c.OnCanvasTapped != nil {
			c.OnCanvasTapped()
		}
	}
	c.Refresh()
}

// -----------------------------------------------------------------------------
// Аффинные преобразования экранных и мировых координат
// -----------------------------------------------------------------------------

func (c *SocialGraphCanvas) worldToScreen(wx, wy float64, size fyne.Size) fyne.Position {
	cx := float64(size.Width / 2.0)
	cy := float64(size.Height / 2.0)
	sx := cx + (wx+float64(c.panX))*float64(c.zoom)
	sy := cy + (wy+float64(c.panY))*float64(c.zoom)
	return fyne.NewPos(float32(sx), float32(sy))
}

func (c *SocialGraphCanvas) screenToWorld(pos fyne.Position, size fyne.Size) (float64, float64) {
	cx := float64(size.Width / 2.0)
	cy := float64(size.Height / 2.0)
	wx := (float64(pos.X)-cx)/float64(c.zoom) - float64(c.panX)
	wy := (float64(pos.Y)-cy)/float64(c.zoom) - float64(c.panY)
	return wx, wy
}

func (c *SocialGraphCanvas) findNodeAt(pos fyne.Position, size fyne.Size) *graph.Node {
	if c.graphService == nil {
		return nil
	}

	nodesMap, _, _ := c.graphService.GetTopologySnapshot()
	snap := c.graphService.GetLatestSnapshot()

	hitRadiusSq := float32(22.0 * 22.0)
	var bestNode *graph.Node
	var bestDistSq = hitRadiusSq

	for id, n := range nodesMap {
		if c.dunbarFilter > 0 && n.DunbarTier > c.dunbarFilter {
			continue
		}

		var wx, wy float64
		if c.mode == GraphModeDunbar {
			wx, wy = calculateDunbarOrbitPos(n, nodesMap)
		} else if snap != nil {
			if p, ok := snap.Positions[id]; ok {
				wx, wy = p.PosX, p.PosY
			} else {
				wx, wy = n.PosX, n.PosY
			}
		} else {
			wx, wy = n.PosX, n.PosY
		}

		sp := c.worldToScreen(wx, wy, size)
		dx := sp.X - pos.X
		dy := sp.Y - pos.Y
		distSq := dx*dx + dy*dy
		if distSq <= bestDistSq {
			bestDistSq = distSq
			bestNode = n
		}
	}
	return bestNode
}

func calculateDunbarOrbitPos(n *graph.Node, allNodes map[int64]*graph.Node) (float64, float64) {
	var radius float64
	switch n.DunbarTier {
	case graph.DunbarSupportClique:
		radius = 70.0
	case graph.DunbarSympathyGroup:
		radius = 135.0
	case graph.DunbarAffinityGroup:
		radius = 205.0
	case graph.DunbarActiveNetwork:
		radius = 280.0
	default:
		radius = 360.0
	}

	// Детерминированный угол по ContactID для стабильного расположения на кольце
	angle := float64(n.ContactID%360) * (math.Pi / 180.0) * 1.61803398875
	return radius * math.Cos(angle), radius * math.Sin(angle)
}

// -----------------------------------------------------------------------------
// Рендерер виджета (fyne.WidgetRenderer)
// -----------------------------------------------------------------------------

// CreateRenderer создаёт специализированный рендерер Fyne для холста.
func (c *SocialGraphCanvas) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(colorCanvasBgDark)

	// Контейнеры слоёв для быстрого переиспользования
	orbitLayer := container.NewWithoutLayout()
	edgeLayer := container.NewWithoutLayout()
	nodeLayer := container.NewWithoutLayout()
	tooltipLayer := container.NewWithoutLayout()

	root := container.NewStack(bg, orbitLayer, edgeLayer, nodeLayer, tooltipLayer)

	r := &graphCanvasRenderer{
		canvas:       c,
		bg:           bg,
		orbitLayer:   orbitLayer,
		edgeLayer:    edgeLayer,
		nodeLayer:    nodeLayer,
		tooltipLayer: tooltipLayer,
		root:         root,
	}
	r.Refresh()
	return r
}

type graphCanvasRenderer struct {
	canvas       *SocialGraphCanvas
	bg           *canvas.Rectangle
	orbitLayer   *fyne.Container
	edgeLayer    *fyne.Container
	nodeLayer    *fyne.Container
	tooltipLayer *fyne.Container
	root         *fyne.Container
}

func (r *graphCanvasRenderer) Destroy() {}

func (r *graphCanvasRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.root.Resize(size)
	r.Refresh()
}

func (r *graphCanvasRenderer) MinSize() fyne.Size {
	return fyne.NewSize(320, 240)
}

func (r *graphCanvasRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.root}
}

func (r *graphCanvasRenderer) Refresh() {
	r.canvas.mu.RLock()
	defer r.canvas.mu.RUnlock()

	size := r.canvas.Size()
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	// 1. Очистка слоёв
	r.orbitLayer.Objects = nil
	r.edgeLayer.Objects = nil
	r.nodeLayer.Objects = nil
	r.tooltipLayer.Objects = nil

	// 2. Отрисовка концентрических орбит Данбара (в режиме Dunbar)
	if r.canvas.mode == GraphModeDunbar {
		tiers := []struct {
			radius float32
			label  string
		}{
			{70, "Орбита 5 (Ядро)"},
			{135, "Орбита 15 (Симпатия)"},
			{205, "Орбита 50 (Доверие)"},
			{280, "Орбита 150 (Активная сеть)"},
			{360, "Орбита 500 (Знакомые)"},
		}

		cx := size.Width/2.0 + r.canvas.panX*r.canvas.zoom
		cy := size.Height/2.0 + r.canvas.panY*r.canvas.zoom

		for _, t := range tiers {
			screenRadius := t.radius * r.canvas.zoom
			circle := canvas.NewCircle(color.Transparent)
			circle.StrokeColor = colorOrbitRingDark
			circle.StrokeWidth = 1.2
			circle.Resize(fyne.NewSize(screenRadius*2, screenRadius*2))
			circle.Move(fyne.NewPos(cx-screenRadius, cy-screenRadius))
			r.orbitLayer.Add(circle)

			if r.canvas.zoom > 0.40 {
				txt := canvas.NewText(t.label, colorOrbitRingLight)
				txt.TextSize = 10
				txt.Move(fyne.NewPos(cx-screenRadius+8, cy-8))
				r.orbitLayer.Add(txt)
			}
		}
	}

	if r.canvas.graphService == nil {
		return
	}

	nodesMap, edges, _ := r.canvas.graphService.GetTopologySnapshot()
	snap := r.canvas.graphService.GetLatestSnapshot()

	// Кэш экранных позиций узлов на этом кадре
	screenPositions := make(map[int64]fyne.Position, len(nodesMap))
	for id, n := range nodesMap {
		if r.canvas.dunbarFilter > 0 && n.DunbarTier > r.canvas.dunbarFilter {
			continue
		}
		var wx, wy float64
		if r.canvas.mode == GraphModeDunbar {
			wx, wy = calculateDunbarOrbitPos(n, nodesMap)
		} else if snap != nil {
			if p, ok := snap.Positions[id]; ok {
				wx, wy = p.PosX, p.PosY
			} else {
				wx, wy = n.PosX, n.PosY
			}
		} else {
			wx, wy = n.PosX, n.PosY
		}
		screenPositions[id] = r.canvas.worldToScreen(wx, wy, size)
	}

	// 3. Отрисовка рёбер графа
	for _, e := range edges {
		if e == nil {
			continue
		}
		if !r.canvas.showImplicit && e.Type == graph.RelImplicitColleague {
			continue
		}

		p1, ok1 := screenPositions[e.SourceID]
		p2, ok2 := screenPositions[e.TargetID]
		if !ok1 || !ok2 {
			continue
		}

		line := canvas.NewLine(colorEdgeDefault)
		line.Position1 = p1
		line.Position2 = p2
		line.StrokeWidth = float32(1.0 + float32(e.Weight)*0.5)

		if e.Type == graph.RelImplicitColleague {
			line.StrokeColor = colorEdgeImplicit
			line.StrokeWidth = 1.0
		}

		key := fmt.Sprintf("%d-%d", e.SourceID, e.TargetID)
		if r.canvas.highlightedEdges[key] {
			line.StrokeColor = colorEdgeIntroPath
			line.StrokeWidth = 3.5
		}

		r.edgeLayer.Add(line)
	}

	// 4. Отрисовка узлов (кружки + инициалы + бейджи)
	nodeRadius := float32(14.0 * math.Max(0.7, math.Min(1.6, float64(r.canvas.zoom))))

	for id, n := range nodesMap {
		sp, ok := screenPositions[id]
		if !ok {
			continue
		}

		// Внешняя обводка (Брокер сети или Выделение)
		isBroker := n.BetweennessScore > 0.30
		isSelected := r.canvas.selectedNodeID == id
		isHighlighted := r.canvas.highlightedNodes[id]

		if isSelected || isBroker || isHighlighted {
			outerRing := canvas.NewCircle(color.Transparent)
			outerRing.StrokeWidth = 2.5
			if isSelected {
				outerRing.StrokeColor = colorNodeSelected
			} else if isHighlighted {
				outerRing.StrokeColor = colorEdgeIntroPath
			} else {
				outerRing.StrokeColor = colorNodeBroker
			}
			outerRing.Resize(fyne.NewSize((nodeRadius+4)*2, (nodeRadius+4)*2))
			outerRing.Move(fyne.NewPos(sp.X-(nodeRadius+4), sp.Y-(nodeRadius+4)))
			r.nodeLayer.Add(outerRing)
		}

		// Тело узла с цветом сообщества
		commIdx := n.CommunityID % len(communityColors)
		if commIdx < 0 {
			commIdx = 0
		}
		nodeCircle := canvas.NewCircle(communityColors[commIdx])
		nodeCircle.Resize(fyne.NewSize(nodeRadius*2, nodeRadius*2))
		nodeCircle.Move(fyne.NewPos(sp.X-nodeRadius, sp.Y-nodeRadius))
		r.nodeLayer.Add(nodeCircle)

		// Инициалы узла (LOD: скрываем при zoom < 0.45)
		if r.canvas.zoom >= 0.45 {
			txt := canvas.NewText(n.AvatarInitials, color.White)
			txt.TextSize = nodeRadius * 0.85
			txt.TextStyle = fyne.TextStyle{Bold: true}
			txt.Alignment = fyne.TextAlignCenter
			txt.Move(fyne.NewPos(sp.X-nodeRadius, sp.Y-txt.TextSize*0.6))
			r.nodeLayer.Add(txt)
		}

		// Текстовая подпись имени под узлом при нормальном масштабе
		if r.canvas.zoom >= 0.75 {
			nameLabel := canvas.NewText(n.FullName, color.NRGBA{R: 226, G: 232, B: 240, A: 240})
			nameLabel.TextSize = 10
			nameLabel.Alignment = fyne.TextAlignCenter
			nameLabel.Move(fyne.NewPos(sp.X-50, sp.Y+nodeRadius+2))
			r.nodeLayer.Add(nameLabel)
		}
	}

	// 5. Отрисовка плавающего тултипа для hovered узла
	if r.canvas.hoveredNodeID > 0 {
		if hNode, ok := nodesMap[r.canvas.hoveredNodeID]; ok {
			if sp, ok := screenPositions[r.canvas.hoveredNodeID]; ok {
				tipBg := canvas.NewRectangle(color.NRGBA{R: 15, G: 23, B: 42, A: 235})
				tipBg.CornerRadius = 4
				tipBg.StrokeColor = color.NRGBA{R: 51, G: 65, B: 85, A: 255}
				tipBg.StrokeWidth = 1.0

				infoText := fmt.Sprintf("%s\n%s • %s", hNode.FullName, valueOrDash(hNode.Company), hNode.DunbarTier.Label())
				tipTxt := canvas.NewText(infoText, color.White)
				tipTxt.TextSize = 11

				tipWidth := float32(200.0)
				tipHeight := float32(38.0)
				tipX := sp.X + 15
				tipY := sp.Y - 20

				tipBg.Resize(fyne.NewSize(tipWidth, tipHeight))
				tipBg.Move(fyne.NewPos(tipX, tipY))
				tipTxt.Move(fyne.NewPos(tipX+8, tipY+4))

				r.tooltipLayer.Add(tipBg)
				r.tooltipLayer.Add(tipTxt)
			}
		}
	}
}
