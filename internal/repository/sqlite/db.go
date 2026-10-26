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

// OpenAndMigrate открывает базу данных SQLite (или :memory:), настраивает PRAGMA и создаёт схему таблиц.
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

	// Гарантируем включение foreign keys и оптимизации WAL
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ошибка включения foreign_keys: %w", err)
	}

	if err := runMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func hasColumn(db *sql.DB, tableName, colName string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == colName {
			return true, nil
		}
	}
	return false, rows.Err()
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

	-- Таблица организаций (узлы 2-го рода)
	CREATE TABLE IF NOT EXISTS organizations (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT    NOT NULL COLLATE NOCASE UNIQUE,
		industry    TEXT    NOT NULL DEFAULT '',
		domain      TEXT    NOT NULL DEFAULT '',
		city        TEXT    NOT NULL DEFAULT '',
		created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);

	-- Таблица рёбер социального графа
	CREATE TABLE IF NOT EXISTS contact_relationships (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		source_id    INTEGER NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
		target_id    INTEGER NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
		rel_type     TEXT    NOT NULL CHECK (rel_type IN (
						 'colleague', 'reports_to', 'founder_of',
						 'family', 'friend', 'mentor', 'introduced_by', 'partner', 'implicit_colleague'
					 )),
		weight       INTEGER NOT NULL DEFAULT 3 CHECK (weight BETWEEN 1 AND 5),
		is_directed  INTEGER NOT NULL DEFAULT 0 CHECK (is_directed IN (0, 1)),
		context_note TEXT    NOT NULL DEFAULT '',
		since_date   TEXT    NOT NULL DEFAULT '',
		created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		updated_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		CONSTRAINT chk_no_self_loop CHECK (source_id != target_id),
		CONSTRAINT uq_contact_edge UNIQUE (source_id, target_id, rel_type)
	);

	-- Таблица сохранённых координат и SNA-метрик
	CREATE TABLE IF NOT EXISTS graph_layouts (
		contact_id        INTEGER PRIMARY KEY REFERENCES contacts(id) ON DELETE CASCADE,
		pos_x             REAL    NOT NULL DEFAULT 0.0,
		pos_y             REAL    NOT NULL DEFAULT 0.0,
		is_pinned         INTEGER NOT NULL DEFAULT 0 CHECK (is_pinned IN (0, 1)),
		community_id      INTEGER NOT NULL DEFAULT 0,
		betweenness_score REAL    NOT NULL DEFAULT 0.0,
		updated_at        TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);

	-- Покрывающие индексы для графа и организаций
	CREATE INDEX IF NOT EXISTS idx_rel_source_target_weight
		ON contact_relationships(source_id, target_id, weight DESC, rel_type);

	CREATE INDEX IF NOT EXISTS idx_rel_target_source_weight
		ON contact_relationships(target_id, source_id, weight DESC, rel_type);

	CREATE INDEX IF NOT EXISTS idx_rel_type
		ON contact_relationships(rel_type);

	CREATE INDEX IF NOT EXISTS idx_graph_layouts_community
		ON graph_layouts(community_id);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("ошибка создания схемы таблиц: %w", err)
	}

	// Идемпотентное расширение таблицы contacts новыми колонками v0.2.0
	hasOrgID, err := hasColumn(db, "contacts", "org_id")
	if err != nil {
		return fmt.Errorf("ошибка проверки колонки contacts.org_id: %w", err)
	}
	if !hasOrgID {
		if _, err := db.Exec(`ALTER TABLE contacts ADD COLUMN org_id INTEGER REFERENCES organizations(id) ON DELETE SET NULL`); err != nil {
			return fmt.Errorf("ошибка добавления колонки org_id: %w", err)
		}
	}

	hasDunbar, err := hasColumn(db, "contacts", "dunbar_override")
	if err != nil {
		return fmt.Errorf("ошибка проверки колонки contacts.dunbar_override: %w", err)
	}
	if !hasDunbar {
		if _, err := db.Exec(`ALTER TABLE contacts ADD COLUMN dunbar_override INTEGER NOT NULL DEFAULT 0 CHECK (dunbar_override IN (0, 5, 15, 50, 150, 500))`); err != nil {
			return fmt.Errorf("ошибка добавления колонки dunbar_override: %w", err)
		}
	}

	// Индекс по org_id (создаётся после гарантированного добавления колонки org_id)
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_contacts_org_id ON contacts(org_id) WHERE org_id IS NOT NULL;`); err != nil {
		return fmt.Errorf("ошибка создания индекса idx_contacts_org_id: %w", err)
	}

	// Миграция данных v0.1.0 -> v0.2.0: нормализация текстовых компаний в организации
	_, _ = db.Exec(`
		INSERT OR IGNORE INTO organizations (name)
		SELECT DISTINCT TRIM(company) FROM contacts 
		WHERE TRIM(company) != '';

		UPDATE contacts SET org_id = (
			SELECT o.id FROM organizations o WHERE LOWER(o.name) = LOWER(TRIM(contacts.company))
		) WHERE (org_id IS NULL OR org_id = 0) AND TRIM(company) != '';
	`)

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
			Address:  domain.Address{Country: "Казахстан", City: "Алматы", Street: "пр. Достык, 120", PostalCode: "050000"},
			Birthday: "15.05.1994", Tags: []string{"golang", "backend", "teamlead"},
			Notes: "Ведущий архитектор микросервисов и наставник по языку Go", IsFavorite: true,
		},
		{
			GroupID: 2, FirstName: "Айгерим", LastName: "Омарова", MiddleName: "Сериковна",
			Phone: "+7 (707) 234-56-78", PhoneType: domain.PhoneWork,
			Email: "a.omarova@astanahub.kz", Company: "Astana Hub", JobTitle: "Product Manager",
			Address:  domain.Address{Country: "Казахстан", City: "Астана", Street: "пр. Мангилик Ел, 55/8", PostalCode: "010000"},
			Birthday: "22.09.1996", Tags: []string{"product", "startup", "agile"},
			Notes: "Курирует акселерационную программу ИТ-проектов", IsFavorite: true,
		},
		{
			GroupID: 3, FirstName: "Данияр", LastName: "Ким", MiddleName: "Александрович",
			Phone: "+7 (702) 998-77-66", PhoneType: domain.PhoneMobile,
			Email: "d.kim@cloudops.io", Company: "CloudOps Studio", JobTitle: "DevOps Engineer",
			Address:  domain.Address{Country: "Казахстан", City: "Алматы", Street: "ул. Тимирязева, 42", PostalCode: "050040"},
			Birthday: "03.11.1995", Tags: []string{"devops", "kubernetes", "docker"},
			Notes: "Вместе учились в университете, специалист по CI/CD", IsFavorite: false,
		},
		{
			GroupID: 1, FirstName: "Марат", LastName: "Манасов", MiddleName: "Ерланович",
			Phone: "+7 (777) 888-11-22", PhoneType: domain.PhoneMobile,
			Email: "m.manassov@family.kz", Company: "КазМунайГаз", JobTitle: "Главный инженер",
			Address:  domain.Address{Country: "Казахстан", City: "Астана", Street: "ул. Кунаева, 14", PostalCode: "010000"},
			Birthday: "10.02.1972", Tags: []string{"семья", "родственники"},
			Notes: "Семейный контакт, звонить в любое время", IsFavorite: true,
		},
		{
			GroupID: 4, FirstName: "Елена", LastName: "Воронцова", MiddleName: "Дмитриевна",
			Phone: "+7 (705) 444-33-22", WorkPhone: "+7 (717) 299-00-11", PhoneType: domain.PhoneWork,
			Email: "e.vorontsova@fintech.kz", Company: "FinTech Bank", JobTitle: "CTO / Технический директор",
			Address:  domain.Address{Country: "Казахстан", City: "Алматы", Street: "пр. Аль-Фараби, 77/7", PostalCode: "050060"},
			Birthday: "18.07.1988", Tags: []string{"vip", "fintech", "cto"},
			Notes: "Ключевой партнёр по банковским интеграциям и REST API", IsFavorite: true,
		},
		{
			GroupID: 3, FirstName: "Тимур", LastName: "Бектуров", MiddleName: "Нурланович",
			Phone: "+7 (708) 612-34-56", PhoneType: domain.PhoneMobile,
			Email: "timur.b@gamedev.kz", Company: "Nomad Games", JobTitle: "UI/UX Designer",
			Address:  domain.Address{Country: "Казахстан", City: "Караганда", Street: "пр. Бухар-Жырау, 49", PostalCode: "100000"},
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
