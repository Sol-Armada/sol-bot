package dashboard

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/sol-armada/sol-bot/hybridconfig"
	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/stores"
)

//go:embed templates/*.html
var templateFS embed.FS

type Dashboard struct {
	server    *http.Server
	templates *template.Template
	hub       *Hub
	logger    *slog.Logger
	stores    *stores.Client
	config    *hybridconfig.HybridConfig
}

var dashboard *Dashboard

// Setup initializes the dashboard
func Setup() error {
	dashboard = &Dashboard{
		logger: slog.Default().With("component", "dashboard"),
		stores: stores.Get(),
		config: hybridconfig.Get(),
	}

	// Parse templates
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return fmt.Errorf("failed to parse templates: %w", err)
	}
	dashboard.templates = tmpl

	// Initialize WebSocket hub
	dashboard.hub = NewHub()
	go dashboard.hub.Run()

	return nil
}

// Start begins serving the dashboard
func Start(ctx context.Context) error {
	if dashboard == nil {
		return fmt.Errorf("dashboard not initialized, call Setup() first")
	}

	port := settings.GetIntWithDefault("DASHBOARD.PORT", 8080)
	addr := fmt.Sprintf(":%d", port)

	mux := http.NewServeMux()
	dashboard.registerRoutes(mux)

	dashboard.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	dashboard.logger.Info("starting dashboard server", "address", addr)

	// Start metrics broadcaster
	go dashboard.broadcastMetrics(ctx)

	// Start server in goroutine
	go func() {
		if err := dashboard.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			dashboard.logger.Error("dashboard server error", "error", err)
		}
	}()

	return nil
}

// Shutdown gracefully stops the dashboard server
func Shutdown(ctx context.Context) error {
	if dashboard == nil || dashboard.server == nil {
		return nil
	}

	dashboard.logger.Info("shutting down dashboard server")

	// Stop the hub
	close(dashboard.hub.broadcast)

	return dashboard.server.Shutdown(ctx)
}

// registerRoutes sets up all HTTP routes
func (d *Dashboard) registerRoutes(mux *http.ServeMux) {
	// Setup page (must be first, no redirect)
	mux.HandleFunc("/setup", d.handleSetup)
	
	// Pages
	mux.HandleFunc("/", d.handleHome)
	mux.HandleFunc("/members", d.handleMembers)
	mux.HandleFunc("/attendance", d.handleAttendance)
	mux.HandleFunc("/tokens", d.handleTokens)
	mux.HandleFunc("/activity", d.handleActivity)
	mux.HandleFunc("/raffles", d.handleRaffles)
	mux.HandleFunc("/configs", d.handleConfigs)

	// API endpoints
	mux.HandleFunc("/api/metrics", d.handleMetrics)
	mux.HandleFunc("/api/members/search", d.handleMemberSearch)
	mux.HandleFunc("/api/configs/update", d.handleConfigUpdate)
	mux.HandleFunc("/api/configs/reset", d.handleConfigReset)
	mux.HandleFunc("/api/configs/export", d.handleConfigExport)
	mux.HandleFunc("/api/setup/check", d.handleSetupCheck)

	// WebSocket
	mux.HandleFunc("/ws", d.handleWebSocket)
}

// broadcastMetrics periodically sends metric updates to all connected clients
func (d *Dashboard) broadcastMetrics(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics, err := d.collectMetrics(ctx)
			if err != nil {
				d.logger.Error("failed to collect metrics", "error", err)
				continue
			}

			d.hub.Broadcast(Message{
				Type: "metrics_update",
				Data: metrics,
			})
		}
	}
}

// Get returns the dashboard instance
func Get() *Dashboard {
	return dashboard
}
