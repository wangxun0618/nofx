package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nofx/marketdata"
	"nofx/store"
)

// This file is the contract with HyperData Terminal.
//
// The sidecar is an independent project that ships on its own schedule, so the
// adapter is only safe to leave alone if a shape change is caught by a test
// rather than by a live trading cycle. The fixtures below are the response
// bodies this adapter was verified against; when upstream changes something,
// this is the file that has to change, and the failing assertion names the field.
//
// The drift test is the other half of the contract: upstream is allowed to add
// fields, loosen a type, or send null without breaking us. A test that only
// pinned the happy path would make the adapter brittle in exactly the way the
// HTTP boundary was chosen to avoid.

type stubRoute struct {
	status int
	body   string
}

// stubHyperData serves the given routes and returns its origin.
func stubHyperData(t *testing.T, routes map[string]stubRoute) string {
	t.Helper()

	mux := http.NewServeMux()
	for path, route := range routes {
		route := route
		mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if route.status != 0 {
				w.WriteHeader(route.status)
			}
			_, _ = io.WriteString(w, route.body)
		})
	}

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server.URL
}

// --- Recorded upstream fixtures -------------------------------------------

const fixtureHealth = `{
  "status": "warn",
  "failed_components": [],
  "orderflow_venues": {
    "hyperliquid": {"status": "ok", "stale": false, "sockets_open": 4, "sockets_expected": 4, "shards_dark": 0, "dark_symbols": []},
    "binance": {"status": "partial", "stale": true, "sockets_open": 0, "sockets_expected": 2, "shards_dark": 2, "dark_symbols": ["ETH", "SOL"]}
  },
  "feeds": {"liquidation_feed": "ok", "orderflow_engine": "partial"},
  "version": "1.0.0",
  "mode": "live",
  "uptime_seconds": 3600,
  "tracked_assets": 50,
  "tracked_positions": 120,
  "persistence": {"db_size_mb": 12.5, "events_persisted": 90000},
  "docs": "https://github.com/Co-Messi/HyperData-Terminal"
}`

const fixtureOrderflowBTC = `{
  "symbol": "BTC",
  "cumulative_cvd": 1234567,
  "cumulative_cvd_by_venue": {"hyperliquid": 234567, "binance": 1000000},
  "venue_coverage": {"hyperliquid": {"trades": 5000}, "binance": {"trades": 12000}},
  "venues_contributing": ["hyperliquid", "binance"],
  "trades_per_second": 12.5,
  "aggregate_signal": "BUY",
  "timeframes": {
    "1m": {"buy_volume": 100, "sell_volume": 90, "net_volume": 10, "trade_count": 50, "ofi": 0.05, "signal": "NEUTRAL"},
    "15m": {"buy_volume": 1500, "sell_volume": 1200, "net_volume": 300000, "trade_count": 900, "ofi": 0.11, "signal": "BUY"}
  }
}`

const fixtureOrderflowETH = `{
  "symbol": "ETH",
  "cumulative_cvd": -45000,
  "cumulative_cvd_by_venue": {"hyperliquid": -45000},
  "venues_contributing": ["hyperliquid"],
  "trades_per_second": 3.25,
  "aggregate_signal": "SELL",
  "timeframes": {"5m": {"buy_volume": 10, "sell_volume": 55, "net_volume": -45000, "trade_count": 30, "ofi": -0.2, "signal": "SELL"}}
}`

const fixtureLongShort = `{"BTC": {"long_ratio": 54.2, "short_ratio": 45.8, "long_short_ratio": 1.18, "timestamp": 1758500000}}`

const fixtureBasis = `{"BTC": {"spot_price": 63000, "perp_price": 63120, "basis_pct": 0.191, "timestamp": 1758500000}}`

