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
	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
)

// GraphRepo реализует интерфейс repository.GraphRepository и repository.OrganizationRepository
// на базе СУБД SQLite с поддержкой внешних ключей, каскадного удаления и recursive CTE.
type GraphRepo struct {
	db *sql.DB
}

// NewGraphRepo создаёт новый экземпляр репозитория графа связей.
func NewGraphRepo(db *sql.DB) *GraphRepo {
	return &GraphRepo{db: db}
}

// OrgRepo реализует интерфейс repository.OrganizationRepository.
type OrgRepo struct {
	db *sql.DB
}

// NewOrgRepo создаёт новый экземпляр репозитория организаций.
func NewOrgRepo(db *sql.DB) *OrgRepo {
	return &OrgRepo{db: db}
}

// -----------------------------------------------------------------------------
// Репозиторий связей (GraphRepository)
// -----------------------------------------------------------------------------

// CreateRelationship добавляет новое ребро между контактами в contact_relationships.
func (r *GraphRepo) CreateRelationship(ctx context.Context, rel *graph.Relationship) (int64, error) {
	if err := rel.Validate(); err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	if rel.CreatedAt.IsZero() {
		rel.CreatedAt = now
	}
	rel.UpdatedAt = now

	isDir := 0
	if rel.IsDirected {
		isDir = 1
	}

	createdStr := rel.CreatedAt.Format(time.RFC3339)
	updatedStr := rel.UpdatedAt.Format(time.RFC3339)

	query := `
		INSERT INTO contact_relationships (
			source_id, target_id, rel_type, weight, is_directed,
			context_note, since_date, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	res, err := r.db.ExecContext(ctx, query,
		rel.SourceID, rel.TargetID, string(rel.Type), rel.Weight, isDir,
		rel.ContextNote, rel.SinceDate, createdStr, updatedStr,
	)
	if err != nil {
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "foreign key") {
			return 0, domain.ErrContactNotFound
		}
		if strings.Contains(errLower, "unique") {
			return 0, graph.ErrDuplicateRelationship
		}
		if strings.Contains(errLower, "chk_no_self_loop") {
			return 0, graph.ErrSelfLoopNotAllowed
		}
		return 0, fmt.Errorf("ошибка вставки связи в БД: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("ошибка получения LastInsertId: %w", err)
	}

	rel.ID = id
	return id, nil
}

// GetRelationship возвращает связь по её первичному ключу.
func (r *GraphRepo) GetRelationship(ctx context.Context, id int64) (*graph.Relationship, error) {
	query := `
		SELECT id, source_id, target_id, rel_type, weight, is_directed,
		       context_note, since_date, created_at, updated_at
		FROM contact_relationships
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	rel, err := scanRelationshipRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, graph.ErrRelationshipNotFound
		}
		return nil, fmt.Errorf("ошибка выборки связи ID=%d: %w", id, err)
	}
	return rel, nil
}

