// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package ui

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

func labeledField(label string, field fyne.CanvasObject) fyne.CanvasObject {
	lbl := widget.NewLabelWithStyle(label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.NewVBox(lbl, field)
}

// showContactFormDialog открывает прокручиваемое модальное окно создания или редактирования контакта (Неделя 9).
// Использование двухколоночной сетки и container.NewVScroll гарантирует отсутствие наложения кнопок в оконном режиме.
func (u *AppUI) showContactFormDialog(existing *domain.Contact) {
	isEdit := existing != nil
	title := "Новый контакт"
	if isEdit {
		title = "Редактирование: " + existing.FullName()
	}

	firstNameEntry := widget.NewEntry()
	firstNameEntry.SetPlaceHolder("Например: Алексей")

	lastNameEntry := widget.NewEntry()
	lastNameEntry.SetPlaceHolder("Например: Смирнов")

	middleNameEntry := widget.NewEntry()
	middleNameEntry.SetPlaceHolder("Например: Игоревич")

	phoneEntry := widget.NewEntry()
	phoneEntry.SetPlaceHolder("+7 (701) 555-01-99")

	workPhoneEntry := widget.NewEntry()
	workPhoneEntry.SetPlaceHolder("Рабочий / доп. номер")

	phoneTypeOptions := []string{"Мобильный", "Рабочий", "Домашний", "Другой"}
	phoneTypeSelect := &widget.Select{
		Options:     phoneTypeOptions,
		Selected:    "Мобильный",
		PlaceHolder: "Тип телефона",
	}
	phoneTypeSelect.ExtendBaseWidget(phoneTypeSelect)

	emailEntry := widget.NewEntry()
	emailEntry.SetPlaceHolder("user@company.kz")

	companyEntry := widget.NewEntry()
	companyEntry.SetPlaceHolder("Название организации")

	jobTitleEntry := widget.NewEntry()
	jobTitleEntry.SetPlaceHolder("Должность / Роль")

	countryEntry := widget.NewEntry()
	countryEntry.SetPlaceHolder("Казахстан")

	cityEntry := widget.NewEntry()
	cityEntry.SetPlaceHolder("Алматы / Астана")

	streetEntry := widget.NewEntry()
	streetEntry.SetPlaceHolder("Улица, дом, офис")

	postalEntry := widget.NewEntry()
	postalEntry.SetPlaceHolder("050000")

	birthdayEntry := widget.NewEntry()
	birthdayEntry.SetPlaceHolder("ДД.ММ.ГГГГ (15.05.1995)")

	tagsEntry := widget.NewEntry()
	tagsEntry.SetPlaceHolder("golang, backend, vip")

	notesEntry := widget.NewMultiLineEntry()
	notesEntry.SetPlaceHolder("Дополнительные заметки о контакте...")
	notesEntry.SetMinRowsVisible(2)

	favoriteCheck := widget.NewCheck("★ В избранные контакты", nil)

	// Список групп
	groupNames := make([]string, 0, len(u.groups))
	groupMap := make(map[string]int64)
	selectedGroupName := ""

	for _, g := range u.groups {
		groupNames = append(groupNames, g.Name)
		groupMap[g.Name] = g.ID
		if existing != nil && existing.GroupID == g.ID {
			selectedGroupName = g.Name
		}
	}
	if selectedGroupName == "" && len(groupNames) > 0 {
		selectedGroupName = groupNames[0]
	}
	groupSelect := &widget.Select{
		Options:     groupNames,
		Selected:    selectedGroupName,
		PlaceHolder: "Выберите категорию",
	}
	groupSelect.ExtendBaseWidget(groupSelect)

	if isEdit {
		firstNameEntry.SetText(existing.FirstName)
		lastNameEntry.SetText(existing.LastName)
		middleNameEntry.SetText(existing.MiddleName)
		phoneEntry.SetText(existing.Phone)
		workPhoneEntry.SetText(existing.WorkPhone)
		phoneTypeSelect.SetSelected(existing.PhoneType.Label())
		emailEntry.SetText(existing.Email)
		companyEntry.SetText(existing.Company)
		jobTitleEntry.SetText(existing.JobTitle)
		countryEntry.SetText(existing.Address.Country)
		cityEntry.SetText(existing.Address.City)
		streetEntry.SetText(existing.Address.Street)
		postalEntry.SetText(existing.Address.PostalCode)
		birthdayEntry.SetText(existing.Birthday)
		tagsEntry.SetText(strings.Join(existing.Tags, ", "))
		notesEntry.SetText(existing.Notes)
		favoriteCheck.SetChecked(existing.IsFavorite)
	} else {
		countryEntry.SetText("Казахстан")
	}

	// Трёхколоночная компактная сетка полей (все 18 полей помещаются на одном экране без наложения на кнопки!)
	threeColGrid := container.NewGridWithColumns(3,
		labeledField("Фамилия *", lastNameEntry),
		labeledField("Имя *", firstNameEntry),
		labeledField("Отчество", middleNameEntry),

		labeledField("Категория / Группа", groupSelect),
		labeledField("Основной телефон *", phoneEntry),
		labeledField("Тип телефона", phoneTypeSelect),

		labeledField("Рабочий телефон", workPhoneEntry),
		labeledField("Электронная почта", emailEntry),
		labeledField("День рождения", birthdayEntry),

		labeledField("Организация", companyEntry),
		labeledField("Должность", jobTitleEntry),
		labeledField("Теги (через запятую)", tagsEntry),

		labeledField("Страна", countryEntry),
		labeledField("Город", cityEntry),
		labeledField("Почтовый индекс", postalEntry),
	)

	streetAndFavGrid := container.NewGridWithColumns(2,
		labeledField("Улица, дом, офис", streetEntry),
		labeledField("Статус контакта", favoriteCheck),
	)

	formBody := container.NewPadded(container.NewVBox(
		threeColGrid,
		streetAndFavGrid,
		labeledField("Заметки о контакте", notesEntry),
	))

	dlg := dialog.NewCustomConfirm(title, "Сохранить", "Отмена", formBody, func(confirm bool) {
		if !confirm {
			return
		}

		pt := domain.PhoneMobile
		switch phoneTypeSelect.Selected {
		case "Рабочий":
			pt = domain.PhoneWork
		case "Домашний":
			pt = domain.PhoneHome
		case "Другой":
			pt = domain.PhoneOther
		}

		var tags []string
		for _, rawTag := range strings.Split(tagsEntry.Text, ",") {
			clean := strings.TrimSpace(strings.TrimPrefix(rawTag, "#"))
			if clean != "" {
				tags = append(tags, strings.ToLower(clean))
			}
		}

		groupID := groupMap[groupSelect.Selected]
		if groupID <= 0 {
			groupID = 5
		}

		contact := &domain.Contact{
			GroupID:    groupID,
			FirstName:  firstNameEntry.Text,
			LastName:   lastNameEntry.Text,
			MiddleName: middleNameEntry.Text,
			Phone:      phoneEntry.Text,
			WorkPhone:  workPhoneEntry.Text,
			PhoneType:  pt,
			Email:      emailEntry.Text,
			Company:    companyEntry.Text,
			JobTitle:   jobTitleEntry.Text,
			Address: domain.Address{
				Country:    countryEntry.Text,
				City:       cityEntry.Text,
				Street:     streetEntry.Text,
				PostalCode: postalEntry.Text,
			},
			Birthday:   birthdayEntry.Text,
			Tags:       tags,
			Notes:      notesEntry.Text,
			IsFavorite: favoriteCheck.Checked,
		}

		ctx := context.Background()
		var err error
		if isEdit {
			contact.ID = existing.ID
			contact.CreatedAt = existing.CreatedAt
			err = u.service.UpdateContact(ctx, contact)
		} else {
			err = u.service.CreateContact(ctx, contact)
		}

		if err != nil {
			dialog.ShowError(err, u.mainWin)
			return
		}

		u.selectedContact = contact
		u.reloadData()
		u.setStatus(fmt.Sprintf("Контакт «%s» успешно сохранён", contact.FullName()))
	}, u.mainWin)

	dlg.Resize(fyne.NewSize(780, 560))
	dlg.Show()
}

// showNewGroupDialog открывает диалог создания новой группы/категории контактов.
func (u *AppUI) showNewGroupDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Например: Университет / Клиенты")

	colorSelect := &widget.Select{
		Options:     []string{"#3B82F6", "#EF4444", "#10B981", "#F59E0B", "#6B7280"},
		Selected:    "#3B82F6",
		PlaceHolder: "Выберите цвет",
	}
	colorSelect.ExtendBaseWidget(colorSelect)

	descEntry := widget.NewEntry()
	descEntry.SetPlaceHolder("Краткое описание категории")

	items := []*widget.FormItem{
		widget.NewFormItem("Название группы *", nameEntry),
		widget.NewFormItem("Цвет маркера", colorSelect),
		widget.NewFormItem("Описание", descEntry),
	}

	dlg := dialog.NewForm("Новая категория контактов", "Создать", "Отмена", items, func(ok bool) {
		if !ok {
			return
		}
		g, err := u.service.CreateGroup(context.Background(), nameEntry.Text, colorSelect.Selected, descEntry.Text)
		if err != nil {
			dialog.ShowError(err, u.mainWin)
			return
		}
		u.reloadData()
		u.setStatus(fmt.Sprintf("Создана новая категория «%s»", g.Name))
	}, u.mainWin)

	dlg.Resize(fyne.NewSize(460, 280))
	dlg.Show()
}

