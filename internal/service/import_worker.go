// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package service

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// BatchImportSummary содержит итоговый отчёт параллельного импорта контактов.
type BatchImportSummary struct {
	TotalProcessed int
	SuccessCount   int
	DuplicateCount int
	InvalidCount   int
	ErrorMessages  []string
}

type importJob struct {
	index   int
	contact *domain.Contact
}

type importResult struct {
	contact     *domain.Contact
	isDuplicate bool
	err         error
}

// ImportBatchConcurrent параллельно валидирует, нормализует и сохраняет пачку контактов
// с использованием goroutines, channels, sync.WaitGroup и select.
func (s *ContactService) ImportBatchConcurrent(
	ctx context.Context,
	contacts []*domain.Contact,
	progressChan chan<- float64,
) *BatchImportSummary {
	summary := &BatchImportSummary{
		TotalProcessed: len(contacts),
	}
	if len(contacts) == 0 {
		if progressChan != nil {
			select {
			case progressChan <- 1.0:
			default:
			}
		}
		return summary
	}

	jobsChan := make(chan importJob, len(contacts))
	resultsChan := make(chan importResult, len(contacts))

	// Запускаем пул параллельных воркеров (Worker Pool) для валидации и нормализации контактов
	workerCount := s.workerCount
	if workerCount > len(contacts) {
		workerCount = len(contacts)
	}

	var wg sync.WaitGroup
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobsChan:
					if !ok {
						return
					}
					if job.contact == nil {
						resultsChan <- importResult{err: errors.New("пустая запись")}
						continue
					}
					// Валидируем и форматируем поля в параллельной горутине
					err := domain.NormalizeAndValidateContact(job.contact)
					resultsChan <- importResult{
						contact: job.contact,
						err:     err,
					}
				}
			}
		}()
	}

	// Отправляем задачи в канал jobsChan
	for idx, c := range contacts {
		jobsChan <- importJob{index: idx, contact: c}
	}
	close(jobsChan)

	// Закрываем канал результатов после завершения всех воркеров
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	processed := 0
	for res := range resultsChan {
		processed++
		if res.err != nil {
			summary.InvalidCount++
			if len(summary.ErrorMessages) < 10 {
				summary.ErrorMessages = append(summary.ErrorMessages, res.err.Error())
			}
		} else {
			if err := s.CreateContact(ctx, res.contact); err != nil {
				if errors.Is(err, domain.ErrDuplicatePhone) {
					summary.DuplicateCount++
				} else {
					summary.InvalidCount++
					if len(summary.ErrorMessages) < 10 {
						summary.ErrorMessages = append(summary.ErrorMessages, err.Error())
					}
				}
			} else {
				summary.SuccessCount++
			}
		}

		if progressChan != nil {
			progress := float64(processed) / float64(len(contacts))
			select {
			case progressChan <- progress:
			default:
			}
		}
	}

	return summary
}

// AuditIssue описывает замечание по качеству карточки контакта.
type AuditIssue struct {
	ContactID   int64
	ContactName string
	Phone       string
	Issue       string
}

// AuditReport содержит результаты многопоточного аудита адресной книги.
type AuditReport struct {
	TotalChecked        int
	HealthyContacts     int
	MissingEmailCount   int
	MissingAddressCount int
	MissingCompanyCount int
	Issues              []AuditIssue
}

// AuditContactsConcurrent выполняет параллельную проверку качества заполнения карточек контактов.
func (s *ContactService) AuditContactsConcurrent(ctx context.Context) (*AuditReport, error) {
	contacts, err := s.contactRepo.List(ctx, domain.ContactFilter{})
	if err != nil {
		return nil, err
	}

	report := &AuditReport{
		TotalChecked: len(contacts),
	}
	if len(contacts) == 0 {
		return report, nil
	}

	type contactAuditResult struct {
		missingEmail   bool
		missingAddress bool
		missingCompany bool
		issues         []AuditIssue
	}

	jobs := make(chan *domain.Contact, len(contacts))
	results := make(chan contactAuditResult, len(contacts))

	var wg sync.WaitGroup
	workers := s.workerCount
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range jobs {
				var r contactAuditResult
				if strings.TrimSpace(c.Email) == "" {
					r.missingEmail = true
					r.issues = append(r.issues, AuditIssue{
						ContactID:   c.ID,
						ContactName: c.FullName(),
						Phone:       c.Phone,
						Issue:       "Не указан Email",
					})
				}
				if c.Address.IsEmpty() {
					r.missingAddress = true
					r.issues = append(r.issues, AuditIssue{
						ContactID:   c.ID,
						ContactName: c.FullName(),
						Phone:       c.Phone,
						Issue:       "Не заполнен почтовый адрес",
					})
				}
				if strings.TrimSpace(c.Company) == "" {
					r.missingCompany = true
					r.issues = append(r.issues, AuditIssue{
						ContactID:   c.ID,
						ContactName: c.FullName(),
						Phone:       c.Phone,
						Issue:       "Не указана организация",
					})
				} else if (c.GroupID == 2 || c.GroupID == 4) && strings.TrimSpace(c.WorkPhone) == "" {
					r.issues = append(r.issues, AuditIssue{
						ContactID:   c.ID,
						ContactName: c.FullName(),
						Phone:       c.Phone,
						Issue:       "Рекомендуется добавить рабочий номер",
					})
				}
				results <- r
			}
		}()
	}

	for _, c := range contacts {
		jobs <- c
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		if !r.missingEmail && !r.missingAddress {
			report.HealthyContacts++
		}
		if r.missingEmail {
			report.MissingEmailCount++
		}
		if r.missingAddress {
			report.MissingAddressCount++
		}
		if r.missingCompany {
			report.MissingCompanyCount++
		}
		report.Issues = append(report.Issues, r.issues...)
	}

	return report, nil
}
