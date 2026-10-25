// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// handleImportFileDialog открывает диалог выбора файла (.csv / .json) для параллельного импорта.
func (u *AppUI) handleImportFileDialog() {
	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, u.mainWin)
			return
		}
		if reader == nil {
			return
		}
		filePath := reader.URI().Path()
		_ = reader.Close()

		u.runConcurrentFileImport(filePath)
	}, u.mainWin)

	fd.SetFilter(storage.NewExtensionFileFilter([]string{".csv", ".json", ".vcf"}))
	fd.Resize(fyne.NewSize(780, 520))
	fd.Show()
}

// runConcurrentFileImport запускает фоновый пул воркеров (Worker Pool) и обновляет ProgressBar через канал.
func (u *AppUI) runConcurrentFileImport(filePath string) {
	u.setStatus("Параллельный импорт файла: " + filepath.Base(filePath))
	u.progressBar.SetValue(0)
	u.progressBar.Show()

	progressChan := make(chan float64, 30)
	go func() {
		for p := range progressChan {
			val := p
			fyne.Do(func() {
				u.progressBar.SetValue(val)
			})
		}
	}()

	go func() {
		summary, err := u.service.ImportFromFileConcurrent(context.Background(), filePath, progressChan)
		close(progressChan)

		fyne.Do(func() {
			u.progressBar.Hide()
			if err != nil {
				dialog.ShowError(err, u.mainWin)
				return
			}
			u.reloadData()
			reportMsg := fmt.Sprintf(
				"Файл: %s\n\nВсего записей: %d\nДобавлено новых: %d\nДубликатов номера: %d\nОшибок валидации: %d",
				filepath.Base(filePath),
				summary.TotalProcessed,
				summary.SuccessCount,
				summary.DuplicateCount,
				summary.InvalidCount,
			)
			dialog.ShowInformation("Импорт завершён", reportMsg, u.mainWin)
			u.setStatus(fmt.Sprintf("Импортировано %d контактов из %s", summary.SuccessCount, filepath.Base(filePath)))
		})
	}()
}

// handleExportDialog открывает диалог сохранения контактов в форматах .json, .csv или .vcf.
func (u *AppUI) handleExportDialog(ext string) {
	defaultName := fmt.Sprintf("contacts_export_%s%s", time.Now().Format("20060102_1504"), ext)
	fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, u.mainWin)
			return
		}
		if writer == nil {
			return
		}
		filePath := writer.URI().Path()
		_ = writer.Close()

		count, err := u.service.ExportToFile(context.Background(), filePath, u.currentFilter)
		if err != nil {
			dialog.ShowError(err, u.mainWin)
			return
		}
		dialog.ShowInformation("Экспорт выполнен", fmt.Sprintf("Успешно экспортировано %d контактов в файл:\n%s", count, filePath), u.mainWin)
		u.setStatus(fmt.Sprintf("Экспортировано %d контактов в %s", count, filepath.Base(filePath)))
	}, u.mainWin)

	fd.SetFileName(defaultName)
	fd.Resize(fyne.NewSize(780, 520))
	fd.Show()
}

// handleExportSingleVCard сохраняет персональную электронную визитку vCard 3.0 (.vcf).
func (u *AppUI) handleExportSingleVCard(c *domain.Contact) {
	outPath := filepath.Join("data", "exports", fmt.Sprintf("%s_%s.vcf", c.LastName, c.FirstName))
	err := u.exportSingleContactVCard(outPath, c)
	if err != nil {
		dialog.ShowError(err, u.mainWin)
		return
	}
	dialog.ShowInformation("Визитка vCard создана", fmt.Sprintf("Электронная визитка контакта «%s» сохранена:\n%s", c.FullName(), outPath), u.mainWin)
	u.setStatus("Сохранена визитка " + filepath.Base(outPath))
}

func (u *AppUI) exportSingleContactVCard(path string, c *domain.Contact) error {
	_, err := u.service.ExportToFile(context.Background(), path, domain.ContactFilter{Query: c.Phone})
	return err
}

// handleFetchFromAPI загружает контакты из внешнего REST API в фоновой горутине и сохраняет через Worker Pool.
func (u *AppUI) handleFetchFromAPI(count int) {
	u.setStatus("Загрузка контактов из REST API в фоновом пуле горутин...")
	u.progressBar.SetValue(0)
	u.progressBar.Show()

	progressChan := make(chan float64, 20)
	go func() {
		for val := range progressChan {
			v := val
			fyne.Do(func() {
				u.progressBar.SetValue(v)
			})
		}
	}()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		summary, sourceName, err := u.service.FetchAndImportFromAPI(ctx, count, progressChan)
		close(progressChan)

		fyne.Do(func() {
			u.progressBar.Hide()
			if err != nil {
				dialog.ShowError(err, u.mainWin)
				return
			}
			u.reloadData()
			msg := fmt.Sprintf(
				"Источник: %s\n\nОбработано в Worker Pool: %d\nУспешно добавлено: %d\nПропущено дубликатов: %d",
				sourceName, summary.TotalProcessed, summary.SuccessCount, summary.DuplicateCount,
			)
			dialog.ShowInformation("Импорт из REST API завершён", msg, u.mainWin)
			u.setStatus(fmt.Sprintf("Загружено %d контактов из %s", summary.SuccessCount, sourceName))
		})
	}()
}
