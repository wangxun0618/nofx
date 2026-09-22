package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"nofx/store"
)

// DefaultHyperDataBaseURL is where `python run_api.py` listens by default.
//
// HyperData Terminal binds to loopback and refuses a non-loopback bind unless an
// API key is configured, so this is deliberately a local address: the sidecar is
// expected to run on the same host as this backend.
const DefaultHyperDataBaseURL = "http://127.0.0.1:8420"

// Environment overrides for the sidecar settings.
//
// They exist so a deployment can configure the sidecar once instead of asking
// every strategy to carry the same URL and credential. A value set on the
// strategy still wins, which keeps per-strategy routing possible.
const (
	HyperDataBaseURLEnv = "HYPERDATA_BASE_URL"
	HyperDataAPIKeyEnv  = "HYPERDATA_API_KEY"
)

// hyperDataSettings resolves the effective sidecar settings, preferring the
// strategy configuration and falling back to the environment.
func hyperDataSettings(cfg store.IndicatorConfig) (baseURL, apiKey string) {
	baseURL = strings.TrimSpace(cfg.HyperDataBaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv(HyperDataBaseURLEnv))
	}
	apiKey = strings.TrimSpace(cfg.HyperDataAPIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv(HyperDataAPIKeyEnv))
	}
	return baseURL, apiKey
}

// hyperDataClientFor returns a cached client for the resolved settings.
//
// The cache is shared per provider instance so the health payload is not
// re-polled by every source in a cycle. It is rebuilt when the resolved endpoint
// or credential changes, so a strategy edit or an env change takes effect
// without a restart.
func hyperDataClientFor(
	mu *sync.Mutex,
	cached **HyperDataClient,
	cacheKey *string,
	cfg store.IndicatorConfig,
) *HyperDataClient {
	baseURL, apiKey := hyperDataSettings(cfg)
	key := baseURL + "\x00" + apiKey

	mu.Lock()
	defer mu.Unlock()
	if *cached == nil || *cacheKey != key {
		*cached = NewHyperDataClient(baseURL, apiKey)
		*cacheKey = key
	}
	return *cached
}

const (
	// hyperDataTimeout bounds a single upstream call. A healthy sidecar answers
	// from its own in-memory hub in milliseconds; anything slower means it is
	// wedged and must not be allowed to stall a trading cycle.
	hyperDataTimeout = 8 * time.Second
	// hyperDataMaxBody caps one response body.
	hyperDataMaxBody = 8 << 20
	// hyperDataHealthTTL is how long a health payload is reused. Every provider
	// in a cycle would otherwise re-poll it, and the version it carries only
	// needs checking occasionally.
	hyperDataHealthTTL = 30 * time.Second

	// hyperDataTestedMajor is the upstream major version this adapter was built
	// and contract-tested against. A different major produces a warning, not a
	// failure, because the adapter tolerates unknown and reordered fields.
	hyperDataTestedMajor = 1
)

// HyperDataClient is a thin HTTP client over the HyperData Terminal REST API.
//
// The sidecar is an independent project with its own release cadence, so this
// adapter mirrors the upstream JSON instead of normalising it into new shapes:
// the less translation happens here, the less there is to re-verify after an
// upstream upgrade. Compatibility therefore rests on three rules:
//
//   - unknown fields are ignored (encoding/json default), so upstream additions
//     are safe
//   - fields whose type has historically been loose (signals, venue freshness)
//     decode through the tolerant types below, so upstream may send a string or
//     an object without breaking the call
//   - the reported version is compared against hyperDataTestedMajor and a
//     mismatch is logged, so a breaking upgrade is visible in the logs
type HyperDataClient struct {
	baseURL string
	apiKey  string
	http    *http.Client

	mu           sync.Mutex
	cachedHealth *HyperDataHealth
	healthAt     time.Time
}

// NewHyperDataClient builds a client. An empty baseURL falls back to
// DefaultHyperDataBaseURL; an empty apiKey assumes the sidecar runs without
// HYPERDATA_API_KEY.
func NewHyperDataClient(baseURL, apiKey string) *HyperDataClient {
	base := strings.TrimSpace(baseURL)
	if base == "" {
		base = DefaultHyperDataBaseURL
	}
	return &HyperDataClient{
		baseURL: strings.TrimRight(base, "/"),
		apiKey:  strings.TrimSpace(apiKey),
		http:    &http.Client{Timeout: hyperDataTimeout},
	}
}

// BaseURL reports the origin this client talks to, for logs and diagnostics.
func (c *HyperDataClient) BaseURL() string { return c.baseURL }

