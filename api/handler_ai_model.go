package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"nofx/config"
	"nofx/crypto"
	"nofx/logger"
	"nofx/security"
	"nofx/store"

	"github.com/gin-gonic/gin"
)

type ModelConfig struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	Enabled      bool   `json:"enabled"`
	APIKey       string `json:"apiKey,omitempty"`
	CustomAPIURL string `json:"customApiUrl,omitempty"`
}

// SafeModelConfig Safe model configuration structure (does not contain sensitive information)
type SafeModelConfig struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Provider        string `json:"provider"`
	Enabled         bool   `json:"enabled"`
	HasAPIKey       bool   `json:"has_api_key"`
	CustomAPIURL    string `json:"customApiUrl"`    // Custom API URL (usually not sensitive)
	CustomModelName string `json:"customModelName"` // Custom model name (not sensitive)
}

// ModelConfigUpdate is a single model's update payload. It is a named type
// (rather than an inline anonymous struct) so the log-sanitizer in utils.go is
// guaranteed to stay in sync with this shape — a mismatch there is what let
// plaintext credentials reach the logs previously.
type ModelConfigUpdate struct {
	Enabled         bool   `json:"enabled"`
	APIKey          string `json:"api_key"`
	CustomAPIURL    string `json:"custom_api_url"`
	CustomModelName string `json:"custom_model_name"`
}

type UpdateModelConfigRequest struct {
	Models map[string]ModelConfigUpdate `json:"models"`
}

// supportedProviderDefaults is the single source of truth for the AI providers
// the system can talk to. Every entry maps to a native client registered in
// mcp/provider/* (see mcp.ProviderXxx constants).
var supportedProviderDefaults = []struct {
	ID           string
	Name         string
	Provider     string
	DefaultModel string
}{
	{ID: "deepseek", Name: "DeepSeek", Provider: "deepseek", DefaultModel: "deepseek-chat"},
	{ID: "openai", Name: "OpenAI", Provider: "openai", DefaultModel: "gpt-4o"},
	{ID: "claude", Name: "Claude", Provider: "claude", DefaultModel: "claude-sonnet-4-20250514"},
	{ID: "qwen", Name: "Qwen", Provider: "qwen", DefaultModel: "qwen3-max"},
	{ID: "gemini", Name: "Gemini", Provider: "gemini", DefaultModel: "gemini-2.5-pro"},
	{ID: "grok", Name: "Grok", Provider: "grok", DefaultModel: "grok-4"},
	{ID: "kimi", Name: "Kimi", Provider: "kimi", DefaultModel: "kimi-k2-0905-preview"},
	{ID: "minimax", Name: "MiniMax", Provider: "minimax", DefaultModel: "MiniMax-M2.7"},
}

// isSupportedProvider reports whether a provider still has a native client.
// Rows persisted by older releases (e.g. the retired "claw402" gateway) are
// kept in the database for audit but must never be offered to users, because
// no runtime client exists for them any more.
func isSupportedProvider(provider string) bool {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	if normalized == "" {
		return false
	}
	for _, p := range supportedProviderDefaults {
		if p.Provider == normalized || p.ID == normalized {
			return true
		}
	}
	return false
}

// defaultModelConfigs builds the "no configuration saved yet" response so the
// frontend can render an empty, editable model grid.
func defaultModelConfigs() []SafeModelConfig {
	models := make([]SafeModelConfig, 0, len(supportedProviderDefaults))
	for _, p := range supportedProviderDefaults {
		models = append(models, SafeModelConfig{
			ID:       p.ID,
			Name:     p.Name,
			Provider: p.Provider,
			Enabled:  false,
		})
	}
	return models
}

// handleGetModelConfigs Get AI model configurations
func (s *Server) handleGetModelConfigs(c *gin.Context) {
	userID := c.GetString("user_id")
	logger.Infof("🔍 Querying AI model configs for user %s", userID)
	models, err := s.store.AIModel().List(userID)
	if err != nil {
		logger.Infof("❌ Failed to get AI model configs: %v", err)
		SafeInternalError(c, "Failed to get AI model configs", err)
		return
	}

	// If no models in database, return the provider defaults so the UI can
	// still offer every supported provider.
	if len(models) == 0 {
		logger.Infof("⚠️ No AI models in database, returning defaults")
		c.JSON(http.StatusOK, defaultModelConfigs())
		return
	}

	logger.Infof("✅ Found %d AI model configs", len(models))

	// Convert to safe response structure, remove sensitive information
	safeModels := make([]SafeModelConfig, 0, len(models))
	for _, model := range models {
		if !store.IsVisibleAIModel(model) {
			continue
		}
		if !isSupportedProvider(model.Provider) {
			continue
		}
		safeModels = append(safeModels, SafeModelConfig{
			ID:              model.ID,
			Name:            model.Name,
			Provider:        model.Provider,
			Enabled:         model.Enabled,
			HasAPIKey:       model.APIKey != "",
			CustomAPIURL:    model.CustomAPIURL,
			CustomModelName: model.CustomModelName,
		})
	}

	if len(safeModels) == 0 {
		logger.Infof("⚠️ No visible AI models in database, returning defaults")
		c.JSON(http.StatusOK, defaultModelConfigs())
		return
	}

	c.JSON(http.StatusOK, safeModels)
}

