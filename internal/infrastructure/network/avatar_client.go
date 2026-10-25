// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package network

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
)

// AvatarClient формирует и проверяет URL аватаров через внешние сервисы Gravatar / UI-Avatars (FR-5.1).
type AvatarClient struct {
	httpClient *http.Client
}

// NewAvatarClient создаёт HTTP-клиент для работы с аватарами контактов.
func NewAvatarClient(timeout time.Duration) *AvatarClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &AvatarClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

// GenerateAvatarURL формирует детерминированный URL аватара по Email (Gravatar MD5) или по инициалам контакта (UI-Avatars API).
func GenerateAvatarURL(c *domain.Contact) string {
	if c == nil {
		return ""
	}
	if strings.TrimSpace(c.AvatarURL) != "" {
		return strings.TrimSpace(c.AvatarURL)
	}

	email := strings.ToLower(strings.TrimSpace(c.Email))
	if email != "" {
		hash := md5.Sum([]byte(email))
		hashHex := hex.EncodeToString(hash[:])
		return fmt.Sprintf("https://www.gravatar.com/avatar/%s?d=identicon&s=128", hashHex)
	}

	initials := url.QueryEscape(c.Initials())
	return fmt.Sprintf("https://ui-avatars.com/api/?name=%s&background=3B82F6&color=fff&size=128", initials)
}

// CheckAvatarReachable проверяет доступность аватара по HTTP HEAD-запросу.
func (a *AvatarClient) CheckAvatarReachable(ctx context.Context, avatarURL string) bool {
	if strings.TrimSpace(avatarURL) == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, avatarURL, nil)
	if err != nil {
		return false
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
