// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package fileio

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// Exporter определяет стратегию экспорта списка контактов в поток (Паттерн Strategy).
type Exporter interface {
	Export(w io.Writer, contacts []*domain.Contact) error
	FormatName() string
	FileExtension() string
}

// Importer определяет стратегию импорта списка контактов из потока.
type Importer interface {
	Import(r io.Reader) ([]*domain.Contact, error)
}

// ExportToFile сохраняет контакты в файл на диске с использованием переданной стратегии Exporter.
func ExportToFile(path string, exporter Exporter, contacts []*domain.Contact) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию для экспорта: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("ошибка создания файла %s: %w", path, err)
	}
	defer f.Close()

	if err := exporter.Export(f, contacts); err != nil {
		return fmt.Errorf("ошибка записи данных (%s): %w", exporter.FormatName(), err)
	}
	return nil
}

// ImportFromFile читает контакты из файла на диске с использованием переданной стратегии Importer.
func ImportFromFile(path string, importer Importer) ([]*domain.Contact, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла %s: %w", path, err)
	}
	defer f.Close()

	contacts, err := importer.Import(f)
	if err != nil {
		return nil, fmt.Errorf("ошибка разбора файла %s: %w", path, err)
	}
	return contacts, nil
}
