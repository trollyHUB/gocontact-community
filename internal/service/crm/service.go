// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package crm

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/domain/crm"
	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/repository"
)

// CRMService определяет контракт сервисного слоя Personal CRM, скоринга отношений и генератора досье.
type CRMService interface {
	LogInteraction(ctx context.Context, inter *crm.Interaction, promise *crm.ActionPromise) (*crm.Interaction, error)
	GetContactTimeline(ctx context.Context, contactID int64, limit int) ([]*crm.Interaction, error)
	DeleteInteraction(ctx context.Context, id int64) error

	CalculateContactHealth(ctx context.Context, contactID int64) (*crm.HealthSnapshot, error)
	RecalculateAllHealthScores(ctx context.Context) ([]*crm.HealthSnapshot, error)
	GetRadarCandidates(ctx context.Context, limit int) ([]*crm.RadarCandidate, error)
	SnoozeContact(ctx context.Context, contactID int64, duration time.Duration) error

	GeneratePreMeetingDossier(ctx context.Context, contactID int64) (*crm.PreMeetingDossier, error)

	CreateActionPromise(ctx context.Context, promise *crm.ActionPromise) (int64, error)
	UpdatePromiseStatus(ctx context.Context, id int64, status string) error
	ListPendingPromises(ctx context.Context) ([]*crm.ActionPromise, error)
	ListPromisesByContact(ctx context.Context, contactID int64) ([]*crm.ActionPromise, error)

	CreateLifeEvent(ctx context.Context, event *crm.LifeEvent) (int64, error)
	ListLifeEvents(ctx context.Context, contactID int64) ([]*crm.LifeEvent, error)
	DeleteLifeEvent(ctx context.Context, id int64) error
}

type crmService struct {
	mu          sync.RWMutex
	crmRepo     repository.CRMRepository
	contactRepo repository.ContactRepository
	graphRepo   repository.GraphRepository
}

// NewCRMService создаёт новый экземпляр сервисного слоя Personal CRM.
func NewCRMService(
	crmRepo repository.CRMRepository,
	contactRepo repository.ContactRepository,
	graphRepo repository.GraphRepository,
) CRMService {
	return &crmService{
		crmRepo:     crmRepo,
		contactRepo: contactRepo,
		graphRepo:   graphRepo,
	}
}

// LogInteraction логирует касание, опционально создаёт обещание и обновляет здоровье всех участников.
func (s *crmService) LogInteraction(ctx context.Context, inter *crm.Interaction, promise *crm.ActionPromise) (*crm.Interaction, error) {
	if err := inter.Validate(); err != nil {
		return nil, err
	}

	id, err := s.crmRepo.CreateInteraction(ctx, inter)
	if err != nil {
		return nil, fmt.Errorf("ошибка сохранения взаимодействия: %w", err)
	}
	inter.ID = id

	if promise != nil && strings.TrimSpace(promise.Description) != "" {
		promise.SourceInteractionID = id
		if promise.ContactID <= 0 && len(inter.ParticipantIDs) > 0 {
			promise.ContactID = inter.ParticipantIDs[0]
		}
		_, _ = s.crmRepo.CreatePromise(ctx, promise)
	}

	// Обновляем здоровье всех участников
	for _, pid := range inter.ParticipantIDs {
		_ = s.updateHealthAfterInteraction(ctx, pid, inter)
	}

	return inter, nil
}

func (s *crmService) updateHealthAfterInteraction(ctx context.Context, contactID int64, inter *crm.Interaction) error {
	snap, err := s.crmRepo.GetCadencePolicy(ctx, contactID)
	if err != nil {
		return err
	}

	// Добавляем энергию от канала
	newEnergy := ApplyInteractionEnergy(snap.RawEnergy, inter.Channel)
	score, status := EnergyToScore(newEnergy)

	snap.RawEnergy = newEnergy
	snap.HealthScore = score
	snap.Status = status
	snap.LastInteractionAt = inter.OccurredAt

	return s.crmRepo.UpsertCadencePolicy(ctx, snap)
}

// GetContactTimeline возвращает хронологию взаимодействий для контакта.
func (s *crmService) GetContactTimeline(ctx context.Context, contactID int64, limit int) ([]*crm.Interaction, error) {
	return s.crmRepo.ListInteractionsByContact(ctx, contactID, limit)
}

// DeleteInteraction удаляет взаимодействие.
func (s *crmService) DeleteInteraction(ctx context.Context, id int64) error {
	return s.crmRepo.DeleteInteraction(ctx, id)
}

