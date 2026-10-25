// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package service

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// BackupScheduler управляет фоновым созданием резервных копий на базе select и time.Ticker (Неделя 10).
type BackupScheduler struct {
	service        *ContactService
	backupDir      string
	interval       time.Duration
	triggerNowChan chan struct{}
	mu             sync.RWMutex
	lastBackupPath string
	lastBackupTime time.Time
}

// NewBackupScheduler создаёт планировщик автоматического резервного копирования.
func NewBackupScheduler(service *ContactService, backupDir string, intervalMinutes int) *BackupScheduler {
	if intervalMinutes <= 0 {
		intervalMinutes = 10
	}
	return &BackupScheduler{
		service:        service,
		backupDir:      backupDir,
		interval:       time.Duration(intervalMinutes) * time.Minute,
		triggerNowChan: make(chan struct{}, 1),
	}
}

// Start запускает фоновую горутину планировщика с конструкцией select.
func (b *BackupScheduler) Start(ctx context.Context, onBackupDone func(path string, count int, err error)) {
	go func() {
		ticker := time.NewTicker(b.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-b.triggerNowChan:
				path, count, err := b.PerformBackupNow(ctx)
				if onBackupDone != nil {
					onBackupDone(path, count, err)
				}
			case <-ticker.C:
				path, count, err := b.PerformBackupNow(ctx)
				if onBackupDone != nil {
					onBackupDone(path, count, err)
				}
			}
		}
	}()
}

// TriggerNow отправляет сигнал в неблокирующий канал для немедленного создания бэкапа.
func (b *BackupScheduler) TriggerNow() {
	select {
	case b.triggerNowChan <- struct{}{}:
	default:
	}
}

// PerformBackupNow синхронно сохраняет JSON-слепок адресной книги в директорию бэкапов.
func (b *BackupScheduler) PerformBackupNow(ctx context.Context) (string, int, error) {
	fileName := fmt.Sprintf("backup_%s.json", time.Now().Format("20060102_150405"))
	fullPath := filepath.Join(b.backupDir, fileName)

	count, err := b.service.ExportToFile(ctx, fullPath, domain.ContactFilter{})
	if err != nil {
		return "", 0, err
	}

	b.mu.Lock()
	b.lastBackupPath = fullPath
	b.lastBackupTime = time.Now()
	b.mu.Unlock()

	return fullPath, count, nil
}

// LastBackupInfo потокобезопасно возвращает путь и время последнего успешного бэкапа.
func (b *BackupScheduler) LastBackupInfo() (string, time.Time) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.lastBackupPath, b.lastBackupTime
}
