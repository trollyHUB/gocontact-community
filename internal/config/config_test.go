// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package config

import (
	"path/filepath"
	"testing"
)

func TestConfig_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test_config.json")

	// 1. При отсутствии файла должен создаться дефолтный конфиг
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("неожиданная ошибка при первом Load: %v", err)
	}
	if cfg.WindowWidth != 1240 || cfg.Theme != "dark" {
		t.Errorf("неверные параметры по умолчанию: %+v", cfg)
	}

	// 2. Изменяем параметры и сохраняем
	cfg.Theme = "light"
	cfg.WindowWidth = 400 // Меньше минимума 800 — при загрузке должно нормализоваться
	cfg.WorkerPoolSize = -2

	if err := Save(cfgPath, cfg); err != nil {
		t.Fatalf("ошибка Save: %v", err)
	}

	// 3. Повторно загружаем и проверяем нормализацию
	loaded, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("ошибка повторного Load: %v", err)
	}
	if loaded.Theme != "light" {
		t.Errorf("ожидалась тема 'light', получено %q", loaded.Theme)
	}
	if loaded.WindowWidth != 1240 {
		t.Errorf("ожидалась нормализация ширины окна до 1240, получено %d", loaded.WindowWidth)
	}
	if loaded.WorkerPoolSize != 4 {
		t.Errorf("ожидалась нормализация WorkerPoolSize до 4, получено %d", loaded.WorkerPoolSize)
	}
}
