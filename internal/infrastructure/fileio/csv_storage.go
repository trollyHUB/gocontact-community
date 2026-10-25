// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package fileio

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// CSVStorage реализует интерфейсы Exporter и Importer для табличного формата CSV (Неделя 7).
type CSVStorage struct{}

func NewCSVStorage() *CSVStorage {
	return &CSVStorage{}
}

func (c *CSVStorage) FormatName() string    { return "CSV (Таблица Excel / Google Contacts)" }
func (c *CSVStorage) FileExtension() string { return ".csv" }

var csvHeaders = []string{
	"first_name", "last_name", "middle_name", "phone", "work_phone",
	"phone_type", "email", "company", "job_title",
	"country", "city", "street", "postal_code",
	"birthday", "group_id", "is_favorite", "tags", "notes",
}

// Export записывает контакты в CSV-поток с заголовком.
func (c *CSVStorage) Export(w io.Writer, contacts []*domain.Contact) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(csvHeaders); err != nil {
		return fmt.Errorf("ошибка записи заголовка CSV: %w", err)
	}

	for _, item := range contacts {
		if item == nil {
			continue
		}
		record := []string{
			item.FirstName,
			item.LastName,
			item.MiddleName,
			item.Phone,
			item.WorkPhone,
			string(item.PhoneType),
			item.Email,
			item.Company,
			item.JobTitle,
			item.Address.Country,
			item.Address.City,
			item.Address.Street,
			item.Address.PostalCode,
			item.Birthday,
			strconv.FormatInt(item.GroupID, 10),
			strconv.FormatBool(item.IsFavorite),
			strings.Join(item.Tags, ";"),
			item.Notes,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("ошибка записи строки CSV: %w", err)
		}
	}

	writer.Flush()
	return writer.Error()
}

// Import читает CSV-файл, автоматически распознавая колонки по заголовку или порядку.
func (c *CSVStorage) Import(r io.Reader) ([]*domain.Contact, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения CSV: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}

	headerMap := make(map[string]int)
	startRow := 0
	for idx, col := range rows[0] {
		cleanCol := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(col, "\ufeff")))
		headerMap[cleanCol] = idx
	}

	if _, hasFirst := headerMap["first_name"]; hasFirst {
		startRow = 1
	} else {
		// Если заголовка нет, используем стандартный порядок
		for idx, h := range csvHeaders {
			headerMap[h] = idx
		}
	}

	getCol := func(row []string, name string) string {
		if idx, ok := headerMap[name]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	var result []*domain.Contact
	for i := startRow; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		groupID, _ := strconv.ParseInt(getCol(row, "group_id"), 10, 64)
		if groupID <= 0 {
			groupID = 2 // Работа по умолчанию при импорте
		}
		isFav, _ := strconv.ParseBool(getCol(row, "is_favorite"))

		var tags []string
		rawTags := getCol(row, "tags")
		if rawTags != "" {
			for _, t := range strings.Split(rawTags, ";") {
				if clean := strings.TrimSpace(t); clean != "" {
					tags = append(tags, clean)
				}
			}
		}

		contact := &domain.Contact{
			GroupID:    groupID,
			FirstName:  getCol(row, "first_name"),
			LastName:   getCol(row, "last_name"),
			MiddleName: getCol(row, "middle_name"),
			Phone:      getCol(row, "phone"),
			WorkPhone:  getCol(row, "work_phone"),
			PhoneType:  domain.PhoneType(getCol(row, "phone_type")),
			Email:      getCol(row, "email"),
			Company:    getCol(row, "company"),
			JobTitle:   getCol(row, "job_title"),
			Address: domain.Address{
				Country:    getCol(row, "country"),
				City:       getCol(row, "city"),
				Street:     getCol(row, "street"),
				PostalCode: getCol(row, "postal_code"),
			},
			Birthday:   getCol(row, "birthday"),
			Tags:       tags,
			Notes:      getCol(row, "notes"),
			IsFavorite: isFav,
		}
		result = append(result, contact)
	}

	return result, nil
}
