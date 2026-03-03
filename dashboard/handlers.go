package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sol-armada/sol-bot/settings"
	"github.com/sol-armada/sol-bot/utils"
	"go.mongodb.org/mongo-driver/bson"
)

// handleHome renders the main dashboard page
func (d *Dashboard) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Check if initial setup is needed
	if utils.NeedsSetup() {
		http.Redirect(w, r, "/setup", http.StatusTemporaryRedirect)
		return
	}

	ctx := r.Context()
	metrics, err := d.collectMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect metrics", "error", err)
		http.Error(w, "Failed to load metrics", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":   "Sol-Bot Dashboard",
		"Page":    "home",
		"Metrics": metrics,
	}

	if err := d.templates.ExecuteTemplate(w, "home.html", data); err != nil {
		d.logger.Error("failed to render template", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// handleMembers renders the members page
func (d *Dashboard) handleMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metrics, err := d.collectMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect metrics", "error", err)
		http.Error(w, "Failed to load metrics", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":   "Members - Sol-Bot Dashboard",
		"Page":    "members",
		"Metrics": metrics,
	}

	if err := d.templates.ExecuteTemplate(w, "members.html", data); err != nil {
		d.logger.Error("failed to render template", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// handleAttendance renders the attendance page
func (d *Dashboard) handleAttendance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metrics, err := d.collectMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect metrics", "error", err)
		http.Error(w, "Failed to load metrics", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":   "Attendance - Sol-Bot Dashboard",
		"Page":    "attendance",
		"Metrics": metrics,
	}

	if err := d.templates.ExecuteTemplate(w, "attendance.html", data); err != nil {
		d.logger.Error("failed to render template", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// handleTokens renders the tokens page
func (d *Dashboard) handleTokens(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metrics, err := d.collectMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect metrics", "error", err)
		http.Error(w, "Failed to load metrics", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":   "Tokens - Sol-Bot Dashboard",
		"Page":    "tokens",
		"Metrics": metrics,
	}

	if err := d.templates.ExecuteTemplate(w, "tokens.html", data); err != nil {
		d.logger.Error("failed to render template", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// handleActivity renders the activity page
func (d *Dashboard) handleActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metrics, err := d.collectMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect metrics", "error", err)
		http.Error(w, "Failed to load metrics", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":   "Activity - Sol-Bot Dashboard",
		"Page":    "activity",
		"Metrics": metrics,
	}

	if err := d.templates.ExecuteTemplate(w, "activity.html", data); err != nil {
		d.logger.Error("failed to render template", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// handleRaffles renders the raffles page
func (d *Dashboard) handleRaffles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metrics, err := d.collectMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect metrics", "error", err)
		http.Error(w, "Failed to load metrics", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":   "Raffles & Giveaways - Sol-Bot Dashboard",
		"Page":    "raffles",
		"Metrics": metrics,
	}

	if err := d.templates.ExecuteTemplate(w, "raffles.html", data); err != nil {
		d.logger.Error("failed to render template", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// handleMetrics returns metrics as JSON
func (d *Dashboard) handleMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metrics, err := d.collectMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect metrics", "error", err)
		http.Error(w, "Failed to collect metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		d.logger.Error("failed to encode metrics", "error", err)
	}
}

// handleMemberSearch searches for members
func (d *Dashboard) handleMemberSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	membersStore, ok := d.stores.GetMembersStore()
	if !ok {
		http.Error(w, "Members store not available", http.StatusInternalServerError)
		return
	}

	// Search by name (case-insensitive)
	filter := bson.D{{Key: "name", Value: bson.D{{Key: "$regex", Value: query}, {Key: "$options", Value: "i"}}}}

	cur, err := membersStore.List(filter, 0, 20)
	if err != nil {
		d.logger.Error("failed to search members", "error", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)

	var members []map[string]any
	for cur.Next(ctx) {
		var member map[string]any
		if err := cur.Decode(&member); err != nil {
			d.logger.Error("failed to decode member", "error", err)
			continue
		}
		members = append(members, member)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(members); err != nil {
		d.logger.Error("failed to encode members", "error", err)
	}
}

// handleConfigs renders the configuration management page
func (d *Dashboard) handleConfigs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metrics, err := d.collectMetrics(ctx)
	if err != nil {
		d.logger.Error("failed to collect metrics", "error", err)
		http.Error(w, "Failed to load metrics", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":   "Configuration - Sol-Bot Dashboard",
		"Page":    "configs",
		"Metrics": metrics,
	}

	if err := d.templates.ExecuteTemplate(w, "configs.html", data); err != nil {
		d.logger.Error("failed to render template", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// handleConfigUpdate handles updating a configuration value
func (d *Dashboard) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Key       string `json:"key"`
		Value     any    `json:"value"`
		UpdatedBy string `json:"updated_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get config store
	configStore, ok := d.stores.GetConfigsStore()
	if !ok {
		http.Error(w, "Configuration store not available", http.StatusInternalServerError)
		return
	}

	// Store the config override in MongoDB
	override := bson.M{
		"name":       req.Key,
		"value":      req.Value,
		"override":   true,
		"updated_by": req.UpdatedBy,
		"updated_at": time.Now(),
	}

	if err := configStore.UpsertOverride(override); err != nil {
		d.logger.Error("failed to set config override", "key", req.Key, "error", err)
		http.Error(w, fmt.Sprintf("Failed to update config: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Config updated successfully"))
}

// handleConfigReset handles resetting a configuration to default
func (d *Dashboard) handleConfigReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Key string `json:"key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get config store
	configStore, ok := d.stores.GetConfigsStore()
	if !ok {
		http.Error(w, "Configuration store not available", http.StatusInternalServerError)
		return
	}

	// Remove the config from MongoDB
	ctx := context.Background()
	filter := bson.M{"name": req.Key}
	if _, err := configStore.DeleteOne(ctx, filter); err != nil {
		d.logger.Error("failed to remove config override", "key", req.Key, "error", err)
		http.Error(w, fmt.Sprintf("Failed to reset config: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Config reset successfully"))
}

// handleConfigExport handles exporting all configurations
func (d *Dashboard) handleConfigExport(w http.ResponseWriter, r *http.Request) {
	allSettings := settings.AllSettings()

	// Get all overrides from MongoDB
	configStore, ok := d.stores.GetConfigsStore()
	var overrides []any
	if ok {
		ctx := context.Background()
		cursor, err := configStore.GetAll()
		if err == nil {
			defer cursor.Close(ctx)
			for cursor.Next(ctx) {
				var doc bson.M
				if err := cursor.Decode(&doc); err == nil {
					overrides = append(overrides, doc)
				}
			}
		}
	}

	exportData := map[string]any{
		"timestamp": time.Now(),
		"settings":  allSettings,
		"overrides": overrides,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(exportData); err != nil {
		d.logger.Error("failed to encode export data", "error", err)
	}
}

// handleSetup renders the initial setup page
func (d *Dashboard) handleSetup(w http.ResponseWriter, r *http.Request) {
	// If setup is already complete, redirect to dashboard
	if !utils.NeedsSetup() {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	data := map[string]any{
		"Title": "Initial Setup - Sol-Bot Dashboard",
	}

	if err := d.templates.ExecuteTemplate(w, "setup.html", data); err != nil {
		d.logger.Error("failed to render setup template", "error", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// handleSetupCheck returns JSON indicating if setup is needed
func (d *Dashboard) handleSetupCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"needs_setup": utils.NeedsSetup(),
	})
}
