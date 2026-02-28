# PortfolioPulse

Real-time stock and crypto portfolio tracker with live price updates.

## Architecture

```
cmd/server/main.go        Entry point — HTTP server with graceful shutdown
internal/
  api/server.go            REST handlers, WebSocket endpoint, portfolio logic
  db/sqlite.go             SQLite init and schema migration
  market/provider.go       Yahoo Finance (stocks) + CoinGecko (crypto) price fetching
  models/models.go         Shared data types
  realtime/hub.go          WebSocket client hub for broadcasting
  store/store.go           SQLite CRUD for holdings and alerts
web/                       React + Vite frontend with Recharts
```

**Backend**: Go with gorilla/mux (routing), gorilla/websocket (real-time), and mattn/go-sqlite3 (persistence). A background goroutine polls market data every 30 seconds and pushes portfolio snapshots to all connected WebSocket clients. Price alerts are checked each cycle and marked triggered when thresholds are crossed.

**Frontend**: Vite + React 18 with Recharts for the allocation pie chart. Connects via WebSocket for live updates with automatic reconnection.

## Prerequisites

- Go 1.22+
- Node.js 18+ and npm
- GCC (required by go-sqlite3 CGO)

## Quick Start

```bash
# Install frontend dependencies
make frontend-install

# Run backend + frontend dev servers concurrently
make dev
```

Backend serves on `:8080`, frontend dev server on `:5173` (proxies API/WS to backend).

## Makefile Targets

| Target             | Description                                      |
|--------------------|--------------------------------------------------|
| `make build`       | Compile Go backend to `bin/server`               |
| `make run`         | Build and run the backend server                 |
| `make test`        | Run all Go tests                                 |
| `make frontend-install` | Install frontend npm dependencies           |
| `make frontend-build`   | Build frontend for production (`web/dist`)  |
| `make clean`       | Remove build artifacts and node_modules          |
| `make dev`         | Run backend and frontend dev servers in parallel |

## API Documentation

Base URL: `http://localhost:8080`

### Holdings

| Method | Endpoint              | Description          |
|--------|-----------------------|----------------------|
| GET    | `/api/holdings`       | List all holdings    |
| POST   | `/api/holdings`       | Create a holding     |
| DELETE | `/api/holdings/{id}`  | Delete a holding     |

**POST /api/holdings** body:
```json
{
  "ticker": "AAPL",
  "assetType": "stock",
  "quantity": 10,
  "avgCost": 150.00
}
```

### Price Alerts

| Method | Endpoint            | Description        |
|--------|---------------------|--------------------|
| GET    | `/api/alerts`       | List all alerts    |
| POST   | `/api/alerts`       | Create an alert    |
| DELETE | `/api/alerts/{id}`  | Delete an alert    |

**POST /api/alerts** body:
```json
{
  "ticker": "BTC",
  "assetType": "crypto",
  "direction": "above",
  "threshold": 100000
}
```

### Portfolio

| Method | Endpoint          | Description                              |
|--------|-------------------|------------------------------------------|
| GET    | `/api/portfolio`  | Full portfolio snapshot with P&L         |

### WebSocket

Connect to `ws://localhost:8080/ws` for real-time portfolio snapshots. The server pushes a `PortfolioSnapshot` JSON message every 30 seconds and after any CRUD operation.

### Health Check

| Method | Endpoint      | Description   |
|--------|---------------|---------------|
| GET    | `/api/health` | Health check  |

## Running Tests

```bash
make test
```

Tests cover store CRUD operations and HTTP handler behavior using in-memory SQLite databases.
