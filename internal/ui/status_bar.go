// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// buildStatusBar создаёт нижнюю строку состояния с индикатором прогресса фоновых горутин и кнопкой REST API.
func (u *AppUI) buildStatusBar() fyne.CanvasObject {
	u.statusLabel = widget.NewLabel("Готово к работе | СУБД: SQLite (WAL)")
	u.progressBar = widget.NewProgressBar()
	u.progressBar.Hide()

	apiBtn := widget.NewButtonWithIcon(fmt.Sprintf("REST API: %s", u.apiServer.BaseURL()), theme.HomeIcon(), func() {
		u.showLocalAPITesterDialog()
	})
	apiBtn.Importance = widget.LowImportance

	rightBox := container.NewHBox(u.progressBar, apiBtn)
	bar := container.NewBorder(nil, nil, u.statusLabel, rightBox)

	return container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(bar),
	)
}
