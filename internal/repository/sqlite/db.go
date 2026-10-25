// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // Чистый Go-драйвер SQLite (без CGO)

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// OpenAndMigrate открывает базу данных SQLite (или :memory:), настраивает PRAGMA и создаёт схему таблиц (Неделя 12).
func OpenAndMigrate(dbPath string) (*sql.DB, error) {
	var dsn string
	if dbPath == ":memory:" {
		dsn = "file::memory:?cache=shared&_pragma=foreign_keys(1)"
	} else {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return nil, fmt.Errorf("не удалось создать директорию для БД: %w", err)
		}
		dsn = fmt.Sprintf(
			"file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)",
			filepath.ToSlash(dbPath),
		)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия SQLite БД: %w", err)
	}

	db.SetMaxOpenConns(1) // Исключает блокировки записи SQLITE_BUSY в десктопном режиме
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ошибка подключения к SQLite: %w", err)
	}

	if err := runMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS groups (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT NOT NULL UNIQUE,
		color_hex   TEXT NOT NULL DEFAULT '#3B82F6',
		description TEXT NOT NULL DEFAULT '',
		created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS contacts (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id    INTEGER REFERENCES groups(id) ON DELETE SET NULL,
		first_name  TEXT NOT NULL,
		last_name   TEXT NOT NULL,
		middle_name TEXT NOT NULL DEFAULT '',
		phone       TEXT NOT NULL UNIQUE,
		work_phone  TEXT NOT NULL DEFAULT '',
		phone_type  TEXT NOT NULL DEFAULT 'mobile',
		email       TEXT NOT NULL DEFAULT '',
		company     TEXT NOT NULL DEFAULT '',
		job_title   TEXT NOT NULL DEFAULT '',
		street      TEXT NOT NULL DEFAULT '',
		city        TEXT NOT NULL DEFAULT '',
		country     TEXT NOT NULL DEFAULT '',
		postal_code TEXT NOT NULL DEFAULT '',
		birthday    TEXT NOT NULL DEFAULT '',
		tags_json   TEXT NOT NULL DEFAULT '[]',
		notes       TEXT NOT NULL DEFAULT '',
		is_favorite INTEGER NOT NULL DEFAULT 0,
		avatar_url  TEXT NOT NULL DEFAULT '',
		created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_contacts_name ON contacts(last_name, first_name);
	CREATE INDEX IF NOT EXISTS idx_contacts_phone ON contacts(phone);
	CREATE INDEX IF NOT EXISTS idx_contacts_email ON contacts(email);
	CREATE INDEX IF NOT EXISTS idx_contacts_group ON contacts(group_id);
	CREATE INDEX IF NOT EXISTS idx_contacts_favorite ON contacts(is_favorite);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("ошибка создания схемы таблиц: %w", err)
	}

	// Предзаполнение базовых групп из массива DefaultGroups()
	for _, g := range domain.DefaultGroups() {
		_, err := db.Exec(
			`INSERT OR IGNORE INTO groups (id, name, color_hex, description, created_at) VALUES (?, ?, ?, ?, ?)`,
			g.ID, g.Name, g.ColorHex, g.Description, g.CreatedAt.UTC().Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("ошибка инициализации базовых групп: %w", err)
		}
	}

	return nil
}

