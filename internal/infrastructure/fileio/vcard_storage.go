// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package fileio

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// VCardStorage реализует интерфейсы Exporter и Importer для международного стандарта электронных визиток vCard 3.0 (.vcf).
type VCardStorage struct{}

func NewVCardStorage() *VCardStorage {
	return &VCardStorage{}
}

func (v *VCardStorage) FormatName() string    { return "vCard 3.0 (Электронная визитка .vcf)" }
func (v *VCardStorage) FileExtension() string { return ".vcf" }

// Import разбирает поток карточек vCard 3.0/4.0 (.vcf) и возвращает список контактов.
func (v *VCardStorage) Import(r io.Reader) ([]*domain.Contact, error) {
	scanner := bufio.NewScanner(r)
	var contacts []*domain.Contact
	var current *domain.Contact

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		upper := strings.ToUpper(line)
		if upper == "BEGIN:VCARD" {
			current = &domain.Contact{
				PhoneType: domain.PhoneMobile,
				GroupID:   domain.GroupFriends,
			}
			continue
		}
		if upper == "END:VCARD" {
			if current != nil {
				if current.FirstName == "" && current.LastName != "" {
					current.FirstName = current.LastName
					current.LastName = ""
				}
				contacts = append(contacts, current)
				current = nil
			}
			continue
		}
		if current == nil {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		keyPart := strings.ToUpper(parts[0])
		val := unescapeVCard(parts[1])

		switch {
		case keyPart == "N" || strings.HasPrefix(keyPart, "N;"):
			nameFields := strings.Split(parts[1], ";")
			if len(nameFields) > 0 {
				current.LastName = unescapeVCard(nameFields[0])
			}
			if len(nameFields) > 1 {
				current.FirstName = unescapeVCard(nameFields[1])
			}
			if len(nameFields) > 2 {
				current.MiddleName = unescapeVCard(nameFields[2])
			}
		case (keyPart == "FN" || strings.HasPrefix(keyPart, "FN;")) && current.FirstName == "" && current.LastName == "":
			tokens := strings.Fields(val)
			if len(tokens) == 1 {
				current.FirstName = tokens[0]
			} else if len(tokens) >= 2 {
				current.LastName = tokens[0]
				current.FirstName = tokens[1]
				if len(tokens) >= 3 {
					current.MiddleName = tokens[2]
				}
			}
		case keyPart == "ORG" || strings.HasPrefix(keyPart, "ORG;"):
			current.Company = val
			current.GroupID = domain.GroupWork
		case keyPart == "TITLE" || strings.HasPrefix(keyPart, "TITLE;"):
			current.JobTitle = val
		case strings.HasPrefix(keyPart, "TEL"):
			if strings.Contains(keyPart, "WORK") && current.Phone != "" {
				current.WorkPhone = val
			} else if current.Phone == "" {
				current.Phone = val
			} else if current.WorkPhone == "" {
				current.WorkPhone = val
			}
		case strings.HasPrefix(keyPart, "EMAIL"):
			current.Email = val
		case strings.HasPrefix(keyPart, "ADR"):
			adrParts := strings.Split(parts[1], ";")
			if len(adrParts) > 2 {
				current.Address.Street = unescapeVCard(adrParts[2])
			}
			if len(adrParts) > 3 {
				current.Address.City = unescapeVCard(adrParts[3])
			}
			if len(adrParts) > 5 {
				current.Address.PostalCode = unescapeVCard(adrParts[5])
			}
			if len(adrParts) > 6 {
				current.Address.Country = unescapeVCard(adrParts[6])
			}
		case keyPart == "NOTE" || strings.HasPrefix(keyPart, "NOTE;"):
			current.Notes = val
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ошибка чтения vCard потока: %w", err)
	}
	return contacts, nil
}

// Export генерирует карточки контактов в стандарте RFC 2426 (vCard 3.0).
func (v *VCardStorage) Export(w io.Writer, contacts []*domain.Contact) error {
	var sb strings.Builder
	rev := time.Now().UTC().Format("20060102T150405Z")

	for _, c := range contacts {
		if c == nil {
			continue
		}
		sb.WriteString("BEGIN:VCARD\r\n")
		sb.WriteString("VERSION:3.0\r\n")
		sb.WriteString(fmt.Sprintf("N:%s;%s;%s;;\r\n", escapeVCard(c.LastName), escapeVCard(c.FirstName), escapeVCard(c.MiddleName)))
		sb.WriteString(fmt.Sprintf("FN:%s\r\n", escapeVCard(c.FullName())))
		if c.Company != "" {
			sb.WriteString(fmt.Sprintf("ORG:%s\r\n", escapeVCard(c.Company)))
		}
		if c.JobTitle != "" {
			sb.WriteString(fmt.Sprintf("TITLE:%s\r\n", escapeVCard(c.JobTitle)))
		}
		sb.WriteString(fmt.Sprintf("TEL;TYPE=CELL,VOICE:%s\r\n", escapeVCard(c.Phone)))
		if c.WorkPhone != "" {
			sb.WriteString(fmt.Sprintf("TEL;TYPE=WORK,VOICE:%s\r\n", escapeVCard(c.WorkPhone)))
		}
		if c.Email != "" {
			sb.WriteString(fmt.Sprintf("EMAIL;TYPE=INTERNET:%s\r\n", escapeVCard(c.Email)))
		}
		if !c.Address.IsEmpty() {
			sb.WriteString(fmt.Sprintf("ADR;TYPE=HOME:;;%s;%s;;%s;%s\r\n",
				escapeVCard(c.Address.Street),
				escapeVCard(c.Address.City),
				escapeVCard(c.Address.PostalCode),
				escapeVCard(c.Address.Country),
			))
		}
		if c.Notes != "" {
			sb.WriteString(fmt.Sprintf("NOTE:%s\r\n", escapeVCard(c.Notes)))
		}
		sb.WriteString(fmt.Sprintf("REV:%s\r\n", rev))
		sb.WriteString("END:VCARD\r\n")
	}

	_, err := io.WriteString(w, sb.String())
	return err
}

func escapeVCard(s string) string {
	r := strings.NewReplacer("\n", "\\n", "\r", "", ";", "\\;", ",", "\\,")
	return r.Replace(strings.TrimSpace(s))
}

func unescapeVCard(s string) string {
	r := strings.NewReplacer("\\n", "\n", "\\;", ";", "\\,", ",")
	return strings.TrimSpace(r.Replace(s))
}
