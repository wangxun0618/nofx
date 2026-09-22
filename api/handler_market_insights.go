package api

import (
	"net/http"
	"strings"
	"time"

	"nofx/marketdata"
	"nofx/marketdata/providers"
	"nofx/store"

	"github.com/gin-gonic/gin"
)

// marketInsightDefaultLimit is the row count used when the caller does not ask
// for a specific board size.
const marketInsightDefaultLimit = 10

// marketInsightSource describes one registered data source for the UI catalogue.
// It comes straight from the provider registry, so a newly registered source
// appears in the API without any change here.
type marketInsightSource struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// RequiresKey lets the UI badge optional sources and prompt for a credential
	// instead of enabling them silently.
	RequiresKey bool `json:"requires_key"`
	// RequiresService names an external process the source needs, for sources
	// that are locally hosted and therefore need no credential but are still
	// unavailable until the user starts something. Empty when not applicable.
	RequiresService string `json:"requires_service,omitempty"`
}

// handleMarketInsightProviders lists every market-intelligence source this build
// ships, regardless of whether the current strategy enables it.
//
// GET /api/market-insights/providers
func (s *Server) handleMarketInsightProviders(c *gin.Context) {
	registry := providers.Default()

	sources := make([]marketInsightSource, 0, len(registry.All()))
	for _, provider := range registry.All() {
		source := marketInsightSource{
			Name:        provider.Name(),
			Description: provider.Description(),
			RequiresKey: provider.RequiresAPIKey(),
		}
		if service, ok := provider.(marketdata.ServiceBacked); ok {
			source.RequiresService = service.RequiredService()
		}
		sources = append(sources, source)
	}

	c.JSON(http.StatusOK, gin.H{"sources": sources})
}

// handleMarketInsights collects the market-wide insight board on demand and
// returns both the structured payload and the rendered prompt blocks.
//
// It runs the same registry the strategy engine uses, so the terminal displays
// exactly the data the AI is reasoning over.
//
// GET /api/market-insights?lang=en&sources=hyperliquid_flow,directional_signal&hyperdata=1
func (s *Server) handleMarketInsights(c *gin.Context) {
	cfg := store.IndicatorConfig{
		EnableMarketInsights: true,
		MarketInsightLimit:   marketInsightDefaultLimit,
	}
	if raw := c.Query("sources"); raw != "" {
		cfg.MarketInsightSources = marketInsightSourceList(raw)
	}
	// The sidecar-backed sources are opt-in because they need a separate process.
	// A preview caller can request them explicitly; the credential itself is
	// never accepted over the query string.
	if raw := strings.TrimSpace(c.Query("hyperdata")); raw == "1" || strings.EqualFold(raw, "true") {
		cfg.EnableHyperData = true
		if base := strings.TrimSpace(c.Query("hyperdata_url")); base != "" {
			cfg.HyperDataBaseURL = base
		}
	}

	registry := providers.Default()
	insights := registry.Collect(c.Request.Context(), marketdata.Request{
		Language: c.DefaultQuery("lang", "en"),
	}, cfg)

	if insights == nil {
		insights = []*marketdata.Insight{}
	}

	c.JSON(http.StatusOK, gin.H{
		"insights":   insights,
		"fetched_at": time.Now().UTC(),
	})
}

// marketInsightSourceList parses a comma-separated source allow-list.
func marketInsightSourceList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