// SeedDemoContactsIfEmpty заполняет базу данных реалистичными контактами при самом первом запуске приложения.
func SeedDemoContactsIfEmpty(ctx context.Context, db *sql.DB) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM contacts`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	demoContacts := []domain.Contact{
		{
			GroupID: 2, FirstName: "Алексей", LastName: "Смирнов", MiddleName: "Игоревич",
			Phone: "+7 (701) 555-01-99", WorkPhone: "+7 (727) 250-10-20", PhoneType: domain.PhoneMobile,
			Email: "a.smirnov@techcorp.kz", Company: "TechCorp Kazakhstan", JobTitle: "Lead Go Developer",
			Address: domain.Address{Country: "Казахстан", City: "Алматы", Street: "пр. Достык, 120", PostalCode: "050000"},
			Birthday: "15.05.1994", Tags: []string{"golang", "backend", "teamlead"},
			Notes: "Ведущий архитектор микросервисов и наставник по языку Go", IsFavorite: true,
		},
		{
			GroupID: 2, FirstName: "Айгерим", LastName: "Омарова", MiddleName: "Сериковна",
			Phone: "+7 (707) 234-56-78", PhoneType: domain.PhoneWork,
			Email: "a.omarova@astanahub.kz", Company: "Astana Hub", JobTitle: "Product Manager",
			Address: domain.Address{Country: "Казахстан", City: "Астана", Street: "пр. Мангилик Ел, 55/8", PostalCode: "010000"},
			Birthday: "22.09.1996", Tags: []string{"product", "startup", "agile"},
			Notes: "Курирует акселерационную программу ИТ-проектов", IsFavorite: true,
		},
		{
			GroupID: 3, FirstName: "Данияр", LastName: "Ким", MiddleName: "Александрович",
			Phone: "+7 (702) 998-77-66", PhoneType: domain.PhoneMobile,
			Email: "d.kim@cloudops.io", Company: "CloudOps Studio", JobTitle: "DevOps Engineer",
			Address: domain.Address{Country: "Казахстан", City: "Алматы", Street: "ул. Тимирязева, 42", PostalCode: "050040"},
			Birthday: "03.11.1995", Tags: []string{"devops", "kubernetes", "docker"},
			Notes: "Вместе учились в университете, специалист по CI/CD", IsFavorite: false,
		},
		{
			GroupID: 1, FirstName: "Марат", LastName: "Манасов", MiddleName: "Ерланович",
			Phone: "+7 (777) 888-11-22", PhoneType: domain.PhoneMobile,
			Email: "m.manassov@family.kz", Company: "КазМунайГаз", JobTitle: "Главный инженер",
			Address: domain.Address{Country: "Казахстан", City: "Астана", Street: "ул. Кунаева, 14", PostalCode: "010000"},
			Birthday: "10.02.1972", Tags: []string{"семья", "родственники"},
			Notes: "Семейный контакт, звонить в любое время", IsFavorite: true,
		},
		{
			GroupID: 4, FirstName: "Елена", LastName: "Воронцова", MiddleName: "Дмитриевна",
			Phone: "+7 (705) 444-33-22", WorkPhone: "+7 (717) 299-00-11", PhoneType: domain.PhoneWork,
			Email: "e.vorontsova@fintech.kz", Company: "FinTech Bank", JobTitle: "CTO / Технический директор",
			Address: domain.Address{Country: "Казахстан", City: "Алматы", Street: "пр. Аль-Фараби, 77/7", PostalCode: "050060"},
			Birthday: "18.07.1988", Tags: []string{"vip", "fintech", "cto"},
			Notes: "Ключевой партнёр по банковским интеграциям и REST API", IsFavorite: true,
		},
		{
			GroupID: 3, FirstName: "Тимур", LastName: "Бектуров", MiddleName: "Нурланович",
			Phone: "+7 (708) 612-34-56", PhoneType: domain.PhoneMobile,
			Email: "timur.b@gamedev.kz", Company: "Nomad Games", JobTitle: "UI/UX Designer",
			Address: domain.Address{Country: "Казахстан", City: "Караганда", Street: "пр. Бухар-Жырау, 49", PostalCode: "100000"},
			Birthday: "29.12.1999", Tags: []string{"design", "fyne", "ui"},
			Notes: "Помогал с концептом цветовой схемы интерфейса", IsFavorite: false,
		},
	}

	nowStr := time.Now().UTC().Format(time.RFC3339)
	for _, c := range demoContacts {
		tagsBytes, _ := json.Marshal(c.Tags)
		fav := 0
		if c.IsFavorite {
			fav = 1
		}
		_, err := db.ExecContext(ctx, `
			INSERT OR IGNORE INTO contacts (
				group_id, first_name, last_name, middle_name, phone, work_phone,
				phone_type, email, company, job_title, street, city, country,
				postal_code, birthday, tags_json, notes, is_favorite, avatar_url,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			c.GroupID, c.FirstName, c.LastName, c.MiddleName, c.Phone, c.WorkPhone,
			string(c.PhoneType), c.Email, c.Company, c.JobTitle,
			c.Address.Street, c.Address.City, c.Address.Country, c.Address.PostalCode,
			c.Birthday, string(tagsBytes), c.Notes, fav, c.AvatarURL, nowStr, nowStr,
		)
		if err != nil {
			return fmt.Errorf("ошибка предзаполнения демо-контакта %s: %w", c.Phone, err)
		}
	}

	return nil
}
