-- SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
-- Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

BEGIN TRANSACTION;

-- 1. Журнал взаимодействий (встречи, звонки, сообщения, письма, подарки/интро)
CREATE TABLE IF NOT EXISTS interactions (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    channel          TEXT    NOT NULL CHECK (channel IN ('meeting', 'call', 'gift_intro', 'message', 'email')),
    title            TEXT    NOT NULL DEFAULT '',
    notes_md         TEXT    NOT NULL DEFAULT '',
    location         TEXT    NOT NULL DEFAULT '',
    occurred_at_unix INTEGER NOT NULL,
    duration_min     INTEGER NOT NULL DEFAULT 0 CHECK (duration_min >= 0),
    sentiment        INTEGER NOT NULL DEFAULT 0 CHECK (sentiment BETWEEN -1 AND 1),
    created_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- 2. Связующая таблица участников взаимодействия (поддержка групповых встреч M:N)
CREATE TABLE IF NOT EXISTS interaction_participants (
    interaction_id INTEGER NOT NULL REFERENCES interactions(id) ON DELETE CASCADE,
    contact_id     INTEGER NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    role           TEXT    NOT NULL DEFAULT 'participant' CHECK (role IN ('host', 'participant', 'mentioned')),
    PRIMARY KEY (interaction_id, contact_id)
) WITHOUT ROWID;

-- 3. Таблица вех жизни и памятных дат контакта
CREATE TABLE IF NOT EXISTS life_events (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    contact_id         INTEGER NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    event_type         TEXT    NOT NULL CHECK (event_type IN (
                           'birthday', 'anniversary', 'child_birth',
                           'promotion', 'relocation', 'gift_idea', 'custom'
                       )),
    title              TEXT    NOT NULL,
    event_date         TEXT    NOT NULL, -- Формат YYYY-MM-DD или --MM-DD
    is_recurring       INTEGER NOT NULL DEFAULT 1 CHECK (is_recurring IN (0, 1)),
    remind_days_before INTEGER NOT NULL DEFAULT 7 CHECK (remind_days_before BETWEEN 0 AND 90),
    notes              TEXT    NOT NULL DEFAULT '',
    created_at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- 4. Таблица взаимных обещаний и обязательств (Action Promises)
CREATE TABLE IF NOT EXISTS action_promises (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    contact_id            INTEGER NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    source_interaction_id INTEGER REFERENCES interactions(id) ON DELETE SET NULL,
    direction             TEXT    NOT NULL CHECK (direction IN ('i_promised', 'they_promised')),
    description           TEXT    NOT NULL,
    due_at_unix           INTEGER NOT NULL,
    status                TEXT    NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed', 'cancelled')),
    completed_at          TEXT    NOT NULL DEFAULT '',
    created_at            TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- 5. Политика частоты касаний и материализованный индекс здоровья связи (1:1 с contacts)
CREATE TABLE IF NOT EXISTS cadence_policies (
    contact_id            INTEGER PRIMARY KEY REFERENCES contacts(id) ON DELETE CASCADE,
    target_interval_days  INTEGER NOT NULL DEFAULT 0 CHECK (target_interval_days >= 0), -- 0 = авто по кругу Данбара
    health_score          INTEGER NOT NULL DEFAULT 50 CHECK (health_score BETWEEN 0 AND 100),
    health_status         TEXT    NOT NULL DEFAULT 'warm' CHECK (health_status IN ('thriving', 'warm', 'cooling', 'cold')),
    last_interaction_unix INTEGER NOT NULL DEFAULT 0,
    snoozed_until_unix    INTEGER NOT NULL DEFAULT 0,
    raw_energy            REAL    NOT NULL DEFAULT 0.5,
    updated_at            TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- 6. Индексы временных рядов и покрывающие индексы для Keep-in-Touch Radar и Dossier
CREATE INDEX IF NOT EXISTS idx_part_contact_interaction
    ON interaction_participants(contact_id, interaction_id DESC);

CREATE INDEX IF NOT EXISTS idx_interactions_occurred_channel
    ON interactions(occurred_at_unix DESC, channel);

CREATE INDEX IF NOT EXISTS idx_cadence_radar_priority
    ON cadence_policies(health_status, health_score ASC, snoozed_until_unix, last_interaction_unix ASC);

CREATE INDEX IF NOT EXISTS idx_promises_pending_due
    ON action_promises(status, due_at_unix ASC, contact_id)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_life_events_contact_date
    ON life_events(contact_id, event_date);

COMMIT;