const fixtureWhales = `{
  "count": 2,
  "as_of": 1758500000,
  "scan_age_seconds": 42.5,
  "positions": [
    {"address": "0xaaa", "symbol": "BTC", "side": "long", "size_usd": 2500000, "entry_price": 61000, "current_price": 63000, "liq_price": 48000, "distance_pct": 23.8, "leverage": 20, "unrealized_pnl": 80000, "margin_used": 125000, "scanned_at": 1758499957.5},
    {"address": "0xbbb", "symbol": "ETH", "side": "short", "size_usd": 900000, "entry_price": 3300, "current_price": 3250, "liq_price": 3450, "distance_pct": 6.15, "leverage": 10, "unrealized_pnl": 13000, "margin_used": 90000, "scanned_at": 1758499957.5}
  ]
}`

const fixtureDangerZone = `{
  "threshold_pct": 8,
  "count": 1,
  "as_of": 1758500000,
  "scan_age_seconds": 42.5,
  "positions": [
    {"address": "0xbbb", "symbol": "ETH", "side": "short", "size_usd": 900000, "entry_price": 3300, "current_price": 3250, "liq_price": 3450, "distance_pct": 6.15, "leverage": 10, "unrealized_pnl": 13000, "margin_used": 90000, "scanned_at": 1758499957.5}
  ]
}`

// hyperDataConfig points a strategy at a stub instance.
func hyperDataConfig(baseURL string) store.IndicatorConfig {
	return store.IndicatorConfig{
		EnableMarketInsights: true,
		MarketInsightLimit:   10,
		EnableHyperData:      true,
		HyperDataBaseURL:     baseURL,
	}
}

// --- Client contract ------------------------------------------------------

func TestHyperDataClientParsesRecordedShapes(t *testing.T) {
	base := stubHyperData(t, map[string]stubRoute{
		"/v1/health":                {body: fixtureHealth},
		"/v1/orderflow/BTC":         {body: fixtureOrderflowBTC},
		"/v1/long-short-ratio":      {body: fixtureLongShort},
		"/v1/basis":                 {body: fixtureBasis},
		"/v1/whales":                {body: fixtureWhales},
		"/v1/positions/danger-zone": {body: fixtureDangerZone},
	})

	client := NewHyperDataClient(base, "")
	ctx := context.Background()

	health, err := client.Health(ctx)
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if health.Status != "warn" || health.Version != "1.0.0" || health.TrackedPositions != 120 {
		t.Fatalf("health parsed as %+v", health)
	}
	if got := venueStatuses(health); len(got) != 2 {
		t.Fatalf("venue statuses = %v, want both venues", got)
	} else if !strings.Contains(strings.Join(got, "|"), "0/2 sockets") {
		t.Fatalf("venue statuses lost the socket counts: %v", got)
	}

	flow, err := client.Orderflow(ctx, "btc")
	if err != nil {
		t.Fatalf("orderflow: %v", err)
	}
	if flow.CumulativeCVD != 1234567 || flow.CumulativeCVDByVenue["binance"] != 1000000 {
		t.Fatalf("orderflow parsed as %+v", flow)
	}
	if flow.AggregateSignal.String() != "BUY" {
		t.Fatalf("aggregate signal = %q", flow.AggregateSignal)
	}
	if flow.Timeframes["15m"].NetVolume != 300000 {
		t.Fatalf("15m frame parsed as %+v", flow.Timeframes["15m"])
	}

	longShort, err := client.LongShortRatio(ctx)
	if err != nil {
		t.Fatalf("long-short: %v", err)
	}
	if longShort["BTC"].LongShortRatio != 1.18 {
		t.Fatalf("long-short parsed as %+v", longShort)
	}

	basis, err := client.Basis(ctx)
	if err != nil {
		t.Fatalf("basis: %v", err)
	}
	if basis["BTC"].BasisPct != 0.191 {
		t.Fatalf("basis parsed as %+v", basis)
	}

	whales, err := client.Whales(ctx)
	if err != nil {
		t.Fatalf("whales: %v", err)
	}
	if len(whales.Positions) != 2 || whales.Positions[0].DistancePct != 23.8 {
		t.Fatalf("whales parsed as %+v", whales)
	}
	if whales.ScanAgeSeconds == nil || *whales.ScanAgeSeconds != 42.5 {
		t.Fatalf("whales lost the scan age: %+v", whales.ScanAgeSeconds)
	}

	danger, err := client.DangerZone(ctx)
	if err != nil {
		t.Fatalf("danger zone: %v", err)
	}
	if danger.ThresholdPct != 8 || len(danger.Positions) != 1 {
		t.Fatalf("danger zone parsed as %+v", danger)
	}
}

