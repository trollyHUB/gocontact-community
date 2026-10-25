// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package fileio

import (
	"bytes"
	"strings"
	"testing"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

func TestJSONStorage_ExportAndImport(t *testing.T) {
	storage := NewJSONStorage()
	original := []*domain.Contact{
		{
			ID:         1,
			GroupID:    2,
			FirstName:  "Алексей",
			LastName:   "Смирнов",
			Phone:      "+7 (701) 555-01-99",
			Email:      "a.smirnov@techcorp.kz",
			Company:    "TechCorp",
			Address:    domain.Address{City: "Алматы", Country: "Казахстан"},
			Tags:       []string{"golang", "backend"},
			IsFavorite: true,
		},
	}

	var buf bytes.Buffer
	if err := storage.Export(&buf, original); err != nil {
		t.Fatalf("ошибка JSON Export: %v", err)
	}

	imported, err := storage.Import(&buf)
	if err != nil {
		t.Fatalf("ошибка JSON Import: %v", err)
	}
	if len(imported) != 1 || imported[0].FullName() != "Смирнов Алексей" {
		t.Fatalf("данные после JSON импорта не совпадают: %+v", imported)
	}
}

func TestCSVStorage_ExportAndImport(t *testing.T) {
	storage := NewCSVStorage()
	original := []*domain.Contact{
		{
			GroupID:    3,
			FirstName:  "Мария",
			LastName:   "Иванова",
			Phone:      "+7 (707) 123-45-67",
			Email:      "m.ivanova@design.io",
			Company:    "DesignStudio",
			Address:    domain.Address{City: "Астана", Country: "Казахстан"},
			Tags:       []string{"ui", "ux"},
			IsFavorite: true,
		},
	}

	var buf bytes.Buffer
	if err := storage.Export(&buf, original); err != nil {
		t.Fatalf("ошибка CSV Export: %v", err)
	}

	imported, err := storage.Import(&buf)
	if err != nil {
		t.Fatalf("ошибка CSV Import: %v", err)
	}
	if len(imported) != 1 {
		t.Fatalf("ожидалась 1 запись из CSV, получено %d", len(imported))
	}
	if imported[0].FirstName != "Мария" || len(imported[0].Tags) != 2 || !imported[0].IsFavorite {
		t.Errorf("некорректный разбор CSV записи: %+v", imported[0])
	}
}

func TestVCardStorage_ExportAndImport(t *testing.T) {
	storage := NewVCardStorage()
	contacts := []*domain.Contact{
		{
			FirstName:  "Данияр",
			LastName:   "Ким",
			MiddleName: "Серикович",
			Phone:      "+7 (702) 998-77-66",
			WorkPhone:  "+7 (727) 250-10-20",
			Email:      "d.kim@cloudops.io",
			Company:    "CloudOps",
			JobTitle:   "SRE Architect",
			Address:    domain.Address{City: "Алматы", Country: "Казахстан", Street: "пр. Достык, 105", PostalCode: "050010"},
			Notes:      "Инфраструктура и высокая доступность",
		},
	}

	var buf bytes.Buffer
	if err := storage.Export(&buf, contacts); err != nil {
		t.Fatalf("ошибка vCard Export: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "BEGIN:VCARD") || !strings.Contains(out, "FN:Ким Данияр Серикович") {
		t.Errorf("некорректное содержимое vCard:\n%s", out)
	}

	imported, err := storage.Import(&buf)
	if err != nil {
		t.Fatalf("ошибка vCard Import: %v", err)
	}
	if len(imported) != 1 || imported[0].FullName() != "Ким Данияр Серикович" || imported[0].Company != "CloudOps" {
		t.Fatalf("данные после vCard импорта не совпадают: %+v", imported)
	}
}
