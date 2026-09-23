package providers

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"nofx/marketdata"
	"nofx/provider/hyperliquid"
	"nofx/store"
)

// DirectionRow is the structured form of one instrument's directional read.
type DirectionRow struct {
	Symbol     string                          `json:"symbol"`
	Bias       string                          `json:"bias"`
	Score      float64                         `json:"score"`
	Bullish    int                             `json:"bullish"`
	Bearish    int                             `json:"bearish"`
	Neutral    int                             `json:"neutral"`
	Components []marketdata.DirectionComponent `json:"components"`
}

// DirectionPayload is the structured direction board.
type DirectionPayload struct {
	TopBullish    []DirectionRow               `json:"top_bullish"`
	TopBearish    []DirectionRow               `json:"top_bearish"`
	Instruments   []DirectionRow               `json:"instruments"`
	RecentChanges []marketdata.DirectionChange `json:"recent_changes,omitempty"`
	UniverseSize  int                          `json:"universe_size"`
	Components    []string                     `json:"components"`
}

// DirectionalSignalProvider is the in-house replacement for the retired Vergex
// direction board: it turns the free Hyperliquid feed (plus optional HyperData
// order flow) into a per-instrument bias, a signal-strength score and a
// persisted timeline of bias changes.
//
// Design stance, which matters more than the maths below: this is computed
// locally from public data rather than purchased, so its components are exposed
// in full and the model is told the score is a summary of those components
// rather than an oracle. The retired board returned a single opaque verdict that
// the prompt then declared authoritative; this one deliberately does neither.
type DirectionalSignalProvider struct {
	tracker *marketdata.DirectionTracker

	mu     sync.Mutex
	client *HyperDataClient
	config string
}

// NewDirectionalSignalProvider builds the direction source. The tracker is
// resolved lazily so the process-wide history file is only touched on first use.
func NewDirectionalSignalProvider() *DirectionalSignalProvider {
	return &DirectionalSignalProvider{}
}

// Name implements marketdata.Provider.
func (p *DirectionalSignalProvider) Name() string { return "directional_signal" }

// Description implements marketdata.Provider.
func (p *DirectionalSignalProvider) Description() string {
	return "Per-instrument directional bias and signal strength computed in-house from price momentum, perp premium and aggressive order flow — the free replacement for the retired signal board."
}

// RequiresAPIKey reports false: every input is public.
func (p *DirectionalSignalProvider) RequiresAPIKey() bool { return false }

// Enabled implements marketdata.Provider. All inputs are free, so it is on by
// default; it simply loses the order-flow component when HyperData is off.
func (p *DirectionalSignalProvider) Enabled(store.IndicatorConfig) bool { return true }

// history returns the tracker backing the change timeline.
func (p *DirectionalSignalProvider) history() *marketdata.DirectionTracker {
	if p.tracker != nil {
		return p.tracker
	}
	return defaultDirectionTracker()
}

var (
	directionOnce   sync.Once
	directionShared *marketdata.DirectionTracker
)

// defaultDirectionTracker returns the process-wide tracker. It is shared so the
// timeline keeps accumulating across the short-lived provider values the engine
// constructs.
func defaultDirectionTracker() *marketdata.DirectionTracker {
	directionOnce.Do(func() {
		directionShared = marketdata.NewDirectionTracker(marketdata.ResolveDirectionHistoryPath())
	})
	return directionShared
}

// DirectionalStateFor resolves a trading symbol against the shared timeline and
// returns its latest directional read.
//
// The trade layer needs this for signal-managed exits: it compares the bias a
// position was opened on against the bias now. Position symbols arrive in
// product form ("BTCUSDT") while the board records Hyperliquid asset names
// ("BTC"), so the lookup matches on base ticker. A miss means no read has been
// recorded for that instrument yet, never "neutral".
func DirectionalStateFor(symbol string) (marketdata.DirectionState, bool) {
	return defaultDirectionTracker().StateFor(symbol, hyperliquid.NormalizeCoinBase)
}

// flowClientFor returns a HyperData client when the order-flow component is
// available, or nil when it is not. A nil client is not an error: the block
// simply reports fewer components.
func (p *DirectionalSignalProvider) flowClientFor(cfg store.IndicatorConfig) *HyperDataClient {
	if !cfg.EnableHyperData {
		return nil
	}
	return hyperDataClientFor(&p.mu, &p.client, &p.config, cfg)
}