func TestHyperDataClientToleratesUpstreamDrift(t *testing.T) {
	// Everything here is a change upstream is allowed to make without asking us:
	// a new field, a signal that became an object, a component list holding a
	// nested record, a null where a number used to be, and a version string with
	// a pre-release suffix.
	driftedHealth := `{
	  "status": "ok",
	  "failed_components": [{"name": "hlp", "reason": "slow"}],
	  "orderflow_venues": {"hyperliquid": {"status": "ok", "sockets_open": 4, "sockets_expected": 4, "brand_new_field": [1,2,3]}},
	  "version": "2.0.0-rc1",
	  "unknown_top_level": {"nested": true}
	}`
	driftedOrderflow := `{
	  "symbol": "BTC",
	  "cumulative_cvd": 500,
	  "cumulative_cvd_by_venue": {"hyperliquid": 500, "okx": null},
	  "venues_contributing": ["hyperliquid", {"exchange": "okx"}],
	  "signal_v2": {"label": "BUY", "confidence": 0.9},
	  "aggregate_signal": {"label": "BUY", "confidence": 0.9},
	  "timeframes": {"15m": {"buy_volume": 10, "sell_volume": 4, "net_volume": 6000, "trade_count": 3, "ofi": 0.3, "signal": {"label": "BUY"}}}
	}`
	driftedWhales := `{"count": 0, "as_of": null, "scan_age_seconds": null, "positions": []}`

	base := stubHyperData(t, map[string]stubRoute{
		"/v1/health":        {body: driftedHealth},
		"/v1/orderflow/BTC": {body: driftedOrderflow},
		"/v1/whales":        {body: driftedWhales},
	})

	client := NewHyperDataClient(base, "")
	ctx := context.Background()

	health, err := client.Health(ctx)
	if err != nil {
		t.Fatalf("drifted health rejected: %v", err)
	}
	if len(health.FailedComponents) != 1 || !strings.Contains(health.FailedComponents[0], "hlp") {
		t.Fatalf("non-string component entries should survive as compact JSON: %v", health.FailedComponents)
	}
	if warning := VersionWarning(health); warning == "" {
		t.Fatal("a 2.x sidecar should warn that the adapter was verified against 1.x")
	}

	flow, err := client.Orderflow(ctx, "BTC")
	if err != nil {
		t.Fatalf("drifted orderflow rejected: %v", err)
	}
	if !strings.Contains(flow.AggregateSignal.String(), "BUY") {
		t.Fatalf("an object signal should degrade to readable text, got %q", flow.AggregateSignal)
	}
	if len(flow.VenuesContributing) != 2 {
		t.Fatalf("a non-string venue entry should survive: %v", flow.VenuesContributing)
	}
	if flow.CumulativeCVDByVenue["okx"] != 0 {
		t.Fatal("a null venue value should read as zero rather than failing the decode")
	}

	whales, err := client.Whales(ctx)
	if err != nil {
		t.Fatalf("null timestamps rejected: %v", err)
	}
	if whales.AsOf != nil || whales.ScanAgeSeconds != nil {
		t.Fatalf("nulls should stay nil: %+v", whales)
	}
}