// UpdateRelationship обновляет атрибуты существующей связи.
func (r *GraphRepo) UpdateRelationship(ctx context.Context, rel *graph.Relationship) error {
	if rel == nil || rel.ID <= 0 {
		return graph.ErrRelationshipNotFound
	}
	if err := rel.Validate(); err != nil {
		return err
	}

	rel.UpdatedAt = time.Now().UTC()
	isDir := 0
	if rel.IsDirected {
		isDir = 1
	}

	query := `
		UPDATE contact_relationships SET
			source_id = ?, target_id = ?, rel_type = ?, weight = ?,
			is_directed = ?, context_note = ?, since_date = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		rel.SourceID, rel.TargetID, string(rel.Type), rel.Weight,
		isDir, rel.ContextNote, rel.SinceDate, rel.UpdatedAt.Format(time.RFC3339),
		rel.ID,
	)
	if err != nil {
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "foreign key") {
			return domain.ErrContactNotFound
		}
		if strings.Contains(errLower, "unique") {
			return graph.ErrDuplicateRelationship
		}
		if strings.Contains(errLower, "chk_no_self_loop") {
			return graph.ErrSelfLoopNotAllowed
		}
		return fmt.Errorf("ошибка обновления связи ID=%d: %w", rel.ID, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return graph.ErrRelationshipNotFound
	}
	return nil
}

// DeleteRelationship удаляет связь по идентификатору.
func (r *GraphRepo) DeleteRelationship(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM contact_relationships WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления связи ID=%d: %w", id, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return graph.ErrRelationshipNotFound
	}
	return nil
}

// GetRelationshipsByContact возвращает все инцидентные рёбра контакта (исходящие и входящие).
func (r *GraphRepo) GetRelationshipsByContact(ctx context.Context, contactID int64) ([]*graph.Relationship, error) {
	query := `
		SELECT id, source_id, target_id, rel_type, weight, is_directed,
		       context_note, since_date, created_at, updated_at
		FROM contact_relationships
		WHERE source_id = ? OR target_id = ?
		ORDER BY weight DESC, id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, contactID, contactID)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса связей контакта %d: %w", contactID, err)
	}
	defer rows.Close()

	var rels []*graph.Relationship
	for rows.Next() {
		rel, err := scanRelationshipRow(rows)
		if err != nil {
			return nil, err
		}
		rels = append(rels, rel)
	}
	return rels, rows.Err()
}

// GetAllRelationships возвращает весь список рёбер графа из БД.
func (r *GraphRepo) GetAllRelationships(ctx context.Context) ([]*graph.Relationship, error) {
	query := `
		SELECT id, source_id, target_id, rel_type, weight, is_directed,
		       context_note, since_date, created_at, updated_at
		FROM contact_relationships
		ORDER BY id ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки всех связей: %w", err)
	}
	defer rows.Close()

	var rels []*graph.Relationship
	for rows.Next() {
		rel, err := scanRelationshipRow(rows)
		if err != nil {
			return nil, err
		}
		rels = append(rels, rel)
	}
	return rels, rows.Err()
}

const upsertLayoutQuery = `
	INSERT INTO graph_layouts (
		contact_id, pos_x, pos_y, is_pinned, community_id, betweenness_score, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(contact_id) DO UPDATE SET
		pos_x = excluded.pos_x,
		pos_y = excluded.pos_y,
		is_pinned = excluded.is_pinned,
		community_id = excluded.community_id,
		betweenness_score = excluded.betweenness_score,
		updated_at = excluded.updated_at
`

// SaveLayout сохраняет или обновляет координаты одного узла на холсте.
func (r *GraphRepo) SaveLayout(ctx context.Context, layout *graph.GraphLayout) error {
	if err := layout.Validate(); err != nil {
		return err
	}

	layout.UpdatedAt = time.Now().UTC()
	isPinned := 0
	if layout.IsPinned {
		isPinned = 1
	}

	_, err := r.db.ExecContext(ctx, upsertLayoutQuery,
		layout.ContactID, layout.PosX, layout.PosY, isPinned,
		layout.CommunityID, layout.BetweennessScore, layout.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "foreign key") {
			return domain.ErrContactNotFound
		}
		return fmt.Errorf("ошибка сохранения координат узла: %w", err)
	}
	return nil
}

// SaveLayoutBatch выполняет пакетное сохранение координат узлов в единой транзакции (Debounced Batch Flush).
func (r *GraphRepo) SaveLayoutBatch(ctx context.Context, layouts []*graph.GraphLayout) error {
	if len(layouts) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ошибка старта транзакции сохранения лейаута: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, upsertLayoutQuery)
	if err != nil {
		return fmt.Errorf("ошибка подготовки SQL-выражения: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	for _, l := range layouts {
		if err := l.Validate(); err != nil {
			return err
		}
		l.UpdatedAt = now
		isPinned := 0
		if l.IsPinned {
			isPinned = 1
		}

		if _, err := stmt.ExecContext(ctx,
			l.ContactID, l.PosX, l.PosY, isPinned,
			l.CommunityID, l.BetweennessScore, nowStr,
		); err != nil {
			return fmt.Errorf("ошибка пакетной записи координат узла %d: %w", l.ContactID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ошибка фиксации транзакции лейаута: %w", err)
	}
	return nil
}

// GetLayout возвращает сохранённые координаты узла по ContactID.
func (r *GraphRepo) GetLayout(ctx context.Context, contactID int64) (*graph.GraphLayout, error) {
	query := `
		SELECT contact_id, pos_x, pos_y, is_pinned, community_id, betweenness_score, updated_at
		FROM graph_layouts
		WHERE contact_id = ?
	`
	row := r.db.QueryRowContext(ctx, query, contactID)

	var l graph.GraphLayout
	var isPinned int
	var updatedStr string

	err := row.Scan(
		&l.ContactID, &l.PosX, &l.PosY, &isPinned,
		&l.CommunityID, &l.BetweennessScore, &updatedStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, graph.ErrNodeNotFound
		}
		return nil, fmt.Errorf("ошибка выборки размещения узла %d: %w", contactID, err)
	}

	l.IsPinned = isPinned == 1
	if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
		l.UpdatedAt = t
	}
	return &l, nil
}

// GetAllLayouts возвращает карту размещений всех узлов (ContactID -> GraphLayout).
func (r *GraphRepo) GetAllLayouts(ctx context.Context) (map[int64]*graph.GraphLayout, error) {
	query := `
		SELECT contact_id, pos_x, pos_y, is_pinned, community_id, betweenness_score, updated_at
		FROM graph_layouts
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки всех размещений: %w", err)
	}
	defer rows.Close()

	result := make(map[int64]*graph.GraphLayout)
	for rows.Next() {
		var l graph.GraphLayout
		var isPinned int
		var updatedStr string

		if err := rows.Scan(
			&l.ContactID, &l.PosX, &l.PosY, &isPinned,
			&l.CommunityID, &l.BetweennessScore, &updatedStr,
		); err != nil {
			return nil, err
		}

		l.IsPinned = isPinned == 1
		if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
			l.UpdatedAt = t
		}
		result[l.ContactID] = &l
	}
	return result, rows.Err()
}

