// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package graph

import (
	"errors"
)

// Доменные ошибки подсистемы социального графа и реестра организаций.
// Святая зона Clean Architecture: 0 внешних импортов, только стандартная библиотека.
var (
	// ErrSelfLoopNotAllowed возвращается при попытке создать связь контакта с самим собой.
	ErrSelfLoopNotAllowed = errors.New("graph: связь контакта с самим собой запрещена")

	// ErrInvalidWeight возвращается, если вес доверия выходит за границы [1..5].
	ErrInvalidWeight = errors.New("graph: вес доверия связи должен быть в диапазоне [1..5]")

	// ErrRelationshipNotFound возвращается при запросе несуществующей связи.
	ErrRelationshipNotFound = errors.New("graph: связь между контактами не найдена")

	// ErrOrgNotFound возвращается при запросе несуществующей организации.
	ErrOrgNotFound = errors.New("graph: организация не найдена")

	// ErrIntroPathNotFound возвращается, если путь интро между узлами не существует.
	ErrIntroPathNotFound = errors.New("graph: цепочка знакомств между контактами не найдена")

	// ErrNodeNotFound возвращается при запросе несуществующего узла графа.
	ErrNodeNotFound = errors.New("graph: узел контакта не найден в топологии")

	// ErrInvalidRelationshipType возвращается при передаче неизвестного типа связи.
	ErrInvalidRelationshipType = errors.New("graph: недопустимый тип связи")

	// ErrEmptyOrgName возвращается при попытке создать организацию с пустым названием.
	ErrEmptyOrgName = errors.New("graph: название организации не может быть пустым")

	// ErrInvalidContactID возвращается при некорректном идентификаторе контакта.
	ErrInvalidContactID = errors.New("graph: некорректный идентификатор контакта")

	// ErrDuplicateRelationship возвращается при попытке создать связь между контактами одного типа повторно.
	ErrDuplicateRelationship = errors.New("graph: связь между данными контактами такого типа уже существует")

	// ErrDuplicateOrg возвращается при попытке создать организацию с уже занятым именем.
	ErrDuplicateOrg = errors.New("graph: организация с таким названием уже существует")
)
