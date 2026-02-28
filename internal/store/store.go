package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"portfoliopulse/internal/models"
)

type Store interface {
	ListHoldings(ctx context.Context) ([]models.Holding, error)
	CreateHolding(ctx context.Context, h models.Holding) (models.Holding, error)
	DeleteHolding(ctx context.Context, id int64) error
	ListAlerts(ctx context.Context) ([]models.PriceAlert, error)
	CreateAlert(ctx context.Context, alert models.PriceAlert) (models.PriceAlert, error)
	DeleteAlert(ctx context.Context, id int64) error
	MarkAlertTriggered(ctx context.Context, id int64, triggeredAt time.Time) error
}

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) ListHoldings(ctx context.Context) ([]models.Holding, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, ticker, asset_type, quantity, avg_cost, created_at
		FROM holdings ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("query holdings: %w", err)
	}
	defer rows.Close()

	holdings := make([]models.Holding, 0)
	for rows.Next() {
		var h models.Holding
		if err := rows.Scan(&h.ID, &h.Ticker, &h.AssetType, &h.Quantity, &h.AvgCost, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan holding: %w", err)
		}
		holdings = append(holdings, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate holdings: %w", err)
	}
	return holdings, nil
}

func (s *SQLiteStore) CreateHolding(ctx context.Context, h models.Holding) (models.Holding, error) {
	h.Ticker = strings.ToUpper(strings.TrimSpace(h.Ticker))
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO holdings(ticker, asset_type, quantity, avg_cost)
		VALUES (?, ?, ?, ?)`, h.Ticker, h.AssetType, h.Quantity, h.AvgCost)
	if err != nil {
		return models.Holding{}, fmt.Errorf("insert holding: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return models.Holding{}, fmt.Errorf("holding last insert id: %w", err)
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT id, ticker, asset_type, quantity, avg_cost, created_at
		FROM holdings WHERE id = ?`, id)

	var out models.Holding
	if err := row.Scan(&out.ID, &out.Ticker, &out.AssetType, &out.Quantity, &out.AvgCost, &out.CreatedAt); err != nil {
		return models.Holding{}, fmt.Errorf("fetch inserted holding: %w", err)
	}

	return out, nil
}

func (s *SQLiteStore) DeleteHolding(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM holdings WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete holding: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("holding rows affected: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) ListAlerts(ctx context.Context) ([]models.PriceAlert, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, ticker, asset_type, direction, threshold, created_at, triggered, triggered_at
		FROM price_alerts ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("query alerts: %w", err)
	}
	defer rows.Close()

	alerts := make([]models.PriceAlert, 0)
	for rows.Next() {
		var a models.PriceAlert
		var triggeredInt int
		var triggeredAt sql.NullTime
		if err := rows.Scan(&a.ID, &a.Ticker, &a.AssetType, &a.Direction, &a.Threshold, &a.CreatedAt, &triggeredInt, &triggeredAt); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		a.Triggered = triggeredInt == 1
		if triggeredAt.Valid {
			t := triggeredAt.Time
			a.TriggeredAt = &t
		}
		alerts = append(alerts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate alerts: %w", err)
	}
	return alerts, nil
}

func (s *SQLiteStore) CreateAlert(ctx context.Context, alert models.PriceAlert) (models.PriceAlert, error) {
	alert.Ticker = strings.ToUpper(strings.TrimSpace(alert.Ticker))
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO price_alerts(ticker, asset_type, direction, threshold)
		VALUES (?, ?, ?, ?)`, alert.Ticker, alert.AssetType, alert.Direction, alert.Threshold)
	if err != nil {
		return models.PriceAlert{}, fmt.Errorf("insert alert: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return models.PriceAlert{}, fmt.Errorf("alert last insert id: %w", err)
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT id, ticker, asset_type, direction, threshold, created_at, triggered, triggered_at
		FROM price_alerts WHERE id = ?`, id)

	var out models.PriceAlert
	var triggeredInt int
	var triggeredAt sql.NullTime
	if err := row.Scan(&out.ID, &out.Ticker, &out.AssetType, &out.Direction, &out.Threshold, &out.CreatedAt, &triggeredInt, &triggeredAt); err != nil {
		return models.PriceAlert{}, fmt.Errorf("fetch inserted alert: %w", err)
	}
	out.Triggered = triggeredInt == 1
	if triggeredAt.Valid {
		t := triggeredAt.Time
		out.TriggeredAt = &t
	}

	return out, nil
}

func (s *SQLiteStore) DeleteAlert(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM price_alerts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete alert: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("alert rows affected: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteStore) MarkAlertTriggered(ctx context.Context, id int64, triggeredAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE price_alerts
		SET triggered = 1, triggered_at = ?
		WHERE id = ?`, triggeredAt, id)
	if err != nil {
		return fmt.Errorf("mark alert triggered: %w", err)
	}
	return nil
}
