// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package repository

import (
	"context"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// ContactRepository определяет контракт хранилища контактов.
type ContactRepository interface {
	Create(ctx context.Context, contact *domain.Contact) error
	GetByID(ctx context.Context, id int64) (*domain.Contact, error)
	GetByPhone(ctx context.Context, phone string) (*domain.Contact, error)
	Update(ctx context.Context, contact *domain.Contact) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter domain.ContactFilter) ([]*domain.Contact, error)
	Stats(ctx context.Context) (*domain.BookStats, error)
}

// GroupRepository определяет контракт хранилища групп/категорий контактов.
type GroupRepository interface {
	Create(ctx context.Context, group *domain.Group) error
	GetByID(ctx context.Context, id int64) (*domain.Group, error)
	List(ctx context.Context) ([]*domain.Group, error)
	Update(ctx context.Context, group *domain.Group) error
	Delete(ctx context.Context, id int64) error
}