// showEditSelectedGroupDialog открывает диалог переименования и редактирования выбранной группы (FR-2.2).
func (u *AppUI) showEditSelectedGroupDialog() {
	var selected *domain.Group
	for _, g := range u.groups {
		if g.ID == u.currentFilter.GroupID {
			selected = g
			break
		}
	}
	if selected == nil {
		dialog.ShowInformation("Выберите категорию", "Сначала выберите категорию в левой панели для редактирования.", u.mainWin)
		return
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetText(selected.Name)

	colorSelect := &widget.Select{
		Options:     []string{"#3B82F6", "#EF4444", "#10B981", "#F59E0B", "#6B7280"},
		Selected:    selected.ColorHex,
		PlaceHolder: "Выберите цвет",
	}
	colorSelect.ExtendBaseWidget(colorSelect)

	descEntry := widget.NewEntry()
	descEntry.SetText(selected.Description)

	items := []*widget.FormItem{
		widget.NewFormItem("Название группы *", nameEntry),
		widget.NewFormItem("Цвет маркера", colorSelect),
		widget.NewFormItem("Описание", descEntry),
	}

	dlg := dialog.NewForm("Редактирование категории", "Сохранить", "Отмена", items, func(ok bool) {
		if !ok {
			return
		}
		updated := &domain.Group{
			ID:          selected.ID,
			Name:        nameEntry.Text,
			ColorHex:    colorSelect.Selected,
			Description: descEntry.Text,
		}
		if err := u.service.UpdateGroup(context.Background(), updated); err != nil {
			dialog.ShowError(err, u.mainWin)
			return
		}
		u.reloadData()
		u.setStatus(fmt.Sprintf("Категория «%s» обновлена", updated.Name))
	}, u.mainWin)

	dlg.Resize(fyne.NewSize(460, 280))
	dlg.Show()
}

// handleDeleteSelectedGroup удаляет выбранную пользовательскую категорию с подтверждением (FR-2.2).
func (u *AppUI) handleDeleteSelectedGroup() {
	var selected *domain.Group
	for _, g := range u.groups {
		if g.ID == u.currentFilter.GroupID {
			selected = g
			break
		}
	}
	if selected == nil {
		dialog.ShowInformation("Выберите категорию", "Сначала выберите категорию в левой панели для удаления.", u.mainWin)
		return
	}

	dialog.ShowConfirm(
		"Удаление категории",
		fmt.Sprintf("Удалить категорию «%s»? Контакты внутри неё останутся в базе без группы.", selected.Name),
		func(ok bool) {
			if !ok {
				return
			}
			if err := u.service.DeleteGroup(context.Background(), selected.ID); err != nil {
				dialog.ShowError(err, u.mainWin)
				return
			}
			u.currentFilter.GroupID = 0
			u.groupList.UnselectAll()
			u.reloadData()
			u.setStatus(fmt.Sprintf("Категория «%s» удалена", selected.Name))
		},
		u.mainWin,
	)
}
