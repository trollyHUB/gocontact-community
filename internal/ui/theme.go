// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// StudioTheme реализует кастомную тему оформления GoContact Studio с поддержкой светлого и тёмного режимов (Неделя 9).
type StudioTheme struct {
	variant fyne.ThemeVariant
}

// NewStudioTheme создаёт тему оформления ("dark" или "light").
func NewStudioTheme(mode string) *StudioTheme {
	v := theme.VariantDark
	if mode == "light" {
		v = theme.VariantLight
	}
	return &StudioTheme{variant: v}
}

func (t *StudioTheme) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	if name == theme.ColorNamePrimary {
		return color.NRGBA{R: 59, G: 130, B: 246, A: 255} // Насыщенный синий акцент (#3B82F6)
	}
	return theme.DefaultTheme().Color(name, t.variant)
}

func (t *StudioTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *StudioTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *StudioTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNamePadding {
		return 6
	}
	return theme.DefaultTheme().Size(name)
}

// ParseHexColor преобразует HEX-строку (#RRGGBB) в color.NRGBA для цветных индикаторов групп.
func ParseHexColor(hexStr string) color.NRGBA {
	switch hexStr {
	case "#EF4444":
		return color.NRGBA{R: 239, G: 68, B: 68, A: 255} // Красный (Семья)
	case "#3B82F6":
		return color.NRGBA{R: 59, G: 130, B: 246, A: 255} // Синий (Работа)
	case "#10B981":
		return color.NRGBA{R: 16, G: 185, B: 129, A: 255} // Изумрудный (Друзья)
	case "#F59E0B":
		return color.NRGBA{R: 245, G: 158, B: 11, A: 255} // Янтарный (VIP)
	default:
		return color.NRGBA{R: 107, G: 114, B: 128, A: 255} // Серый (Прочее)
	}
}
