// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package graph

import (
	"strings"
	"time"
)

// RelationshipType задаёт семантический тип связи между людьми.
type RelationshipType string

const (
	RelColleague         RelationshipType = "colleague"
	RelReportsTo         RelationshipType = "reports_to"
	RelFounderOf         RelationshipType = "founder_of"
	RelFamily            RelationshipType = "family"
	RelFriend            RelationshipType = "friend"
	RelMentor            RelationshipType = "mentor"
	RelIntroducedBy      RelationshipType = "introduced_by"
	RelPartner           RelationshipType = "partner"
	RelImplicitColleague RelationshipType = "implicit_colleague"
)

// IsValid проверяет принадлежность типа связи допустимому перечислению.
func (t RelationshipType) IsValid() bool {
	switch t {
	case RelColleague, RelReportsTo, RelFounderOf,
		RelFamily, RelFriend, RelMentor,
		RelIntroducedBy, RelPartner, RelImplicitColleague:
		return true
	default:
		return false
	}
}

// Label возвращает русскоязычное наименование типа связи для интерфейса.
func (t RelationshipType) Label() string {
	switch t {
	case RelColleague:
		return "Коллега"
	case RelReportsTo:
		return "Руководитель"
	case RelFounderOf:
		return "Основатель"
	case RelFamily:
		return "Семья"
	case RelFriend:
		return "Друг"
	case RelMentor:
		return "Наставник"
	case RelIntroducedBy:
		return "Познакомил"
	case RelPartner:
		return "Партнёр"
	case RelImplicitColleague:
		return "Общая организация"
	default:
		return "Связь"
	}
}

// DunbarTier представляет концентрический круг близости по числу Данбара.
type DunbarTier int

const (
	DunbarNone          DunbarTier = 0
	DunbarSupportClique DunbarTier = 5   // Ядро поддержки (семья, лучшие друзья)
	DunbarSympathyGroup DunbarTier = 15  // Группа симпатии (высокое доверие)
	DunbarAffinityGroup DunbarTier = 50  // Активный рабочий и дружеский круг
	DunbarActiveNetwork DunbarTier = 150 // Стабильная социальная сеть (число Данбара)
	DunbarAcquaintances DunbarTier = 500 // Внешний пояс знакомых
)

// IsValid проверяет допустимость значения круга Данбара.
func (d DunbarTier) IsValid() bool {
	switch d {
	case DunbarNone, DunbarSupportClique, DunbarSympathyGroup,
		DunbarAffinityGroup, DunbarActiveNetwork, DunbarAcquaintances:
		return true
	default:
		return false
	}
}

// Label возвращает текстовое описание круга Данбара.
func (d DunbarTier) Label() string {
	switch d {
	case DunbarSupportClique:
		return "Круг поддержки (≤5)"
	case DunbarSympathyGroup:
		return "Группа симпатии (≤15)"
	case DunbarAffinityGroup:
		return "Круг доверия (≤50)"
	case DunbarActiveNetwork:
		return "Активная сеть (≤150)"
	case DunbarAcquaintances:
		return "Знакомые (≤500)"
	default:
		return "Не классифицирован"
	}
}

// Relationship описывает связь (ребро мультиграфа) между двумя контактами.
type Relationship struct {
	ID          int64            `json:"id"`
	SourceID    int64            `json:"source_id"`
	TargetID    int64            `json:"target_id"`
	Type        RelationshipType `json:"rel_type"`
	Weight      int              `json:"weight"` // 1 (слабое знакомство) .. 5 (максимальное доверие)
	IsDirected  bool             `json:"is_directed"`
	ContextNote string           `json:"context_note"`
	SinceDate   string           `json:"since_date"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// Edge — псевдоним типа Relationship для алгоритмов социального графа.
type Edge = Relationship

// Validate проверяет корректность параметров связи.
func (r *Relationship) Validate() error {
	if r == nil {
		return ErrRelationshipNotFound
	}
	if r.SourceID <= 0 || r.TargetID <= 0 {
		return ErrInvalidContactID
	}
	if r.SourceID == r.TargetID {
		return ErrSelfLoopNotAllowed
	}
	if r.Weight < 1 || r.Weight > 5 {
		return ErrInvalidWeight
	}
	if !r.Type.IsValid() {
		return ErrInvalidRelationshipType
	}
	return nil
}

// Organization описывает нормализованную компанию или организацию (узел 2-го рода).
type Organization struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Industry  string    `json:"industry"`
	Domain    string    `json:"domain"`
	City      string    `json:"city"`
	CreatedAt time.Time `json:"created_at"`
}

// Validate проверяет корректность полей организации.
func (o *Organization) Validate() error {
	if o == nil {
		return ErrOrgNotFound
	}
	if strings.TrimSpace(o.Name) == "" {
		return ErrEmptyOrgName
	}
	return nil
}

// GraphLayout хранит сохранённые координаты узла на холсте и предрассчитанные SNA-метрики.
type GraphLayout struct {
	ContactID        int64     `json:"contact_id"`
	PosX             float64   `json:"pos_x"`
	PosY             float64   `json:"pos_y"`
	IsPinned         bool      `json:"is_pinned"`
	CommunityID      int       `json:"community_id"`
	BetweennessScore float64   `json:"betweenness_score"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Validate проверяет валидность данных размещения узла.
func (l *GraphLayout) Validate() error {
	if l == nil {
		return ErrNodeNotFound
	}
	if l.ContactID <= 0 {
		return ErrInvalidContactID
	}
	return nil
}

// Node описывает вершину графа с физическими и топологическими характеристиками для визуализации и анализа.
type Node struct {
	ContactID        int64      `json:"contact_id"`
	FullName         string     `json:"full_name"`
	Company          string     `json:"company"`
	AvatarInitials   string     `json:"avatar_initials"`
	DunbarTier       DunbarTier `json:"dunbar_tier"`
	CommunityID      int        `json:"community_id"`
	Degree           int        `json:"degree"`
	BetweennessScore float64    `json:"betweenness_score"`

	// Физическое состояние для симуляции Barnes-Hut / Force-Directed
	PosX     float64 `json:"pos_x"`
	PosY     float64 `json:"pos_y"`
	VelX     float64 `json:"-"`
	VelY     float64 `json:"-"`
	ForceX   float64 `json:"-"`
	ForceY   float64 `json:"-"`
	Mass     float64 `json:"-"`
	IsPinned bool    `json:"is_pinned"`
}

// IntroHop представляет единичный шаг в цепочке тёплого интро.
type IntroHop struct {
	FromContactID int64            `json:"from_contact_id"`
	ToContactID   int64            `json:"to_contact_id"`
	ViaEdgeType   RelationshipType `json:"via_edge_type"`
	EdgeWeight    int              `json:"edge_weight"`
	ContextNote   string           `json:"context_note"`
}

// IntroPath — результат алгоритма поиска пути знакомств между двумя контактами.
type IntroPath struct {
	SourceID        int64      `json:"source_id"`
	TargetID        int64      `json:"target_id"`
	Hops            []IntroHop `json:"hops"`
	TotalResistance float64    `json:"total_resistance"`
	CompoundTrust   float64    `json:"compound_trust"` // [0.0 .. 1.0]
}

// EgoNode представляет узел в Ego-Network с метриками шага и доверия.
type EgoNode struct {
	ContactID     int64   `json:"contact_id"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	Company       string  `json:"company"`
	MinHop        int     `json:"min_hop"`
	MaxTrustScore float64 `json:"max_trust_score"`
}
