// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package domain

import "time"

// Количество базовых предустановленных групп (константа — Неделя 4).
const DefaultGroupsCount = 5

const (
	GroupFamily  int64 = 1
	GroupWork    int64 = 2
	GroupFriends int64 = 3
	GroupVIP     int64 = 4
	GroupOther   int64 = 5
)

// Group представляет категорию/группу контактов (Неделя 5: struct).
type Group struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	ColorHex     string    `json:"color_hex"`
	Description  string    `json:"description"`
	ContactCount int       `json:"contact_count,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// DefaultGroups возвращает массив фиксированной длины [DefaultGroupsCount]Group (Неделя 5: arrays).
func DefaultGroups() [DefaultGroupsCount]Group {
	now := time.Now()
	return [DefaultGroupsCount]Group{
		{ID: 1, Name: "Семья", ColorHex: "#EF4444", Description: "Близкие родственники и члены семьи", CreatedAt: now},
		{ID: 2, Name: "Работа", ColorHex: "#3B82F6", Description: "Коллеги, партнёры и клиенты", CreatedAt: now},
		{ID: 3, Name: "Друзья", ColorHex: "#10B981", Description: "Друзья и товарищи", CreatedAt: now},
		{ID: 4, Name: "VIP", ColorHex: "#F59E0B", Description: "Особо важные контакты", CreatedAt: now},
		{ID: 5, Name: "Прочее", ColorHex: "#6B7280", Description: "Общие контакты", CreatedAt: now},
	}
}

// SortBy определяет поле сортировки списка контактов.
type SortBy string

const (
	SortByNameAsc     SortBy = "name_asc"
	SortByNameDesc    SortBy = "name_desc"
	SortByCompanyAsc  SortBy = "company_asc"
	SortByCreatedDesc SortBy = "created_desc"
)

// ContactFilter задаёт параметры поиска, фильтрации и сортировки (Неделя 12).
type ContactFilter struct {
	Query        string // Поисковый запрос (ФИО, телефон, email, компания, город)
	GroupID      int64  // 0 = все группы
	OnlyFavorite bool   // true = только избранные контакты
	City         string // Фильтр по городу
	SortBy       SortBy // Режим сортировки
}

// BookStats агрегирует статистику адресной книги для аналитики и статус-бара.
type BookStats struct {
	TotalContacts    int            `json:"total_contacts"`
	FavoriteContacts int            `json:"favorite_contacts"`
	WithEmailCount   int            `json:"with_email_count"`
	WithCompanyCount int            `json:"with_company_count"`
	ByGroup          map[string]int `json:"by_group"` // Неделя 5: maps
	ByCity           map[string]int `json:"by_city"`
}