func TestHyperDataClientSurfacesUpstreamErrors(t *testing.T) {
	base := stubHyperData(t, map[string]stubRoute{
		"/v1/orderflow/BTC": {status: http.StatusNotFound, body: `{"error": "No orderflow data for BTC"}`},
		"/v1/orderflow/ETH": {status: http.StatusInternalServerError, body: ``},
		"/v1/orderflow/SOL": {status: http.StatusOK, body: `{not json`},
	})

	client := NewHyperDataClient(base, "")
	ctx := context.Background()

	if _, err := client.Orderflow(ctx, "BTC"); err == nil || !strings.Contains(err.Error(), "No orderflow data for BTC") {
		t.Fatalf("a documented per-symbol error should reach the caller verbatim, got %v", err)
	}
	if _, err := client.Orderflow(ctx, "ETH"); err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("a body-less failure should still report its status, got %v", err)
	}
	if _, err := client.Orderflow(ctx, "SOL"); err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("malformed JSON should be reported as a decode failure, got %v", err)
	}
}

func TestHyperDataClientReportsUnreachableSidecar(t *testing.T) {
	// A closed port is the normal state when the feature is enabled but the
	// process was never started, so it must read as a clean error, not a hang.
	client := NewHyperDataClient("http://127.0.0.1:1", "")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := client.Health(ctx); err == nil {
		t.Fatal("an unreachable sidecar should return an error")
	}
}

