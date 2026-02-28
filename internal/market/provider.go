package market

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"portfoliopulse/internal/models"
)

type Provider struct {
	httpClient *http.Client
	mu         sync.RWMutex
	prices     map[string]float64
}

func NewProvider() *Provider {
	return &Provider{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		prices:     make(map[string]float64),
	}
}

func key(assetType models.AssetType, ticker string) string {
	return string(assetType) + ":" + strings.ToUpper(strings.TrimSpace(ticker))
}

func (p *Provider) GetPrice(assetType models.AssetType, ticker string) (float64, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	price, ok := p.prices[key(assetType, ticker)]
	return price, ok
}

func (p *Provider) Snapshot() map[string]float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make(map[string]float64, len(p.prices))
	for k, v := range p.prices {
		out[k] = v
	}
	return out
}

func (p *Provider) Refresh(ctx context.Context, holdings []models.Holding) error {
	stocks := make([]string, 0)
	cryptos := make([]string, 0)
	seen := map[string]bool{}

	for _, h := range holdings {
		ticker := strings.ToUpper(strings.TrimSpace(h.Ticker))
		k := key(h.AssetType, ticker)
		if seen[k] {
			continue
		}
		seen[k] = true
		switch h.AssetType {
		case models.AssetCrypto:
			if id, ok := coinGeckoIDs[ticker]; ok {
				cryptos = append(cryptos, id)
			}
		default:
			stocks = append(stocks, ticker)
		}
	}

	updates := make(map[string]float64)
	if len(stocks) > 0 {
		stockUpdates, err := p.fetchYahooQuotes(ctx, stocks)
		if err != nil {
			return err
		}
		for k, v := range stockUpdates {
			updates[k] = v
		}
	}

	if len(cryptos) > 0 {
		cryptoUpdates, err := p.fetchCoinGeckoPrices(ctx, cryptos)
		if err != nil {
			return err
		}
		for k, v := range cryptoUpdates {
			updates[k] = v
		}
	}

	p.mu.Lock()
	for k, v := range updates {
		if !math.IsNaN(v) && v > 0 {
			p.prices[k] = v
		}
	}
	p.mu.Unlock()

	return nil
}

func (p *Provider) fetchYahooQuotes(ctx context.Context, symbols []string) (map[string]float64, error) {
	updates := make(map[string]float64)

	for _, symbol := range symbols {
		endpoint := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", url.PathEscape(symbol))

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36")

		resp, err := p.httpClient.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		var payload struct {
			Chart struct {
				Result []struct {
					Meta struct {
						Symbol             string  `json:"symbol"`
						RegularMarketPrice float64 `json:"regularMarketPrice"`
					} `json:"meta"`
				} `json:"result"`
			} `json:"chart"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		if len(payload.Chart.Result) > 0 {
			meta := payload.Chart.Result[0].Meta
			updates[key(models.AssetStock, meta.Symbol)] = meta.RegularMarketPrice
		}
	}

	return updates, nil
}

func (p *Provider) fetchCoinGeckoPrices(ctx context.Context, ids []string) (map[string]float64, error) {
	values := url.Values{}
	values.Set("ids", strings.Join(ids, ","))
	values.Set("vs_currencies", "usd")
	endpoint := "https://api.coingecko.com/api/v3/simple/price?" + values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create coingecko request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch coingecko prices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("coingecko status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload map[string]struct {
		USD float64 `json:"usd"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode coingecko prices: %w", err)
	}

	updates := make(map[string]float64)
	for ticker, id := range coinGeckoIDs {
		if val, ok := payload[id]; ok {
			updates[key(models.AssetCrypto, ticker)] = val.USD
		}
	}
	return updates, nil
}

var coinGeckoIDs = map[string]string{
	"BTC":     "bitcoin",
	"BITCOIN": "bitcoin",
	"ETH":     "ethereum",
	"ETHEREUM":"ethereum",
	"SOL":     "solana",
	"SOLANA":  "solana",
	"DOGE":    "dogecoin",
	"ADA":     "cardano",
	"XRP":     "ripple",
	"DOT":     "polkadot",
	"AVAX":    "avalanche-2",
	"MATIC":   "matic-network",
	"LINK":    "chainlink",
}