// handleUpdateModelConfigs Update AI model configurations (supports both encrypted and plain text based on config)
func (s *Server) handleUpdateModelConfigs(c *gin.Context) {
	userID := c.GetString("user_id")
	cfg := config.Get()

	// Read raw request body
	bodyBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	var req UpdateModelConfigRequest

	// Check if transport encryption is enabled
	if !cfg.TransportEncryption {
		// Transport encryption disabled, accept plain JSON
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			logger.Infof("❌ Failed to parse plain JSON request: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}
		logger.Infof("📝 Received plain text model config (UserID: %s)", userID)
	} else {
		// Transport encryption enabled, require encrypted payload
		var encryptedPayload crypto.EncryptedPayload
		if err := json.Unmarshal(bodyBytes, &encryptedPayload); err != nil {
			logger.Infof("❌ Failed to parse encrypted payload: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format, encrypted transmission required"})
			return
		}

		// Verify encrypted data
		if encryptedPayload.WrappedKey == "" {
			logger.Infof("❌ Detected unencrypted request (UserID: %s)", userID)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "This endpoint only supports encrypted transmission, please use encrypted client",
				"code":    "ENCRYPTION_REQUIRED",
				"message": "Encrypted transmission is required for security reasons",
			})
			return
		}

		// Decrypt data
		decrypted, err := s.cryptoHandler.cryptoService.DecryptSensitiveData(&encryptedPayload)
		if err != nil {
			logger.Infof("❌ Failed to decrypt model config (UserID: %s): %v", userID, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decrypt data"})
			return
		}

		// Parse decrypted data
		if err := json.Unmarshal([]byte(decrypted), &req); err != nil {
			logger.Infof("❌ Failed to parse decrypted data: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse decrypted data"})
			return
		}
		logger.Infof("🔓 Decrypted model config data (UserID: %s)", userID)
	}

	// Update each model's configuration and track traders that need reload.
	// The request key may be either the model row id or the provider name
	// (legacy clients send the provider, e.g. "deepseek", while trader rows
	// reference the full model id) — resolve both, mirroring the matching in
	// AIModelStore.Update, otherwise running traders keep the old model.
	modelIDCandidates := func(modelID string) map[string]bool {
		candidates := map[string]bool{modelID: true}
		if models, listErr := s.store.AIModel().List(userID); listErr == nil {
			for _, m := range models {
				if m.ID == modelID || m.Provider == modelID {
					candidates[m.ID] = true
					candidates[m.Provider] = true
				}
			}
		}
		return candidates
	}

	tradersToReload := make(map[string]bool)
	for modelID, modelData := range req.Models {
		candidates := modelIDCandidates(modelID)

		// Reject rows for providers that no longer have a native client, so a
		// stale cached UI or agent cannot resurrect a retired gateway.
		supported := false
		for candidateID := range candidates {
			if isSupportedProvider(candidateID) {
				supported = true
				break
			}
		}
		if !supported {
			logger.Warnf("Skipping AI model config update for unsupported provider %q", modelID)
			continue
		}

		// SSRF protection: validate custom_api_url before storing
		if modelData.CustomAPIURL != "" {
			cleanURL := strings.TrimSuffix(modelData.CustomAPIURL, "#")
			if err := security.ValidateURL(cleanURL); err != nil {
				logger.Warnf("Invalid custom_api_url for model %s: %v", modelID, err)
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid custom_api_url for model %s: URL must be a valid HTTPS endpoint", modelID)})
				return
			}
		}

		// Find traders using this AI model BEFORE updating
		for candidateID := range candidates {
			traders, _ := s.store.Trader().ListByAIModelID(userID, candidateID)
			for _, t := range traders {
				tradersToReload[t.ID] = true
			}
		}

		err := s.store.AIModel().Update(userID, modelID, modelData.Enabled, modelData.APIKey, modelData.CustomAPIURL, modelData.CustomModelName)
		if err != nil {
			SafeInternalError(c, fmt.Sprintf("Update model %s", modelID), err)
			return
		}
	}

	// Remove affected traders from memory BEFORE reloading to pick up new config
	for traderID := range tradersToReload {
		logger.Infof("🔄 Removing trader %s from memory to reload with new AI model config", traderID)
		s.traderManager.RemoveTrader(traderID)
	}

	// Reload all traders for this user to make new config take effect immediately
	err = s.traderManager.LoadUserTradersFromStore(s.store, userID)
	if err != nil {
		logger.Infof("⚠️ Failed to reload user traders into memory: %v", err)
		// Don't return error here since model config was successfully updated to database
	}

	logger.Infof("✓ AI model config updated: %+v", SanitizeModelConfigForLog(req.Models))
	c.JSON(http.StatusOK, gin.H{"message": "Model configuration updated"})
}

// handleGetSupportedModels Get list of AI models supported by the system
func (s *Server) handleGetSupportedModels(c *gin.Context) {
	supportedModels := make([]map[string]interface{}, 0, len(supportedProviderDefaults))
	for _, p := range supportedProviderDefaults {
		supportedModels = append(supportedModels, map[string]interface{}{
			"id":           p.ID,
			"name":         p.Name,
			"provider":     p.Provider,
			"defaultModel": p.DefaultModel,
		})
	}

	c.JSON(http.StatusOK, supportedModels)
}
