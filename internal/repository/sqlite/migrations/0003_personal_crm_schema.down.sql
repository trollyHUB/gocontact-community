-- SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
-- Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

BEGIN TRANSACTION;

DROP INDEX IF EXISTS idx_life_events_contact_date;
DROP INDEX IF EXISTS idx_promises_pending_due;
DROP INDEX IF EXISTS idx_cadence_radar_priority;
DROP INDEX IF EXISTS idx_interactions_occurred_channel;
DROP INDEX IF EXISTS idx_part_contact_interaction;

DROP TABLE IF EXISTS cadence_policies;
DROP TABLE IF EXISTS action_promises;
DROP TABLE IF EXISTS life_events;
DROP TABLE IF EXISTS interaction_participants;
DROP TABLE IF EXISTS interactions;

COMMIT;