// get performs one request. All upstream errors funnel through here so a caller
// can rely on a single failure style: the sidecar is either talking to us or it
// is not, and every provider degrades the same way when it is not.
func (c *HyperDataClient) get(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("hyperdata %s: %w", path, err)
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		// Upstream accepts either header; sending both avoids caring which build
		// is running on the other end.
		req.Header.Set("X-API-Key", c.apiKey)
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("hyperdata %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, hyperDataMaxBody))
	if err != nil {
		return fmt.Errorf("hyperdata %s: read body: %w", path, err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		// The API reports "no data for this symbol" and similar conditions as
		// {"error": "..."} rather than an empty body.
		var apiErr struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
			return fmt.Errorf("hyperdata %s: HTTP %d: %s", path, resp.StatusCode, apiErr.Error)
		}
		return fmt.Errorf("hyperdata %s: HTTP %d", path, resp.StatusCode)
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("hyperdata %s: decode: %w", path, err)
	}
	return nil
}

// flexStrings decodes a JSON array of strings while tolerating entries that are
// not strings. An unexpected entry is kept in its compact JSON form rather than
// failing the payload: a source list is diagnostic detail, never the point of
// the call.
type flexStrings []string

// UnmarshalJSON implements json.Unmarshaler.
func (f *flexStrings) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		// null, a bare string or any other shape: treat as "nothing listed".
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		var s string
		if err := json.Unmarshal(item, &s); err == nil {
			out = append(out, s)
			continue
		}
		out = append(out, string(item))
	}
	*f = out
	return nil
}

// flexText decodes a value that upstream has shipped as both a plain label and a
// nested object. Objects are flattened to compact JSON so the label still
// reaches the prompt instead of voiding the whole response.
type flexText string

// UnmarshalJSON implements json.Unmarshaler.
func (f *flexText) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = flexText(s)
		return nil
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil
	}
	*f = flexText(compactJSON(v))
	return nil
}

// String returns the decoded text.
func (f flexText) String() string { return string(f) }

// compactJSON renders a decoded value on one line, so an unexpected object does
// not inject newlines into a markdown table cell.
func compactJSON(v any) string {
	encoded, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(encoded)
}

// --- Response types -------------------------------------------------------
//
// Field names below were taken verbatim from the upstream API surface. Numeric
// fields use concrete types; anything whose upstream type is loose keeps a
// tolerant type so a shape change cannot void the call.

// HyperDataHealth mirrors GET /v1/health.
type HyperDataHealth struct {
	Status           string                    `json:"status"`
	Version          string                    `json:"version"`
	Mode             string                    `json:"mode"`
	UptimeSeconds    float64                   `json:"uptime_seconds"`
	FailedComponents flexStrings               `json:"failed_components"`
	OrderflowVenues  map[string]map[string]any `json:"orderflow_venues"`
	Feeds            map[string]any            `json:"feeds"`
	TrackedAssets    int                       `json:"tracked_assets"`
	TrackedPositions int                       `json:"tracked_positions"`
}

// HyperDataOrderflowFrame is one timeframe of GET /v1/orderflow/{symbol}.
type HyperDataOrderflowFrame struct {
	BuyVolume  float64  `json:"buy_volume"`
	SellVolume float64  `json:"sell_volume"`
	NetVolume  float64  `json:"net_volume"`
	TradeCount int      `json:"trade_count"`
	OFI        float64  `json:"ofi"`
	Signal     flexText `json:"signal"`
}

// HyperDataOrderflow mirrors GET /v1/orderflow/{symbol}.
type HyperDataOrderflow struct {
	Symbol               string                             `json:"symbol"`
	CumulativeCVD        float64                            `json:"cumulative_cvd"`
	CumulativeCVDByVenue map[string]float64                 `json:"cumulative_cvd_by_venue"`
	VenueCoverage        map[string]any                     `json:"venue_coverage"`
	VenuesContributing   flexStrings                        `json:"venues_contributing"`
	TradesPerSecond      float64                            `json:"trades_per_second"`
	AggregateSignal      flexText                           `json:"aggregate_signal"`
	Timeframes           map[string]HyperDataOrderflowFrame `json:"timeframes"`
}

// HyperDataLongShort is one entry of GET /v1/long-short-ratio.
type HyperDataLongShort struct {
	LongRatio      float64 `json:"long_ratio"`
	ShortRatio     float64 `json:"short_ratio"`
	LongShortRatio float64 `json:"long_short_ratio"`
	Timestamp      float64 `json:"timestamp"`
}

