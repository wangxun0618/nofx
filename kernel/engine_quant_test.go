package kernel

import (
	"net/http"
	"strings"
	"testing"

	"nofx/provider/nofxos"
	"nofx/store"
)

func quantTestEngine(enabled bool) *StrategyEngine {
	return NewStrategyEngine(&store.StrategyConfig{
		Indicators: store.IndicatorConfig{EnableQuantData: enabled},
	})
}

func TestQuantDataStaysOffWhenTheSwitchIsOff(t *testing.T) {
	engine := quantTestEngine(false)
	// A disabled switch must not produce a request, a note, or a latched flag.
	if got := engine.FetchQuantDataBatch([]string{"BTCUSDT"}); len(got) != 0 {
		t.Fatalf("disabled quant data must return nothing, got %v", got)
	}
	if notes := engine.TakeCoinSourceNotes(); len(notes) != 0 {
		t.Fatalf("a disabled switch must not explain itself as a failure, got %v", notes)
	}
	if engine.quantDataUnavailable() {
		t.Fatal("a disabled switch must not latch the endpoint as unavailable")
	}
}

func TestQuantDataLatchesOnceOnEntitlementFailure(t *testing.T) {
	engine := quantTestEngine(true)
	cause := &nofxos.APIError{StatusCode: http.StatusPaymentRequired, Message: "payment required"}

	// The failure is reported and latched by the first caller...
	engine.disableQuantData(cause)
	if !engine.quantDataUnavailable() {
		t.Fatal("an entitlement failure must latch the endpoint off")
	}
	// ...and only explained once, however many cycles hit it afterwards.
	engine.disableQuantData(cause)
	engine.disableQuantData(cause)

	notes := engine.TakeCoinSourceNotes()
	if len(notes) != 1 {
		t.Fatalf("the reason must be stated exactly once, got %d notes: %v", len(notes), notes)
	}
	if !strings.Contains(notes[0], "unavailable") {
		t.Fatalf("the note must say the source is unavailable, got %q", notes[0])
	}

	// Once latched, a cycle must not spend a request per symbol re-learning it.
	if got := engine.FetchQuantDataBatch([]string{"BTCUSDT", "ETHUSDT"}); len(got) != 0 {
		t.Fatalf("a latched endpoint must not be queried again, got %v", got)
	}
}
