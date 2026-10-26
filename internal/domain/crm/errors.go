// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package crm

import "errors"

// Доменные ошибки Personal CRM.
var (
	ErrEmptyParticipants   = errors.New("crm: взаимодействие должно включать минимум одного участника")
	ErrInvalidChannel      = errors.New("crm: неизвестный канал коммуникации")
	ErrPromiseEmptyDesc    = errors.New("crm: описание обещания не может быть пустым")
	ErrInvalidCadenceDays  = errors.New("crm: целевой интервал дней должен быть в диапазоне [0..365]")
	ErrInteractionNotFound = errors.New("crm: взаимодействие не найдено")
	ErrPromiseNotFound     = errors.New("crm: обещание не найдено")
	ErrLifeEventNotFound   = errors.New("crm: памятная дата не найдена")
)
