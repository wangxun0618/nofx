package kernel

import (
	"strings"
	"testing"

	"nofx/store"
)

// TestBuildSystemPromptUsesHyperliquidAutoTraderPrompt pins the canonical
// system prompt contract. The strategy engine no longer has a paid-gateway
// (Claw402/Vergex/x402) prompt flavour: every strategy, whether created with
// the English or the Chinese UI, must produce the same generic
// Hyperliquid-native auto-trader prompt.
func TestBuildSystemPromptUsesHyperliquidAutoTraderPrompt(t *testing.T) {
	for _, lang := range []string{"en", "zh"} {
		t.Run(lang, func(t *testing.T) {
			cfg := store.GetDefaultStrategyConfig(lang)

			engine := NewStrategyEngine(&cfg)
			prompt := engine.BuildSystemPrompt(30, "balanced")

			required := []string{
				"Data Dictionary & Trading Rules",
				"NOFX auto-trader",
				"Trade only the Hyperliquid instruments presented in this cycle's candidate list",
			}
			for _, phrase := range required {
				if !strings.Contains(prompt, phrase) {
					t.Fatalf("prompt missing %q:\n%s", phrase, prompt)
				}
			}
			if containsCJK(prompt) {
				t.Fatalf("system prompt must be English-only, got CJK text:\n%s", prompt)
			}
			for _, retired := range []string{"Claw402", "claw402", "Vergex", "vergex", "x402", "Direction Board", "Signal Lab"} {
				if strings.Contains(prompt, retired) {
					t.Fatalf("prompt still references the retired payment/data gateway (%q):\n%s", retired, prompt)
				}
			}
		})
	}
}

// TestBuildSystemPromptDropsChineseCustomSections verifies that stored prompt
// sections written in Chinese are discarded in favour of the built-in English
// fallbacks, so the model contract never mixes languages.
func TestBuildSystemPromptDropsChineseCustomSections(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("zh")
	cfg.CoinSource.SourceType = "static"
	cfg.CoinSource.StaticCoins = []string{"BTCUSDT", "ETHUSDT"}
	cfg.PromptSections.RoleDefinition = "# 你是一个中文系统提示"
	cfg.PromptSections.TradingFrequency = "# 高频交易\n每分钟交易一次。"
	cfg.PromptSections.EntryStandards = "# 入场\n自由开仓。"
	cfg.PromptSections.DecisionProcess = "# 决策\n直接输出。"
	cfg.CustomPrompt = "中文偏好不应进入系统提示。"

	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(30, "balanced")

	required := []string{
		"Data Dictionary & Trading Rules",
		"You are a professional Hyperliquid USDC multi-asset trading AI",
		"Trading Frequency Awareness",
		"Entry Standards",
		"Decision Process",
	}
	for _, phrase := range required {
		if !strings.Contains(prompt, phrase) {
			t.Fatalf("English fallback prompt missing %q:\n%s", phrase, prompt)
		}
	}
	if containsCJK(prompt) {
		t.Fatalf("system prompt must be English-only, got CJK text:\n%s", prompt)
	}
}

// TestBuildSystemPromptKeepsEnglishCustomSections verifies that an
// English-language custom override is still honoured after the legacy
// whole-config wipe was removed.
func TestBuildSystemPromptKeepsEnglishCustomSections(t *testing.T) {
	cfg := store.GetDefaultStrategyConfig("zh")
	cfg.CoinSource.SourceType = "static"
	cfg.CoinSource.StaticCoins = []string{"BTCUSDT", "ETHUSDT"}
	cfg.PromptSections.RoleDefinition = "# You are a disciplined systematic trader"
	cfg.CustomPrompt = "Prefer fewer, higher-quality entries."

	engine := NewStrategyEngine(&cfg)
	prompt := engine.BuildSystemPrompt(30, "balanced")

	for _, phrase := range []string{
		"# You are a disciplined systematic trader",
		"Prefer fewer, higher-quality entries.",
	} {
		if !strings.Contains(prompt, phrase) {
			t.Fatalf("prompt dropped English custom section %q:\n%s", phrase, prompt)
		}
	}
	if containsCJK(prompt) {
		t.Fatalf("system prompt must be English-only, got CJK text:\n%s", prompt)
	}
}

func TestBuildSystemPromptDoesNotForceLongOnlyForSingleXYZ(t *testing.T) {
	prompt := buildXYZStockCustomPrompt("XYZ:INTC")

	required := []string{
		"DIRECTIONAL, SIGNAL-DRIVEN",
		"You may open long or short",
		"open_short",
	}
	for _, phrase := range required {
		if !strings.Contains(prompt, phrase) {
			t.Fatalf("single XYZ prompt missing %q:\n%s", phrase, prompt)
		}
	}

	forbidden := []string{
		"LONG-ONLY",
		"Do not short",
		"MUST open a long",
		"Probing > waiting",
		"Claw402",
	}
	for _, phrase := range forbidden {
		if strings.Contains(prompt, phrase) {
			t.Fatalf("single XYZ prompt still contains forced-long phrase %q:\n%s", phrase, prompt)
		}
	}
}

func containsCJK(text string) bool {
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}
