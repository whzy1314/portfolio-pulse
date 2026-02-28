package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"portfoliopulse/internal/db"
	"portfoliopulse/internal/models"
)

func setupStore(t *testing.T) (*SQLiteStore, *sql.DB) {
	t.Helper()
	dbFile := filepath.Join(t.TempDir(), "test.db")
	sqlDB, err := db.Open(dbFile)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return NewSQLiteStore(sqlDB), sqlDB
}

func TestHoldingCRUD(t *testing.T) {
	s, sqlDB := setupStore(t)
	defer sqlDB.Close()

	ctx := context.Background()
	created, err := s.CreateHolding(ctx, models.Holding{
		Ticker:    "aapl",
		AssetType: models.AssetStock,
		Quantity:  2,
		AvgCost:   150,
	})
	if err != nil {
		t.Fatalf("create holding: %v", err)
	}
	if created.ID == 0 || created.Ticker != "AAPL" {
		t.Fatalf("unexpected created holding: %+v", created)
	}

	holdings, err := s.ListHoldings(ctx)
	if err != nil {
		t.Fatalf("list holdings: %v", err)
	}
	if len(holdings) != 1 {
		t.Fatalf("expected 1 holding, got %d", len(holdings))
	}

	if err := s.DeleteHolding(ctx, created.ID); err != nil {
		t.Fatalf("delete holding: %v", err)
	}
	if err := s.DeleteHolding(ctx, created.ID); err == nil {
		t.Fatalf("expected error deleting same holding twice")
	}
}

func TestAlertCRUDAndTrigger(t *testing.T) {
	s, sqlDB := setupStore(t)
	defer sqlDB.Close()

	ctx := context.Background()
	created, err := s.CreateAlert(ctx, models.PriceAlert{
		Ticker:    "btc",
		AssetType: models.AssetCrypto,
		Direction: models.AlertAbove,
		Threshold: 50000,
	})
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}
	if created.ID == 0 || created.Ticker != "BTC" {
		t.Fatalf("unexpected created alert: %+v", created)
	}

	now := time.Now().UTC().Truncate(time.Second)
	if err := s.MarkAlertTriggered(ctx, created.ID, now); err != nil {
		t.Fatalf("mark alert triggered: %v", err)
	}

	alerts, err := s.ListAlerts(ctx)
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if !alerts[0].Triggered || alerts[0].TriggeredAt == nil {
		t.Fatalf("expected triggered alert, got %+v", alerts[0])
	}

	if err := s.DeleteAlert(ctx, created.ID); err != nil {
		t.Fatalf("delete alert: %v", err)
	}
}