// HyperDataBasis is one entry of GET /v1/basis.
type HyperDataBasis struct {
	SpotPrice float64 `json:"spot_price"`
	PerpPrice float64 `json:"perp_price"`
	BasisPct  float64 `json:"basis_pct"`
	Timestamp float64 `json:"timestamp"`
}

// HyperDataIV is one entry of GET /v1/deribit/iv.
type HyperDataIV struct {
	MarkIV     float64 `json:"mark_iv"`
	IndexPrice float64 `json:"index_price"`
	Timestamp  float64 `json:"timestamp"`
}

// HyperDataPosition is one tracked position from /v1/whales or
// /v1/positions/danger-zone. The upstream field set is a flat scalar record.
type HyperDataPosition struct {
	Address       string  `json:"address"`
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	SizeUSD       float64 `json:"size_usd"`
	EntryPrice    float64 `json:"entry_price"`
	CurrentPrice  float64 `json:"current_price"`
	LiqPrice      float64 `json:"liq_price"`
	DistancePct   float64 `json:"distance_pct"`
	Leverage      float64 `json:"leverage"`
	UnrealizedPnL float64 `json:"unrealized_pnl"`
	MarginUsed    float64 `json:"margin_used"`
	ScannedAt     float64 `json:"scanned_at"`
}

// HyperDataPositions covers both /v1/whales and /v1/positions/danger-zone; the
// latter adds ThresholdPct, which decodes to zero for the former.
type HyperDataPositions struct {
	Count          int                 `json:"count"`
	AsOf           *float64            `json:"as_of"`
	ScanAgeSeconds *float64            `json:"scan_age_seconds"`
	ThresholdPct   float64             `json:"threshold_pct"`
	Positions      []HyperDataPosition `json:"positions"`
}

// HyperDataAsset is one entry of GET /v1/market.
type HyperDataAsset struct {
	Symbol            string  `json:"symbol"`
	Price             float64 `json:"price"`
	FundingRate       float64 `json:"funding_rate"`
	OpenInterest      float64 `json:"open_interest"`
	Volume24h         float64 `json:"volume_24h"`
	PriceChange24hPct float64 `json:"price_change_24h_pct"`
	MarkPrice         float64 `json:"mark_price"`
	IndexPrice        float64 `json:"index_price"`
	PremiumPct        float64 `json:"premium_pct"`
}

// HyperDataMarket mirrors GET /v1/market.
type HyperDataMarket struct {
	Count  int              `json:"count"`
	Assets []HyperDataAsset `json:"assets"`
}

// --- Endpoint methods -----------------------------------------------------