// QueryEgoSubgraph выполняет рекурсивный CTE запрос подграфа вокруг выбранного контакта глубиной maxHops.
func (r *GraphRepo) QueryEgoSubgraph(ctx context.Context, egoID int64, maxHops int) ([]*graph.EgoNode, error) {
	if maxHops <= 0 {
		maxHops = 1
	}

	query := `
	WITH RECURSIVE ego_subgraph(contact_id, hop_depth, visited_path, cumulative_trust) AS (
		SELECT 
			CAST(? AS INTEGER) AS contact_id,
			0 AS hop_depth,
			',' || CAST(? AS TEXT) || ',' AS visited_path,
			1.0 AS cumulative_trust

		UNION ALL

		SELECT
			CASE 
				WHEN r.source_id = eg.contact_id THEN r.target_id 
				ELSE r.source_id 
			END AS next_contact_id,
			eg.hop_depth + 1 AS hop_depth,
			eg.visited_path || CAST(
				CASE WHEN r.source_id = eg.contact_id THEN r.target_id ELSE r.source_id END 
				AS TEXT
			) || ',',
			eg.cumulative_trust * (r.weight / 5.0)
		FROM ego_subgraph eg
		JOIN contact_relationships r
		  ON (r.source_id = eg.contact_id OR (r.target_id = eg.contact_id AND r.is_directed = 0))
		WHERE eg.hop_depth < ?
		  AND instr(
				eg.visited_path,
				',' || CAST(CASE WHEN r.source_id = eg.contact_id THEN r.target_id ELSE r.source_id END AS TEXT) || ','
			  ) = 0
	)
	SELECT 
		c.id,
		c.first_name,
		c.last_name,
		c.company,
		MIN(eg.hop_depth) AS min_hop,
		MAX(eg.cumulative_trust) AS max_trust_score
	FROM ego_subgraph eg
	JOIN contacts c ON c.id = eg.contact_id
	GROUP BY c.id
	ORDER BY min_hop ASC, max_trust_score DESC;
	`

	rows, err := r.db.QueryContext(ctx, query, egoID, egoID, maxHops)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения ego_subgraph CTE: %w", err)
	}
	defer rows.Close()

	var nodes []*graph.EgoNode
	for rows.Next() {
		var n graph.EgoNode
		if err := rows.Scan(&n.ContactID, &n.FirstName, &n.LastName, &n.Company, &n.MinHop, &n.MaxTrustScore); err != nil {
			return nil, err
		}
		nodes = append(nodes, &n)
	}
	return nodes, rows.Err()
}

