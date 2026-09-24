package trader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/marketdata"
)

func TestEvaluateSignalExitUsesRecordedEntryForBothRules(t *testing.T) {
	entry := signalRead{Bias: marketdata.BiasBullish, Score: 2.0}

	if got := evaluateSignalExit("long", entry, signalRead{Bias: marketdata.BiasBullish, Score: 2.3}, true, 0.5); got != "" {
		t.Fatalf("an intact thesis must not close, got %q", got)
	}
	if got := evaluateSignalExit("long", entry, signalRead{Bias: marketdata.BiasBearish, Score: -2.0}, true, 0.5); got != signalReasonFlip {
		t.Fatalf("a reversed read must close on flip, got %q", got)
	}
	if got := evaluateSignalExit("long", entry, signalRead{Bias: marketdata.BiasBullish, Score: 0.1}, true, 0.5); got != signalReasonDecay {
		t.Fatalf("a faded read must close on decay, got %q", got)
	}
	if got := evaluateSignalExit("long", entry, signalRead{}, true, 0.5); got != "" {
		t.Fatalf("an unobserved symbol carries no evidence and must not close, got %q", got)
	}
}

func TestEvaluateSignalExitWithoutRecordedEntryOnlyFlips(t *testing.T) {
	// A position whose entry thesis was never recorded (book lost, or opened
	// before the book persisted). Decay compares strength against the recorded
	// entry, so with no entry it must stay silent — otherwise every position is
	// closed on the first cycle after a restart, for a reason nobody can audit.
	faded := signalRead{Bias: marketdata.BiasBullish, Score: 0.1}
	if got := evaluateSignalExit("long", signalRead{}, faded, false, 0.5); got != "" {
		t.Fatalf("decay against a fabricated entry must not fire, got %q", got)
	}

	// The flip half still works: it only needs the side's implied direction.
	reversed := signalRead{Bias: marketdata.BiasBearish, Score: -1.0}
	if got := evaluateSignalExit("long", signalRead{}, reversed, false, 0.5); got != signalReasonFlip {
		t.Fatalf("a flip against the position's own side must still close, got %q", got)
	}
	// A short position flips on a bullish read.
	if got := evaluateSignalExit("short", signalRead{}, signalRead{Bias: marketdata.BiasBullish, Score: 1.0}, false, 0.5); got != signalReasonFlip {
		t.Fatalf("a short must close when the read turns bullish, got %q", got)
	}
	// Neutral is not a flip, and without a recorded entry it is not decay
	// either: the read losing its opinion is only actionable as decay.
	if got := evaluateSignalExit("long", signalRead{}, signalRead{Bias: marketdata.BiasNeutral}, false, 0.5); got != "" {
		t.Fatalf("a neutral read must not close an unevidenced position, got %q", got)
	}
}

func TestSignalBookSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(signalBookDirEnv, dir)

	path := SignalBookPath("trader 1")
	if filepath.Dir(path) != dir {
		t.Fatalf("book should live in the configured directory, got %s", path)
	}
	if !strings.HasSuffix(path, "signal_book_trader_1.json") {
		t.Fatalf("trader id should be sanitised into the file name, got %s", path)
	}

	first := newSignalBook(path)
	first.Record("BTCUSDT", "long", signalRead{Bias: marketdata.BiasBullish, Score: 1.5})

	// A new process: the book is rebuilt from disk, not from memory.
	restarted := newSignalBook(SignalBookPath("trader 1"))
	entry, ok := restarted.Read("BTCUSDT", "long")
	if !ok {
		t.Fatal("the recorded entry must survive a restart")
	}
	if entry.Read.Bias != marketdata.BiasBullish || entry.Read.Score != 1.5 {
		t.Fatalf("restored entry lost its values: %+v", entry.Read)
	}

	// With the entry restored, decay is evaluated against the real thesis again.
	if got := evaluateSignalExit("long", entry.Read, signalRead{Bias: marketdata.BiasBullish, Score: 0.1}, true, 0.5); got != signalReasonDecay {
		t.Fatalf("restored entry must make decay actionable again, got %q", got)
	}
}

func TestSignalBookPruneForgetsClosedPositionsOnDisk(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(signalBookDirEnv, dir)
	path := SignalBookPath("t2")

	book := newSignalBook(path)
	book.Record("BTCUSDT", "long", signalRead{Bias: marketdata.BiasBullish, Score: 1})
	book.Record("ETHUSDT", "short", signalRead{Bias: marketdata.BiasBearish, Score: -1})

	book.Prune(map[string]bool{signalKey("BTCUSDT", "long"): true})

	reloaded := newSignalBook(path)
	if _, ok := reloaded.Read("BTCUSDT", "long"); !ok {
		t.Fatal("the still-open position must stay in the book")
	}
	if _, ok := reloaded.Read("ETHUSDT", "short"); ok {
		t.Fatal("a closed position must not come back after a restart")
	}
}

func TestSignalBookToleratesCorruptFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(signalBookDirEnv, dir)
	path := SignalBookPath("t3")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	book := newSignalBook(path)
	if _, ok := book.Read("BTCUSDT", "long"); ok {
		t.Fatal("a corrupt book must be treated as empty")
	}
	// And it must be usable again, not permanently poisoned.
	book.Record("BTCUSDT", "long", signalRead{Bias: marketdata.BiasBullish, Score: 1})
	if _, ok := newSignalBook(path).Read("BTCUSDT", "long"); !ok {
		t.Fatal("a corrupt book must be overwritten by the next write")
	}
}
