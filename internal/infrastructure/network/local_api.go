// SPDX-License-Identifier: AGPL-3.0-only OR LicenseRef-GoContact-Commercial
// Copyright (C) 2026 Tolegen Manassov (trollyHUB). See NOTICE.

package network

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/repository"
)

// LocalAPIServer — встроенный HTTP REST API сервер для доступа к адресной книге по сети.
type LocalAPIServer struct {
	port        int
	contactRepo repository.ContactRepository
	server      *http.Server
	addr        string
}

// NewLocalAPIServer создаёт новый экземпляр локального REST API сервера.
func NewLocalAPIServer(port int, repo repository.ContactRepository) *LocalAPIServer {
	if port <= 0 {
		port = 8089
	}
	return &LocalAPIServer{
		port:        port,
		contactRepo: repo,
		addr:        fmt.Sprintf("127.0.0.1:%d", port),
	}
}

// Start запускает HTTP-сервер в фоновой горутине.
func (s *LocalAPIServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/contacts", s.handleContacts)
	mux.HandleFunc("/api/v1/stats", s.handleStats)

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		// Если стандартный порт занят, выбираем свободный порт ОС
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return fmt.Errorf("не удалось запустить локальный REST API сервер: %w", err)
		}
		s.addr = ln.Addr().String()
	}

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		_ = s.server.Serve(ln)
	}()

	return nil
}

// Stop корректно останавливает HTTP-сервер.
func (s *LocalAPIServer) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// BaseURL возвращает полный базовый URL запущенного локального REST API.
func (s *LocalAPIServer) BaseURL() string {
	return "http://" + s.addr
}

func (s *LocalAPIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"service":   "GoContact Studio REST API",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (s *LocalAPIServer) handleContacts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		query := r.URL.Query().Get("q")
		contacts, err := s.contactRepo.List(ctx, domain.ContactFilter{Query: query})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"count":    len(contacts),
			"contacts": contacts,
		})

	case http.MethodPost:
		var c domain.Contact
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный JSON запрос: " + err.Error()})
			return
		}
		if err := s.contactRepo.Create(ctx, &c); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, c)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "метод не поддерживается"})
	}
}

func (s *LocalAPIServer) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.contactRepo.Stats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
}

// CreateContactViaHTTP отправляет JSON POST-запрос к REST API через http.Client (для демонстрации цепочки Клиент -> Сервер).
func CreateContactViaHTTP(ctx context.Context, baseURL string, contact *domain.Contact) (*domain.Contact, error) {
	bodyBytes, err := json.Marshal(contact)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/contacts", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("ошибка REST API (%d): %s", resp.StatusCode, errResp["error"])
	}

	var created domain.Contact
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, err
	}
	return &created, nil
}
