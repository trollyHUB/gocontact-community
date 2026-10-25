// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package domain

import (
	"errors"
	"fmt"
)

// Сентинел-ошибки предметной области (Неделя 8: Обработка ошибок).
var (
	ErrContactNotFound = errors.New("контакт не найден")
	ErrGroupNotFound   = errors.New("группа не найдена")
	ErrDuplicatePhone  = errors.New("контакт с таким номером телефона уже существует")
	ErrDuplicateGroup  = errors.New("группа с таким названием уже существует")
	ErrEmptyFirstName  = errors.New("имя контакта не может быть пустым")
	ErrEmptyLastName   = errors.New("фамилия контакта не может быть пустой")
	ErrInvalidPhone    = errors.New("некорректный формат номера телефона")
	ErrInvalidEmail    = errors.New("некорректный формат электронной почты")
	ErrInvalidBirthday = errors.New("некорректный формат даты рождения (ожидается ДД.ММ.ГГГГ)")
)

// ValidationError представляет собственную (кастомную) ошибку валидации поля (Неделя 8).
// Реализует стандартный интерфейс error и интерфейс Unwrap для совместимости с errors.Is / errors.As.
type ValidationError struct {
	Field   string // Название поля, в котором допущена ошибка
	Value   string // Переданное некорректное значение
	Message string // Человекочитаемое описание проблемы
	Err     error  // Базовая ошибка (для цепочки ошибок)
}

// Error возвращает строковое представление ошибки (реализация интерфейса error).
func (e *ValidationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("ошибка валидации поля «%s» (значение %q): %s", e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("ошибка валидации поля «%s»: %s", e.Field, e.Message)
}

// Unwrap позволяет проверять вложенную ошибку через errors.Is(err, ErrInvalidPhone).
func (e *ValidationError) Unwrap() error {
	return e.Err
}

// NewValidationError создаёт новый экземпляр кастомной ошибки валидации.
func NewValidationError(field, value, message string, baseErr error) error {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
		Err:     baseErr,
	}
}
