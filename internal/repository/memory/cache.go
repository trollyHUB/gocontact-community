// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package memory

import (
	"sync"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// ContactCache — потокобезопасный In-Memory кэш и индекс контактов на базе map и sync.RWMutex (Недели 5, 10).
type ContactCache struct {
	mu      sync.RWMutex
	byID    map[int64]*domain.Contact
	byPhone map[string]int64
}

// NewContactCache создаёт новый экземпляр потокобезопасного кэша.
func NewContactCache() *ContactCache {
	return &ContactCache{
		byID:    make(map[int64]*domain.Contact),
		byPhone: make(map[string]int64),
	}
}

// Set сохраняет или обновляет контакт в кэше (эксклюзивная блокировка).
func (c *ContactCache) Set(contact *domain.Contact) {
	if contact == nil || contact.ID == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if old, exists := c.byID[contact.ID]; exists && old.Phone != contact.Phone {
		delete(c.byPhone, old.Phone)
	}

	clone := *contact
	if len(contact.Tags) > 0 {
		clone.Tags = append([]string(nil), contact.Tags...)
	}
	c.byID[contact.ID] = &clone
	c.byPhone[contact.Phone] = contact.ID
}

// SetBatch массово перезаполняет кэш срезом контактов.
func (c *ContactCache) SetBatch(contacts []*domain.Contact) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.byID = make(map[int64]*domain.Contact, len(contacts))
	c.byPhone = make(map[string]int64, len(contacts))

	for _, item := range contacts {
		if item == nil || item.ID == 0 {
			continue
		}
		clone := *item
		if len(item.Tags) > 0 {
			clone.Tags = append([]string(nil), item.Tags...)
		}
		c.byID[item.ID] = &clone
		c.byPhone[item.Phone] = item.ID
	}
}

// GetByID возвращает копию контакта из кэша по ID (разделяемая блокировка чтения).
func (c *ContactCache) GetByID(id int64) (*domain.Contact, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.byID[id]
	if !ok {
		return nil, false
	}
	clone := *item
	if len(item.Tags) > 0 {
		clone.Tags = append([]string(nil), item.Tags...)
	}
	return &clone, true
}

// ExistsPhone быстро проверяет наличие номера телефона в памяти без обращения к БД.
func (c *ContactCache) ExistsPhone(phone string) (int64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	id, ok := c.byPhone[phone]
	return id, ok
}

// Delete удаляет контакт из кэша.
func (c *ContactCache) Delete(id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if old, exists := c.byID[id]; exists {
		delete(c.byPhone, old.Phone)
		delete(c.byID, id)
	}
}

// Size возвращает текущее количество контактов в кэше.
func (c *ContactCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.byID)
}
