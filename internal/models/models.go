package models

import "time"

type AssetType string

const (
	AssetStock  AssetType = "stock"
	AssetCrypto AssetType = "crypto"
)

type Holding struct {
	ID        int64     `json:"id"`
	Ticker    string    `json:"ticker"`
	AssetType AssetType `json:"assetType"`
	Quantity  float64   `json:"quantity"`
	AvgCost   float64   `json:"avgCost"`
	CreatedAt time.Time `json:"createdAt"`
}

type AlertDirection string

const (
	AlertAbove AlertDirection = "above"
	AlertBelow AlertDirection = "below"
)

type PriceAlert struct {
	ID         int64          `json:"id"`
	Ticker     string         `json:"ticker"`
	AssetType  AssetType      `json:"assetType"`
	Direction  AlertDirection `json:"direction"`
	Threshold  float64        `json:"threshold"`
	CreatedAt  time.Time      `json:"createdAt"`
	Triggered  bool           `json:"triggered"`
	TriggeredAt *time.Time    `json:"triggeredAt,omitempty"`
}

type HoldingWithPrice struct {
	Holding
	Price      float64 `json:"price"`
	MarketValue float64 `json:"marketValue"`
	CostBasis  float64 `json:"costBasis"`
	PnL        float64 `json:"pnl"`
	PnLPct     float64 `json:"pnlPct"`
}

type PortfolioSnapshot struct {
	Holdings   []HoldingWithPrice `json:"holdings"`
	TotalValue float64            `json:"totalValue"`
	TotalCost  float64            `json:"totalCost"`
	TotalPnL   float64            `json:"totalPnl"`
	UpdatedAt  time.Time          `json:"updatedAt"`
	AlertsFired []PriceAlert      `json:"alertsFired,omitempty"`
}
