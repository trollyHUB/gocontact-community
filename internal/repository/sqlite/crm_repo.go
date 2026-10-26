// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain/crm"
	"github.com/trollyHUB/gocontact-studio/internal/repository"
)

type sqliteCRMRepo struct {
	db *sql.DB
}

// NewSQLiteCRMRepo создаёт новый экземпляр репозитория сущностей Personal CRM.
func NewSQLiteCRMRepo(db *sql.DB) repository.CRMRepository {
	return &sqliteCRMRepo{db: db}
}

// NewCRMRepo — удобный фабричный конструктор, соответствующий сигнатуре остальных репозиториев SQLite.
func NewCRMRepo(db *sql.DB) repository.CRMRepository {
	return NewSQLiteCRMRepo(db)
}

// CreateInteraction создаёт запись взаимодействия и привязывает участников (M:N).
func (r *sqliteCRMRepo) CreateInteraction(ctx context.Context, inter *crm.Interaction) (int64, error) {
	if err := inter.Validate(); err != nil {
		return 0, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("ошибка открытия транзакции: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	occurredUnix := inter.OccurredAt.Unix()
	res, err := tx.ExecContext(ctx, `
		INSERT INTO interactions (channel, title, notes_md, location, occurred_at_unix, duration_min, sentiment)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, string(inter.Channel), inter.Title, inter.NotesMD, inter.Location, occurredUnix, inter.DurationMin, inter.Sentiment)
	if err != nil {
		return 0, fmt.Errorf("ошибка сохранения взаимодействия: %w", err)
	}

	interactionID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	for _, pid := range inter.ParticipantIDs {
		if pid <= 0 {
			continue
		}
		_, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO interaction_participants (interaction_id, contact_id, role)
			VALUES (?, ?, 'participant')
		`, interactionID, pid)
		if err != nil {
			return 0, fmt.Errorf("ошибка привязки участника %d: %w", pid, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("ошибка фиксации транзакции: %w", err)
	}

	inter.ID = interactionID
	return interactionID, nil
}

// GetInteraction возвращает взаимодействие по ID вместе со списком участников.
func (r *sqliteCRMRepo) GetInteraction(ctx context.Context, id int64) (*crm.Interaction, error) {
	var inter crm.Interaction
	var channelStr string
	var occurredUnix int64
	var createdAtStr, updatedAtStr string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, channel, title, notes_md, location, occurred_at_unix, duration_min, sentiment, created_at, updated_at
		FROM interactions
		WHERE id = ?
	`, id).Scan(
		&inter.ID, &channelStr, &inter.Title, &inter.NotesMD, &inter.Location,
		&occurredUnix, &inter.DurationMin, &inter.Sentiment, &createdAtStr, &updatedAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, crm.ErrInteractionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки взаимодействия: %w", err)
	}

	inter.Channel = crm.ChannelType(channelStr)
	inter.OccurredAt = time.Unix(occurredUnix, 0).UTC()

	// Выборка участников
	rows, err := r.db.QueryContext(ctx, `
		SELECT contact_id FROM interaction_participants WHERE interaction_id = ? ORDER BY contact_id ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки участников: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pid int64
		if err := rows.Scan(&pid); err == nil {
			inter.ParticipantIDs = append(inter.ParticipantIDs, pid)
		}
	}

	return &inter, nil
}

// ListInteractionsByContact возвращает хронологию касаний контакта в обратном порядке (новейшие первыми).
func (r *sqliteCRMRepo) ListInteractionsByContact(ctx context.Context, contactID int64, limit int) ([]*crm.Interaction, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT i.id, i.channel, i.title, i.notes_md, i.location, i.occurred_at_unix, i.duration_min, i.sentiment, i.created_at, i.updated_at
		FROM interactions i
		INNER JOIN interaction_participants ip ON i.id = ip.interaction_id
		WHERE ip.contact_id = ?
		ORDER BY i.occurred_at_unix DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, contactID, limit)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки таймлайна контакта: %w", err)
	}
	defer rows.Close()

	var result []*crm.Interaction
	for rows.Next() {
		var inter crm.Interaction
		var channelStr string
		var occurredUnix int64
		var createdAtStr, updatedAtStr string

		if err := rows.Scan(
			&inter.ID, &channelStr, &inter.Title, &inter.NotesMD, &inter.Location,
			&occurredUnix, &inter.DurationMin, &inter.Sentiment, &createdAtStr, &updatedAtStr,
		); err != nil {
			return nil, err
		}
		inter.Channel = crm.ChannelType(channelStr)
		inter.OccurredAt = time.Unix(occurredUnix, 0).UTC()
		result = append(result, &inter)
	}

	// Заполняем участников для каждого взаимодействия
	for _, inter := range result {
		pRows, err := r.db.QueryContext(ctx, `SELECT contact_id FROM interaction_participants WHERE interaction_id = ?`, inter.ID)
		if err == nil {
			for pRows.Next() {
				var pid int64
				if err := pRows.Scan(&pid); err == nil {
					inter.ParticipantIDs = append(inter.ParticipantIDs, pid)
				}
			}
			pRows.Close()
		}
	}

	return result, nil
}