// Fetch implements marketdata.Provider.
func (p *DirectionalSignalProvider) Fetch(ctx context.Context, req marketdata.Request, cfg store.IndicatorConfig) (*marketdata.Insight, error) {
	board, err := hyperliquid.GetAssetContexts(ctx)
	if err != nil {
		return nil, fmt.Errorf("directional signal board: %w", err)
	}
	if len(board) == 0 {
		return nil, nil
	}

	universe := rankBy(board, directionUniverseSize, func(a, b hyperliquid.CoinInfo) bool {
		return a.Volume24h > b.Volume24h
	})

	momentumMean, momentumStd := meanStd(universe, func(c hyperliquid.CoinInfo) float64 {
		return c.Change24hPct
	})
	premiumMean, premiumStd := meanStd(universe, func(c hyperliquid.CoinInfo) float64 {
		return c.Premium
	})

	// Order flow is fetched only for the instruments actually in play, and it is
	// a secondary input: it is bounded by its own short timeout so a wedged
	// sidecar cannot consume the budget that the primary Hyperliquid read needs.
	// Losing order flow drops one component; losing the board loses the block.
	flow := p.fetchFlow(ctx, cfg, req)

	limit := req.NormalizeLimit(defaultRowLimit)

	ranked := make([]DirectionRow, 0, len(universe))
	for _, coin := range universe {
		ranked = append(ranked, directionRow(coin, momentumMean, momentumStd, premiumMean, premiumStd, flow))
	}

	instruments := make([]DirectionRow, 0, maxDirectionInstruments)
	instruments = append(instruments, directionRowsFor(universe, req.Positions, momentumMean, momentumStd, premiumMean, premiumStd, flow)...)
	instruments = append(instruments, directionRowsFor(universe, req.Symbols, momentumMean, momentumStd, premiumMean, premiumStd, flow)...)

	// Record the timeline before rendering so a flip observed this cycle is
	// visible in the same prompt as the bias that caused it.
	recent := p.record(instruments)

	bullish := rankDirectionRows(ranked, limit, func(r DirectionRow) bool { return r.Bias == marketdata.BiasBullish })
	bearish := rankDirectionRows(ranked, limit, func(r DirectionRow) bool { return r.Bias == marketdata.BiasBearish })

	var sb strings.Builder
	sb.WriteString("## Directional Signal (computed in-house)\n\n")
	sb.WriteString("This block is computed by the platform from free public data, not bought from a signal vendor. Every verdict lists the components behind it so you can audit it: each component votes bullish, bearish or neutral, and the bias is whichever side has more votes.\n\n")
	sb.WriteString(fmt.Sprintf("Scoring: price momentum over 24h and mark-vs-oracle premium are standardised against the instrument's peers (a z-score, so +2σ means far above its peers). Aggressive order flow contributes its sign. The score is the mean of those contributions clipped to ±%.0f — it is a summary of the components shown, never a number to trade on its own.\n\n", directionScoreClip))
	if len(flow) == 0 {
		sb.WriteString("Order flow is not among the components this cycle, so the verdict rests on price momentum and premium alone.\n\n")
	}

	sb.WriteString("### Strongest supported directions\n\n")
	sb.WriteString(directionTable(bullish, "bullish"))
	sb.WriteString("\n")
	sb.WriteString(directionTable(bearish, "bearish"))
	sb.WriteString("\n")

	if len(instruments) > 0 {
		sb.WriteString("### Your instruments\n\n")
		sb.WriteString("| Instrument | Bias | Score | Component evidence |\n")
		sb.WriteString("| :--- | :--- | ---: | :--- |\n")
		for _, row := range instruments {
			sb.WriteString(fmt.Sprintf("| %s | %s | %+.2f | %s |\n",
				row.Symbol, row.Bias, row.Score, describeComponents(row.Components)))
		}
		sb.WriteString("\n")
	}

	if len(recent) > 0 {
		sb.WriteString("### Recent direction changes\n\n")
		sb.WriteString("A change means the vote crossed to the other side, not that price reversed. The reason lists which component's vote flipped.\n\n")
		for _, change := range recent {
			sb.WriteString(fmt.Sprintf("- %s %s → %s, %s ago — %s\n",
				change.Symbol, change.FromBias, change.ToBias,
				humanizeAge(time.Since(change.ChangedAt)), change.Reason))
		}
		sb.WriteString("\n")
	}

	payload := DirectionPayload{
		TopBullish:    bullish,
		TopBearish:    bearish,
		Instruments:   instruments,
		RecentChanges: recent,
		UniverseSize:  len(board),
		Components:    directionComponentNames(flow),
	}

	return &marketdata.Insight{
		Provider: p.Name(),
		Title:    "Directional signal",
		Markdown: sb.String(),
		Payload:  payload,
	}, nil
}

