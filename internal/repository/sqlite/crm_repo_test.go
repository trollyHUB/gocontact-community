// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/domain/crm"
	"github.com/trollyHUB/gocontact-studio/internal/repository"
	"github.com/trollyHUB/gocontact-studio/internal/repository/sqlite"
)

func setupTestCRMRepo(t *testing.T) (repository.ContactRepository, repository.GroupRepository, repository.GraphRepository, repository.CRMRepository) {
	t.Helper()
	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("не удалось инициализировать тестовую БД: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	contactRepo := sqlite.NewContactRepo(db)
	groupRepo := sqlite.NewGroupRepo(db)
	graphRepo := sqlite.NewGraphRepo(db)
	crmRepo := sqlite.NewSQLiteCRMRepo(db)

	return contactRepo, groupRepo, graphRepo, crmRepo
}

func TestCRMRepo_InteractionsCRUD_AndParticipantsMN(t *testing.T) {
	ctx := context.Background()
	contactRepo, _, _, crmRepo := setupTestCRMRepo(t)

	// Создаём 2 контакта
	c1 := &domain.Contact{FirstName: "Ерлан", LastName: "Тулегенов", Phone: "+77011112233"}
	c2 := &domain.Contact{FirstName: "Диана", LastName: "Рахимова", Phone: "+77012223344"}
	if err := contactRepo.Create(ctx, c1); err != nil {
		t.Fatalf("ошибка создания контакта 1: %v", err)
	}
	if err := contactRepo.Create(ctx, c2); err != nil {
		t.Fatalf("ошибка создания контакта 2: %v", err)
	}

	// 1. Создаём взаимодействие (Встреча с обоими участниками M:N)
	touch := &crm.Interaction{
		Channel:        crm.ChannelMeeting,
		Title:          "Обед в Del Papa",
		NotesMD:        "Обсудили программу акселерации",
		Location:       "Алматы, ул. Достык",
		OccurredAt:     time.Date(2026, 10, 20, 13, 0, 0, 0, time.UTC),
		DurationMin:    45,
		Sentiment:      1,
		ParticipantIDs: []int64{c1.ID, c2.ID},
	}

	id, err := crmRepo.CreateInteraction(ctx, touch)
	if err != nil {
		t.Fatalf("ошибка создания взаимодействия: %v", err)
	}
	if id <= 0 {
		t.Fatalf("некорректный ID взаимодействия: %d", id)
	}

	// 2. Получаем взаимодействие и проверяем участников
	fetched, err := crmRepo.GetInteraction(ctx, id)
	if err != nil {
		t.Fatalf("ошибка получения взаимодействия: %v", err)
	}
	if fetched.Title != "Обед в Del Papa" {
		t.Fatalf("ожидался заголовок 'Обед в Del Papa', получено '%s'", fetched.Title)
	}
	if len(fetched.ParticipantIDs) != 2 {
		t.Fatalf("ожидалось 2 участника, получено %d", len(fetched.ParticipantIDs))
	}

	// 3. Выборка таймлайна по каждому контакту
	timeline1, err := crmRepo.ListInteractionsByContact(ctx, c1.ID, 10)
	if err != nil || len(timeline1) != 1 {
		t.Fatalf("ожидалось 1 касание для контакта 1, получено: %d, err: %v", len(timeline1), err)
	}

	timeline2, err := crmRepo.ListInteractionsByContact(ctx, c2.ID, 10)
	if err != nil || len(timeline2) != 1 {
		t.Fatalf("ожидалось 1 касание для контакта 2, получено: %d, err: %v", len(timeline2), err)
	}

	// 4. Удаление взаимодействия
	if err := crmRepo.DeleteInteraction(ctx, id); err != nil {
		t.Fatalf("ошибка удаления взаимодействия: %v", err)
	}
	if _, err := crmRepo.GetInteraction(ctx, id); err != crm.ErrInteractionNotFound {
		t.Fatalf("ожидалась ошибка ErrInteractionNotFound, получено: %v", err)
	}
}

func TestCRMRepo_PromisesCRUD(t *testing.T) {
	ctx := context.Background()
	contactRepo, _, _, crmRepo := setupTestCRMRepo(t)

	c1 := &domain.Contact{FirstName: "Даурен", LastName: "Бектуров", Phone: "+77019998877"}
	if err := contactRepo.Create(ctx, c1); err != nil {
		t.Fatalf("ошибка создания контакта: %v", err)
	}

	promise := &crm.ActionPromise{
		ContactID:   c1.ID,
		Direction:   "i_promised",
		Description: "Познакомить с юристом МФЦА",
		DueAt:       time.Now().Add(48 * time.Hour),
		Status:      "pending",
	}

	pID, err := crmRepo.CreatePromise(ctx, promise)
	if err != nil || pID <= 0 {
		t.Fatalf("ошибка сохранения обещания: %v", err)
	}

	// Проверяем список pending
	pending, err := crmRepo.ListPendingPromises(ctx)
	if err != nil || len(pending) != 1 {
		t.Fatalf("ожидалось 1 активное обещание, получено: %d, err: %v", len(pending), err)
	}

	// Завершаем обещание
	if err := crmRepo.UpdatePromiseStatus(ctx, pID, "completed"); err != nil {
		t.Fatalf("ошибка обновления статуса обещания: %v", err)
	}

	pendingAfter, err := crmRepo.ListPendingPromises(ctx)
	if err != nil || len(pendingAfter) != 0 {
		t.Fatalf("ожидалось 0 активных обещаний после завершения, получено: %d", len(pendingAfter))
	}

	byContact, err := crmRepo.ListPromisesByContact(ctx, c1.ID)
	if err != nil || len(byContact) != 1 || byContact[0].Status != "completed" {
		t.Fatalf("ожидалось 1 завершённое обещание у контакта, получено: %v", byContact)
	}
}

func TestCRMRepo_LifeEventsCRUD(t *testing.T) {
	ctx := context.Background()
	contactRepo, _, _, crmRepo := setupTestCRMRepo(t)

	c1 := &domain.Contact{FirstName: "Диана", LastName: "Рахимова", Phone: "+77014445566"}
	if err := contactRepo.Create(ctx, c1); err != nil {
		t.Fatalf("ошибка создания контакта: %v", err)
	}

	ev := &crm.LifeEvent{
		ContactID:        c1.ID,
		EventType:        "birthday",
		Title:            "День рождения (32 года)",
		EventDate:        "1994-10-28",
		IsRecurring:      true,
		RemindDaysBefore: 7,
		Notes:            "Увлекается марафонами",
	}

	evID, err := crmRepo.CreateLifeEvent(ctx, ev)
	if err != nil || evID <= 0 {
		t.Fatalf("ошибка создания вехи жизни: %v", err)
	}

	events, err := crmRepo.ListLifeEventsByContact(ctx, c1.ID)
	if err != nil || len(events) != 1 {
		t.Fatalf("ожидалась 1 веха жизни, получено: %d", len(events))
	}
	if !events[0].IsRecurring {
		t.Fatalf("ожидался IsRecurring = true")
	}

	if err := crmRepo.DeleteLifeEvent(ctx, evID); err != nil {
		t.Fatalf("ошибка удаления вехи жизни: %v", err)
	}
}

func TestCRMRepo_CadencePolicyUpsertAndList(t *testing.T) {
	ctx := context.Background()
	contactRepo, _, _, crmRepo := setupTestCRMRepo(t)

	c1 := &domain.Contact{FirstName: "Аскар", LastName: "Мамырбаев", Phone: "+77017778899"}
	if err := contactRepo.Create(ctx, c1); err != nil {
		t.Fatalf("ошибка создания контакта: %v", err)
	}

	// Дефолтная политика для контакта без записей
	defPolicy, err := crmRepo.GetCadencePolicy(ctx, c1.ID)
	if err != nil || defPolicy.HealthScore != 50 {
		t.Fatalf("ожидался дефолтный скор 50, получено: %v, err: %v", defPolicy, err)
	}

	// Upsert обновлённого скоринга
	now := time.Now().UTC()
	updatedSnap := &crm.HealthSnapshot{
		ContactID:          c1.ID,
		TargetIntervalDays: 14,
		HealthScore:        85,
		Status:             crm.StatusThriving,
		LastInteractionAt:  now,
		RawEnergy:          0.85,
	}
	if err := crmRepo.UpsertCadencePolicy(ctx, updatedSnap); err != nil {
		t.Fatalf("ошибка upsert cadence: %v", err)
	}

	fetched, err := crmRepo.GetCadencePolicy(ctx, c1.ID)
	if err != nil {
		t.Fatalf("ошибка получения cadence: %v", err)
	}
	if fetched.HealthScore != 85 || fetched.Status != crm.StatusThriving {
		t.Fatalf("некорректный скор: %d, статус: %s", fetched.HealthScore, fetched.Status)
	}

	list, err := crmRepo.ListCadencePolicies(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("ожидалась 1 политика в списке, получено: %d", len(list))
	}
}
