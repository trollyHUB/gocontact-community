// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/trollyHUB/gocontact-studio/internal/service"
)

// showAuditReportDialog выполняет многопоточный аудит базы и выводит результат в таблицу widget.NewTable.
func (u *AppUI) showAuditReportDialog() {
	u.setStatus("Запуск параллельного аудита контактов в горутинах...")
	u.progressBar.Show()
	u.progressBar.SetValue(0.3)

	go func() {
		report, err := u.service.AuditContactsConcurrent(context.Background())
		fyne.Do(func() {
			u.progressBar.SetValue(1.0)
			u.progressBar.Hide()

			if err != nil {
				dialog.ShowError(err, u.mainWin)
				return
			}

			u.renderAuditModal(report)
			u.setStatus(fmt.Sprintf("Аудит завершён: проверено %d контактов, замечаний: %d", report.TotalChecked, len(report.Issues)))
		})
	}()
}

func (u *AppUI) renderAuditModal(report *service.AuditReport) {
	summaryLabel := widget.NewLabel(fmt.Sprintf(
		"Всего проверено карточек: %d  |  Полностью заполнены: %d  |  Без Email: %d  |  Без адреса: %d",
		report.TotalChecked, report.HealthyContacts, report.MissingEmailCount, report.MissingAddressCount,
	))
	summaryLabel.TextStyle = fyne.TextStyle{Bold: true}

	rowsData := report.Issues
	if len(rowsData) == 0 {
		rowsData = []service.AuditIssue{
			{
				ContactID:   0,
				ContactName: "Все карточки заполнены",
				Phone:       "—",
				Issue:       "Замечаний не обнаружено (100% качество)",
			},
		}
	}

	headers := []string{"ID", "ФИО Контакта", "Телефон", "Замечание аудитора"}
	table := widget.NewTable(
		func() (int, int) {
			return len(rowsData) + 1, 4
		},
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("Шаблон ячейки таблицы")
			lbl.Truncation = fyne.TextTruncateEllipsis
			return lbl
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			lbl := cell.(*widget.Label)
			if id.Row == 0 {
				lbl.TextStyle = fyne.TextStyle{Bold: true}
				lbl.SetText(headers[id.Col])
				return
			}
			lbl.TextStyle = fyne.TextStyle{}
			issue := rowsData[id.Row-1]
			switch id.Col {
			case 0:
				if issue.ContactID == 0 {
					lbl.SetText("OK")
				} else {
					lbl.SetText(fmt.Sprintf("#%d", issue.ContactID))
				}
			case 1:
				lbl.SetText(issue.ContactName)
			case 2:
				lbl.SetText(issue.Phone)
			case 3:
				lbl.SetText(issue.Issue)
			}
		},
	)
	table.SetColumnWidth(0, 50)
	table.SetColumnWidth(1, 240)
	table.SetColumnWidth(2, 165)
	table.SetColumnWidth(3, 330)

	content := container.NewBorder(
		container.NewVBox(summaryLabel, widget.NewSeparator()),
		nil, nil, nil,
		table,
	)

	dlg := dialog.NewCustom("Отчёт многопоточного аудита адресной книги (Goroutines)", "Закрыть", content, u.mainWin)
	dlg.Resize(fyne.NewSize(840, 480))
	dlg.Show()
}

// showLocalAPITesterDialog демонстрирует работу встроенного REST API сервера и HTTP-клиента.
func (u *AppUI) showLocalAPITesterDialog() {
	baseURL := u.apiServer.BaseURL()
	urlLabel := widget.NewLabel(fmt.Sprintf("Локальный сервер запущен на: %s", baseURL))
	urlLabel.TextStyle = fyne.TextStyle{Bold: true}

	responseBox := widget.NewMultiLineEntry()
	responseBox.SetMinRowsVisible(12)
	responseBox.SetText("Нажмите на одну из кнопок выше для выполнения HTTP-запроса...")

	fetchEndpoint := func(path string) {
		go func() {
			client := &http.Client{Timeout: 4 * time.Second}
			resp, err := client.Get(baseURL + path)
			if err != nil {
				fyne.Do(func() {
					responseBox.SetText("Ошибка HTTP-клиента: " + err.Error())
				})
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			fyne.Do(func() {
				responseBox.SetText(fmt.Sprintf("HTTP %d %s\nURL: %s%s\n\n%s",
					resp.StatusCode, resp.Status, baseURL, path, string(body)))
			})
		}()
	}

	btnHealth := widget.NewButtonWithIcon("GET /api/v1/health", theme.ConfirmIcon(), func() {
		fetchEndpoint("/api/v1/health")
	})
	btnStats := widget.NewButtonWithIcon("GET /api/v1/stats", theme.InfoIcon(), func() {
		fetchEndpoint("/api/v1/stats")
	})
	btnContacts := widget.NewButtonWithIcon("GET /api/v1/contacts", theme.FolderOpenIcon(), func() {
		fetchEndpoint("/api/v1/contacts")
	})

	fetchEndpoint("/api/v1/stats")

	topBar := container.NewVBox(
		urlLabel,
		container.NewHBox(btnHealth, btnStats, btnContacts),
		widget.NewSeparator(),
	)

	content := container.NewBorder(topBar, nil, nil, nil, responseBox)
	dlg := dialog.NewCustom("Панель тестирования REST API и HTTP-клиента", "Закрыть", content, u.mainWin)
	dlg.Resize(fyne.NewSize(720, 500))
	dlg.Show()
}

// showAboutDialog выводит информацию о программе и ключевых архитектурных возможностях.
func (u *AppUI) showAboutDialog() {
	info := `GoContact Studio Community Edition (v0.1.0)
Открытая платформа «Адресная книга» / Local-First Personal CRM.
Разработано на языке Go с графическим интерфейсом Fyne v2.

Архитектура и ключевые возможности:
• Чистая архитектура (Clean Architecture) и паттерн MVP (Model-View-Presenter).
• Встраиваемая реляционная база данных SQLite в высокопроизводительном режиме WAL.
• Потокобезопасный In-Memory кэш с защитой через sync.RWMutex.
• Пакетный многопоточный импорт через Worker Pool на горутинах и каналах.
• Экспорт и импорт промышленных форматов: JSON, CSV и визитные карточки vCard 3.0 (.vcf).
• Встроенный локальный HTTP REST API сервер (:8089) с поддержкой JSON-эндпоинтов.
• Автоматический планировщик резервного копирования базы данных (select + time.Ticker).
• Двухрежимная адаптивная тема оформления StudioTheme (Dark / Light).
• Лицензия: GNU AGPL v3 с официальным исключением для плагинов (Plugin Exception).

Copyright © 2026 Tolegen Manassov (trollyHUB).`

	dialog.ShowInformation("О программе GoContact Studio", info, u.mainWin)
}