func TestHyperDataClientSendsAPIKeyWhenConfigured(t *testing.T) {
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("X-API-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok","version":"1.0.0"}`)
	}))
	t.Cleanup(server.Close)

	if _, err := NewHyperDataClient(server.URL, "secret").Health(context.Background()); err != nil {
		t.Fatalf("health: %v", err)
	}
	if seen != "secret" {
		t.Fatalf("X-API-Key = %q, want the configured key", seen)
	}
}

func TestVersionWarningOnlyFiresOnMajorMismatch(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		{"1.0.0", false},
		{"1.9.9", false},
		{"2.0.0", true},
		{"2.0.0-rc1", true},
		{"", false},
		{"unversioned", false},
	}
	for _, tc := range cases {
		got := VersionWarning(&HyperDataHealth{Version: tc.version})
		if (got != "") != tc.want {
			t.Fatalf("version %q produced warning %q, want warn=%v", tc.version, got, tc.want)
		}
	}
}

// --- Markdown table integrity --------------------------------------------
//
// Prompt tables are read positionally, so a row with the wrong number of cells
// silently reassigns every value to its right. That is a quiet way to feed the
// model wrong numbers, and it is exactly what happened once while this file was
// being written, so it is asserted rather than eyeballed.

// assertTablesAligned fails when any markdown table in the block has a row or
// separator whose column count differs from its header.
func assertTablesAligned(t *testing.T, markdown string) {
	t.Helper()

	var header []string
	inTable := false

	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			header, inTable = nil, false
			continue
		}

		current := tableCells(trimmed)
		switch {
		case !inTable:
			header, inTable = current, true
		case isSeparatorRow(current):
			if len(current) != len(header) {
				t.Fatalf("separator has %d columns, header has %d:\n%s", len(current), len(header), markdown)
			}
		default:
			if len(current) != len(header) {
				t.Fatalf("row %v has %d columns, header %v has %d:\n%s",
					current, len(current), header, len(header), markdown)
			}
		}
	}
}

// tableCells splits a markdown table line into trimmed cells.
func tableCells(line string) []string {
	parts := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}

// isSeparatorRow reports whether a row is the "| ---: |" alignment row.
func isSeparatorRow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		if strings.Trim(cell, ":-") != "" {
			return false
		}
	}
	return true
}

// --- Provider behaviour ---------------------------------------------------

func TestHyperDataProvidersAreOptIn(t *testing.T) {
	for _, provider := range []marketdata.Provider{
		NewHyperDataOrderflowProvider(),
		NewHyperDataPositioningProvider(),
	} {
		if provider.Enabled(store.IndicatorConfig{}) {
			t.Fatalf("%s must stay off until the sidecar is requested", provider.Name())
		}
		if provider.Enabled(store.IndicatorConfig{EnableHyperData: true}) != true {
			t.Fatalf("%s should run once the sidecar is enabled", provider.Name())
		}
		// It is locally hosted, so it needs no credential — but the UI still has
		// to be able to say that something has to be started.
		if provider.RequiresAPIKey() {
			t.Fatalf("%s should not claim to need a key", provider.Name())
		}
		service, ok := provider.(marketdata.ServiceBacked)
		if !ok || strings.TrimSpace(service.RequiredService()) == "" {
			t.Fatalf("%s should name the service it needs", provider.Name())
		}
		if !strings.Contains(provider.Description(), "HyperData") {
			t.Fatalf("%s description should say where the data comes from", provider.Name())
		}
	}
}

func TestOrderflowProviderRendersVenueHonesty(t *testing.T) {
	base := stubHyperData(t, map[string]stubRoute{
		"/v1/health":           {body: fixtureHealth},
		"/v1/orderflow/ETH":    {body: fixtureOrderflowETH},
		"/v1/orderflow/BTC":    {status: http.StatusNotFound, body: `{"error": "No orderflow data for BTC"}`},
		"/v1/orderflow/SOL":    {status: http.StatusNotFound, body: `{"error": "No orderflow data for SOL"}`},
		"/v1/long-short-ratio": {body: fixtureLongShort},
		"/v1/basis":            {body: fixtureBasis},
	})

	provider := NewHyperDataOrderflowProvider()
	insight, err := provider.Fetch(context.Background(), marketdata.Request{
		Symbols: []string{"BTC", "ETH", "SOL"},
	}, hyperDataConfig(base))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if insight == nil {
		t.Fatal("expected an insight")
	}

	// Venue coverage must be stated, and the quiet Binance stream must be named
	// rather than silently folded into a single-venue number.
	for _, want := range []string{
		"binance", "0/2 sockets", "2 symbols dark",
		"single-venue evidence",
		"Not tracked by this instance", "BTC", "SOL",
		"Leverage sentiment", "Perpetual basis",
	} {
		if !strings.Contains(insight.Markdown, want) {
			t.Fatalf("order-flow block missing %q:\n%s", want, insight.Markdown)
		}
	}

	payload, ok := insight.Payload.(OrderflowPayload)
	if !ok {
		t.Fatalf("payload type = %T", insight.Payload)
	}
	if len(payload.Rows) != 1 || payload.Rows[0].Symbol != "ETH" {
		t.Fatalf("rows = %+v, want just the tracked symbol", payload.Rows)
	}
	if payload.Rows[0].NetVolumeWindow != "5m" {
		t.Fatalf("window = %q, want the shortest reported window", payload.Rows[0].NetVolumeWindow)
	}
	if len(payload.Unavailable) != 2 {
		t.Fatalf("unavailable = %v, want BTC and SOL recorded", payload.Unavailable)
	}

	assertTablesAligned(t, insight.Markdown)
}

func TestOrderflowProviderFailsSoftWithoutSidecar(t *testing.T) {
	provider := NewHyperDataOrderflowProvider()
	_, err := provider.Fetch(context.Background(), marketdata.Request{Symbols: []string{"BTC"}},
		hyperDataConfig("http://127.0.0.1:1"))
	if err == nil {
		t.Fatal("an unreachable sidecar should surface as an error so the registry can report the gap")
	}
	if !strings.Contains(err.Error(), "hyperdata") {
		t.Fatalf("error should name the source: %v", err)
	}
}

func TestOrderflowProviderRejectsDegradedSidecar(t *testing.T) {
	base := stubHyperData(t, map[string]stubRoute{
		"/v1/health": {body: `{"status": "degraded", "version": "1.0.0", "failed_components": ["orderflow"]}`},
	})

	_, err := NewHyperDataOrderflowProvider().Fetch(context.Background(),
		marketdata.Request{Symbols: []string{"BTC"}}, hyperDataConfig(base))
	if err == nil || !strings.Contains(err.Error(), "degraded") {
		t.Fatalf("a degraded sidecar should not be queried silently, got %v", err)
	}
}

func TestOrderflowProviderToleratesHealthWarn(t *testing.T) {
	// "warn" means one venue is quiet, which is exactly the case the block is
	// built to describe honestly — it must not be treated as a failure.
	base := stubHyperData(t, map[string]stubRoute{
		"/v1/health":           {body: fixtureHealth},
		"/v1/orderflow/ETH":    {body: fixtureOrderflowETH},
		"/v1/long-short-ratio": {body: fixtureLongShort},
		"/v1/basis":            {body: fixtureBasis},
	})

	insight, err := NewHyperDataOrderflowProvider().Fetch(context.Background(),
		marketdata.Request{Symbols: []string{"ETH"}}, hyperDataConfig(base))
	if err != nil {
		t.Fatalf("warn status should still produce a block: %v", err)
	}
	if insight == nil {
		t.Fatal("expected an insight")
	}
}

func TestPositioningProviderStatesSamplingCaveat(t *testing.T) {
	base := stubHyperData(t, map[string]stubRoute{
		"/v1/health":                {body: fixtureHealth},
		"/v1/whales":                {body: fixtureWhales},
		"/v1/positions/danger-zone": {body: fixtureDangerZone},
	})

	insight, err := NewHyperDataPositioningProvider().Fetch(context.Background(),
		marketdata.Request{}, hyperDataConfig(base))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if insight == nil {
		t.Fatal("expected an insight")
	}

	// The single most important thing this block must not do is imply it holds
	// the market's total positioning.
	for _, want := range []string{
		"Sample, not a census",
		"discovered",
		"never as the market's total open interest",
		"Position scan is",
		"s old",
		"Largest tracked positions",
		"Closest to liquidation",
		"within 8.0% of liquidation",
	} {
		if !strings.Contains(insight.Markdown, want) {
			t.Fatalf("positioning block missing %q:\n%s", want, insight.Markdown)
		}
	}

	payload, ok := insight.Payload.(PositioningPayload)
	if !ok {
		t.Fatalf("payload type = %T", insight.Payload)
	}
	if len(payload.Largest) != 2 || payload.Largest[0].Symbol != "BTC" {
		t.Fatalf("largest = %+v", payload.Largest)
	}
	if len(payload.NearLiquidation) != 1 || payload.NearLiquidation[0].DistancePct != 6.15 {
		t.Fatalf("near liquidation = %+v", payload.NearLiquidation)
	}
	if payload.DangerThreshold != 8 {
		t.Fatalf("danger threshold = %v", payload.DangerThreshold)
	}

	assertTablesAligned(t, insight.Markdown)
}

func TestPositioningTableKeepsDistanceInItsOwnColumn(t *testing.T) {
	// Distance sits between Liq and Lev in the header; building the row out of
	// order would misalign everything to its right.
	rows := []PositioningRow{{
		Symbol: "ETH", Side: "short", SizeUSD: 900000, EntryPrice: 3300,
		MarkPrice: 3250, LiqPrice: 3450, DistancePct: 6.15, Leverage: 10, UnrealizedPnL: 13000,
	}}

	for _, withDistance := range []bool{true, false} {
		rendered := positioningTable(rows, withDistance)
		assertTablesAligned(t, rendered)

		if !withDistance {
			continue
		}
		lines := strings.Split(strings.TrimSpace(rendered), "\n")
		header := tableCells(lines[0])
		row := tableCells(lines[2])

		liqIndex := -1
		for i, cell := range header {
			if cell == "Liq" {
				liqIndex = i
			}
		}
		if liqIndex < 0 || header[liqIndex+1] != "Distance" {
			t.Fatalf("distance is not adjacent to Liq in the header: %v", header)
		}
		if row[liqIndex+1] != "6.15%" {
			t.Fatalf("distance value landed outside the distance column: %v", row)
		}
	}
}

func TestPositioningProviderFailsOnlyWhenBothEndpointsFail(t *testing.T) {
	// A failing danger-zone must not discard the whale table, which is the
	// larger half of the block.
	base := stubHyperData(t, map[string]stubRoute{
		"/v1/whales":                {body: fixtureWhales},
		"/v1/positions/danger-zone": {status: http.StatusInternalServerError, body: ``},
	})

	insight, err := NewHyperDataPositioningProvider().Fetch(context.Background(),
		marketdata.Request{}, hyperDataConfig(base))
	if err != nil {
		t.Fatalf("a single failing endpoint should not discard the block: %v", err)
	}
	if !strings.Contains(insight.Markdown, "Largest tracked positions") {
		t.Fatalf("whale table missing:\n%s", insight.Markdown)
	}
	if !strings.Contains(insight.Markdown, "danger zone unavailable") {
		t.Fatalf("the gap should be stated:\n%s", insight.Markdown)
	}
}

func TestHyperDataSettingsPreferStrategyThenEnvironment(t *testing.T) {
	// The environment fallback exists so a deployment configures the sidecar
	// once; the strategy still wins so per-strategy routing stays possible.
	t.Setenv(HyperDataBaseURLEnv, "http://env:8420")
	t.Setenv(HyperDataAPIKeyEnv, "env-key")

	base, key := hyperDataSettings(store.IndicatorConfig{})
	if base != "http://env:8420" || key != "env-key" {
		t.Fatalf("env fallback = %q / %q", base, key)
	}

	base, key = hyperDataSettings(store.IndicatorConfig{
		HyperDataBaseURL: "http://strategy:1",
		HyperDataAPIKey:  "strategy-key",
	})
	if base != "http://strategy:1" || key != "strategy-key" {
		t.Fatalf("strategy settings should win, got %q / %q", base, key)
	}

	// With nothing configured anywhere the client falls back to the documented
	// local default rather than an empty base URL.
	t.Setenv(HyperDataBaseURLEnv, "")
	t.Setenv(HyperDataAPIKeyEnv, "")
	base, key = hyperDataSettings(store.IndicatorConfig{})
	if base != "" || key != "" {
		t.Fatalf("unset settings should resolve empty, got %q / %q", base, key)
	}
	if got := NewHyperDataClient(base, key).BaseURL(); got != DefaultHyperDataBaseURL {
		t.Fatalf("client default = %q, want %q", got, DefaultHyperDataBaseURL)
	}
}

func TestHyperDataTargetsNormalizesAndCaps(t *testing.T) {
	got := hyperDataTargets(marketdata.Request{
		Positions: []string{"xyz:TSLA"},
		Symbols:   []string{"btc", "BTC", " eth ", "SOL", "XRP", "DOGE"},
	}, 4)

	want := []string{"TSLA", "BTC", "ETH", "SOL"}
	if len(got) != len(want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("targets = %v, want %v", got, want)
		}
	}
}

func TestOrderflowRowPrefersShortestReportedWindow(t *testing.T) {
	var flow HyperDataOrderflow
	if err := json.Unmarshal([]byte(fixtureOrderflowBTC), &flow); err != nil {
		t.Fatalf("fixture: %v", err)
	}

	row := orderflowRow(flow)
	if row.NetVolumeWindow != "15m" {
		t.Fatalf("window = %q, want 15m (5m was not reported)", row.NetVolumeWindow)
	}
	if row.NetVolume != 300000 {
		t.Fatalf("net volume = %v", row.NetVolume)
	}
	if !row.MetricsAvailable {
		t.Fatal("a reported window should mark the metrics available")
	}
	if row.CVDHyperliquid != 234567 || row.CVDBinance != 1000000 {
		t.Fatalf("venue split lost: %+v", row)
	}
}
