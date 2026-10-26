// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sqlite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/repository/sqlite"
)

func TestOrgRepo_CRUD(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("ошибка инициализации БД: %v", err)
	}
	defer db.Close()

	orgRepo := sqlite.NewOrgRepo(db)

	// 1. Создание организации
	org := &graph.Organization{
		Name:     "Kolesa Group",
		Industry: "IT & Auto",
		Domain:   "kolesa.kz",
		City:     "Алматы",
	}

	id, err := orgRepo.CreateOrganization(ctx, org)
	if err != nil {
		t.Fatalf("ошибка создания организации: %v", err)
	}
	if id <= 0 || org.ID != id {
		t.Fatalf("некорректный ID организации: %d", id)
	}

	// 2. Ошибка дубликата имени (COLLATE NOCASE)
	dupOrg := &graph.Organization{Name: "kolesa group"}
	_, err = orgRepo.CreateOrganization(ctx, dupOrg)
	if !errors.Is(err, graph.ErrDuplicateOrg) {
		t.Fatalf("ожидалась ошибка ErrDuplicateOrg, получено: %v", err)
	}

	// 3. Ошибка пустого имени
	emptyOrg := &graph.Organization{Name: "   "}
	_, err = orgRepo.CreateOrganization(ctx, emptyOrg)
	if !errors.Is(err, graph.ErrEmptyOrgName) {
		t.Fatalf("ожидалась ошибка ErrEmptyOrgName, получено: %v", err)
	}

	// 4. Получение по ID
	found, err := orgRepo.GetOrganization(ctx, id)
	if err != nil {
		t.Fatalf("ошибка получения организации по ID: %v", err)
	}
	if found.Name != "Kolesa Group" || found.City != "Алматы" {
		t.Fatalf("неверные данные организации: %+v", found)
	}

	// 5. Поиск по имени
	byName, err := orgRepo.GetOrganizationByName(ctx, "KOLESA GROUP")
	if err != nil {
		t.Fatalf("ошибка поиска по имени: %v", err)
	}
	if byName.ID != id {
		t.Fatalf("неверный ID найденной организации: %d", byName.ID)
	}

	// 6. Несуществующая организация
	_, err = orgRepo.GetOrganization(ctx, 99999)
	if !errors.Is(err, graph.ErrOrgNotFound) {
		t.Fatalf("ожидалась ошибка ErrOrgNotFound, получено: %v", err)
	}

	// 7. Обновление
	found.City = "Астана"
	if err := orgRepo.UpdateOrganization(ctx, found); err != nil {
		t.Fatalf("ошибка обновления организации: %v", err)
	}
	updated, _ := orgRepo.GetOrganization(ctx, id)
	if updated.City != "Астана" {
		t.Fatalf("город не обновился: %s", updated.City)
	}

	// 8. Список
	org2 := &graph.Organization{Name: "Astana Hub", Industry: "Technopark"}
	_, _ = orgRepo.CreateOrganization(ctx, org2)
	list, err := orgRepo.ListOrganizations(ctx)
	if err != nil {
		t.Fatalf("ошибка получения списка организаций: %v", err)
	}
	if len(list) < 2 {
		t.Fatalf("ожидалось как минимум 2 организации, получено: %d", len(list))
	}
	// Алфавитный порядок: "Astana Hub" перед "Kolesa Group"
	if list[0].Name != "Astana Hub" {
		t.Errorf("ожидалась сортировка по алфавиту, первая: %s", list[0].Name)
	}

	// 9. Удаление
	if err := orgRepo.DeleteOrganization(ctx, id); err != nil {
		t.Fatalf("ошибка удаления организации: %v", err)
	}
	_, err = orgRepo.GetOrganization(ctx, id)
	if !errors.Is(err, graph.ErrOrgNotFound) {
		t.Fatalf("после удаления ожидался ErrOrgNotFound, получено: %v", err)
	}
}