// GetImplicitRelationships возвращает неявные рёбра между контактами, работающими в одной организации.
func (r *GraphRepo) GetImplicitRelationships(ctx context.Context) ([]*graph.Relationship, error) {
	query := `
		SELECT 
			c1.id AS source_id,
			c2.id AS target_id,
			'implicit_colleague' AS rel_type,
			2 AS weight,
			o.name AS context_note
		FROM contacts c1
		JOIN contacts c2 ON c1.org_id = c2.org_id AND c1.id < c2.id
		JOIN organizations o ON o.id = c1.org_id
		WHERE c1.org_id IS NOT NULL
		ORDER BY c1.id ASC, c2.id ASC;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки неявных связей: %w", err)
	}
	defer rows.Close()

	now := time.Now().UTC()
	var rels []*graph.Relationship
	for rows.Next() {
		var rel graph.Relationship
		var relTypeStr string
		if err := rows.Scan(&rel.SourceID, &rel.TargetID, &relTypeStr, &rel.Weight, &rel.ContextNote); err != nil {
			return nil, err
		}
		rel.Type = graph.RelationshipType(relTypeStr)
		rel.IsDirected = false
		rel.CreatedAt = now
		rel.UpdatedAt = now
		rels = append(rels, &rel)
	}
	return rels, rows.Err()
}

func scanRelationshipRow(s rowScanner) (*graph.Relationship, error) {
	var r graph.Relationship
	var isDirInt int
	var relTypeStr, createdStr, updatedStr string

	err := s.Scan(
		&r.ID, &r.SourceID, &r.TargetID, &relTypeStr, &r.Weight, &isDirInt,
		&r.ContextNote, &r.SinceDate, &createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}

	r.Type = graph.RelationshipType(relTypeStr)
	r.IsDirected = isDirInt == 1

	if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
		r.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
		r.UpdatedAt = t
	}
	return &r, nil
}

// -----------------------------------------------------------------------------
// Репозиторий организаций (OrganizationRepository)
// -----------------------------------------------------------------------------

// CreateOrganization создаёт запись организации.
func (r *OrgRepo) CreateOrganization(ctx context.Context, org *graph.Organization) (int64, error) {
	if err := org.Validate(); err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	if org.CreatedAt.IsZero() {
		org.CreatedAt = now
	}
	createdStr := org.CreatedAt.Format(time.RFC3339)

	query := `
		INSERT INTO organizations (name, industry, domain, city, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	res, err := r.db.ExecContext(ctx, query,
		strings.TrimSpace(org.Name), org.Industry, org.Domain, org.City, createdStr,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return 0, graph.ErrDuplicateOrg
		}
		return 0, fmt.Errorf("ошибка создания организации: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	org.ID = id
	return id, nil
}

// GetOrganization возвращает организацию по ID.
func (r *OrgRepo) GetOrganization(ctx context.Context, id int64) (*graph.Organization, error) {
	query := `
		SELECT id, name, industry, domain, city, created_at
		FROM organizations
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	org, err := scanOrgRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, graph.ErrOrgNotFound
		}
		return nil, fmt.Errorf("ошибка выборки организации ID=%d: %w", id, err)
	}
	return org, nil
}

// GetOrganizationByName возвращает организацию по названию (регистронезависимо).
func (r *OrgRepo) GetOrganizationByName(ctx context.Context, name string) (*graph.Organization, error) {
	query := `
		SELECT id, name, industry, domain, city, created_at
		FROM organizations
		WHERE name = ? COLLATE NOCASE
	`
	row := r.db.QueryRowContext(ctx, query, strings.TrimSpace(name))
	org, err := scanOrgRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, graph.ErrOrgNotFound
		}
		return nil, fmt.Errorf("ошибка выборки организации по имени: %w", err)
	}
	return org, nil
}

// ListOrganizations возвращает полный список организаций, отсортированный по алфавиту.
func (r *OrgRepo) ListOrganizations(ctx context.Context) ([]*graph.Organization, error) {
	query := `
		SELECT id, name, industry, domain, city, created_at
		FROM organizations
		ORDER BY name ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("ошибка списка организаций: %w", err)
	}
	defer rows.Close()

	var orgs []*graph.Organization
	for rows.Next() {
		org, err := scanOrgRow(rows)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}
	return orgs, rows.Err()
}

// UpdateOrganization обновляет поля существующей организации.
func (r *OrgRepo) UpdateOrganization(ctx context.Context, org *graph.Organization) error {
	if org == nil || org.ID <= 0 {
		return graph.ErrOrgNotFound
	}
	if err := org.Validate(); err != nil {
		return err
	}

	query := `
		UPDATE organizations SET
			name = ?, industry = ?, domain = ?, city = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		strings.TrimSpace(org.Name), org.Industry, org.Domain, org.City, org.ID,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return graph.ErrDuplicateOrg
		}
		return fmt.Errorf("ошибка обновления организации ID=%d: %w", org.ID, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return graph.ErrOrgNotFound
	}
	return nil
}

// DeleteOrganization удаляет организацию по идентификатору.
func (r *OrgRepo) DeleteOrganization(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM organizations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления организации ID=%d: %w", id, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return graph.ErrOrgNotFound
	}
	return nil
}

// GraphRepo также делегирует методы OrganizationRepository для универсального использования:
func (r *GraphRepo) CreateOrganization(ctx context.Context, org *graph.Organization) (int64, error) {
	return (&OrgRepo{db: r.db}).CreateOrganization(ctx, org)
}

func (r *GraphRepo) GetOrganization(ctx context.Context, id int64) (*graph.Organization, error) {
	return (&OrgRepo{db: r.db}).GetOrganization(ctx, id)
}

func (r *GraphRepo) GetOrganizationByName(ctx context.Context, name string) (*graph.Organization, error) {
	return (&OrgRepo{db: r.db}).GetOrganizationByName(ctx, name)
}

func (r *GraphRepo) ListOrganizations(ctx context.Context) ([]*graph.Organization, error) {
	return (&OrgRepo{db: r.db}).ListOrganizations(ctx)
}

func (r *GraphRepo) UpdateOrganization(ctx context.Context, org *graph.Organization) error {
	return (&OrgRepo{db: r.db}).UpdateOrganization(ctx, org)
}

func (r *GraphRepo) DeleteOrganization(ctx context.Context, id int64) error {
	return (&OrgRepo{db: r.db}).DeleteOrganization(ctx, id)
}

func scanOrgRow(s rowScanner) (*graph.Organization, error) {
	var o graph.Organization
	var createdStr string
	err := s.Scan(&o.ID, &o.Name, &o.Industry, &o.Domain, &o.City, &createdStr)
	if err != nil {
		return nil, err
	}
	if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
		o.CreatedAt = t
	}
	return &o, nil
}
