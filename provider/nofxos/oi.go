package nofxos

import (
	"encoding/json"
	"fmt"
)

// OIPosition represents open interest data for a single coin
type OIPosition struct {
	Symbol            string  `json:"symbol"`
	Rank              int     `json:"rank"`
	Price             float64 `json:"price"`
	CurrentOI         float64 `json:"current_oi"`
	OIDelta           float64 `json:"oi_delta"`
	OIDeltaPercent    float64 `json:"oi_delta_percent"`    // Already x100 (5.0 = 5%)
	OIDeltaValue      float64 `json:"oi_delta_value"`      // USDT value
	PriceDeltaPercent float64 `json:"price_delta_percent"` // Already x100 (5.0 = 5%)
	NetLong           float64 `json:"net_long"`
	NetShort          float64 `json:"net_short"`
}

// OIRankingResponse is the API response structure for OI ranking
type OIRankingResponse struct {
	Success bool `json:"success"`
	Code    int  `json:"code"`
	Data    struct {
		Positions      []OIPosition `json:"positions"`
		Count          int          `json:"count"`
		Exchange       string       `json:"exchange"`
		TimeRange      string       `json:"time_range"`
		TimeRangeParam string       `json:"time_range_param"`
		RankType       string       `json:"rank_type"`
		Limit          int          `json:"limit"`
	} `json:"data"`
}

func (c *Client) fetchOIRanking(rankType, duration string, limit int) ([]OIPosition, string, error) {
	endpoint := fmt.Sprintf("/api/oi/%s-ranking?limit=%d&duration=%s", rankType, limit, duration)

	body, err := c.doRequest(endpoint)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}

	var response OIRankingResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, "", fmt.Errorf("JSON parsing failed: %w", err)
	}

	// Check for success (support both success field and code field)
	if !response.Success && response.Code != 0 {
		return nil, "", fmt.Errorf("API returned error code: %d", response.Code)
	}

	return response.Data.Positions, response.Data.TimeRange, nil
}

// GetOITopPositions retrieves top OI increase positions (legacy compatibility)
func (c *Client) GetOITopPositions() ([]OIPosition, error) {
	positions, _, err := c.fetchOIRanking("top", "1h", 20)
	if err != nil {
		return nil, err
	}
	return positions, nil
}

// GetOITopSymbols retrieves OI top coin symbol list
func (c *Client) GetOITopSymbols() ([]string, error) {
	positions, err := c.GetOITopPositions()
	if err != nil {
		return nil, err
	}

	var symbols []string
	for _, pos := range positions {
		symbol := NormalizeSymbol(pos.Symbol)
		symbols = append(symbols, symbol)
	}

	return symbols, nil
}

// GetOILowPositions retrieves OI decrease positions (for short opportunities)
func (c *Client) GetOILowPositions() ([]OIPosition, error) {
	positions, _, err := c.fetchOIRanking("low", "1h", 20)
	if err != nil {
		return nil, err
	}
	return positions, nil
}

// GetOILowSymbols retrieves OI low coin symbol list
func (c *Client) GetOILowSymbols() ([]string, error) {
	positions, err := c.GetOILowPositions()
	if err != nil {
		return nil, err
	}

	var symbols []string
	for _, pos := range positions {
		symbol := NormalizeSymbol(pos.Symbol)
		symbols = append(symbols, symbol)
	}

	return symbols, nil
}