// Health calls GET /v1/health.
func (c *HyperDataClient) Health(ctx context.Context) (*HyperDataHealth, error) {
	var out HyperDataHealth
	if err := c.get(ctx, "/v1/health", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// HealthCached returns a recent health payload when one is available, so the
// providers in a single cycle share one poll instead of racing each other.
func (c *HyperDataClient) HealthCached(ctx context.Context) (*HyperDataHealth, error) {
	c.mu.Lock()
	cached, at := c.cachedHealth, c.healthAt
	c.mu.Unlock()

	if cached != nil && time.Since(at) < hyperDataHealthTTL {
		return cached, nil
	}

	health, err := c.Health(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cachedHealth = health
	c.healthAt = time.Now()
	c.mu.Unlock()
	return health, nil
}

// Orderflow calls GET /v1/orderflow/{symbol}. A symbol the upstream instance
// does not track comes back as an HTTP 404 carrying {"error": ...}, which
// surfaces here as a normal error for the caller to skip.
func (c *HyperDataClient) Orderflow(ctx context.Context, symbol string) (*HyperDataOrderflow, error) {
	var out HyperDataOrderflow
	if err := c.get(ctx, "/v1/orderflow/"+url.PathEscape(strings.ToUpper(symbol)), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// LongShortRatio calls GET /v1/long-short-ratio, keyed by instrument.
func (c *HyperDataClient) LongShortRatio(ctx context.Context) (map[string]HyperDataLongShort, error) {
	out := map[string]HyperDataLongShort{}
	if err := c.get(ctx, "/v1/long-short-ratio", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Basis calls GET /v1/basis, keyed by instrument.
func (c *HyperDataClient) Basis(ctx context.Context) (map[string]HyperDataBasis, error) {
	out := map[string]HyperDataBasis{}
	if err := c.get(ctx, "/v1/basis", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeribitIV calls GET /v1/deribit/iv, keyed by instrument.
//
// Not used by any provider yet, and therefore absent from the recorded contract
// fixtures in hyperdata_contract_test.go and from the live drift report. Verify
// the shape against a running sidecar before relying on it.
func (c *HyperDataClient) DeribitIV(ctx context.Context) (map[string]HyperDataIV, error) {
	out := map[string]HyperDataIV{}
	if err := c.get(ctx, "/v1/deribit/iv", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Whales calls GET /v1/whales.
func (c *HyperDataClient) Whales(ctx context.Context) (*HyperDataPositions, error) {
	var out HyperDataPositions
	if err := c.get(ctx, "/v1/whales", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DangerZone calls GET /v1/positions/danger-zone.
func (c *HyperDataClient) DangerZone(ctx context.Context) (*HyperDataPositions, error) {
	var out HyperDataPositions
	if err := c.get(ctx, "/v1/positions/danger-zone", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Market calls GET /v1/market.
//
// Not used by any provider yet, and therefore absent from the recorded contract
// fixtures in hyperdata_contract_test.go and from the live drift report. Note
// that HyperDataAsset.OpenInterest is mirrored without a unit: upstream's
// documented "$" suffix suggests USD, but nothing here verifies it. Confirm
// before building a provider on it.
func (c *HyperDataClient) Market(ctx context.Context) (*HyperDataMarket, error) {
	var out HyperDataMarket
	if err := c.get(ctx, "/v1/market", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// --- Compatibility helpers ------------------------------------------------

// VersionWarning reports whether the running sidecar is a major version this
// adapter was not verified against. Empty means "no concern".
//
// The check is advisory on purpose: the adapter is written to tolerate upstream
// field additions, so a major bump is a prompt to re-run the contract test
// rather than a reason to stop trading.
func VersionWarning(health *HyperDataHealth) string {
	if health == nil || strings.TrimSpace(health.Version) == "" {
		return ""
	}
	major, ok := majorVersion(health.Version)
	if !ok || major == hyperDataTestedMajor {
		return ""
	}
	return fmt.Sprintf("HyperData reports version %s; this adapter was verified against %d.x. Re-run the contract test before trusting new fields.",
		health.Version, hyperDataTestedMajor)
}

// majorVersion extracts the leading integer of a dotted version string.
func majorVersion(version string) (int, bool) {
	head, _, _ := strings.Cut(strings.TrimSpace(version), ".")
	if head == "" {
		return 0, false
	}
	n := 0
	for _, r := range head {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

// venueStatuses renders the per-venue order-flow freshness map as compact text.
//
// The map's inner shape has changed upstream before, so it is read generically:
// only the keys we recognise are formatted, and an unrecognised payload is
// summarised instead of dropped. When Binance Futures is geo-blocked its socket
// connects but never delivers, which upstream reports here rather than hiding.
func venueStatuses(health *HyperDataHealth) []string {
	if health == nil || len(health.OrderflowVenues) == 0 {
		return nil
	}
	names := make([]string, 0, len(health.OrderflowVenues))
	for name := range health.OrderflowVenues {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]string, 0, len(names))
	for _, name := range names {
		venue := health.OrderflowVenues[name]
		parts := make([]string, 0, 3)
		if status := anyString(venue["status"]); status != "" {
			parts = append(parts, status)
		}
		if stale, ok := venue["stale"].(bool); ok && stale {
			parts = append(parts, "stale")
		}
		if open, ok := anyFloat(venue["sockets_open"]); ok {
			if expected, ok := anyFloat(venue["sockets_expected"]); ok {
				parts = append(parts, fmt.Sprintf("%d/%d sockets", int(open), int(expected)))
			}
		}
		if dark := anyStringSlice(venue["dark_symbols"]); len(dark) > 0 {
			parts = append(parts, fmt.Sprintf("%d symbols dark", len(dark)))
		}
		if len(parts) == 0 {
			parts = append(parts, "reported")
		}
		out = append(out, name+" ("+strings.Join(parts, ", ")+")")
	}
	return out
}

// anyString coerces a generically decoded JSON value to a display string.
func anyString(v any) string {
	switch typed := v.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return trimFloatString(typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return compactJSON(typed)
	}
}

// anyFloat coerces a generically decoded JSON value to a float.
func anyFloat(v any) (float64, bool) {
	switch typed := v.(type) {
	case float64:
		return typed, true
	case json.Number:
		f, err := typed.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// anyStringSlice coerces a generically decoded JSON value to a string list.
func anyStringSlice(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s := anyString(item); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// trimFloatString renders a float without a trailing ".0".
func trimFloatString(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return fmt.Sprintf("%.2f", v)
}
