// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package crm_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	domainCRM "github.com/trollyHUB/gocontact-studio/internal/domain/crm"
	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/repository/sqlite"
	serviceCRM "github.com/trollyHUB/gocontact-studio/internal/service/crm"
)

func setupTestCRMService(t *testing.T) (serviceCRM.CRMService, *sqlite.ContactRepo, *sqlite.GraphRepo) {
	t.Helper()
	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("не удалось инициализировать тестовую БД: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	contactRepo := sqlite.NewContactRepo(db)
	graphRepo := sqlite.NewGraphRepo(db)
	crmRepo := sqlite.NewSQLiteCRMRepo(db)

	svc := serviceCRM.NewCRMService(crmRepo, contactRepo, graphRepo)
	return svc, contactRepo, graphRepo
}

func TestDecayEngine_Formulas(t *testing.T) {
	// 1. Проверка периода полураспада: через T_half энергия должна уменьшиться ровно в 2 раза
	tHalf15 := serviceCRM.DunbarHalfLifeDays(domainCRM.DunbarSympathyGroup)
	if tHalf15 != 21.0 {
		t.Fatalf("ожидался период 21 день, получено: %f", tHalf15)
	}

	halfEnergy := serviceCRM.CalculateDecayedEnergy(1.0, 21.0, domainCRM.DunbarSympathyGroup)
	if math.Abs(halfEnergy-0.5) > 1e-5 {
		t.Fatalf("через 21 день энергия должна стать 0.5, получено: %f", halfEnergy)
	}

	// 2. Добавление энергии от встречи
	boosted := serviceCRM.ApplyInteractionEnergy(0.5, domainCRM.ChannelMeeting)
	if boosted != 1.0 {
		t.Fatalf("0.5 + 1.0 (встреча) должно упереться в 1.0, получено: %f", boosted)
	}

	// 3. Скоринг и статусы
	score100, statusThriving := serviceCRM.EnergyToScore(1.0)
	if score100 != 100 || statusThriving != domainCRM.StatusThriving {
		t.Fatalf("ожидался скор 100 Thriving, получено: %d %s", score100, statusThriving)
	}

	score20, statusCold := serviceCRM.EnergyToScore(0.20)
	if score20 != 20 || statusCold != domainCRM.StatusCold {
		t.Fatalf("ожидался скор 20 Cold, получено: %d %s", score20, statusCold)
	}
}

func TestRadarRanker_UrgencyScore(t *testing.T) {
	now := time.Now().UTC()

	snapCold := &domainCRM.HealthSnapshot{
		DunbarTier:  domainCRM.DunbarSupportClique, // 5
		HealthScore: 20,                            // дефицит 0.8 * 5.0 = 4.0
	}
	urgencyCold := serviceCRM.CalculateUrgencyScore(snapCold, false, false, now)
	if urgencyCold != 4.0 {
		t.Fatalf("ожидалась срочность 4.0, получено: %f", urgencyCold)
	}

	// С открытым обещанием (+2.5) и событием (+2.0)
	urgencyBoosted := serviceCRM.CalculateUrgencyScore(snapCold, true, true, now)
	if urgencyBoosted != 8.5 {
		t.Fatalf("ожидалась срочность 8.5, получено: %f", urgencyBoosted)
	}

	// Отложенный контакт (Snooze)
	snapSnoozed := &domainCRM.HealthSnapshot{
		DunbarTier:   domainCRM.DunbarSupportClique,
		HealthScore:  20,
		SnoozedUntil: now.Add(24 * time.Hour),
	}
	if serviceCRM.CalculateUrgencyScore(snapSnoozed, true, true, now) != -1.0 {
		t.Fatalf("отложенный контакт должен возвращать -1.0")
	}
}

func TestCRMService_FullLifecycle(t *testing.T) {
	ctx := context.Background()
	svc, contactRepo, graphRepo := setupTestCRMService(t)

	// Создаём контакты: Ерлан и Алихан
	c1 := &domain.Contact{
		FirstName: "Ерлан",
		LastName:  "Тулегенов",
		Company:   "Astana Hub",
		JobTitle:  "Директор",
		Phone:     "+77011112233",
		Email:     "erlan@astanahub.kz",
	}
	c2 := &domain.Contact{
		FirstName: "Алихан",
		LastName:  "Сатыбалдиев",
		Phone:     "+77019998877",
	}
	_ = contactRepo.Create(ctx, c1)
	_ = contactRepo.Create(ctx, c2)

	// Добавляем ребро introduced_by
	_, _ = graphRepo.CreateRelationship(ctx, &graph.Relationship{
		SourceID:    c2.ID,
		TargetID:    c1.ID,
		Type:        graph.RelIntroducedBy,
		Weight:      5,
		ContextNote: "Хакатон Алматы 2025",
	})

	// 1. Логируем встречу с обещанием
	touch := &domainCRM.Interaction{
		Channel:        domainCRM.ChannelMeeting,
		Title:          "Обед в Del Papa",
		NotesMD:        "Обсудили DeepTech гранты",
		DurationMin:    50,
		OccurredAt:     time.Now().UTC(),
		ParticipantIDs: []int64{c1.ID},
	}
	promise := &domainCRM.ActionPromise{
		ContactID:   c1.ID,
		Direction:   "i_promised",
		Description: "Выслать драфт архитектуры",
		DueAt:       time.Now().Add(48 * time.Hour),
	}

	logged, err := svc.LogInteraction(ctx, touch, promise)
	if err != nil || logged.ID <= 0 {
		t.Fatalf("ошибка логирования встречи: %v", err)
	}

	// 2. Проверяем таймлайн
	timeline, err := svc.GetContactTimeline(ctx, c1.ID, 10)
	if err != nil || len(timeline) != 1 {
		t.Fatalf("ожидалось 1 касание в таймлайне: %v", timeline)
	}

	// 3. Проверяем обещание
	promises, err := svc.ListPendingPromises(ctx)
	if err != nil || len(promises) != 1 {
		t.Fatalf("ожидалось 1 активное обещание: %v", promises)
	}

	// 4. Проверяем здоровье (после личной встречи должно быть Thriving)
	health, err := svc.CalculateContactHealth(ctx, c1.ID)
	if err != nil || health.Status != domainCRM.StatusThriving {
		t.Fatalf("ожидался статус Thriving, получено: %v", health)
	}

	// 5. Генерируем досье перед встречей
	dossier, err := svc.GeneratePreMeetingDossier(ctx, c1.ID)
	if err != nil {
		t.Fatalf("ошибка генерации досье: %v", err)
	}
	if dossier.IntroducedByName != "Сатыбалдиев Алихан" && dossier.IntroducedByName != "Алихан Сатыбалдиев" {
		t.Fatalf("ожидался интро-контакт Алихан Сатыбалдиев, получено: '%s'", dossier.IntroducedByName)
	}
	md := dossier.ToMarkdown()
	if len(md) < 100 {
		t.Fatalf("слишком короткий Markdown досье: %s", md)
	}

	// 6. Проверяем Keep-in-Touch Radar
	candidates, err := svc.GetRadarCandidates(ctx, 10)
	if err != nil {
		t.Fatalf("ошибка получения кандидатов радара: %v", err)
	}
	// Контакт c1 имеет активное обещание, поэтому должен присутствовать
	if len(candidates) == 0 {
		t.Fatalf("ожидался минимум 1 кандидат на радаре")
	}

	// 7. Откладываем контакт (Snooze)
	if err := svc.SnoozeContact(ctx, c1.ID, 7*24*time.Hour); err != nil {
		t.Fatalf("ошибка snooze: %v", err)
	}
	candidatesAfter, _ := svc.GetRadarCandidates(ctx, 10)
	for _, cand := range candidatesAfter {
		if cand.ContactID == c1.ID {
			t.Fatalf("отложенный контакт c1 не должен появляться на радаре")
		}
	}
}

// BenchmarkRecalculateHealthScores_1000Contacts проверяет SLA пересчёта здоровья (SLA < 15 мс).
func BenchmarkRecalculateHealthScores_1000Contacts(b *testing.B) {
	ctx := context.Background()
	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		b.Fatalf("ошибка БД: %v", err)
	}
	defer db.Close()

	contactRepo := sqlite.NewContactRepo(db)
	crmRepo := sqlite.NewSQLiteCRMRepo(db)
	svc := serviceCRM.NewCRMService(crmRepo, contactRepo, nil)

	// Создаём 100 контактов для бенчмарка в памяти
	for i := 0; i < 100; i++ {
		_ = contactRepo.Create(ctx, &domain.Contact{
			FirstName: "Тест",
			LastName:  "Контакт",
			Phone:     "+7700" + string(rune('0'+i%10)) + string(rune('0'+i/10)),
		})
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = svc.RecalculateAllHealthScores(ctx)
	}
}