func TestGraphRepo_RelationshipsCRUD_AndCascade(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("ошибка инициализации БД: %v", err)
	}
	defer db.Close()

	contactRepo := sqlite.NewContactRepo(db)
	graphRepo := sqlite.NewGraphRepo(db)

	// Создаём тестовые контакты
	c1 := &domain.Contact{
		FirstName: "Иван", LastName: "Иванов", Phone: "+7 (701) 111-22-33", PhoneType: domain.PhoneMobile,
	}
	c2 := &domain.Contact{
		FirstName: "Петр", LastName: "Петров", Phone: "+7 (701) 222-33-44", PhoneType: domain.PhoneMobile,
	}
	c3 := &domain.Contact{
		FirstName: "Анна", LastName: "Сидорова", Phone: "+7 (701) 333-44-55", PhoneType: domain.PhoneMobile,
	}
	if err := contactRepo.Create(ctx, c1); err != nil {
		t.Fatalf("ошибка создания c1: %v", err)
	}
	if err := contactRepo.Create(ctx, c2); err != nil {
		t.Fatalf("ошибка создания c2: %v", err)
	}
	if err := contactRepo.Create(ctx, c3); err != nil {
		t.Fatalf("ошибка создания c3: %v", err)
	}

	// 1. Создание корректной связи c1 -> c2
	rel := &graph.Relationship{
		SourceID:    c1.ID,
		TargetID:    c2.ID,
		Type:        graph.RelColleague,
		Weight:      4,
		IsDirected:  false,
		ContextNote: "Работали в одном отделе",
		SinceDate:   "2021-05-01",
	}

	relID, err := graphRepo.CreateRelationship(ctx, rel)
	if err != nil {
		t.Fatalf("ошибка создания связи: %v", err)
	}
	if relID <= 0 || rel.ID != relID {
		t.Fatalf("некорректный ID связи: %d", relID)
	}

	// 2. Запрет петли (Self Loop)
	loopRel := &graph.Relationship{
		SourceID: c1.ID,
		TargetID: c1.ID,
		Type:     graph.RelFriend,
		Weight:   3,
	}
	_, err = graphRepo.CreateRelationship(ctx, loopRel)
	if !errors.Is(err, graph.ErrSelfLoopNotAllowed) {
		t.Fatalf("ожидалась ошибка ErrSelfLoopNotAllowed, получено: %v", err)
	}

	// 3. Запрет недопустимого веса
	badWeightRel := &graph.Relationship{
		SourceID: c1.ID,
		TargetID: c3.ID,
		Type:     graph.RelFriend,
		Weight:   10,
	}
	_, err = graphRepo.CreateRelationship(ctx, badWeightRel)
	if !errors.Is(err, graph.ErrInvalidWeight) {
		t.Fatalf("ожидалась ошибка ErrInvalidWeight, получено: %v", err)
	}

	// 4. Запрет дубликата связи того же типа
	dupRel := &graph.Relationship{
		SourceID: c1.ID,
		TargetID: c2.ID,
		Type:     graph.RelColleague,
		Weight:   5,
	}
	_, err = graphRepo.CreateRelationship(ctx, dupRel)
	if !errors.Is(err, graph.ErrDuplicateRelationship) {
		t.Fatalf("ожидалась ошибка ErrDuplicateRelationship, получено: %v", err)
	}

	// 5. Внешний ключ: несуществующий контакт
	fkRel := &graph.Relationship{
		SourceID: c1.ID,
		TargetID: 99999,
		Type:     graph.RelFriend,
		Weight:   3,
	}
	_, err = graphRepo.CreateRelationship(ctx, fkRel)
	if !errors.Is(err, domain.ErrContactNotFound) {
		t.Fatalf("ожидалась ошибка ErrContactNotFound при FK violation, получено: %v", err)
	}

	// 6. Получение связи по ID
	foundRel, err := graphRepo.GetRelationship(ctx, relID)
	if err != nil {
		t.Fatalf("ошибка получения связи: %v", err)
	}
	if foundRel.SourceID != c1.ID || foundRel.TargetID != c2.ID || foundRel.Weight != 4 {
		t.Fatalf("неверные атрибуты связи: %+v", foundRel)
	}

	// 7. Получение связей по контакту
	c1Rels, err := graphRepo.GetRelationshipsByContact(ctx, c1.ID)
	if err != nil {
		t.Fatalf("ошибка связей c1: %v", err)
	}
	if len(c1Rels) != 1 {
		t.Fatalf("ожидалась 1 связь для c1, получено: %d", len(c1Rels))
	}

	// Также для c2 (входящая)
	c2Rels, err := graphRepo.GetRelationshipsByContact(ctx, c2.ID)
	if err != nil {
		t.Fatalf("ошибка связей c2: %v", err)
	}
	if len(c2Rels) != 1 {
		t.Fatalf("ожидалась 1 связь для c2, получено: %d", len(c2Rels))
	}

	// 8. Обновление связи
	foundRel.Weight = 5
	foundRel.ContextNote = "Повышен уровень доверия"
	if err := graphRepo.UpdateRelationship(ctx, foundRel); err != nil {
		t.Fatalf("ошибка обновления связи: %v", err)
	}
	updatedRel, _ := graphRepo.GetRelationship(ctx, relID)
	if updatedRel.Weight != 5 || updatedRel.ContextNote != "Повышен уровень доверия" {
		t.Fatalf("связь не обновилась: %+v", updatedRel)
	}

	// 9. Все связи
	allRels, err := graphRepo.GetAllRelationships(ctx)
	if err != nil || len(allRels) != 1 {
		t.Fatalf("ожидалась ровно 1 связь, получено %d, err: %v", len(allRels), err)
	}

	// 10. Каскадное удаление (ON DELETE CASCADE)
	// Удаляем c2 из contacts -> ребро relID должно удалиться автоматически!
	if err := contactRepo.Delete(ctx, c2.ID); err != nil {
		t.Fatalf("ошибка удаления контакта c2: %v", err)
	}
	_, err = graphRepo.GetRelationship(ctx, relID)
	if !errors.Is(err, graph.ErrRelationshipNotFound) {
		t.Fatalf("ожидалось каскадное удаление связи при удалении контакта, err: %v", err)
	}
}

