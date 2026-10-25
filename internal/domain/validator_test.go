// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package domain

import (
	"errors"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "Казахстанский номер с 8",
			input:   "8 (701) 555-01-99",
			want:    "+7 (701) 555-01-99",
			wantErr: false,
		},
		{
			name:    "Казахстанский номер с +7 без скобок",
			input:   "+77071234567",
			want:    "+7 (707) 123-45-67",
			wantErr: false,
		},
		{
			name:    "Десятизначный номер",
			input:   "705 999 88 77",
			want:    "+7 (705) 999-88-77",
			wantErr: false,
		},
		{
			name:    "Международный номер",
			input:   "+44 20 7946 0958",
			want:    "+442079460958",
			wantErr: false,
		},
		{
			name:    "Слишком короткий номер",
			input:   "12345",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Пустой номер",
			input:   "   ",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizePhone(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizePhone(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("NormalizePhone(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if tt.wantErr && !errors.Is(err, ErrInvalidPhone) {
				t.Errorf("ожидалась обёртка над ErrInvalidPhone, получено: %v", err)
			}
		})
	}
}

func TestNormalizeAndValidateContact(t *testing.T) {
	c := &Contact{
		FirstName: "  алексей ",
		LastName:  " смирнов",
		Phone:     "87015550199",
		Email:     "  A.Smirnov@TechCorp.kz ",
		Birthday:  "15.05.1995",
	}

	if err := NormalizeAndValidateContact(c); err != nil {
		t.Fatalf("неожиданная ошибка валидации: %v", err)
	}

	if c.FirstName != "Алексей" || c.LastName != "Смирнов" {
		t.Errorf("ФИО нормализовано неверно: %s %s", c.FirstName, c.LastName)
	}
	if c.Phone != "+7 (701) 555-01-99" {
		t.Errorf("телефон нормализован неверно: %s", c.Phone)
	}
	if c.Email != "a.smirnov@techcorp.kz" {
		t.Errorf("email нормализован неверно: %s", c.Email)
	}
	if c.FullName() != "Смирнов Алексей" {
		t.Errorf("FullName() = %q", c.FullName())
	}
	if c.Initials() != "СА" {
		t.Errorf("Initials() = %q, ожидалось СА", c.Initials())
	}
}

func TestContactMethodsAndTags(t *testing.T) {
	c := &Contact{
		FirstName: "Данияр",
		LastName:  "Ким",
		Phone:     "+77021112233",
	}

	c.AddTag("#Golang")
	c.AddTag("backend")
	c.AddTag("golang") // дубликат не должен добавиться

	if len(c.Tags) != 2 {
		t.Fatalf("ожидалось 2 уникальных тега, получено %d: %v", len(c.Tags), c.Tags)
	}

	if !c.ToggleFavorite() {
		t.Errorf("ожидалось IsFavorite = true после первого ToggleFavorite")
	}

	c.RemoveTag("backend")
	if len(c.Tags) != 1 || c.Tags[0] != "golang" {
		t.Errorf("некорректное удаление тега: %v", c.Tags)
	}
}

func TestValidationErrors(t *testing.T) {
	badEmailContact := &Contact{
		FirstName: "Иван",
		LastName:  "Петров",
		Phone:     "87011234567",
		Email:     "not-an-email",
	}

	err := NormalizeAndValidateContact(badEmailContact)
	if err == nil {
		t.Fatal("ожидалась ошибка валидации неверного email")
	}
	if !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("ожидалась ошибка ErrInvalidEmail, получено: %v", err)
	}

	var valErr *ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("ожидался кастомный тип *ValidationError")
	}
	if valErr.Field != "Email" {
		t.Errorf("ожидалось поле 'Email', получено %q", valErr.Field)
	}
}