const (
	// directionUniverseSize bounds the board before ranking.
	directionUniverseSize = 120
	// directionVoteSigma is how far a standardised component must sit from the
	// universe mean before it casts a vote. Below it the component abstains:
	// forcing a side on noise manufactures conviction.
	directionVoteSigma = 0.5
	// directionPremiumFloor is the mark-vs-oracle premium (decimal) below which
	// the premium axis abstains. Premium is a small number by construction, so it
	// is thresholded absolutely rather than relative to its peers.
	directionPremiumFloor = 0.0002
	// directionFlowFloor is the net volume delta (USD) below which order flow
	// abstains, so tape noise does not cast a vote.
	directionFlowFloor = 25_000
	// directionScoreClip bounds the reported z-score.
	directionScoreClip = 3.0
	// maxDirectionInstruments caps the per-instrument table.
	maxDirectionInstruments = 12
	// directionHistoryWindow is how far back the change timeline reaches.
	directionHistoryWindow = 24 * time.Hour
	// directionHistoryRows caps the rendered timeline.
	directionHistoryRows = 8
	// directionFlowTimeout bounds the optional order-flow enrichment.
	directionFlowTimeout = 3 * time.Second
)

// directionRow computes one instrument's verdict from the standardised inputs.
func directionRow(
	coin hyperliquid.CoinInfo,
	momentumMean, momentumStd, premiumMean, premiumStd float64,
	flow map[string]float64,
) DirectionRow {
	components := make([]marketdata.DirectionComponent, 0, 3)

	momentumZ := standardize(coin.Change24hPct, momentumMean, momentumStd)
	components = append(components, marketdata.DirectionComponent{
		Name: "momentum",
		Vote: voteBySigma(momentumZ),
		// Change24hPct is already a percentage; Premium is a decimal.
		Detail: fmt.Sprintf("%+.2fσ (%+.2f%%/24h)", momentumZ, coin.Change24hPct),
	})

	premium := coin.Premium
	premiumVote := marketdata.BiasNeutral
	if premium > directionPremiumFloor {
		premiumVote = marketdata.BiasBullish
	} else if premium < -directionPremiumFloor {
		premiumVote = marketdata.BiasBearish
	}
	components = append(components, marketdata.DirectionComponent{
		Name:   "premium",
		Vote:   premiumVote,
		Detail: fmt.Sprintf("%+.3f%%", premium*100),
	})

	if net, ok := flow[coin.Symbol]; ok {
		flowVote := marketdata.BiasNeutral
		if net > directionFlowFloor {
			flowVote = marketdata.BiasBullish
		} else if net < -directionFlowFloor {
			flowVote = marketdata.BiasBearish
		}
		components = append(components, marketdata.DirectionComponent{
			Name:   "flow",
			Vote:   flowVote,
			Detail: formatUsd(net),
		})
	}

	votes := make([]string, 0, len(components))
	for _, component := range components {
		votes = append(votes, component.Vote)
	}
	bias, bullish, bearish, neutral := marketdata.ScoreBias(votes)

	// The score averages standardised contributions. Momentum and premium carry
	// their z-score; flow carries its sign, because a handful of instruments have
	// no cross-section to standardise against.
	sum := momentumZ + standardize(premium, premiumMean, premiumStd)
	weight := 2.0
	if net, ok := flow[coin.Symbol]; ok {
		sum += clamp(math.Copysign(1, net), -directionScoreClip, directionScoreClip)
		weight++
	}

	return DirectionRow{
		Symbol:     coin.Symbol,
		Bias:       bias,
		Score:      clamp(sum/weight, -directionScoreClip, directionScoreClip),
		Bullish:    bullish,
		Bearish:    bearish,
		Neutral:    neutral,
		Components: components,
	}
}