// CalculateContactHealth пересчитывает актуальный HealthScore контакта с учётом времени с последнего касания.
func (s *crmService) CalculateContactHealth(ctx context.Context, contactID int64) (*crm.HealthSnapshot, error) {
	contact, err := s.contactRepo.GetByID(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("контакт не найден: %w", err)
	}

	snap, err := s.crmRepo.GetCadencePolicy(ctx, contactID)
	if err != nil {
		return nil, err
	}

	// Определяем круг Данбара
	tier := crm.DunbarAffinityGroup
	if contact.DunbarOverride > 0 {
		tier = crm.DunbarTier(contact.DunbarOverride)
	} else if s.graphRepo != nil {
		rels, _ := s.graphRepo.GetRelationshipsByContact(ctx, contactID)
		if len(rels) >= 5 {
			tier = crm.DunbarSympathyGroup
		}
	}
	snap.DunbarTier = tier

	// Затухание энергии
	if !snap.LastInteractionAt.IsZero() {
		daysElapsed := time.Since(snap.LastInteractionAt).Hours() / 24.0
		decayedEnergy := CalculateDecayedEnergy(snap.RawEnergy, daysElapsed, tier)
		score, status := EnergyToScore(decayedEnergy)
		snap.RawEnergy = decayedEnergy
		snap.HealthScore = score
		snap.Status = status
	}

	_ = s.crmRepo.UpsertCadencePolicy(ctx, snap)
	return snap, nil
}

// RecalculateAllHealthScores пересчитывает здоровье для всех контактов в адресной книге.
func (s *crmService) RecalculateAllHealthScores(ctx context.Context) ([]*crm.HealthSnapshot, error) {
	contacts, err := s.contactRepo.List(ctx, domain.ContactFilter{})
	if err != nil {
		return nil, err
	}

	var results []*crm.HealthSnapshot
	for _, c := range contacts {
		snap, err := s.CalculateContactHealth(ctx, c.ID)
		if err == nil {
			results = append(results, snap)
		}
	}
	return results, nil
}

// GetRadarCandidates возвращает приоритизированный список контактов для Keep-in-Touch Radar.
func (s *crmService) GetRadarCandidates(ctx context.Context, limit int) ([]*crm.RadarCandidate, error) {
	if limit <= 0 {
		limit = 10
	}

	contacts, err := s.contactRepo.List(ctx, domain.ContactFilter{})
	if err != nil {
		return nil, err
	}

	contactMap := make(map[int64]*domain.Contact, len(contacts))
	for _, c := range contacts {
		contactMap[c.ID] = c
	}

	// Индексируем обещания и вехи
	pendingPromises, _ := s.crmRepo.ListPendingPromises(ctx)
	promisesByContact := make(map[int64][]*crm.ActionPromise)
	for _, p := range pendingPromises {
		promisesByContact[p.ContactID] = append(promisesByContact[p.ContactID], p)
	}

	now := time.Now().UTC()
	var candidates []*crm.RadarCandidate

	for _, c := range contacts {
		snap, err := s.CalculateContactHealth(ctx, c.ID)
		if err != nil {
			continue
		}

		promises := promisesByContact[c.ID]
		hasPromise := len(promises) > 0

		// Проверяем приближающиеся вехи жизни (в ближайшие 7 дней)
		events, _ := s.crmRepo.ListLifeEventsByContact(ctx, c.ID)
		hasUpcomingEvent := len(events) > 0 // упрощённый маркер

		urgency := CalculateUrgencyScore(snap, hasPromise, hasUpcomingEvent, now)
		if urgency <= 0 {
			continue
		}

		daysSince := 0
		if !snap.LastInteractionAt.IsZero() {
			daysSince = int(time.Since(snap.LastInteractionAt).Hours() / 24.0)
		}

		promiseNote := ""
		if hasPromise {
			promiseNote = fmt.Sprintf("Обещание: %s", promises[0].Description)
		}

		eventNote := ""
		if len(events) > 0 {
			eventNote = fmt.Sprintf("%s (%s)", events[0].Title, events[0].EventDate)
		}

		candidates = append(candidates, &crm.RadarCandidate{
			ContactID:          c.ID,
			FullName:           c.FullName(),
			JobTitle:           c.JobTitle,
			Company:            c.Company,
			DunbarTier:         snap.DunbarTier,
			Health:             *snap,
			DaysSinceLastTouch: daysSince,
			TargetIntervalDays: int(DunbarHalfLifeDays(snap.DunbarTier)),
			PendingPromiseNote: promiseNote,
			UpcomingEventNote:  eventNote,
			UrgencyScore:       urgency,
		})
	}

	// Сортировка по убыванию UrgencyScore
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].UrgencyScore > candidates[j].UrgencyScore
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates, nil
}