// DeleteInteraction удаляет взаимодействие (участники удаляются каскадно).
func (r *sqliteCRMRepo) DeleteInteraction(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM interactions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления взаимодействия: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return crm.ErrInteractionNotFound
	}
	return nil
}

// CreatePromise сохраняет новое взаимное обещание.
func (r *sqliteCRMRepo) CreatePromise(ctx context.Context, promise *crm.ActionPromise) (int64, error) {
	if err := promise.Validate(); err != nil {
		return 0, err
	}

	var sourceID *int64
	if promise.SourceInteractionID > 0 {
		sourceID = &promise.SourceInteractionID
	}

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO action_promises (contact_id, source_interaction_id, direction, description, due_at_unix, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`, promise.ContactID, sourceID, promise.Direction, promise.Description, promise.DueAt.Unix(), promise.Status)
	if err != nil {
		return 0, fmt.Errorf("ошибка сохранения обещания: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	promise.ID = id
	return id, nil
}

// UpdatePromiseStatus обновляет статус обещания (completed/cancelled/pending).
func (r *sqliteCRMRepo) UpdatePromiseStatus(ctx context.Context, id int64, status string) error {
	completedAt := ""
	if status == "completed" {
		completedAt = time.Now().UTC().Format(time.RFC3339)
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE action_promises SET status = ?, completed_at = ? WHERE id = ?
	`, status, completedAt, id)
	if err != nil {
		return fmt.Errorf("ошибка обновления статуса обещания: %w", err)
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return crm.ErrPromiseNotFound
	}
	return nil
}

// ListPendingPromises возвращает все незавершённые обещания, отсортированные по срочности.
func (r *sqliteCRMRepo) ListPendingPromises(ctx context.Context) ([]*crm.ActionPromise, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, contact_id, COALESCE(source_interaction_id, 0), direction, description, due_at_unix, status, completed_at, created_at
		FROM action_promises
		WHERE status = 'pending'
		ORDER BY due_at_unix ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки активных обещаний: %w", err)
	}
	defer rows.Close()

	var result []*crm.ActionPromise
	for rows.Next() {
		var p crm.ActionPromise
		var dueUnix int64
		var createdStr string

		if err := rows.Scan(
			&p.ID, &p.ContactID, &p.SourceInteractionID, &p.Direction,
			&p.Description, &dueUnix, &p.Status, &p.CompletedAt, &createdStr,
		); err != nil {
			return nil, err
		}
		p.DueAt = time.Unix(dueUnix, 0).UTC()
		result = append(result, &p)
	}
	return result, nil
}

// ListPromisesByContact возвращает обещания для указанного контакта.
func (r *sqliteCRMRepo) ListPromisesByContact(ctx context.Context, contactID int64) ([]*crm.ActionPromise, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, contact_id, COALESCE(source_interaction_id, 0), direction, description, due_at_unix, status, completed_at, created_at
		FROM action_promises
		WHERE contact_id = ?
		ORDER BY status DESC, due_at_unix ASC
	`, contactID)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки обещаний контакта: %w", err)
	}
	defer rows.Close()

	var result []*crm.ActionPromise
	for rows.Next() {
		var p crm.ActionPromise
		var dueUnix int64
		var createdStr string

		if err := rows.Scan(
			&p.ID, &p.ContactID, &p.SourceInteractionID, &p.Direction,
			&p.Description, &dueUnix, &p.Status, &p.CompletedAt, &createdStr,
		); err != nil {
			return nil, err
		}
		p.DueAt = time.Unix(dueUnix, 0).UTC()
		result = append(result, &p)
	}
	return result, nil
}

// CreateLifeEvent сохраняет памятную дату или веху жизни.
func (r *sqliteCRMRepo) CreateLifeEvent(ctx context.Context, event *crm.LifeEvent) (int64, error) {
	isRec := 0
	if event.IsRecurring {
		isRec = 1
	}
	remindDays := event.RemindDaysBefore
	if remindDays < 0 {
		remindDays = 7
	}

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO life_events (contact_id, event_type, title, event_date, is_recurring, remind_days_before, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, event.ContactID, event.EventType, event.Title, event.EventDate, isRec, remindDays, event.Notes)
	if err != nil {
		return 0, fmt.Errorf("ошибка сохранения вехи жизни: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	event.ID = id
	return id, nil
}

// ListLifeEventsByContact возвращает список вех жизни для контакта.
func (r *sqliteCRMRepo) ListLifeEventsByContact(ctx context.Context, contactID int64) ([]*crm.LifeEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, contact_id, event_type, title, event_date, is_recurring, remind_days_before, notes, created_at
		FROM life_events
		WHERE contact_id = ?
		ORDER BY event_date ASC
	`, contactID)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки вех жизни: %w", err)
	}
	defer rows.Close()

	var result []*crm.LifeEvent
	for rows.Next() {
		var ev crm.LifeEvent
		var isRec int
		var createdStr string

		if err := rows.Scan(
			&ev.ID, &ev.ContactID, &ev.EventType, &ev.Title, &ev.EventDate,
			&isRec, &ev.RemindDaysBefore, &ev.Notes, &createdStr,
		); err != nil {
			return nil, err
		}
		ev.IsRecurring = isRec == 1
		result = append(result, &ev)
	}
	return result, nil
}

