// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// GroupRepo реализует интерфейс repository.GroupRepository на базе SQLite.
type GroupRepo struct {
	db *sql.DB
}

// NewGroupRepo создаёт новый экземпляр репозитория групп.
func NewGroupRepo(db *sql.DB) *GroupRepo {
	return &GroupRepo{db: db}
}

// Create добавляет новую группу контактов (SQL INSERT).
func (r *GroupRepo) Create(ctx context.Context, group *domain.Group) error {
	if group == nil || strings.TrimSpace(group.Name) == "" {
		return errors.New("название группы не может быть пустым")
	}
	group.Name = strings.TrimSpace(group.Name)
	if group.ColorHex == "" {
		group.ColorHex = "#3B82F6"
	}
	if group.CreatedAt.IsZero() {
		group.CreatedAt = time.Now()
	}

	res, err := r.db.ExecContext(ctx,
		`INSERT INTO groups (name, color_hex, description, created_at) VALUES (?, ?, ?, ?)`,
		group.Name, group.ColorHex, group.Description, group.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return domain.ErrDuplicateGroup
		}
		return fmt.Errorf("ошибка создания группы: %w", err)
	}

	id, err := res.LastInsertId()
	if err == nil {
		group.ID = id
	}
	return nil
}

// GetByID возвращает группу по идентификатору (SQL SELECT).
func (r *GroupRepo) GetByID(ctx context.Context, id int64) (*domain.Group, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT g.id, g.name, g.color_hex, g.description, g.created_at, COUNT(c.id)
		FROM groups g
		LEFT JOIN contacts c ON c.group_id = g.id
		WHERE g.id = ?
		GROUP BY g.id
	`, id)

	var g domain.Group
	var createdStr string
	if err := row.Scan(&g.ID, &g.Name, &g.ColorHex, &g.Description, &createdStr, &g.ContactCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrGroupNotFound
		}
		return nil, fmt.Errorf("ошибка чтения группы: %w", err)
	}
	g.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	return &g, nil
}

// List возвращает все группы с количеством контактов в каждой (SQL SELECT + LEFT JOIN + GROUP BY).
func (r *GroupRepo) List(ctx context.Context) ([]*domain.Group, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT g.id, g.name, g.color_hex, g.description, g.created_at, COUNT(c.id) AS total
		FROM groups g
		LEFT JOIN contacts c ON c.group_id = g.id
		GROUP BY g.id, g.name, g.color_hex, g.description, g.created_at
		ORDER BY g.id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка групп: %w", err)
	}
	defer rows.Close()

	var result []*domain.Group
	for rows.Next() {
		var g domain.Group
		var createdStr string
		if err := rows.Scan(&g.ID, &g.Name, &g.ColorHex, &g.Description, &createdStr, &g.ContactCount); err != nil {
			return nil, err
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		result = append(result, &g)
	}
	return result, rows.Err()
}

// Update обновляет название и цвет группы (SQL UPDATE).
func (r *GroupRepo) Update(ctx context.Context, group *domain.Group) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE groups SET name = ?, color_hex = ?, description = ? WHERE id = ?`,
		strings.TrimSpace(group.Name), group.ColorHex, group.Description, group.ID,
	)
	if err != nil {
		return fmt.Errorf("ошибка обновления группы: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return domain.ErrGroupNotFound
	}
	return nil
}

// Delete удаляет группу по ID (SQL DELETE).
func (r *GroupRepo) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM groups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления группы: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return domain.ErrGroupNotFound
	}
	return nil
}