// SnoozeContact откладывает контакт на указанное время (он не будет отображаться на радаре).
func (s *crmService) SnoozeContact(ctx context.Context, contactID int64, duration time.Duration) error {
	snap, err := s.crmRepo.GetCadencePolicy(ctx, contactID)
	if err != nil {
		return err
	}
	snap.SnoozedUntil = time.Now().UTC().Add(duration)
	return s.crmRepo.UpsertCadencePolicy(ctx, snap)
}

// GeneratePreMeetingDossier синтезирует 1-страничный бриф перед встречей.
func (s *crmService) GeneratePreMeetingDossier(ctx context.Context, contactID int64) (*crm.PreMeetingDossier, error) {
	contact, err := s.contactRepo.GetByID(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("контакт не найден: %w", err)
	}

	healthSnap, err := s.CalculateContactHealth(ctx, contactID)
	if err != nil {
		healthSnap = &crm.HealthSnapshot{
			ContactID:   contactID,
			HealthScore: 50,
			Status:      crm.StatusWarm,
		}
	}

	dossier := &crm.PreMeetingDossier{
		ContactID:      contact.ID,
		FullName:       contact.FullName(),
		RoleAndCompany: fmt.Sprintf("%s @ %s", contact.JobTitle, contact.Company),
		Email:          contact.Email,
		Phone:          contact.Phone,
		Health:         *healthSnap,
		KeyNotes:       contact.Notes,
		GeneratedAt:    time.Now().UTC(),
	}

	// Поиск кто познакомил и прямых связей в графе
	if s.graphRepo != nil {
		rels, err := s.graphRepo.GetRelationshipsByContact(ctx, contactID)
		if err == nil {
			for _, r := range rels {
				peerID := r.TargetID
				if peerID == contactID {
					peerID = r.SourceID
				}
				peerContact, pErr := s.contactRepo.GetByID(ctx, peerID)
				if pErr == nil {
					dossier.DirectConnections = append(dossier.DirectConnections, peerContact.FullName())
					if r.Type == graph.RelIntroducedBy && r.TargetID == contactID {
						dossier.IntroducedByName = peerContact.FullName()
						dossier.IntroducedContext = r.ContextNote
					}
				}
			}
		}
	}

	// Выборка последних 3 касаний
	touches, _ := s.crmRepo.ListInteractionsByContact(ctx, contactID, 3)
	dossier.RecentTouches = touches

	// Выборка открытых обещаний
	promises, _ := s.crmRepo.ListPromisesByContact(ctx, contactID)
	for _, p := range promises {
		if p.Status == "pending" {
			dossier.PendingPromises = append(dossier.PendingPromises, p)
		}
	}

	// Выборка вех жизни
	events, _ := s.crmRepo.ListLifeEventsByContact(ctx, contactID)
	dossier.UpcomingEvents = events

	return dossier, nil
}

// CreateActionPromise создаёт новое обещание.
func (s *crmService) CreateActionPromise(ctx context.Context, promise *crm.ActionPromise) (int64, error) {
	return s.crmRepo.CreatePromise(ctx, promise)
}

// UpdatePromiseStatus обновляет статус обещания.
func (s *crmService) UpdatePromiseStatus(ctx context.Context, id int64, status string) error {
	return s.crmRepo.UpdatePromiseStatus(ctx, id, status)
}

// ListPendingPromises возвращает все активные обещания.
func (s *crmService) ListPendingPromises(ctx context.Context) ([]*crm.ActionPromise, error) {
	return s.crmRepo.ListPendingPromises(ctx)
}

// ListPromisesByContact возвращает обещания для указанного контакта.
func (s *crmService) ListPromisesByContact(ctx context.Context, contactID int64) ([]*crm.ActionPromise, error) {
	return s.crmRepo.ListPromisesByContact(ctx, contactID)
}

// CreateLifeEvent создаёт новую веху жизни.
func (s *crmService) CreateLifeEvent(ctx context.Context, event *crm.LifeEvent) (int64, error) {
	return s.crmRepo.CreateLifeEvent(ctx, event)
}

// ListLifeEvents возвращает вехи жизни контакта.
func (s *crmService) ListLifeEvents(ctx context.Context, contactID int64) ([]*crm.LifeEvent, error) {
	return s.crmRepo.ListLifeEventsByContact(ctx, contactID)
}

// DeleteLifeEvent удаляет веху жизни.
func (s *crmService) DeleteLifeEvent(ctx context.Context, id int64) error {
	return s.crmRepo.DeleteLifeEvent(ctx, id)
}