func TestGraphRepo_Layouts(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("ошибка инициализации БД: %v", err)
	}
	defer db.Close()

	contactRepo := sqlite.NewContactRepo(db)
	graphRepo := sqlite.NewGraphRepo(db)

	c1 := &domain.Contact{FirstName: "Тест1", LastName: "Лейаут1", Phone: "+7 (701) 555-11-11", PhoneType: domain.PhoneMobile}
	c2 := &domain.Contact{FirstName: "Тест2", LastName: "Лейаут2", Phone: "+7 (701) 555-22-22", PhoneType: domain.PhoneMobile}
	_ = contactRepo.Create(ctx, c1)
	_ = contactRepo.Create(ctx, c2)

	// 1. Одиночное сохранение (SaveLayout)
	layout1 := &graph.GraphLayout{
		ContactID:        c1.ID,
		PosX:             150.5,
		PosY:             -30.2,
		IsPinned:         true,
		CommunityID:      2,
		BetweennessScore: 0.75,
	}
	if err := graphRepo.SaveLayout(ctx, layout1); err != nil {
		t.Fatalf("ошибка SaveLayout: %v", err)
	}

	// 2. Чтение лейаута
	got, err := graphRepo.GetLayout(ctx, c1.ID)
	if err != nil {
		t.Fatalf("ошибка GetLayout: %v", err)
	}
	if got.PosX != 150.5 || !got.IsPinned || got.CommunityID != 2 {
		t.Fatalf("неверные данные лейаута: %+v", got)
	}

	// 3. Пакетное сохранение (SaveLayoutBatch)
	layout1.PosX = 200.0
	layout2 := &graph.GraphLayout{
		ContactID:        c2.ID,
		PosX:             -80.0,
		PosY:             120.0,
		IsPinned:         false,
		CommunityID:      2,
		BetweennessScore: 0.15,
	}

	if err := graphRepo.SaveLayoutBatch(ctx, []*graph.GraphLayout{layout1, layout2}); err != nil {
		t.Fatalf("ошибка SaveLayoutBatch: %v", err)
	}

	all, err := graphRepo.GetAllLayouts(ctx)
	if err != nil {
		t.Fatalf("ошибка GetAllLayouts: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("ожидалось 2 лейаута, получено: %d", len(all))
	}
	if all[c1.ID].PosX != 200.0 {
		t.Errorf("ожидалась обновлённая координата PosX=200.0, получено: %f", all[c1.ID].PosX)
	}

	// 4. Несуществующий узел
	_, err = graphRepo.GetLayout(ctx, 99999)
	if !errors.Is(err, graph.ErrNodeNotFound) {
		t.Fatalf("ожидалась ошибка ErrNodeNotFound, получено: %v", err)
	}

	// 5. Каскадное удаление координат узла при удалении контакта
	_ = contactRepo.Delete(ctx, c1.ID)
	_, err = graphRepo.GetLayout(ctx, c1.ID)
	if !errors.Is(err, graph.ErrNodeNotFound) {
		t.Fatalf("ожидалось каскадное удаление лейаута при удалении контакта")
	}
}

