// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Константы валидации (Неделя 4).
const (
	MinPhoneDigits = 7
	MaxPhoneDigits = 15
	MaxNameLength  = 100
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// NormalizePhone очищает телефонный номер от лишних символов и приводит к аккуратному виду (Неделя 4: циклы, условия, switch).
func NormalizePhone(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", NewValidationError("Телефон", raw, "номер телефона обязателен для заполнения", ErrInvalidPhone)
	}

	var digits []rune
	for _, r := range trimmed {
		if unicode.IsDigit(r) {
			digits = append(digits, r)
		}
	}

	count := len(digits)
	if count < MinPhoneDigits || count > MaxPhoneDigits {
		return "", NewValidationError(
			"Телефон",
			raw,
			fmt.Sprintf("должен содержать от %d до %d цифр (получено %d)", MinPhoneDigits, MaxPhoneDigits, count),
			ErrInvalidPhone,
		)
	}

	// Если номер 11-значный и начинается с 8 или 7 (Казахстан / РФ), приводим к единому международному формату +7 (XXX) XXX-XX-XX
	if count == 11 && (digits[0] == '8' || digits[0] == '7') {
		s := string(digits)
		return fmt.Sprintf("+7 (%s) %s-%s-%s", s[1:4], s[4:7], s[7:9], s[9:11]), nil
	}

	// Если 10 цифр — предполагаем код +7
	if count == 10 {
		s := string(digits)
		return fmt.Sprintf("+7 (%s) %s-%s-%s", s[0:3], s[3:6], s[6:8], s[8:10]), nil
	}

	// Для остальных международных номеров сохраняем формат +<цифры>
	return "+" + string(digits), nil
}

// ValidateEmail проверяет корректность адреса электронной почты (пустой email допустим, если не указан).
func ValidateEmail(email string) error {
	clean := strings.TrimSpace(email)
	if clean == "" {
		return nil
	}
	if !emailRegex.MatchString(clean) {
		return NewValidationError("Email", email, "неверный формат электронной почты (пример: user@domain.com)", ErrInvalidEmail)
	}
	return nil
}

// ValidateBirthday проверяет формат даты рождения (ДД.ММ.ГГГГ или ГГГГ-ММ-ДД).
func ValidateBirthday(birthday string) error {
	clean := strings.TrimSpace(birthday)
	if clean == "" {
		return nil
	}
	layouts := []string{"02.01.2006", "2006-01-02"}
	for _, layout := range layouts {
		if _, err := time.Parse(layout, clean); err == nil {
			return nil
		}
	}
	return NewValidationError("День рождения", birthday, "используйте формат ДД.ММ.ГГГГ (например, 15.05.1998)", ErrInvalidBirthday)
}

// NormalizeAndValidateContact проверяет и нормализует все поля структуры Contact по указателю (Недели 4, 5, 8).
func NormalizeAndValidateContact(c *Contact) error {
	if c == nil {
		return NewValidationError("Contact", "", "объект контакта равен nil", ErrEmptyFirstName)
	}

	c.FirstName = capitalizeWords(strings.TrimSpace(c.FirstName))
	c.LastName = capitalizeWords(strings.TrimSpace(c.LastName))
	c.MiddleName = capitalizeWords(strings.TrimSpace(c.MiddleName))
	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	c.Company = strings.TrimSpace(c.Company)
	c.JobTitle = strings.TrimSpace(c.JobTitle)
	c.Notes = strings.TrimSpace(c.Notes)

	if c.FirstName == "" {
		return NewValidationError("Имя", "", "имя контакта обязательно для заполнения", ErrEmptyFirstName)
	}
	if len([]rune(c.FirstName)) > MaxNameLength {
		return NewValidationError("Имя", c.FirstName, "превышена максимальная длина имени", ErrEmptyFirstName)
	}

	if c.LastName == "" {
		return NewValidationError("Фамилия", "", "фамилия контакта обязательна для заполнения", ErrEmptyLastName)
	}
	if len([]rune(c.LastName)) > MaxNameLength {
		return NewValidationError("Фамилия", c.LastName, "превышена максимальная длина фамилии", ErrEmptyLastName)
	}

	normalizedPhone, err := NormalizePhone(c.Phone)
	if err != nil {
		return err
	}
	c.Phone = normalizedPhone

	if strings.TrimSpace(c.WorkPhone) != "" {
		normalizedWork, err := NormalizePhone(c.WorkPhone)
		if err != nil {
			return NewValidationError("Рабочий телефон", c.WorkPhone, "некорректный рабочий номер", ErrInvalidPhone)
		}
		c.WorkPhone = normalizedWork
	}

	if err := ValidateEmail(c.Email); err != nil {
		return err
	}

	if err := ValidateBirthday(c.Birthday); err != nil {
		return err
	}

	if c.PhoneType == "" {
		c.PhoneType = PhoneMobile
	}

	if c.GroupID <= 0 {
		c.GroupID = 5 // Группа "Прочее" по умолчанию
	}

	c.Touch()
	return nil
}

// capitalizeWords делает первую букву каждого слова заглавной.
func capitalizeWords(s string) string {
	if s == "" {
		return ""
	}
	words := strings.Fields(s)
	for i, w := range words {
		runes := []rune(w)
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}
