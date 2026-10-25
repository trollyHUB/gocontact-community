// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// PhoneType определяет классификацию номера телефона (Неделя 4: константы и пользовательские типы).
type PhoneType string

const (
	PhoneMobile PhoneType = "mobile"
	PhoneWork   PhoneType = "work"
	PhoneHome   PhoneType = "home"
	PhoneOther  PhoneType = "other"
)

// Label возвращает русскоязычное название типа телефона (Неделя 4: switch).
func (pt PhoneType) Label() string {
	switch pt {
	case PhoneMobile:
		return "Мобильный"
	case PhoneWork:
		return "Рабочий"
	case PhoneHome:
		return "Домашний"
	default:
		return "Другой"
	}
}

// AuditInfo хранит временные метки создания и изменения записи.
// Используется для демонстрации композиции (встраивания структур — Неделя 6).
type AuditInfo struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Touch обновляет время модификации записи (метод с указателем — Недели 5, 6).
func (a *AuditInfo) Touch() {
	now := time.Now()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
}

// Address представляет физический/почтовый адрес контакта (Недели 5, 6: композиция).
type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	Country    string `json:"country"`
	PostalCode string `json:"postal_code"`
}

// Format возвращает отформатированный строковый адрес.
func (a Address) Format() string {
	parts := make([]string, 0, 4)
	if strings.TrimSpace(a.Country) != "" {
		parts = append(parts, strings.TrimSpace(a.Country))
	}
	if strings.TrimSpace(a.City) != "" {
		parts = append(parts, "г. "+strings.TrimSpace(a.City))
	}
	if strings.TrimSpace(a.Street) != "" {
		parts = append(parts, strings.TrimSpace(a.Street))
	}
	if strings.TrimSpace(a.PostalCode) != "" {
		parts = append(parts, fmt.Sprintf("(%s)", strings.TrimSpace(a.PostalCode)))
	}
	if len(parts) == 0 {
		return "Адрес не указан"
	}
	return strings.Join(parts, ", ")
}

// IsEmpty проверяет, заполнено ли хотя бы одно поле адреса.
func (a Address) IsEmpty() bool {
	return strings.TrimSpace(a.Street) == "" &&
		strings.TrimSpace(a.City) == "" &&
		strings.TrimSpace(a.Country) == "" &&
		strings.TrimSpace(a.PostalCode) == ""
}

// Contact — главная доменная сущность «Контакт» адресной книги (Недели 4, 5, 6).
type Contact struct {
	ID         int64     `json:"id"`
	GroupID    int64     `json:"group_id"`
	GroupName  string    `json:"group_name,omitempty"`
	FirstName  string    `json:"first_name"`
	LastName   string    `json:"last_name"`
	MiddleName string    `json:"middle_name,omitempty"`
	Phone      string    `json:"phone"`
	WorkPhone  string    `json:"work_phone,omitempty"`
	PhoneType  PhoneType `json:"phone_type"`
	Email      string    `json:"email,omitempty"`
	Company    string    `json:"company,omitempty"`
	JobTitle   string    `json:"job_title,omitempty"`
	Address    Address   `json:"address"` // Вложенная структура (композиция)
	Birthday   string    `json:"birthday,omitempty"`
	Tags       []string  `json:"tags"` // Срез (slice) тегов (Неделя 5)
	Notes      string    `json:"notes,omitempty"`
	IsFavorite bool      `json:"is_favorite"`
	AvatarURL  string    `json:"avatar_url,omitempty"`
	AuditInfo            // Встроенная структура (Struct Embedding — Неделя 6)
}

// FullName возвращает полное ФИО контакта в формате «Фамилия Имя Отчество».
func (c Contact) FullName() string {
	parts := make([]string, 0, 3)
	if ln := strings.TrimSpace(c.LastName); ln != "" {
		parts = append(parts, ln)
	}
	if fn := strings.TrimSpace(c.FirstName); fn != "" {
		parts = append(parts, fn)
	}
	if mn := strings.TrimSpace(c.MiddleName); mn != "" {
		parts = append(parts, mn)
	}
	if len(parts) == 0 {
		return "Без имени"
	}
	return strings.Join(parts, " ")
}

// DisplayTitle возвращает краткую подпись: должность и компанию.
func (c Contact) DisplayTitle() string {
	job := strings.TrimSpace(c.JobTitle)
	comp := strings.TrimSpace(c.Company)
	switch {
	case job != "" && comp != "":
		return fmt.Sprintf("%s @ %s", job, comp)
	case comp != "":
		return comp
	case job != "":
		return job
	default:
		return "Личный контакт"
	}
}

// Initials возвращает 1–2 заглавные буквы имени и фамилии для аватарки.
func (c Contact) Initials() string {
	var res []rune
	if r := firstRuneUpper(c.LastName); r != 0 {
		res = append(res, r)
	}
	if r := firstRuneUpper(c.FirstName); r != 0 {
		res = append(res, r)
	}
	if len(res) == 0 {
		return "К"
	}
	return string(res)
}

func firstRuneUpper(s string) rune {
	for _, r := range strings.TrimSpace(s) {
		return unicode.ToUpper(r)
	}
	return 0
}

// ToggleFavorite переключает статус избранного контакта (работа через указатель *Contact — Неделя 5).
func (c *Contact) ToggleFavorite() bool {
	c.IsFavorite = !c.IsFavorite
	c.Touch()
	return c.IsFavorite
}

// AddTag добавляет уникальный тег в срез Tags контакта (работа со слайсами и циклами — Недели 4, 5).
func (c *Contact) AddTag(tag string) {
	clean := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(tag, "#")))
	if clean == "" {
		return
	}
	for _, existing := range c.Tags {
		if existing == clean {
			return
		}
	}
	c.Tags = append(c.Tags, clean)
	c.Touch()
}

// RemoveTag удаляет тег из среза Tags.
func (c *Contact) RemoveTag(tag string) {
	clean := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(tag, "#")))
	filtered := make([]string, 0, len(c.Tags))
	for _, existing := range c.Tags {
		if existing != clean {
			filtered = append(filtered, existing)
		}
	}
	c.Tags = filtered
	c.Touch()
}

// FormattedTags возвращает строку тегов с префиксом # (например, "#golang #work").
func (c Contact) FormattedTags() string {
	if len(c.Tags) == 0 {
		return "Нет тегов"
	}
	out := make([]string, 0, len(c.Tags))
	for _, t := range c.Tags {
		if strings.TrimSpace(t) != "" {
			out = append(out, "#"+strings.TrimSpace(t))
		}
	}
	return strings.Join(out, " ")
}
