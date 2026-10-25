// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// ContactRepo реализует интерфейс repository.ContactRepository на базе СУБД SQLite.
type ContactRepo struct {
	db *sql.DB
}

// NewContactRepo создаёт новый экземпляр SQLite-репозитория контактов.
func NewContactRepo(db *sql.DB) *ContactRepo {
	return &ContactRepo{db: db}
}

// Create добавляет новый контакт в таблицу contacts (SQL INSERT).
func (r *ContactRepo) Create(ctx context.Context, c *domain.Contact) error {
	if err := domain.NormalizeAndValidateContact(c); err != nil {
		return err
	}

	tagsBytes, err := json.Marshal(c.Tags)
	if err != nil {
		tagsBytes = []byte("[]")
	}

	fav := 0
	if c.IsFavorite {
		fav = 1
	}

	var groupID sql.NullInt64
	if c.GroupID > 0 {
		groupID = sql.NullInt64{Int64: c.GroupID, Valid: true}
	}

	nowStr := c.UpdatedAt.UTC().Format(time.RFC3339)
	createdStr := c.CreatedAt.UTC().Format(time.RFC3339)

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO contacts (
			group_id, first_name, last_name, middle_name, phone, work_phone,
			phone_type, email, company, job_title, street, city, country,
			postal_code, birthday, tags_json, notes, is_favorite, avatar_url,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		groupID, c.FirstName, c.LastName, c.MiddleName, c.Phone, c.WorkPhone,
		string(c.PhoneType), c.Email, c.Company, c.JobTitle,
		c.Address.Street, c.Address.City, c.Address.Country, c.Address.PostalCode,
		c.Birthday, string(tagsBytes), c.Notes, fav, c.AvatarURL, createdStr, nowStr,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return domain.ErrDuplicatePhone
		}
		return fmt.Errorf("ошибка SQL INSERT контакта: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		c.ID = id
	}
	return nil
}

// GetByID возвращает контакт по его уникальному ID (SQL SELECT + LEFT JOIN).
func (r *ContactRepo) GetByID(ctx context.Context, id int64) (*domain.Contact, error) {
	row := r.db.QueryRowContext(ctx, baseSelectQuery+` WHERE c.id = ?`, id)
	c, err := scanContactRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrContactNotFound
		}
		return nil, fmt.Errorf("ошибка получения контакта по ID=%d: %w", id, err)
	}
	return c, nil
}

// GetByPhone ищет контакт по номеру телефона (SQL SELECT).
func (r *ContactRepo) GetByPhone(ctx context.Context, phone string) (*domain.Contact, error) {
	norm, err := domain.NormalizePhone(phone)
	if err != nil {
		norm = strings.TrimSpace(phone)
	}
	row := r.db.QueryRowContext(ctx, baseSelectQuery+` WHERE c.phone = ?`, norm)
	c, err := scanContactRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrContactNotFound
		}
		return nil, fmt.Errorf("ошибка получения контакта по телефону: %w", err)
	}
	return c, nil
}

