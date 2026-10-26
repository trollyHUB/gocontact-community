// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package graph

import (
	"errors"
	"testing"
	"time"
)

func TestRelationshipType_ValidationAndLabels(t *testing.T) {
	validTypes := []RelationshipType{
		RelColleague,
		RelReportsTo,
		RelFounderOf,
		RelFamily,
		RelFriend,
		RelMentor,
		RelIntroducedBy,
		RelPartner,
		RelImplicitColleague,
	}

	for _, vt := range validTypes {
		if !vt.IsValid() {
			t.Errorf("ожидался валидный тип %s", vt)
		}
		if vt.Label() == "Связь" && vt != "invalid" {
			// Должна быть специфическая подпись
			t.Logf("тип %s получил метку %s", vt, vt.Label())
		}
	}

	invalidType := RelationshipType("unknown_custom_rel")
	if invalidType.IsValid() {
		t.Errorf("невалидный тип должен возвращать false")
	}
	if invalidType.Label() != "Связь" {
		t.Errorf("для неизвестного типа ожидалась дефолтная метка, получено: %s", invalidType.Label())
	}
}

func TestDunbarTier_ValidationAndLabels(t *testing.T) {
	tiers := []struct {
		tier  DunbarTier
		valid bool
	}{
		{DunbarNone, true},
		{DunbarSupportClique, true},
		{DunbarSympathyGroup, true},
		{DunbarAffinityGroup, true},
		{DunbarActiveNetwork, true},
		{DunbarAcquaintances, true},
		{DunbarTier(999), false},
	}

	for _, tt := range tiers {
		if tt.tier.IsValid() != tt.valid {
			t.Errorf("для тира %d ожидалась валидность %v", tt.tier, tt.valid)
		}
		label := tt.tier.Label()
		if !tt.valid && label != "Не классифицирован" {
			t.Errorf("невалидный тир должен иметь дефолтную метку, получено: %s", label)
		}
	}
}

func TestRelationship_Validate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		rel     *Relationship
		wantErr error
	}{
		{
			name:    "nil relationship",
			rel:     nil,
			wantErr: ErrRelationshipNotFound,
		},
		{
			name: "invalid source id",
			rel: &Relationship{
				SourceID:  0,
				TargetID:  2,
				Type:      RelFriend,
				Weight:    3,
				CreatedAt: now,
			},
			wantErr: ErrInvalidContactID,
		},
		{
			name: "invalid target id",
			rel: &Relationship{
				SourceID:  1,
				TargetID:  -5,
				Type:      RelFriend,
				Weight:    3,
				CreatedAt: now,
			},
			wantErr: ErrInvalidContactID,
		},
		{
			name: "self-loop forbidden",
			rel: &Relationship{
				SourceID:  10,
				TargetID:  10,
				Type:      RelColleague,
				Weight:    4,
				CreatedAt: now,
			},
			wantErr: ErrSelfLoopNotAllowed,
		},
		{
			name: "weight too low",
			rel: &Relationship{
				SourceID:  10,
				TargetID:  20,
				Type:      RelColleague,
				Weight:    0,
				CreatedAt: now,
			},
			wantErr: ErrInvalidWeight,
		},
		{
			name: "weight too high",
			rel: &Relationship{
				SourceID:  10,
				TargetID:  20,
				Type:      RelColleague,
				Weight:    6,
				CreatedAt: now,
			},
			wantErr: ErrInvalidWeight,
		},
		{
			name: "invalid relationship type",
			rel: &Relationship{
				SourceID:  10,
				TargetID:  20,
				Type:      RelationshipType("hacker"),
				Weight:    3,
				CreatedAt: now,
			},
			wantErr: ErrInvalidRelationshipType,
		},
		{
			name: "valid relationship",
			rel: &Relationship{
				ID:          1,
				SourceID:    10,
				TargetID:    20,
				Type:        RelColleague,
				Weight:      5,
				IsDirected:  false,
				ContextNote: "Бывшие коллеги по разработке",
				SinceDate:   "2020-01-01",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rel.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ожидалась ошибка %v, получено %v", tt.wantErr, err)
			}
		})
	}
}

func TestOrganization_Validate(t *testing.T) {
	tests := []struct {
		name    string
		org     *Organization
		wantErr error
	}{
		{
			name:    "nil organization",
			org:     nil,
			wantErr: ErrOrgNotFound,
		},
		{
			name: "empty name",
			org: &Organization{
				Name: "   ",
			},
			wantErr: ErrEmptyOrgName,
		},
		{
			name: "valid organization",
			org: &Organization{
				ID:       1,
				Name:     "TechCorp",
				Industry: "IT",
				Domain:   "techcorp.kz",
				City:     "Алматы",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.org.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ожидалась ошибка %v, получено %v", tt.wantErr, err)
			}
		})
	}
}

func TestGraphLayout_Validate(t *testing.T) {
	tests := []struct {
		name    string
		layout  *GraphLayout
		wantErr error
	}{
		{
			name:    "nil layout",
			layout:  nil,
			wantErr: ErrNodeNotFound,
		},
		{
			name: "invalid contact id",
			layout: &GraphLayout{
				ContactID: 0,
			},
			wantErr: ErrInvalidContactID,
		},
		{
			name: "valid layout",
			layout: &GraphLayout{
				ContactID:        42,
				PosX:             100.5,
				PosY:             -50.2,
				IsPinned:         true,
				CommunityID:      1,
				BetweennessScore: 0.85,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.layout.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ожидалась ошибка %v, получено %v", tt.wantErr, err)
			}
		})
	}
}

func TestDomainStructs_Instantiation(t *testing.T) {
	// Проверка узла Node
	node := Node{
		ContactID:        1,
		FullName:         "Тест Тестов",
		Company:          "ТестОрг",
		AvatarInitials:   "ТТ",
		DunbarTier:       DunbarSupportClique,
		CommunityID:      1,
		Degree:           5,
		BetweennessScore: 0.12,
		PosX:             10,
		PosY:             20,
		IsPinned:         false,
	}
	if node.ContactID != 1 || node.DunbarTier != DunbarSupportClique {
		t.Errorf("неверная инициализация узла: %+v", node)
	}

	// Проверка пути интро
	hop := IntroHop{
		FromContactID: 1,
		ToContactID:   2,
		ViaEdgeType:   RelMentor,
		EdgeWeight:    4,
		ContextNote:   "Ментор",
	}
	path := IntroPath{
		SourceID:        1,
		TargetID:        2,
		Hops:            []IntroHop{hop},
		TotalResistance: 1.25,
		CompoundTrust:   0.8,
	}
	if len(path.Hops) != 1 || path.CompoundTrust != 0.8 {
		t.Errorf("неверная инициализация пути: %+v", path)
	}

	// Проверка эго-узла
	ego := EgoNode{
		ContactID:     2,
		FirstName:     "Иван",
		LastName:      "Иванов",
		Company:       "Horns and Hoofs",
		MinHop:        1,
		MaxTrustScore: 0.9,
	}
	if ego.MinHop != 1 || ego.FirstName != "Иван" {
		t.Errorf("неверная инициализация эго-узла: %+v", ego)
	}

	// Проверка псевдонима Edge = Relationship
	var edge Edge = Relationship{
		SourceID: 1,
		TargetID: 2,
		Type:     RelFriend,
		Weight:   4,
	}
	if edge.SourceID != 1 || edge.Type != RelFriend {
		t.Errorf("псевдоним Edge работает некорректно")
	}
}
