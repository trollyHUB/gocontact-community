// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/trollyHUB/gocontact-studio/internal/domain/crm"
)

// HealthRingBadge — кастомный виджет Fyne v2, отображающий кольцевой индикатор теплоты отношений [0..100%].
type HealthRingBadge struct {
	widget.BaseWidget
	score  int
	status crm.HealthStatus
}

// NewHealthRingBadge создаёт новый экземпляр индикатора здоровья связи.
func NewHealthRingBadge(score int, status crm.HealthStatus) *HealthRingBadge {
	b := &HealthRingBadge{
		score:  score,
		status: status,
	}
	b.ExtendBaseWidget(b)
	return b
}

// SetScore обновляет значение и статус индикатора.
func (b *HealthRingBadge) SetScore(score int, status crm.HealthStatus) {
	b.score = score
	b.status = status
	b.Refresh()
}

// CreateRenderer создаёт рендерер кастомного виджета.
func (b *HealthRingBadge) CreateRenderer() fyne.WidgetRenderer {
	outerCircle := canvas.NewCircle(color.Transparent)
	outerCircle.StrokeWidth = 3.5

	innerBg := canvas.NewCircle(color.NRGBA{R: 20, G: 25, B: 35, A: 220})

	lbl := canvas.NewText(fmt.Sprintf("%d%%", b.score), color.White)
	lbl.TextSize = 10
	lbl.TextStyle = fyne.TextStyle{Bold: true}
	lbl.Alignment = fyne.TextAlignCenter

	r := &healthRingBadgeRenderer{
		badge:       b,
		outerCircle: outerCircle,
		innerBg:     innerBg,
		text:        lbl,
	}
	r.Refresh()
	return r
}

type healthRingBadgeRenderer struct {
	badge       *HealthRingBadge
	outerCircle *canvas.Circle
	innerBg     *canvas.Circle
	text        *canvas.Text
}

func (r *healthRingBadgeRenderer) Destroy() {}

func (r *healthRingBadgeRenderer) Layout(size fyne.Size) {
	dim := fyne.Min(size.Width, size.Height)
	if dim < 16 {
		dim = 16
	}
	r.outerCircle.Resize(fyne.NewSize(dim, dim))
	r.outerCircle.Move(fyne.NewPos((size.Width-dim)/2, (size.Height-dim)/2))

	innerDim := dim - 6
	r.innerBg.Resize(fyne.NewSize(innerDim, innerDim))
	r.innerBg.Move(fyne.NewPos((size.Width-innerDim)/2, (size.Height-innerDim)/2))

	r.text.Resize(size)
	r.text.Move(fyne.NewPos(0, (size.Height-r.text.MinSize().Height)/2))
}

func (r *healthRingBadgeRenderer) MinSize() fyne.Size {
	return fyne.NewSize(38, 38)
}

func (r *healthRingBadgeRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.innerBg, r.outerCircle, r.text}
}

func (r *healthRingBadgeRenderer) Refresh() {
	var strokeColor color.Color
	switch r.badge.status {
	case crm.StatusThriving:
		strokeColor = color.NRGBA{R: 16, G: 185, B: 129, A: 255} // Изумрудный
	case crm.StatusWarm:
		strokeColor = color.NRGBA{R: 59, G: 130, B: 246, A: 255} // Синий / Тонус
	case crm.StatusCooling:
		strokeColor = color.NRGBA{R: 249, G: 115, B: 22, A: 255} // Оранжевый
	default:
		strokeColor = color.NRGBA{R: 239, G: 68, B: 68, A: 255} // Красный / Cold
	}

	r.outerCircle.StrokeColor = strokeColor
	r.text.Text = fmt.Sprintf("%d%%", r.badge.score)
	r.text.Color = strokeColor

	canvas.Refresh(r.outerCircle)
	canvas.Refresh(r.innerBg)
	canvas.Refresh(r.text)
}

// Cursor реализует desktop.Cursorable.
func (b *HealthRingBadge) Cursor() desktop.Cursor {
	return desktop.DefaultCursor
}