// Update обновляет существующий контакт в БД (SQL UPDATE).
func (r *ContactRepo) Update(ctx context.Context, c *domain.Contact) error {
	if c == nil || c.ID <= 0 {
		return domain.ErrContactNotFound
	}
	if err := domain.NormalizeAndValidateContact(c); err != nil {
		return err
	}

	tagsBytes, _ := json.Marshal(c.Tags)
	fav := 0
	if c.IsFavorite {
		fav = 1
	}

	var groupID sql.NullInt64
	if c.GroupID > 0 {
		groupID = sql.NullInt64{Int64: c.GroupID, Valid: true}
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE contacts SET
			group_id = ?, first_name = ?, last_name = ?, middle_name = ?,
			phone = ?, work_phone = ?, phone_type = ?, email = ?,
			company = ?, job_title = ?, street = ?, city = ?, country = ?,
			postal_code = ?, birthday = ?, tags_json = ?, notes = ?,
			is_favorite = ?, avatar_url = ?, updated_at = ?
		WHERE id = ?
	`,
		groupID, c.FirstName, c.LastName, c.MiddleName,
		c.Phone, c.WorkPhone, string(c.PhoneType), c.Email,
		c.Company, c.JobTitle, c.Address.Street, c.Address.City, c.Address.Country,
		c.Address.PostalCode, c.Birthday, string(tagsBytes), c.Notes,
		fav, c.AvatarURL, c.UpdatedAt.UTC().Format(time.RFC3339), c.ID,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return domain.ErrDuplicatePhone
		}
		return fmt.Errorf("ошибка SQL UPDATE контакта ID=%d: %w", c.ID, err)
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		return domain.ErrContactNotFound
	}
	return nil
}

// Delete удаляет контакт по его ID (SQL DELETE).
func (r *ContactRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM contacts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("ошибка SQL DELETE контакта ID=%d: %w", id, err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return domain.ErrContactNotFound
	}
	return nil
}

// List выполняет выборку, фильтрацию, регистронезависимый поиск (включая кириллицу) и сортировку (SQL SELECT).
func (r *ContactRepo) List(ctx context.Context, filter domain.ContactFilter) ([]*domain.Contact, error) {
	var conditions []string
	var args []any

	if filter.GroupID > 0 {
		conditions = append(conditions, "c.group_id = ?")
		args = append(args, filter.GroupID)
	}
	if filter.OnlyFavorite {
		conditions = append(conditions, "c.is_favorite = 1")
	}
	if strings.TrimSpace(filter.City) != "" {
		conditions = append(conditions, "c.city = ?")
		args = append(args, strings.TrimSpace(filter.City))
	}

	query := baseSelectQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	switch filter.SortBy {
	case domain.SortByNameDesc:
		query += " ORDER BY c.last_name DESC, c.first_name DESC"
	case domain.SortByCompanyAsc:
		query += " ORDER BY CASE WHEN c.company = '' THEN 1 ELSE 0 END, c.company ASC, c.last_name ASC"
	case domain.SortByCreatedDesc:
		query += " ORDER BY c.created_at DESC, c.id DESC"
	default:
		query += " ORDER BY c.is_favorite DESC, c.last_name ASC, c.first_name ASC"
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ошибка SQL SELECT списка контактов: %w", err)
	}
	defer rows.Close()

	qLower := strings.ToLower(strings.TrimSpace(filter.Query))
	var result []*domain.Contact

	for rows.Next() {
		c, err := scanContactRow(rows)
		if err != nil {
			return nil, err
		}
		if qLower != "" && !matchesSearchQuery(c, qLower) {
			continue
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// Stats вычисляет сводную статистику по всей адресной книге (SQL COUNT / GROUP BY + maps).
func (r *ContactRepo) Stats(ctx context.Context) (*domain.BookStats, error) {
	stats := &domain.BookStats{
		ByGroup: make(map[string]int),
		ByCity:  make(map[string]int),
	}

	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN is_favorite = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN email <> '' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN company <> '' THEN 1 ELSE 0 END), 0)
		FROM contacts
	`).Scan(&stats.TotalContacts, &stats.FavoriteContacts, &stats.WithEmailCount, &stats.WithCompanyCount)
	if err != nil {
		return nil, fmt.Errorf("ошибка подсчёта общей статистики: %w", err)
	}

	groupRows, err := r.db.QueryContext(ctx, `
		SELECT COALESCE(g.name, 'Без группы'), COUNT(c.id)
		FROM groups g
		LEFT JOIN contacts c ON c.group_id = g.id
		GROUP BY g.id, g.name
	`)
	if err == nil {
		defer groupRows.Close()
		for groupRows.Next() {
			var gName string
			var cnt int
			if err := groupRows.Scan(&gName, &cnt); err == nil {
				stats.ByGroup[gName] = cnt
			}
		}
	}

	cityRows, err := r.db.QueryContext(ctx, `
		SELECT CASE WHEN city = '' THEN 'Не указан' ELSE city END AS city_name, COUNT(*)
		FROM contacts
		GROUP BY city_name
	`)
	if err == nil {
		defer cityRows.Close()
		for cityRows.Next() {
			var cName string
			var cnt int
			if err := cityRows.Scan(&cName, &cnt); err == nil {
				stats.ByCity[cName] = cnt
			}
		}
	}

	return stats, nil
}

const baseSelectQuery = `
	SELECT
		c.id, COALESCE(c.group_id, 0), COALESCE(g.name, 'Без группы'),
		c.first_name, c.last_name, c.middle_name, c.phone, c.work_phone,
		c.phone_type, c.email, c.company, c.job_title,
		c.street, c.city, c.country, c.postal_code,
		c.birthday, c.tags_json, c.notes, c.is_favorite, c.avatar_url,
		c.created_at, c.updated_at
	FROM contacts c
	LEFT JOIN groups g ON c.group_id = g.id
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanContactRow(s rowScanner) (*domain.Contact, error) {
	var c domain.Contact
	var phoneTypeStr, tagsJSON, createdStr, updatedStr string
	var favInt int

	err := s.Scan(
		&c.ID, &c.GroupID, &c.GroupName,
		&c.FirstName, &c.LastName, &c.MiddleName, &c.Phone, &c.WorkPhone,
		&phoneTypeStr, &c.Email, &c.Company, &c.JobTitle,
		&c.Address.Street, &c.Address.City, &c.Address.Country, &c.Address.PostalCode,
		&c.Birthday, &tagsJSON, &c.Notes, &favInt, &c.AvatarURL,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	c.PhoneType = domain.PhoneType(phoneTypeStr)
	c.IsFavorite = favInt == 1
	_ = json.Unmarshal([]byte(tagsJSON), &c.Tags)
	if c.Tags == nil {
		c.Tags = []string{}
	}

	if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
		c.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
		c.UpdatedAt = t
	}

	return &c, nil
}

// matchesSearchQuery проверяет вхождение поисковой подстроки (регистронезависимо для кириллицы и латиницы).
func matchesSearchQuery(c *domain.Contact, qLower string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		c.FirstName,
		c.LastName,
		c.MiddleName,
		c.Phone,
		c.WorkPhone,
		c.Email,
		c.Company,
		c.JobTitle,
		c.Address.City,
		c.Address.Country,
		c.Address.Street,
		c.GroupName,
		c.Notes,
		strings.Join(c.Tags, " "),
	}, " "))
	return strings.Contains(haystack, qLower)
}
