package store

import "testing"

func TestDefaultStrategyUsesHyperliquidVolumeUniverse(t *testing.T) {
	cfg := GetDefaultStrategyConfig("zh")
	assertHyperMainDefault(t, cfg)

	ind := cfg.Indicators
	if ind.NofxOSAPIKey != "" {
		t.Fatalf("default should not include a NofxOS API key")
	}
	if ind.EnableQuantData || ind.EnableQuantOI || ind.EnableQuantNetflow {
		t.Fatalf("default strategy must not enable NofxOS datasets: %+v", ind)
	}
	if !ind.EnableMarketInsights {
		t.Fatalf("default strategy must enable the free market-insight sources")
	}
	if len(ind.MarketInsightSources) != 0 {
		t.Fatalf("default source allow-list must be empty (= run every registered provider), got %v", ind.MarketInsightSources)
	}
	if ind.CoinankAPIKey != "" {
		t.Fatalf("default strategy must not ship a third-party CoinAnk key")
	}
	if !ind.EnableRawKlines {
		t.Fatalf("raw Hyperliquid klines must stay enabled")
	}
}

func TestHyperMainDefaultSurvivesClampAndNormalize(t *testing.T) {
	cfg := GetDefaultStrategyConfig("zh")
	cfg.CoinSource.UseAI500 = true
	cfg.ClampLimits()
	assertHyperMainDefault(t, cfg)
	if cfg.CoinSource.UseAI500 {
		t.Fatalf("hyper_main strategy must clear the stale AI500 flag: %+v", cfg.CoinSource)
	}
}

func TestEmptyCoinSourceInfersHyperMain(t *testing.T) {
	cfg := GetDefaultStrategyConfig("zh")
	cfg.CoinSource = CoinSourceConfig{}
	cfg.NormalizeProductSchema()
	assertHyperMainDefault(t, cfg)
}

func assertHyperMainDefault(t *testing.T, cfg StrategyConfig) {
	t.Helper()
	if cfg.CoinSource.SourceType != "hyper_main" || !cfg.CoinSource.UseHyperMain || cfg.CoinSource.HyperMainLimit != 30 {
		t.Fatalf("coin source = %+v, want the Hyperliquid native top-30 volume universe", cfg.CoinSource)
	}
	if cfg.RiskControl.MaxPositions != AutopilotDefaultMaxPositions {
		t.Fatalf("max positions = %d, want %d", cfg.RiskControl.MaxPositions, AutopilotDefaultMaxPositions)
	}
	if cfg.RiskControl.BTCETHMaxPositionValueRatio != AutopilotMaxPositionValueRatio ||
		cfg.RiskControl.AltcoinMaxPositionValueRatio != AutopilotMaxPositionValueRatio {
		t.Fatalf("position value ratios = %+v, want the Autopilot hard cap", cfg.RiskControl)
	}
}
