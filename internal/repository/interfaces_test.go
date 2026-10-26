// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package repository_test

import (
	"testing"

	"github.com/trollyHUB/gocontact-studio/internal/repository"
	"github.com/trollyHUB/gocontact-studio/internal/repository/sqlite"
)

// TestRepositoryInterfaces_CompileTimeCompliance проверяет соответствие реализаций интерфейсам репозиториев.
func TestRepositoryInterfaces_CompileTimeCompliance(t *testing.T) {
	var _ repository.ContactRepository = (*sqlite.ContactRepo)(nil)
	var _ repository.GroupRepository = (*sqlite.GroupRepo)(nil)
	var _ repository.GraphRepository = (*sqlite.GraphRepo)(nil)
	var _ repository.OrganizationRepository = (*sqlite.OrgRepo)(nil)
	var _ repository.OrganizationRepository = (*sqlite.GraphRepo)(nil)
}
