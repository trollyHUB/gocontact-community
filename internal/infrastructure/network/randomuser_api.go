// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package network

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// RandomUserClient — HTTP-клиент для получения контактов из внешнего REST API (Неделя 11: HTTP-клиент, JSON-запросы и ответы).
type RandomUserClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewRandomUserClient создаёт настроенный HTTP-клиент с таймаутом.
func NewRandomUserClient(baseURL string, timeout time.Duration) *RandomUserClient {
	if baseURL == "" {
		baseURL = "https://randomuser.me/api/"
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &RandomUserClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Структуры для десериализации JSON-ответа от REST API RandomUser.me
type randomUserResponse struct {
	Results []struct {
		Name struct {
			First string `json:"first"`
			Last  string `json:"last"`
		} `json:"name"`
		Location struct {
			Street struct {
				Number int    `json:"number"`
				Name   string `json:"name"`
			} `json:"street"`
			City     string `json:"city"`
			Country  string `json:"country"`
			Postcode any    `json:"postcode"`
		} `json:"location"`
		Email string `json:"email"`
		Phone string `json:"phone"`
		Cell  string `json:"cell"`
		Dob   struct {
			Date string `json:"date"`
		} `json:"dob"`
		Picture struct {
			Large string `json:"large"`
		} `json:"picture"`
	} `json:"results"`
}

// FetchContacts запрашивает count контактов по протоколу HTTP GET из внешнего REST API.
// При отсутствии интернета автоматически переключается на встроенный генератор (Graceful Fallback).
func (c *RandomUserClient) FetchContacts(ctx context.Context, count int) ([]*domain.Contact, string, error) {
	if count <= 0 {
		count = 5
	}
	if count > 50 {
		count = 50
	}

	reqURL := fmt.Sprintf("%s?results=%d&nat=us,gb,de,fr", c.baseURL, count)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err == nil {
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "GoContactStudio/1.0")

		resp, err := c.httpClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var parsed randomUserResponse
				if err := json.NewDecoder(resp.Body).Decode(&parsed); err == nil && len(parsed.Results) > 0 {
					contacts := make([]*domain.Contact, 0, len(parsed.Results))
					for i, item := range parsed.Results {
						birthday := ""
						if t, err := time.Parse(time.RFC3339, item.Dob.Date); err == nil {
							birthday = t.Format("02.01.2006")
						}
						phone := fmt.Sprintf("+7 (701) %03d-%02d-%02d", 100+rand.Intn(899), 10+rand.Intn(89), (10+i+rand.Intn(80))%100)
						street := fmt.Sprintf("%s %d", item.Location.Street.Name, item.Location.Street.Number)
						contacts = append(contacts, &domain.Contact{
							GroupID:   int64(2 + (i % 3)),
							FirstName: item.Name.First,
							LastName:  item.Name.Last,
							Phone:     phone,
							PhoneType: domain.PhoneMobile,
							Email:     item.Email,
							Company:   "Global Tech Partner",
							JobTitle:  "Software Engineer",
							Address: domain.Address{
								Country:    item.Location.Country,
								City:       item.Location.City,
								Street:     street,
								PostalCode: fmt.Sprintf("%v", item.Location.Postcode),
							},
							Birthday:  birthday,
							Tags:      []string{"api-import", "remote"},
							AvatarURL: item.Picture.Large,
							Notes:     "Импортировано по сети из внешнего REST API (RandomUser.me)",
						})
					}
					return contacts, "Внешний REST API (randomuser.me)", nil
				}
			}
		}
	}

	// Офлайн-фоллбэк, чтобы демонстрация работала даже без интернета
	return generateOfflineAPIContacts(count), "Локальный REST-эмулятор (офлайн-режим)", nil
}

func generateOfflineAPIContacts(count int) []*domain.Contact {
	firstNames := []string{"Аскар", "Бауржан", "Ержан", "Камила", "Мадина", "Руслан", "Жанна", "Олжас", "Динара", "Максим"}
	lastNames := []string{"Ибрагимов", "Сариев", "Касымов", "Алиева", "Нурланова", "Тастанов", "Жумагулова", "Калиев", "Досова", "Ковалёв"}
	cities := []string{"Алматы", "Астана", "Шымкент", "Караганда", "Актобе", "Атырау"}
	companies := []string{"Kaspi Lab", "BTS Digital", "ChocoFamily", "Kolesa Group", "SergeK Vision", "Astana Hub"}
	jobs := []string{"Go Backend Developer", "SRE Engineer", "System Architect", "QA Automation", "Data Analyst", "Team Lead"}

	result := make([]*domain.Contact, 0, count)
	for i := 0; i < count; i++ {
		fn := firstNames[(rand.Int()+i)%len(firstNames)]
		ln := lastNames[(rand.Int()+i*3)%len(lastNames)]
		phone := fmt.Sprintf("+7 (70%d) %03d-%02d-%02d", 1+(i%8), 100+rand.Intn(899), 10+rand.Intn(89), 10+rand.Intn(89))
		city := cities[i%len(cities)]
		comp := companies[i%len(companies)]
		job := jobs[i%len(jobs)]

		result = append(result, &domain.Contact{
			GroupID:   int64(1 + (i % 4)),
			FirstName: fn,
			LastName:  ln,
			Phone:     phone,
			PhoneType: domain.PhoneMobile,
			Email:     fmt.Sprintf("user%d.%d@tech.kz", time.Now().Unix()%1000, i+1),
			Company:   comp,
			JobTitle:  job,
			Address: domain.Address{
				Country:    "Казахстан",
				City:       city,
				Street:     fmt.Sprintf("пр. Республики, %d", 10+i*5),
				PostalCode: "010000",
			},
			Birthday:   fmt.Sprintf("%02d.%02d.199%d", 1+(i%28), 1+(i%12), i%9),
			Tags:       []string{"api", "golang", "kz"},
			IsFavorite: i%3 == 0,
			Notes:      "Сгенерировано сетевым модулем REST API",
		})
	}
	return result
}
