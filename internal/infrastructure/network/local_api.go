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
	"strconv"
	"time"

	"github.com/trollyHUB/gocontact-studio/internal/domain"
	"github.com/trollyHUB/gocontact-studio/internal/domain/graph"
	"github.com/trollyHUB/gocontact-studio/internal/repository"
	"github.com/trollyHUB/gocontact-studio/internal/service/sna"
)

// LocalAPIServer — встроенный HTTP REST API сервер для доступа к контактам и социальному графу.
type LocalAPIServer struct {
	port         int
	contactRepo  repository.ContactRepository
	graphService sna.GraphService
	server       *http.Server
	addr         string
}

// NewLocalAPIServer создаёт новый экземпляр локального REST API сервера с поддержкой GraphService.
func NewLocalAPIServer(port int, repo repository.ContactRepository, graphSvc sna.GraphService) *LocalAPIServer {
	if port <= 0 {
		port = 8089
	}
	return &LocalAPIServer{
		port:         port,
		contactRepo:  repo,
		graphService: graphSvc,
		addr:         fmt.Sprintf("127.0.0.1:%d", port),
	}
}

// Start запускает HTTP-сервер в фоновой горутине.
func (s *LocalAPIServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/contacts", s.handleContacts)
	mux.HandleFunc("/api/v1/stats", s.handleStats)

	// Новые эндпоинты релиза v0.2.0 (Спринт 2.4)
	mux.HandleFunc("/api/v1/graph", s.handleGraph)
	mux.HandleFunc("/api/v1/graph/shortest-path", s.handleShortestPath)

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
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
		"version":   "v0.2.0",
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

// handleGraph возвращает полную топологию социального графа (узлы, координаты, рёбра и метрики).
func (s *LocalAPIServer) handleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "метод не поддерживается"})
		return
	}

	if s.graphService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "сервис социального графа недоступен"})
		return
	}

	includeImplicit := r.URL.Query().Get("include_implicit") == "true"
	dunbarMax, _ := strconv.Atoi(r.URL.Query().Get("dunbar_max"))

	nodesMap, edges, _ := s.graphService.GetTopologySnapshot()
	snap := s.graphService.GetLatestSnapshot()

	communities := make(map[int]bool)
	var filteredNodes []*graph.Node
	for id, n := range nodesMap {
		if dunbarMax > 0 && int(n.DunbarTier) > dunbarMax {
			continue
		}
		cp := *n
		if snap != nil {
			if pos, ok := snap.Positions[id]; ok {
				cp.PosX = pos.PosX
				cp.PosY = pos.PosY
				cp.IsPinned = pos.IsPinned
			}
		}
		communities[cp.CommunityID] = true
		filteredNodes = append(filteredNodes, &cp)
	}

	var filteredEdges []*graph.Relationship
	for _, e := range edges {
		if !includeImplicit && e.Type == graph.RelImplicitColleague {
			continue
		}
		filteredEdges = append(filteredEdges, e)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"meta": map[string]any{
			"node_count":      len(filteredNodes),
			"edge_count":      len(filteredEdges),
			"community_count": len(communities),
			"generated_at":    time.Now().UTC().Format(time.RFC3339),
		},
		"nodes": filteredNodes,
		"edges": filteredEdges,
	})
}

// handleShortestPath находит самый тёплый путь рукопожатий между from_id и to_id.
func (s *LocalAPIServer) handleShortestPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "метод не поддерживается"})
		return
	}

	if s.graphService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "сервис социального графа недоступен"})
		return
	}

	fromID, err1 := strconv.ParseInt(r.URL.Query().Get("from_id"), 10, 64)
	toID, err2 := strconv.ParseInt(r.URL.Query().Get("to_id"), 10, 64)
	if err1 != nil || err2 != nil || fromID <= 0 || toID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "параметры from_id и to_id обязательны и должны быть положительными числами"})
		return
	}

	if fromID == toID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "from_id и to_id не могут быть одинаковыми"})
		return
	}

	path, err := s.graphService.FindWarmIntroPath(r.Context(), fromID, toID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "путь знакомств между контактами не найден"})
		return
	}

	writeJSON(w, http.StatusOK, path)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
}

// CreateContactViaHTTP отправляет JSON POST-запрос к REST API через http.Client.
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