func TestGraphRepo_RecursiveCTE_EgoSubgraph(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("ошибка инициализации БД: %v", err)
	}
	defer db.Close()

	contactRepo := sqlite.NewContactRepo(db)
	graphRepo := sqlite.NewGraphRepo(db)

	// Создаём цепочку: A -> B -> C -> D
	cA := &domain.Contact{FirstName: "Alice", LastName: "A", Phone: "+7 (701) 001-00-01", Company: "Alpha"}
	cB := &domain.Contact{FirstName: "Bob", LastName: "B", Phone: "+7 (701) 002-00-02", Company: "Beta"}
	cC := &domain.Contact{FirstName: "Charlie", LastName: "C", Phone: "+7 (701) 003-00-03", Company: "Gamma"}
	cD := &domain.Contact{FirstName: "Diana", LastName: "D", Phone: "+7 (701) 004-00-04", Company: "Delta"}
	_ = contactRepo.Create(ctx, cA)
	_ = contactRepo.Create(ctx, cB)
	_ = contactRepo.Create(ctx, cC)
	_ = contactRepo.Create(ctx, cD)

	// A <-> B (weight 5)
	_, _ = graphRepo.CreateRelationship(ctx, &graph.Relationship{
		SourceID: cA.ID, TargetID: cB.ID, Type: graph.RelFriend, Weight: 5, IsDirected: false,
	})
	// B <-> C (weight 4)
	_, _ = graphRepo.CreateRelationship(ctx, &graph.Relationship{
		SourceID: cB.ID, TargetID: cC.ID, Type: graph.RelColleague, Weight: 4, IsDirected: false,
	})
	// C <-> D (weight 3)
	_, _ = graphRepo.CreateRelationship(ctx, &graph.Relationship{
		SourceID: cC.ID, TargetID: cD.ID, Type: graph.RelPartner, Weight: 3, IsDirected: false,
	})
	// Цикл для проверки защиты: D <-> A
	_, _ = graphRepo.CreateRelationship(ctx, &graph.Relationship{
		SourceID: cD.ID, TargetID: cA.ID, Type: graph.RelMentor, Weight: 2, IsDirected: false,
	})

	// Запрос Ego-сети вокруг Alice глубиной 2 шага (должны попасть A, B, C, D)
	egoNodes, err := graphRepo.QueryEgoSubgraph(ctx, cA.ID, 2)
	if err != nil {
		t.Fatalf("ошибка выполнения QueryEgoSubgraph: %v", err)
	}

	if len(egoNodes) == 0 {
		t.Fatalf("ожидались узлы эго-сети")
	}

	// Первый узел должен быть Alice (min_hop = 0)
	if egoNodes[0].ContactID != cA.ID || egoNodes[0].MinHop != 0 {
		t.Fatalf("первый узел должен быть сам ego (hop=0), получено: %+v", egoNodes[0])
	}

	// Проверяем наличие Bob (hop=1)
	var foundBob, foundCharlie bool
	for _, n := range egoNodes {
		if n.ContactID == cB.ID && n.MinHop == 1 {
			foundBob = true
		}
		if n.ContactID == cC.ID && n.MinHop == 2 {
			foundCharlie = true
		}
	}
	if !foundBob {
		t.Errorf("Bob (hop 1) не найден в ego-сети")
	}
	if !foundCharlie {
		t.Errorf("Charlie (hop 2) не найден в ego-сети")
	}
}

func TestGraphRepo_ImplicitRelationships(t *testing.T) {
	ctx := context.Background()
	db, err := sqlite.OpenAndMigrate(":memory:")
	if err != nil {
		t.Fatalf("ошибка инициализации БД: %v", err)
	}
	defer db.Close()

	contactRepo := sqlite.NewContactRepo(db)
	graphRepo := sqlite.NewGraphRepo(db)
	orgRepo := sqlite.NewOrgRepo(db)

	// Создаём организацию
	orgID, err := orgRepo.CreateOrganization(ctx, &graph.Organization{Name: "Kaspi Bank", City: "Алматы"})
	if err != nil {
		t.Fatalf("ошибка создания организации: %v", err)
	}

	// Создаём 2 контакта с общим org_id
	c1 := &domain.Contact{FirstName: "Тимур", LastName: "Турлов", Phone: "+7 (701) 900-11-22", OrgID: orgID}
	c2 := &domain.Contact{FirstName: "Михаил", LastName: "Ломтадзе", Phone: "+7 (701) 900-33-44", OrgID: orgID}
	c3 := &domain.Contact{FirstName: "Иван", LastName: "Одинокий", Phone: "+7 (701) 900-55-66"}

	_ = contactRepo.Create(ctx, c1)
	_ = contactRepo.Create(ctx, c2)
	_ = contactRepo.Create(ctx, c3)

	// Записываем org_id контактов напрямую в БД
	_, err = db.Exec(`UPDATE contacts SET org_id = ? WHERE id IN (?, ?)`, orgID, c1.ID, c2.ID)
	if err != nil {
		t.Fatalf("ошибка установки org_id: %v", err)
	}

	implicitRels, err := graphRepo.GetImplicitRelationships(ctx)
	if err != nil {
		t.Fatalf("ошибка GetImplicitRelationships: %v", err)
	}

	if len(implicitRels) != 1 {
		t.Fatalf("ожидалась ровно 1 неявная связь между c1 и c2, получено: %d", len(implicitRels))
	}

	ir := implicitRels[0]
	if ir.SourceID != c1.ID || ir.TargetID != c2.ID || ir.Type != graph.RelImplicitColleague {
		t.Fatalf("неверная неявная связь: %+v", ir)
	}
	if ir.ContextNote != "Kaspi Bank" {
		t.Errorf("ожидалось название организации в контексте, получено: %s", ir.ContextNote)
	}
}
