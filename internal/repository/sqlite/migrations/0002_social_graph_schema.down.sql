-- SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
-- Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

BEGIN TRANSACTION;

DROP INDEX IF EXISTS idx_graph_layouts_community;
DROP INDEX IF EXISTS idx_contacts_org_id;
DROP INDEX IF EXISTS idx_rel_type;
DROP INDEX IF EXISTS idx_rel_target_source_weight;
DROP INDEX IF EXISTS idx_rel_source_target_weight;

DROP TABLE IF EXISTS graph_layouts;
DROP TABLE IF EXISTS contact_relationships;

ALTER TABLE contacts DROP COLUMN dunbar_override;
ALTER TABLE contacts DROP COLUMN org_id;

DROP TABLE IF EXISTS organizations;

COMMIT;