// fetchFlow collects the net volume delta for the instruments in play, keyed by
// symbol. It returns nil whenever order flow is off or unavailable, which is a
// normal state rather than an error.
func (p *DirectionalSignalProvider) fetchFlow(ctx context.Context, cfg store.IndicatorConfig, req marketdata.Request) map[string]float64 {
	client := p.flowClientFor(cfg)
	if client == nil {
		return nil
	}

	targets := hyperDataTargets(req, maxOrderflowSymbols)
	if len(targets) == 0 {
		return nil
	}

	// A dedicated, tighter budget: this is an enrichment, so it must not be able
	// to eat the timeout the primary market read depends on.
	flowCtx, cancel := context.WithTimeout(ctx, directionFlowTimeout)
	defer cancel()

	out := make(map[string]float64, len(targets))
	for _, symbol := range targets {
		select {
		case <-flowCtx.Done():
			// Out of budget: stop early rather than returning a half-populated
			// picture, which would make a partial vote look complete.
			if len(out) == 0 {
				return nil
			}
			return out
		default:
		}

		flow, err := client.Orderflow(flowCtx, symbol)
		if err != nil || flow == nil {
			continue
		}
		out[strings.ToUpper(flow.Symbol)] = flow.CumulativeCVD
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// record writes the observed states into the timeline and returns the recent
// changes eligible for the prompt.
func (p *DirectionalSignalProvider) record(rows []DirectionRow) []marketdata.DirectionChange {
	tracker := p.history()
	if tracker == nil || len(rows) == 0 {
		return nil
	}

	now := time.Now()
	states := make([]marketdata.DirectionState, 0, len(rows))
	for _, row := range rows {
		states = append(states, marketdata.DirectionState{
			Symbol:     row.Symbol,
			Bias:       row.Bias,
			Score:      row.Score,
			Bullish:    row.Bullish,
			Bearish:    row.Bearish,
			Neutral:    row.Neutral,
			Components: row.Components,
			UpdatedAt:  now,
		})
	}

	tracker.Observe(now, states)
	return tracker.Recent(directionHistoryRows, directionHistoryWindow)
}

// directionRowsFor resolves the requested symbols against the board, preserving
// request order and dropping duplicates.
func directionRowsFor(
	board []hyperliquid.CoinInfo,
	symbols []string,
	momentumMean, momentumStd, premiumMean, premiumStd float64,
	flow map[string]float64,
) []DirectionRow {
	rows := make([]DirectionRow, 0, len(symbols))
	seen := make(map[string]bool, len(symbols))

	for _, want := range symbols {
		coin, ok := findInBoard(board, want)
		if !ok || seen[coin.Symbol] {
			continue
		}
		seen[coin.Symbol] = true
		rows = append(rows, directionRow(coin, momentumMean, momentumStd, premiumMean, premiumStd, flow))
		if len(rows) >= maxDirectionInstruments {
			break
		}
	}
	return rows
}

// rankDirectionRows filters and ranks a board by signed score, most extreme
// first, so the strongest evidence for a side leads.
func rankDirectionRows(rows []DirectionRow, limit int, keep func(DirectionRow) bool) []DirectionRow {
	matched := make([]DirectionRow, 0, len(rows))
	for _, row := range rows {
		if keep(row) {
			matched = append(matched, row)
		}
	}
	sortRowsByAbsScore(matched)
	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}
	return matched
}

// directionTable renders a ranked side.
func directionTable(rows []DirectionRow, label string) string {
	var sb strings.Builder
	if len(rows) == 0 {
		return "No instrument currently clears the " + label + " threshold.\n"
	}
	sb.WriteString(fmt.Sprintf("Strongest %s (by score): %s\n", label, directionBrief(rows)))
	return sb.String()
}

// directionBrief renders a one-line summary of a ranked side.
func directionBrief(rows []DirectionRow) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		parts = append(parts, fmt.Sprintf("%s %+.2f", row.Symbol, row.Score))
	}
	return strings.Join(parts, ", ")
}

// describeComponents renders the audit line for one instrument.
func describeComponents(components []marketdata.DirectionComponent) string {
	if len(components) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(components))
	for _, component := range components {
		parts = append(parts, fmt.Sprintf("%s %s (%s)", component.Name, component.Detail, component.Vote))
	}
	return strings.Join(parts, "; ")
}

// directionComponentNames lists which components were available, for the payload.
func directionComponentNames(flow map[string]float64) []string {
	names := []string{"momentum", "premium"}
	if len(flow) > 0 {
		names = append(names, "flow")
	}
	return names
}

// voteBySigma converts a standardised value into a vote.
func voteBySigma(z float64) string {
	switch {
	case z >= directionVoteSigma:
		return marketdata.BiasBullish
	case z <= -directionVoteSigma:
		return marketdata.BiasBearish
	default:
		return marketdata.BiasNeutral
	}
}

// standardize returns the z-score of value, or 0 when the universe has no
// dispersion. A flat cross-section carries no relative information, so an
// abstention is the correct answer rather than a division by zero.
func standardize(value, mean, std float64) float64 {
	if std <= 0 {
		return 0
	}
	return (value - mean) / std
}

// meanStd computes the mean and population standard deviation of a projection
// over the board.
func meanStd(coins []hyperliquid.CoinInfo, value func(hyperliquid.CoinInfo) float64) (float64, float64) {
	if len(coins) == 0 {
		return 0, 0
	}
	var sum float64
	for _, coin := range coins {
		sum += value(coin)
	}
	mean := sum / float64(len(coins))

	var variance float64
	for _, coin := range coins {
		diff := value(coin) - mean
		variance += diff * diff
	}
	return mean, math.Sqrt(variance / float64(len(coins)))
}

// sortRowsByAbsScore orders rows by descending |score|, strongest evidence first.
func sortRowsByAbsScore(rows []DirectionRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		return math.Abs(rows[i].Score) > math.Abs(rows[j].Score)
	})
}

// clamp bounds v to [low, high].
func clamp(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

// humanizeAge renders a duration the way a trader reads a timeline.
func humanizeAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
