package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"portfoliopulse/internal/models"
	"portfoliopulse/internal/realtime"
	"portfoliopulse/internal/store"
)

type Server struct {
	store    store.Store
	market   PriceProvider
	hub      *realtime.Hub
	router   *mux.Router
	upgrader websocket.Upgrader
}

type PriceProvider interface {
	Refresh(ctx context.Context, holdings []models.Holding) error
	Snapshot() map[string]float64
}

func NewServer(s store.Store, p PriceProvider, hub *realtime.Hub) *Server {
	server := &Server{
		store:  s,
		market: p,
		hub:    hub,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}

	r := mux.NewRouter()
	r.Use(corsMiddleware)

	r.HandleFunc("/api/health", server.handleHealth).Methods(http.MethodGet)
	r.HandleFunc("/api/holdings", server.handleListHoldings).Methods(http.MethodGet)
	r.HandleFunc("/api/holdings", server.handleCreateHolding).Methods(http.MethodPost)
	r.HandleFunc("/api/holdings/{id}", server.handleDeleteHolding).Methods(http.MethodDelete)
	r.HandleFunc("/api/alerts", server.handleListAlerts).Methods(http.MethodGet)
	r.HandleFunc("/api/alerts", server.handleCreateAlert).Methods(http.MethodPost)
	r.HandleFunc("/api/alerts/{id}", server.handleDeleteAlert).Methods(http.MethodDelete)
	r.HandleFunc("/api/portfolio", server.handlePortfolioSnapshot).Methods(http.MethodGet)
	r.HandleFunc("/ws", server.handleWebSocket).Methods(http.MethodGet)

	// Serve React SPA (catch-all, must be last)
	spa := spaHandler{staticPath: "web/dist", indexPath: "index.html"}
	r.PathPrefix("/").Handler(spa)

	server.router = r
	return server
}

type spaHandler struct {
	staticPath string
	indexPath  string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(h.staticPath, r.URL.Path)
	fi, err := os.Stat(path)
	if os.IsNotExist(err) || fi.IsDir() {
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) StartPolling(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	_ = s.RefreshAndBroadcast(context.Background())
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.RefreshAndBroadcast(context.Background()); err != nil {
				log.Printf("polling refresh failed: %v", err)
			}
		}
	}
}

func (s *Server) RefreshAndBroadcast(ctx context.Context) error {
	holdings, err := s.store.ListHoldings(ctx)
	if err != nil {
		return err
	}

	if err := s.market.Refresh(ctx, holdings); err != nil {
		return err
	}

	snapshot, err := s.BuildSnapshot(ctx)
	if err != nil {
		return err
	}

	s.hub.BroadcastJSON(snapshot)
	return nil
}

