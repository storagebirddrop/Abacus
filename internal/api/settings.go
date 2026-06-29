package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

type settingsStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	GetAll(ctx context.Context) (map[string]string, error)
}

// AppSettings holds all persisted application settings.
type AppSettings struct {
	SyncEnabled      bool   `json:"sync_enabled"`
	BlockchainBackend string `json:"blockchain_backend"`
	EsploraURL       string `json:"esplora_url"`
	EsploraRateMS    int    `json:"esplora_rate_ms"`
	ElectrumHost     string `json:"electrum_host"`
	ElectrumPort     int    `json:"electrum_port"`
	ElectrumTLS      bool   `json:"electrum_tls"`
}

var settingsDefaults = AppSettings{
	SyncEnabled:       false,
	BlockchainBackend: "esplora",
	EsploraURL:        "https://mempool.space/api",
	EsploraRateMS:     100,
	ElectrumHost:      "electrum.blockstream.info",
	ElectrumPort:      50002,
	ElectrumTLS:       true,
}

// SettingsHandler handles GET/PATCH /settings.
type SettingsHandler struct {
	repo settingsStore
}

func NewSettingsHandler(repo settingsStore) *SettingsHandler {
	return &SettingsHandler{repo: repo}
}

func loadSettings(ctx context.Context, repo settingsStore) (AppSettings, error) {
	m, err := repo.GetAll(ctx)
	if err != nil {
		return settingsDefaults, err
	}
	s := settingsDefaults
	if v, ok := m["sync_enabled"]; ok {
		s.SyncEnabled = v == "true"
	}
	if v, ok := m["blockchain_backend"]; ok {
		s.BlockchainBackend = v
	}
	if v, ok := m["esplora_url"]; ok {
		s.EsploraURL = v
	}
	if v, ok := m["esplora_rate_ms"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			s.EsploraRateMS = n
		}
	}
	if v, ok := m["electrum_host"]; ok {
		s.ElectrumHost = v
	}
	if v, ok := m["electrum_port"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			s.ElectrumPort = n
		}
	}
	if v, ok := m["electrum_tls"]; ok {
		s.ElectrumTLS = v == "true"
	}
	return s, nil
}

// GetSettings handles GET /settings.
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	s, err := loadSettings(r.Context(), h.repo)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// UpdateSettings handles PATCH /settings.
func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SyncEnabled       *bool   `json:"sync_enabled"`
		BlockchainBackend *string `json:"blockchain_backend"`
		EsploraURL        *string `json:"esplora_url"`
		EsploraRateMS     *int    `json:"esplora_rate_ms"`
		ElectrumHost      *string `json:"electrum_host"`
		ElectrumPort      *int    `json:"electrum_port"`
		ElectrumTLS       *bool   `json:"electrum_tls"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	ctx := r.Context()
	set := func(key, value string) error {
		return h.repo.Set(ctx, key, value)
	}

	if req.SyncEnabled != nil {
		if err := set("sync_enabled", strconv.FormatBool(*req.SyncEnabled)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if req.BlockchainBackend != nil {
		switch *req.BlockchainBackend {
		case "esplora", "electrum":
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "blockchain_backend must be esplora or electrum"})
			return
		}
		if err := set("blockchain_backend", *req.BlockchainBackend); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if req.EsploraURL != nil {
		if *req.EsploraURL == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "esplora_url cannot be empty"})
			return
		}
		if err := set("esplora_url", *req.EsploraURL); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if req.EsploraRateMS != nil {
		if err := set("esplora_rate_ms", strconv.Itoa(*req.EsploraRateMS)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if req.ElectrumHost != nil {
		if err := set("electrum_host", *req.ElectrumHost); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if req.ElectrumPort != nil {
		if err := set("electrum_port", strconv.Itoa(*req.ElectrumPort)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	if req.ElectrumTLS != nil {
		if err := set("electrum_tls", strconv.FormatBool(*req.ElectrumTLS)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	// Return updated settings.
	s, err := loadSettings(ctx, h.repo)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// IsSyncEnabled is a helper used by the sync handler to gate sync.
func IsSyncEnabled(ctx context.Context, repo settingsStore) (bool, error) {
	v, err := repo.Get(ctx, "sync_enabled")
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil // default: disabled
	}
	if err != nil {
		return false, err
	}
	return v == "true", nil
}
