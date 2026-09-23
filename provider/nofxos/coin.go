package nofxos

import (
	"encoding/json"
	"fmt"
	"strings"
)

// QuantData represents quantitative data for a single coin
type QuantData struct {
	Symbol      string             `json:"symbol"`
	Price       float64            `json:"price"`
	Netflow     *NetflowData       `json:"netflow,omitempty"`
	OI          map[string]*OIData `json:"oi,omitempty"`           // keyed by exchange: "binance", "bybit"
	PriceChange map[string]float64 `json:"price_change,omitempty"` // keyed by duration: "1h", "4h", etc.
}

// NetflowData contains fund flow data
type NetflowData struct {
	Institution *FlowTypeData `json:"institution,omitempty"`
	Personal    *FlowTypeData `json:"personal,omitempty"`
}

// FlowTypeData contains flow data by trade type
type FlowTypeData struct {
	Future map[string]float64 `json:"future,omitempty"` // keyed by duration
	Spot   map[string]float64 `json:"spot,omitempty"`   // keyed by duration
}

// OIData contains open interest data for an exchange
type OIData struct {
	CurrentOI float64                 `json:"current_oi"`
	NetLong   float64                 `json:"net_long"`
	NetShort  float64                 `json:"net_short"`
	Delta     map[string]*OIDeltaData `json:"delta,omitempty"` // keyed by duration
}

// OIDeltaData contains OI change data
type OIDeltaData struct {
	OIDelta        float64 `json:"oi_delta"`
	OIDeltaValue   float64 `json:"oi_delta_value"`
	OIDeltaPercent float64 `json:"oi_delta_percent"` // Already x100
}

// CoinResponse is the API response structure for coin details
type CoinResponse struct {
	Success bool       `json:"success"`
	Code    int        `json:"code"`
	Data    *QuantData `json:"data"`
}

// GetCoinData retrieves quantitative data for a single coin
func (c *Client) GetCoinData(symbol string, include string) (*QuantData, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	if include == "" {
		include = "netflow,oi,price"
	}

	// Normalize symbol (remove USDT suffix for API call if needed)
	symbol = strings.TrimSuffix(strings.ToUpper(symbol), "USDT")

	endpoint := fmt.Sprintf("/api/coin/%s?include=%s", symbol, include)

	body, err := c.doRequest(endpoint)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	var response CoinResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("JSON parsing failed: %w", err)
	}

	// Check for success (support both success field and code field)
	if !response.Success && response.Code != 0 {
		return nil, fmt.Errorf("API returned error code: %d", response.Code)
	}

	return response.Data, nil
}
