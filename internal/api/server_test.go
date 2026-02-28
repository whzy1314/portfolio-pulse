package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"portfoliopulse/internal/db"
	"portfoliopulse/internal/models"
	"portfoliopulse/internal/realtime"
	"portfoliopulse/internal/store"
)

type fakeMarket struct {
	prices map[string]float64
}

func (f *fakeMarket) Refresh(_ context.Context, _ []models.Holding) error { return nil }
func (f *fakeMarket) Snapshot() map[string]float64 {
	out := make(map[string]float64, len(f.prices))
	for k, v := range f.prices {
		out[k] = v
	}
	return out
}

func setupServer(t *testing.T) (*Server, *sql.DB) {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "api.db")
	sqlDB, err := db.Open(dbFile)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	st := store.NewSQLiteStore(sqlDB)
	fm := &fakeMarket{prices: map[string]float64{"stock:AAPL": 200}}
	server := NewServer(st, fm, realtime.NewHub())
	return server, sqlDB
}

func TestCreateListDeleteHoldingHandlers(t *testing.T) {
	server, sqlDB := setupServer(t)
	defer sqlDB.Close()

	payload := map[string]any{
		"ticker":    "aapl",
		"assetType": "stock",
		"quantity":  2,
		"avgCost":   100,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/holdings", bytes.NewReader(body))
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body=%s", resp.Code, resp.Body.String())
	}

	var created models.Holding
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created holding: %v", err)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/holdings", nil)
	resp = httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var holdings []models.Holding
	if err := json.Unmarshal(resp.Body.Bytes(), &holdings); err != nil {
		t.Fatalf("decode holdings list: %v", err)
	}
	if len(holdings) != 1 || holdings[0].Ticker != "AAPL" {
		t.Fatalf("unexpected holdings response: %+v", holdings)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/holdings/"+itoa(created.ID), nil)
	resp = httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
}

func TestPortfolioSnapshotHandler(t *testing.T) {
	server, sqlDB := setupServer(t)
	defer sqlDB.Close()

	createReq := httptest.NewRequest(http.MethodPost, "/api/holdings", bytes.NewReader([]byte(`{"ticker":"AAPL","assetType":"stock","quantity":2,"avgCost":100}`)))
	createResp := httptest.NewRecorder()
	server.Handler().ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create holding failed: %d", createResp.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/portfolio", nil)
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var snapshot models.PortfolioSnapshot
	if err := json.Unmarshal(resp.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snapshot.TotalValue != 400 || snapshot.TotalCost != 200 || snapshot.TotalPnL != 200 {
		t.Fatalf("unexpected totals: %+v", snapshot)
	}
}

func itoa(v int64) string {
	return fmt.Sprintf("%d", v)
}
