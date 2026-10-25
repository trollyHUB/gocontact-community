// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/infrastructure/fileio"
	"github.com/trollyHUB/gocontact-studio/internal/infrastructure/network"
	"github.com/trollyHUB/gocontact-studio/internal/repository"
	"github.com/trollyHUB/gocontact-studio/internal/repository/memory"
)

// ContactService объединяет бизнес-логику работы с контактами, группами, кэшем, файлами и сетью (Недели 6, 10).
type ContactService struct {
	contactRepo repository.ContactRepository
	groupRepo   repository.GroupRepository
	cache       *memory.ContactCache
	apiClient   *network.RandomUserClient
	workerCount int
}

// NewContactService создаёт экземпляр сервисного слоя с внедрением зависимостей (Dependency Injection).
func NewContactService(
	contactRepo repository.ContactRepository,
	groupRepo repository.GroupRepository,
	cache *memory.ContactCache,
	apiClient *network.RandomUserClient,
	workerCount int,
) *ContactService {
	if cache == nil {
		cache = memory.NewContactCache()
	}
	if workerCount <= 0 {
		workerCount = 4
	}
	return &ContactService{
		contactRepo: contactRepo,
		groupRepo:   groupRepo,
		cache:       cache,
		apiClient:   apiClient,
		workerCount: workerCount,
	}
}

// CreateContact валидирует и сохраняет новый контакт в БД и кэш.
func (s *ContactService) CreateContact(ctx context.Context, c *domain.Contact) error {
	if c != nil && strings.TrimSpace(c.AvatarURL) == "" {
		c.AvatarURL = network.GenerateAvatarURL(c)
	}
	if err := s.contactRepo.Create(ctx, c); err != nil {
		return err
	}
	s.cache.Set(c)
	return nil
}

// UpdateContact обновляет данные существующего контакта в БД и кэше.
func (s *ContactService) UpdateContact(ctx context.Context, c *domain.Contact) error {
	if c != nil && strings.TrimSpace(c.AvatarURL) == "" {
		c.AvatarURL = network.GenerateAvatarURL(c)
	}
	if err := s.contactRepo.Update(ctx, c); err != nil {
		return err
	}
	s.cache.Set(c)
	return nil
}

// DeleteContact удаляет контакт из БД и очищает кэш.
func (s *ContactService) DeleteContact(ctx context.Context, id int64) error {
	if err := s.contactRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.cache.Delete(id)
	return nil
}

// ToggleFavorite быстро переключает флаг «Избранное» у контакта.
func (s *ContactService) ToggleFavorite(ctx context.Context, id int64) (*domain.Contact, error) {
	c, err := s.contactRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	c.ToggleFavorite()
	if err := s.contactRepo.Update(ctx, c); err != nil {
		return nil, err
	}
	s.cache.Set(c)
	return c, nil
}

// GetContactByID возвращает контакт (сначала проверяет быстрый кэш в памяти, затем БД).
func (s *ContactService) GetContactByID(ctx context.Context, id int64) (*domain.Contact, error) {
	if cached, ok := s.cache.GetByID(id); ok && cached.GroupName != "" {
		return cached, nil
	}
	c, err := s.contactRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.cache.Set(c)
	return c, nil
}

// ListContacts возвращает отфильтрованный и отсортированный список контактов.
func (s *ContactService) ListContacts(ctx context.Context, filter domain.ContactFilter) ([]*domain.Contact, error) {
	list, err := s.contactRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	if filter.Query == "" && filter.GroupID == 0 && !filter.OnlyFavorite {
		s.cache.SetBatch(list)
	}
	return list, nil
}

// GetStats возвращает аналитическую сводку по адресной книге.
func (s *ContactService) GetStats(ctx context.Context) (*domain.BookStats, error) {
	return s.contactRepo.Stats(ctx)
}

// ListGroups возвращает список всех групп контактов.
func (s *ContactService) ListGroups(ctx context.Context) ([]*domain.Group, error) {
	return s.groupRepo.List(ctx)
}

// CreateGroup создаёт новую пользовательскую категорию контактов.
func (s *ContactService) CreateGroup(ctx context.Context, name, colorHex, description string) (*domain.Group, error) {
	g := &domain.Group{
		Name:        name,
		ColorHex:    colorHex,
		Description: description,
	}
	if err := s.groupRepo.Create(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

// UpdateGroup обновляет существующую категорию контактов (FR-2.2).
func (s *ContactService) UpdateGroup(ctx context.Context, group *domain.Group) error {
	return s.groupRepo.Update(ctx, group)
}

// DeleteGroup удаляет категорию контактов по ID (FR-2.2).
func (s *ContactService) DeleteGroup(ctx context.Context, id int64) error {
	return s.groupRepo.Delete(ctx, id)
}

// ExportToFile экспортирует контакты в файл по его расширению (.json, .csv, .vcf).
func (s *ContactService) ExportToFile(ctx context.Context, path string, filter domain.ContactFilter) (int, error) {
	contacts, err := s.contactRepo.List(ctx, filter)
	if err != nil {
		return 0, err
	}

	var exporter fileio.Exporter
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		exporter = fileio.NewCSVStorage()
	case ".vcf":
		exporter = fileio.NewVCardStorage()
	default:
		exporter = fileio.NewJSONStorage()
	}

	if err := fileio.ExportToFile(path, exporter, contacts); err != nil {
		return 0, err
	}
	return len(contacts), nil
}

// ImportFromFileConcurrent читает файл (.json или .csv) и импортирует контакты через конкурентный пул горутин.
func (s *ContactService) ImportFromFileConcurrent(
	ctx context.Context,
	path string,
	progressChan chan<- float64,
) (*BatchImportSummary, error) {
	var importer fileio.Importer
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		importer = fileio.NewCSVStorage()
	case ".json":
		importer = fileio.NewJSONStorage()
	case ".vcf":
		importer = fileio.NewVCardStorage()
	default:
		return nil, fmt.Errorf("неподдерживаемый формат файла для импорта: %s (используйте .csv, .json или .vcf)", ext)
	}

	rawContacts, err := fileio.ImportFromFile(path, importer)
	if err != nil {
		return nil, err
	}

	return s.ImportBatchConcurrent(ctx, rawContacts, progressChan), nil
}

// FetchAndImportFromAPI загружает контакты по сети из REST API и обрабатывает их через Worker Pool.
func (s *ContactService) FetchAndImportFromAPI(
	ctx context.Context,
	count int,
	progressChan chan<- float64,
) (*BatchImportSummary, string, error) {
	if s.apiClient == nil {
		return nil, "", fmt.Errorf("HTTP-клиент не инициализирован")
	}

	rawContacts, sourceName, err := s.apiClient.FetchContacts(ctx, count)
	if err != nil {
		return nil, "", err
	}

	summary := s.ImportBatchConcurrent(ctx, rawContacts, progressChan)
	return summary, sourceName, nil
}