// DeleteLifeEvent удаляет веху жизни.
func (r *sqliteCRMRepo) DeleteLifeEvent(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM life_events WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления вехи жизни: %w", err)
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return crm.ErrLifeEventNotFound
	}
	return nil
}

// GetCadencePolicy возвращает политику частоты касаний и материализованный статус здоровья контакта.
func (r *sqliteCRMRepo) GetCadencePolicy(ctx context.Context, contactID int64) (*crm.HealthSnapshot, error) {
	var snap crm.HealthSnapshot
	var statusStr string
	var lastTouchUnix, snoozeUnix int64
	var updatedStr string

	err := r.db.QueryRowContext(ctx, `
		SELECT contact_id, target_interval_days, health_score, health_status, last_interaction_unix, snoozed_until_unix, raw_energy, updated_at
		FROM cadence_policies
		WHERE contact_id = ?
	`, contactID).Scan(
		&snap.ContactID, &snap.TargetIntervalDays, &snap.HealthScore, &statusStr,
		&lastTouchUnix, &snoozeUnix, &snap.RawEnergy, &updatedStr,
	)
	if err == sql.ErrNoRows {
		// Дефолтная политика для контакта
		return &crm.HealthSnapshot{
			ContactID:          contactID,
			DunbarTier:         crm.DunbarAffinityGroup,
			TargetIntervalDays: 0,
			HealthScore:        50,
			Status:             crm.StatusWarm,
			RawEnergy:          0.5,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки политики cadence: %w", err)
	}

	snap.Status = crm.HealthStatus(statusStr)
	if lastTouchUnix > 0 {
		snap.LastInteractionAt = time.Unix(lastTouchUnix, 0).UTC()
	}
	if snoozeUnix > 0 {
		snap.SnoozedUntil = time.Unix(snoozeUnix, 0).UTC()
	}
	return &snap, nil
}

// UpsertCadencePolicy обновляет или вставляет вычисленные метрики здоровья контакта.
func (r *sqliteCRMRepo) UpsertCadencePolicy(ctx context.Context, snap *crm.HealthSnapshot) error {
	lastTouchUnix := int64(0)
	if !snap.LastInteractionAt.IsZero() {
		lastTouchUnix = snap.LastInteractionAt.Unix()
	}
	snoozeUnix := int64(0)
	if !snap.SnoozedUntil.IsZero() {
		snoozeUnix = snap.SnoozedUntil.Unix()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO cadence_policies (
			contact_id, target_interval_days, health_score, health_status,
			last_interaction_unix, snoozed_until_unix, raw_energy, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		ON CONFLICT(contact_id) DO UPDATE SET
			target_interval_days  = excluded.target_interval_days,
			health_score          = excluded.health_score,
			health_status         = excluded.health_status,
			last_interaction_unix = excluded.last_interaction_unix,
			snoozed_until_unix    = excluded.snoozed_until_unix,
			raw_energy            = excluded.raw_energy,
			updated_at            = excluded.updated_at
	`, snap.ContactID, snap.TargetIntervalDays, snap.HealthScore, string(snap.Status), lastTouchUnix, snoozeUnix, snap.RawEnergy)
	if err != nil {
		return fmt.Errorf("ошибка upsert cadence: %w", err)
	}
	return nil
}

// ListCadencePolicies возвращает все сохранённые политики здоровья.
func (r *sqliteCRMRepo) ListCadencePolicies(ctx context.Context) ([]*crm.HealthSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT contact_id, target_interval_days, health_score, health_status, last_interaction_unix, snoozed_until_unix, raw_energy, updated_at
		FROM cadence_policies
		ORDER BY health_score ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ошибка выборки политик cadence: %w", err)
	}
	defer rows.Close()

	var result []*crm.HealthSnapshot
	for rows.Next() {
		var snap crm.HealthSnapshot
		var statusStr string
		var lastTouchUnix, snoozeUnix int64
		var updatedStr string

		if err := rows.Scan(
			&snap.ContactID, &snap.TargetIntervalDays, &snap.HealthScore, &statusStr,
			&lastTouchUnix, &snoozeUnix, &snap.RawEnergy, &updatedStr,
		); err != nil {
			return nil, err
		}
		snap.Status = crm.HealthStatus(statusStr)
		if lastTouchUnix > 0 {
			snap.LastInteractionAt = time.Unix(lastTouchUnix, 0).UTC()
		}
		if snoozeUnix > 0 {
			snap.SnoozedUntil = time.Unix(snoozeUnix, 0).UTC()
		}
		result = append(result, &snap)
	}
	return result, nil
}
