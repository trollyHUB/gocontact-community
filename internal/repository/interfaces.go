// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package repository

import (
	"context"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/domain/crm"
	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
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

// GraphRepository определяет контракт персистентного хранилища связей социального графа и координат узлов.
type GraphRepository interface {
	CreateRelationship(ctx context.Context, rel *graph.Relationship) (int64, error)
	GetRelationship(ctx context.Context, id int64) (*graph.Relationship, error)
	UpdateRelationship(ctx context.Context, rel *graph.Relationship) error
	DeleteRelationship(ctx context.Context, id int64) error
	GetRelationshipsByContact(ctx context.Context, contactID int64) ([]*graph.Relationship, error)
	GetAllRelationships(ctx context.Context) ([]*graph.Relationship, error)
	SaveLayout(ctx context.Context, layout *graph.GraphLayout) error
	SaveLayoutBatch(ctx context.Context, layouts []*graph.GraphLayout) error
	GetLayout(ctx context.Context, contactID int64) (*graph.GraphLayout, error)
	GetAllLayouts(ctx context.Context) (map[int64]*graph.GraphLayout, error)
	QueryEgoSubgraph(ctx context.Context, egoID int64, maxHops int) ([]*graph.EgoNode, error)
	GetImplicitRelationships(ctx context.Context) ([]*graph.Relationship, error)
}

// OrganizationRepository определяет контракт хранилища нормализованных компаний/организаций.
type OrganizationRepository interface {
	CreateOrganization(ctx context.Context, org *graph.Organization) (int64, error)
	GetOrganization(ctx context.Context, id int64) (*graph.Organization, error)
	GetOrganizationByName(ctx context.Context, name string) (*graph.Organization, error)
	ListOrganizations(ctx context.Context) ([]*graph.Organization, error)
	UpdateOrganization(ctx context.Context, org *graph.Organization) error
	DeleteOrganization(ctx context.Context, id int64) error
}

// CRMRepository определяет контракт персистентного хранения сущностей Personal CRM:
// журнала взаимодействий (M:N), обещаний, вех жизни и политик частоты контактов.
type CRMRepository interface {
	// Взаимодействия
	CreateInteraction(ctx context.Context, inter *crm.Interaction) (int64, error)
	GetInteraction(ctx context.Context, id int64) (*crm.Interaction, error)
	ListInteractionsByContact(ctx context.Context, contactID int64, limit int) ([]*crm.Interaction, error)
	DeleteInteraction(ctx context.Context, id int64) error

	// Обещания
	CreatePromise(ctx context.Context, promise *crm.ActionPromise) (int64, error)
	UpdatePromiseStatus(ctx context.Context, id int64, status string) error
	ListPendingPromises(ctx context.Context) ([]*crm.ActionPromise, error)
	ListPromisesByContact(ctx context.Context, contactID int64) ([]*crm.ActionPromise, error)

	// Вехи жизни
	CreateLifeEvent(ctx context.Context, event *crm.LifeEvent) (int64, error)
	ListLifeEventsByContact(ctx context.Context, contactID int64) ([]*crm.LifeEvent, error)
	DeleteLifeEvent(ctx context.Context, id int64) error

	// Политики частоты контактов и здоровье
	GetCadencePolicy(ctx context.Context, contactID int64) (*crm.HealthSnapshot, error)
	UpsertCadencePolicy(ctx context.Context, snap *crm.HealthSnapshot) error
	ListCadencePolicies(ctx context.Context) ([]*crm.HealthSnapshot, error)
}
