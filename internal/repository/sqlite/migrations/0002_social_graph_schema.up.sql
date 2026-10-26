-- SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
-- Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

-- Включение внешних ключей и оптимизация журнала WAL
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;

BEGIN TRANSACTION;

-- 1. Таблица нормализованных организаций (узлы 2-го рода / контекстные хабы)
CREATE TABLE IF NOT EXISTS organizations (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL COLLATE NOCASE UNIQUE,
    industry    TEXT    NOT NULL DEFAULT '',
    domain      TEXT    NOT NULL DEFAULT '',
    city        TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- 2. Добавление ссылки на организацию и ручного переопределения круга Данбара в contacts
ALTER TABLE contacts ADD COLUMN org_id INTEGER REFERENCES organizations(id) ON DELETE SET NULL;
ALTER TABLE contacts ADD COLUMN dunbar_override INTEGER NOT NULL DEFAULT 0 CHECK (dunbar_override IN (0, 5, 15, 50, 150, 500));

-- 3. Таблица типизированных рёбер социального графа
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

-- 4. Таблица сохранённых координат узлов и предрассчитанных SNA-метрик
CREATE TABLE IF NOT EXISTS graph_layouts (
    contact_id        INTEGER PRIMARY KEY REFERENCES contacts(id) ON DELETE CASCADE,
    pos_x             REAL    NOT NULL DEFAULT 0.0,
    pos_y             REAL    NOT NULL DEFAULT 0.0,
    is_pinned         INTEGER NOT NULL DEFAULT 0 CHECK (is_pinned IN (0, 1)),
    community_id      INTEGER NOT NULL DEFAULT 0,
    betweenness_score REAL    NOT NULL DEFAULT 0.0,
    updated_at        TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- 5. Покрывающие индексы для обхода графа за O(1) по смежности
CREATE INDEX IF NOT EXISTS idx_rel_source_target_weight
    ON contact_relationships(source_id, target_id, weight DESC, rel_type);

CREATE INDEX IF NOT EXISTS idx_rel_target_source_weight
    ON contact_relationships(target_id, source_id, weight DESC, rel_type);

CREATE INDEX IF NOT EXISTS idx_rel_type
    ON contact_relationships(rel_type);

CREATE INDEX IF NOT EXISTS idx_contacts_org_id
    ON contacts(org_id) WHERE org_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_graph_layouts_community
    ON graph_layouts(community_id);

COMMIT;
