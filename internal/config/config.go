// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config хранит конфигурационные параметры приложения (Неделя 14).
type Config struct {
	AppName           string `json:"app_name"`
	Version           string `json:"version"`
	DatabasePath      string `json:"database_path"`
	BackupDir         string `json:"backup_dir"`
	AutoBackupMinutes int    `json:"auto_backup_minutes"`
	Theme             string `json:"theme"`
	WindowWidth       int    `json:"window_width"`
	WindowHeight      int    `json:"window_height"`
	NetworkTimeoutSec int    `json:"network_timeout_sec"`
	RandomUserAPIURL  string `json:"random_user_api_url"`
	LocalAPIPort      int    `json:"local_api_port"`
	WorkerPoolSize    int    `json:"worker_pool_size"`
}

// DefaultConfig возвращает конфигурацию по умолчанию, если файл не найден.
func DefaultConfig() *Config {
	return &Config{
		AppName:           "GoContact Studio — Адресная книга",
		Version:           "1.0.0",
		DatabasePath:      "data/addressbook.db",
		BackupDir:         "data/backups",
		AutoBackupMinutes: 10,
		Theme:             "dark",
		WindowWidth:       1240,
		WindowHeight:      780,
		NetworkTimeoutSec: 8,
		RandomUserAPIURL:  "https://randomuser.me/api/",
		LocalAPIPort:      8089,
		WorkerPoolSize:    4,
	}
}

// Load читает конфигурационный файл JSON с диска или создаёт дефолтный.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Автоматически сохраняем дефолтный конфиг при первом запуске
			_ = Save(path, cfg)
			return cfg, nil
		}
		return cfg, fmt.Errorf("ошибка чтения файла конфигурации %s: %w", path, err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return DefaultConfig(), fmt.Errorf("ошибка парсинга JSON конфигурации: %w", err)
	}

	cfg.normalize()
	return cfg, nil
}

// Save сериализует конфигурацию в JSON и записывает на диск.
func Save(path string, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("конфигурация не может быть nil")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию конфигурации: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации конфигурации: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

func (c *Config) normalize() {
	if c.WindowWidth < 800 {
		c.WindowWidth = 1240
	}
	if c.WindowHeight < 500 {
		c.WindowHeight = 780
	}
	if c.WorkerPoolSize <= 0 {
		c.WorkerPoolSize = 4
	}
	if c.NetworkTimeoutSec <= 0 {
		c.NetworkTimeoutSec = 8
	}
	if c.DatabasePath == "" {
		c.DatabasePath = "data/addressbook.db"
	}
	if c.BackupDir == "" {
		c.BackupDir = "data/backups"
	}
}
