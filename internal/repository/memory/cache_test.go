// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package memory

import (
	"fmt"
	"sync"
	"testing"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

func TestContactCache_OperationsAndConcurrency(t *testing.T) {
	cache := NewContactCache()

	c1 := &domain.Contact{
		ID:        1,
		FirstName: "Алексей",
		LastName:  "Смирнов",
		Phone:     "+7 (701) 555-01-99",
		Tags:      []string{"go", "lead"},
	}

	cache.Set(c1)

	if cache.Size() != 1 {
		t.Fatalf("ожидался размер кэша 1, получено %d", cache.Size())
	}

	got, ok := cache.GetByID(1)
	if !ok || got.FullName() != "Смирнов Алексей" {
		t.Fatalf("контакт не найден в кэше: %+v", got)
	}

	if id, exists := cache.ExistsPhone("+7 (701) 555-01-99"); !exists || id != 1 {
		t.Errorf("индекс телефона вернул неверный результат: id=%d, exists=%v", id, exists)
	}

	// Проверка потокобезопасности при параллельном чтении и записи из 20 горутин
	var wg sync.WaitGroup
	for i := 2; i <= 20; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			cache.Set(&domain.Contact{
				ID:        id,
				FirstName: "Тест",
				LastName:  "Параллельный",
				Phone:     fmt.Sprintf("+7 (701) 000-00-%02d", id),
			})
			_, _ = cache.GetByID(1)
			_ = cache.Size()
		}(int64(i))
	}
	wg.Wait()

	if cache.Size() != 20 {
		t.Errorf("ожидалось 20 элементов после конкурентной записи, получено %d", cache.Size())
	}

	cache.Delete(1)
	if _, ok := cache.GetByID(1); ok {
		t.Errorf("контакт ID=1 должен быть удалён из кэша")
	}
}
