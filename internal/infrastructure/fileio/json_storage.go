// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package fileio

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// JSONExportPayload представляет корневую структуру файла резервной копии JSON (Неделя 7).
type JSONExportPayload struct {
	AppName    string            `json:"app_name"`
	Version    string            `json:"version"`
	ExportedAt time.Time         `json:"exported_at"`
	TotalCount int               `json:"total_count"`
	Contacts   []*domain.Contact `json:"contacts"`
}

// JSONStorage реализует интерфейсы Exporter и Importer для формата JSON.
type JSONStorage struct{}

func NewJSONStorage() *JSONStorage {
	return &JSONStorage{}
}

func (j *JSONStorage) FormatName() string    { return "JSON (JavaScript Object Notation)" }
func (j *JSONStorage) FileExtension() string { return ".json" }

// Export сериализует контакты в отформатированный JSON (encoding/json).
func (j *JSONStorage) Export(w io.Writer, contacts []*domain.Contact) error {
	payload := JSONExportPayload{
		AppName:    "GoContact Studio",
		Version:    "1.0",
		ExportedAt: time.Now().UTC(),
		TotalCount: len(contacts),
		Contacts:   contacts,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		return fmt.Errorf("ошибка JSON сериализации: %w", err)
	}
	return nil
}

// Import десериализует контакты из JSON-потока (поддерживает как обёртку JSONExportPayload, так и прямой массив []*Contact).
func (j *JSONStorage) Import(r io.Reader) ([]*domain.Contact, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения JSON потока: %w", err)
	}

	var payload JSONExportPayload
	if err := json.Unmarshal(data, &payload); err == nil && payload.Contacts != nil {
		return payload.Contacts, nil
	}

	var directList []*domain.Contact
	if err := json.Unmarshal(data, &directList); err != nil {
		return nil, fmt.Errorf("некорректный формат JSON файла: %w", err)
	}
	return directList, nil
}
