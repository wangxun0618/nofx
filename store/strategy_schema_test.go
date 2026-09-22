package store

import (
	"encoding/json"
	"testing"
)

func TestStrategyConfigMarshalSeparatesGridAndAIConfig(t *testing.T) {
	cfg := GetDefaultStrategyConfig("zh")
	cfg.StrategyType = "grid_trading"
	cfg.GridConfig = &GridStrategyConfig{
		Symbol:          "BTCUSDT",
		GridCount:       20,
		TotalInvestment: 200,
		Leverage:        2,
		UseATRBounds:    true,
		ATRMultiplier:   2,
		Distribution:    "uniform",
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal grid config: %v", err)
	}

	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatalf("unmarshal grid config map: %v", err)
	}
	if asMap["strategy_type"] != "grid_trading" {
		t.Fatalf("expected grid strategy_type, got %v", asMap["strategy_type"])
	}
	if _, ok := asMap["grid_config"]; !ok {
		t.Fatalf("expected grid_config in grid strategy JSON: %s", string(raw))
	}
	for _, key := range []string{"ai_config", "coin_source", "indicators", "risk_control", "prompt_sections", "custom_prompt"} {
		if _, ok := asMap[key]; ok {
			t.Fatalf("did not expect %s in grid strategy JSON: %s", key, string(raw))
		}
	}
}

func TestStrategyConfigUnmarshalLegacyFlatAIConfig(t *testing.T) {
	raw := []byte(`{
		"strategy_type":"ai_trading",
		"coin_source":{"source_type":"static","static_coins":["ETHUSDT"]},
		"indicators":{"klines":{"primary_timeframe":"15m"}},
		"risk_control":{"max_positions":2,"min_confidence":80},
		"prompt_sections":{"entry_standards":"trend only"},
		"custom_prompt":"prefer ETH"
	}`)

	var cfg StrategyConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("unmarshal legacy flat config: %v", err)
	}
	if cfg.CoinSource.SourceType != "static" || len(cfg.CoinSource.StaticCoins) != 1 || cfg.CoinSource.StaticCoins[0] != "ETHUSDT" {
		t.Fatalf("legacy coin source was not normalized: %+v", cfg.CoinSource)
	}
	if cfg.Indicators.Klines.PrimaryTimeframe != "15m" {
		t.Fatalf("legacy indicators were not normalized: %+v", cfg.Indicators.Klines)
	}

	normalized, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal normalized config: %v", err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(normalized, &asMap); err != nil {
		t.Fatalf("unmarshal normalized map: %v", err)
	}
	if _, ok := asMap["ai_config"]; !ok {
		t.Fatalf("expected ai_config after normalizing legacy config: %s", string(normalized))
	}
	if _, ok := asMap["coin_source"]; ok {
		t.Fatalf("did not expect legacy coin_source at top level: %s", string(normalized))
	}
}

func TestStrategyConfigNormalizeProductSchemaForLLMLabels(t *testing.T) {
	cfg := GetDefaultStrategyConfig("zh")
	patch := map[string]any{
		"strategy_type": "AI strategy",
		"ai_config": map[string]any{
			"coin_source": map[string]any{
				"source_type": "AI500",
			},
			"indicators": map[string]any{
				"klines": map[string]any{
					"primary_timeframe":   "1 minute",
					"selected_timeframes": []any{`["1m"`, `"5m"`, `"15m"]`},
				},
			},
		},
	}

	merged, err := MergeStrategyConfig(cfg, patch)
	if err != nil {
		t.Fatalf("merge strategy config: %v", err)
	}
	merged.ClampLimits()

	if merged.StrategyType != "ai_trading" {
		t.Fatalf("strategy_type = %q, want ai_trading", merged.StrategyType)
	}
	if merged.CoinSource.SourceType != "ai500" {
		t.Fatalf("source_type = %q, want ai500", merged.CoinSource.SourceType)
	}
	if !merged.CoinSource.UseAI500 || merged.CoinSource.UseOITop || merged.CoinSource.UseOILow {
		t.Fatalf("coin source flags not normalized: %+v", merged.CoinSource)
	}
	if merged.Indicators.Klines.PrimaryTimeframe != "1m" {
		t.Fatalf("primary_timeframe = %q, want 1m", merged.Indicators.Klines.PrimaryTimeframe)
	}
	want := []string{"1m", "5m", "15m"}
	if len(merged.Indicators.Klines.SelectedTimeframes) != len(want) {
		t.Fatalf("selected_timeframes = %+v, want %+v", merged.Indicators.Klines.SelectedTimeframes, want)
	}
	for i := range want {
		if merged.Indicators.Klines.SelectedTimeframes[i] != want[i] {
			t.Fatalf("selected_timeframes = %+v, want %+v", merged.Indicators.Klines.SelectedTimeframes, want)
		}
	}
}

func TestStrategyConfigNormalizeProductSchemaForRetiredVergexBoard(t *testing.T) {
	cfg := GetDefaultStrategyConfig("zh")
	cfg.CoinSource = CoinSourceConfig{
		SourceType: "Claw402 Vergex signal board",
	}

	cfg.NormalizeProductSchema()

	if cfg.CoinSource.SourceType != "hyper_main" || !cfg.CoinSource.UseHyperMain {
		t.Fatalf("retired direction board should fall back to hyper_main, got %+v", cfg.CoinSource)
	}
	if cfg.CoinSource.HyperMainLimit != 30 {
		t.Fatalf("hyper_main_limit = %d, want 30", cfg.CoinSource.HyperMainLimit)
	}
}

func TestStrategyConfigNormalizeProductSchemaForHyperMainLimits(t *testing.T) {
	t.Run("zero limit falls back to the thirty-symbol volume board", func(t *testing.T) {
		cfg := GetDefaultStrategyConfig("zh")
		cfg.CoinSource = CoinSourceConfig{
			SourceType:     "hyper_main",
			HyperMainLimit: 0,
			StaticCoins:    nil,
		}

		cfg.NormalizeProductSchema()

		if cfg.CoinSource.HyperMainLimit != 30 {
			t.Fatalf("hyper_main_limit = %d, want 30", cfg.CoinSource.HyperMainLimit)
		}
	})

	t.Run("static coins are normalized to upper-case symbols", func(t *testing.T) {
		cfg := GetDefaultStrategyConfig("zh")
		cfg.CoinSource = CoinSourceConfig{
			SourceType:  "static",
			StaticCoins: []string{"xyz:nvda", "XYZ:AAPL"},
		}

		cfg.NormalizeProductSchema()

		if cfg.CoinSource.SourceType != "static" {
			t.Fatalf("source_type = %q, want static", cfg.CoinSource.SourceType)
		}
		if got := cfg.CoinSource.StaticCoins; len(got) != 2 || got[0] != "XYZ:NVDA" || got[1] != "XYZ:AAPL" {
			t.Fatalf("static_coins = %+v, want normalized xyz symbols", got)
		}
	})
}