func (s *Server) BuildSnapshot(ctx context.Context) (models.PortfolioSnapshot, error) {
	holdings, err := s.store.ListHoldings(ctx)
	if err != nil {
		return models.PortfolioSnapshot{}, err
	}

	alerts, err := s.store.ListAlerts(ctx)
	if err != nil {
		return models.PortfolioSnapshot{}, err
	}

	out := models.PortfolioSnapshot{
		Holdings: make([]models.HoldingWithPrice, 0, len(holdings)),
		UpdatedAt: time.Now().UTC(),
	}

	prices := s.market.Snapshot()
	alertsFired := make([]models.PriceAlert, 0)
	for _, h := range holdings {
		price := prices[assetKey(h.AssetType, h.Ticker)]
		marketValue := h.Quantity * price
		costBasis := h.Quantity * h.AvgCost
		pnl := marketValue - costBasis
		pnlPct := 0.0
		if costBasis > 0 {
			pnlPct = (pnl / costBasis) * 100
		}

		out.Holdings = append(out.Holdings, models.HoldingWithPrice{
			Holding:     h,
			Price:       round2(price),
			MarketValue: round2(marketValue),
			CostBasis:   round2(costBasis),
			PnL:         round2(pnl),
			PnLPct:      round2(pnlPct),
		})

		out.TotalValue += marketValue
		out.TotalCost += costBasis
	}

	out.TotalPnL = out.TotalValue - out.TotalCost
	out.TotalValue = round2(out.TotalValue)
	out.TotalCost = round2(out.TotalCost)
	out.TotalPnL = round2(out.TotalPnL)

	for _, alert := range alerts {
		if alert.Triggered {
			continue
		}
		price, ok := prices[assetKey(alert.AssetType, alert.Ticker)]
		if !ok || price <= 0 {
			continue
		}

		fired := (alert.Direction == models.AlertAbove && price >= alert.Threshold) ||
			(alert.Direction == models.AlertBelow && price <= alert.Threshold)
		if fired {
			now := time.Now().UTC()
			if err := s.store.MarkAlertTriggered(ctx, alert.ID, now); err != nil {
				log.Printf("failed to mark alert triggered %d: %v", alert.ID, err)
				continue
			}
			alert.Triggered = true
			alert.TriggeredAt = &now
			alertsFired = append(alertsFired, alert)
		}
	}

	if len(alertsFired) > 0 {
		out.AlertsFired = alertsFired
	}

	return out, nil
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListHoldings(w http.ResponseWriter, r *http.Request) {
	holdings, err := s.store.ListHoldings(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, holdings)
}

func (s *Server) handleCreateHolding(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Ticker    string           `json:"ticker"`
		AssetType models.AssetType `json:"assetType"`
		Quantity  float64          `json:"quantity"`
		AvgCost   float64          `json:"avgCost"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	req.Ticker = strings.TrimSpace(strings.ToUpper(req.Ticker))
	if req.Ticker == "" || req.Quantity <= 0 || req.AvgCost < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid holding payload"})
		return
	}
	if req.AssetType != models.AssetStock && req.AssetType != models.AssetCrypto {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "assetType must be stock or crypto"})
		return
	}

	created, err := s.store.CreateHolding(r.Context(), models.Holding{
		Ticker:    req.Ticker,
		AssetType: req.AssetType,
		Quantity:  req.Quantity,
		AvgCost:   req.AvgCost,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	_ = s.RefreshAndBroadcast(context.Background())
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleDeleteHolding(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	err = s.store.DeleteHolding(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "holding not found"})
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	_ = s.RefreshAndBroadcast(context.Background())
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := s.store.ListAlerts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, alerts)
}

func (s *Server) handleCreateAlert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Ticker    string                `json:"ticker"`
		AssetType models.AssetType      `json:"assetType"`
		Direction models.AlertDirection `json:"direction"`
		Threshold float64               `json:"threshold"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	req.Ticker = strings.ToUpper(strings.TrimSpace(req.Ticker))
	if req.Ticker == "" || req.Threshold <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid alert payload"})
		return
	}
	if req.AssetType != models.AssetStock && req.AssetType != models.AssetCrypto {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "assetType must be stock or crypto"})
		return
	}
	if req.Direction != models.AlertAbove && req.Direction != models.AlertBelow {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "direction must be above or below"})
		return
	}

	created, err := s.store.CreateAlert(r.Context(), models.PriceAlert{
		Ticker:    req.Ticker,
		AssetType: req.AssetType,
		Direction: req.Direction,
		Threshold: req.Threshold,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	_ = s.RefreshAndBroadcast(context.Background())
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleDeleteAlert(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	err = s.store.DeleteAlert(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "alert not found"})
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	_ = s.RefreshAndBroadcast(context.Background())
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePortfolioSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.BuildSnapshot(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.hub.AddClient(conn)

	if snapshot, err := s.BuildSnapshot(r.Context()); err == nil {
		_ = conn.WriteJSON(snapshot)
	}

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			s.hub.RemoveClient(conn)
			return
		}
	}
}

func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func assetKey(assetType models.AssetType, ticker string) string {
	return string(assetType) + ":" + strings.ToUpper(strings.TrimSpace(ticker))
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
