// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package assets

import (
	"bytes"
	"image/png"
	"testing"
)

func TestAppIconPNG_EmbeddedResource(t *testing.T) {
	if len(AppIconPNG) == 0 {
		t.Fatal("встроенная иконка AppIconPNG (go:embed icon.png) не должна быть пустой")
	}

	// Проверяем сигнатуру PNG и валидность изображения
	img, err := png.Decode(bytes.NewReader(AppIconPNG))
	if err != nil {
		t.Fatalf("встроенный ресурс icon.png не является корректным PNG изображением: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		t.Errorf("некорректные размеры иконки: %dx%d", bounds.Dx(), bounds.Dy())
	}
}
