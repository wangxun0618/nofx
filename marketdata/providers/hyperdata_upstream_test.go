package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// This file answers the question `hyperdata_contract_test.go` cannot: "I just
// upgraded HyperData Terminal — did anything I depend on change?"
//
// The contract test pins the shapes this adapter was written against, but it
// runs against recorded bodies, so it only fails when *we* change the code. A
// real upstream upgrade is invisible to it. This report compares the recorded
// contract against a *running* sidecar and prints what moved.
//
// It is opt-in: without HYPERDATA_LIVE_URL it skips, so the normal suite stays
// hermetic and does not depend on which upstream build a developer happens to
// have running.
//
// Run it after every upstream bump:
//
//	HYPERDATA_LIVE_URL=http://127.0.0.1:8420 \
//	  go test ./marketdata/providers/ -run HyperDataUpstream -v -count=1

// hyperDataLiveURLEnv points the drift report at a running sidecar.
const hyperDataLiveURLEnv = "HYPERDATA_LIVE_URL"

// hyperDataLiveKeyEnv supplies the credential when the sidecar was started with
// HYPERDATA_API_KEY. Optional: an unprotected sidecar needs no key.
const hyperDataLiveKeyEnv = "HYPERDATA_LIVE_API_KEY"

// upstreamProbe is one endpoint paired with the recorded body it must still
// agree with.
type upstreamProbe struct {
	name    string
	path    string
	fixture string
	// dynamicMaps lists response paths that are keyed by an open-ended
	// dimension but hold scalars, so they cannot be recognised as containers.
	// See flattenShape for why this has to be declared instead of guessed.
	dynamicMaps []string
}

func upstreamProbes() []upstreamProbe {
	return []upstreamProbe{
		{name: "health", path: "/v1/health", fixture: fixtureHealth},
		{
			name:    "orderflow",
			path:    "/v1/orderflow/BTC",
			fixture: fixtureOrderflowBTC,
			// Per-venue CVD is a map from venue name to a number. Venues go
			// dark routinely, so a venue disappearing is not a shape change.
			dynamicMaps: []string{"cumulative_cvd_by_venue"},
		},
		{name: "long-short-ratio", path: "/v1/long-short-ratio", fixture: fixtureLongShort},
		{name: "basis", path: "/v1/basis", fixture: fixtureBasis},
		{name: "whales", path: "/v1/whales", fixture: fixtureWhales},
		{name: "danger-zone", path: "/v1/positions/danger-zone", fixture: fixtureDangerZone},
	}
}

// dynamic returns the probe's declared dynamic map paths as a set.
func (p upstreamProbe) dynamic() map[string]bool {
	if len(p.dynamicMaps) == 0 {
		return nil
	}
	set := make(map[string]bool, len(p.dynamicMaps))
	for _, path := range p.dynamicMaps {
		set[path] = true
	}
	return set
}

// driftFinding is one endpoint's probe result: either the classified shape
// differences, or the reason the probe could not be judged at all.
type driftFinding struct {
	probe upstreamProbe
	diff  shapeDiff
	err   error
}

// breaking reports whether this finding should fail the run. A probe that could
// not complete counts as breaking: an unreachable sidecar must not be mistaken
// for a clean bill of health.
func (f driftFinding) breaking() bool {
	return f.err != nil || len(f.diff.missing) > 0 || len(f.diff.typeChanged) > 0
}

// collectDrift probes every endpoint and classifies each response against its
// recorded fixture.
func collectDrift(ctx context.Context, client *HyperDataClient, probes []upstreamProbe) []driftFinding {
	findings := make([]driftFinding, 0, len(probes))

	for _, probe := range probes {
		dynamic := probe.dynamic()

		recorded, err := decodeShape(probe.fixture, dynamic)
		if err != nil {
			findings = append(findings, driftFinding{
				probe: probe,
				err:   fmt.Errorf("recorded fixture is not valid JSON: %w", err),
			})
			continue
		}

		// Decode into `any` rather than the typed struct on purpose: the whole
		// point is to see fields the adapter does not model yet, which a typed
		// decode would silently drop.
		var live any
		if err := client.get(ctx, probe.path, nil, &live); err != nil {
			findings = append(findings, driftFinding{probe: probe, err: err})
			continue
		}

		liveShape := map[string]string{}
		flattenShape(live, "", liveShape, dynamic)
		findings = append(findings, driftFinding{probe: probe, diff: diffShapes(recorded, liveShape)})
	}

	return findings
}

// TestHyperDataUpstreamHasNotDrifted compares the recorded contract with a live
// sidecar and fails only on changes that would actually break this adapter.
//
// Read the output as a triage list, not a verdict:
//
//	MISSING       our parser expects a field upstream no longer sends. Breaking:
//	              the value would render as zero and quietly weaken the prompt.
//	TYPE CHANGED  the field is still there but is no longer the same JSON kind.
//	              Breaking, because tolerant decoding is per-field, not global.
//	ADDED         upstream grew a field we ignore. Harmless, but it is usually
//	              where new capability lives — worth reading the release notes.
func TestHyperDataUpstreamHasNotDrifted(t *testing.T) {
	base := strings.TrimSpace(os.Getenv(hyperDataLiveURLEnv))
	if base == "" {
		t.Skipf("set %s to a running sidecar origin to generate the upstream drift report", hyperDataLiveURLEnv)
	}

	client := NewHyperDataClient(base, os.Getenv(hyperDataLiveKeyEnv))
	t.Logf("comparing recorded contract against %s", client.BaseURL())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var breaking, additive int
	for _, finding := range collectDrift(ctx, client, upstreamProbes()) {
		if finding.err != nil {
			t.Errorf("[%s] probe failed, cannot judge drift: %v", finding.probe.name, finding.err)
			breaking++
			continue
		}
		if finding.breaking() {
			breaking++
		}
		additive += len(finding.diff.added)
		reportFinding(t, finding)
	}

	t.Logf("drift summary: %d endpoint(s) with breaking changes, %d additive field(s)", breaking, additive)
	if breaking > 0 {
		t.Errorf("upstream no longer matches the recorded contract on %d endpoint(s); see the report above", breaking)
	}
}

// TestDriftReportNamesARemovedField proves the report is wired to real HTTP and
// not merely correct in its helpers: a sidecar that drops a field the adapter
// reads must be reported as breaking, and one that conforms must not.
//
// This is what makes the opt-in report trustworthy. Without it, the only
// evidence that it works would be a developer remembering that it once did.
func TestDriftReportNamesARemovedField(t *testing.T) {
	routes := map[string]stubRoute{}
	for _, probe := range upstreamProbes() {
		routes[probe.path] = stubRoute{body: probe.fixture}
	}
	// Health loses almost everything, and keeps uptime_seconds under a
	// different JSON kind. Every other endpoint still matches its fixture.
	routes["/v1/health"] = stubRoute{
		body: `{"status":"ok","version":"2.0.0","uptime_seconds":"3600"}`,
	}

	origin := stubHyperData(t, routes)
	client := NewHyperDataClient(origin, "")

	findings := collectDrift(context.Background(), client, upstreamProbes())

	byName := map[string]driftFinding{}
	for _, finding := range findings {
		byName[finding.probe.name] = finding
	}

	health, ok := byName["health"]
	if !ok {
		t.Fatal("health endpoint was not probed")
	}
	if health.err != nil {
		t.Fatalf("health probe failed: %v", health.err)
	}
	if !health.breaking() {
		t.Fatal("a body missing most of the contract was reported as clean")
	}

	// The field that changed kind is the subtle case: still present, so a
	// presence-only check would miss it.
	if len(health.diff.typeChanged) != 1 || !strings.Contains(health.diff.typeChanged[0], "uptime_seconds") {
		t.Errorf("typeChanged = %v, want exactly uptime_seconds", health.diff.typeChanged)
	}
	// A nested field, to prove the collapse rule still reaches inside objects.
	if !containsSubstring(health.diff.missing, "orderflow_venues.*.status") {
		t.Errorf("missing = %v, want it to name orderflow_venues.*.status", health.diff.missing)
	}
	// A flat sibling of the same type must be reported by name, not swallowed
	// by the collapse rule — that rule is about containers only.
	if !containsSubstring(health.diff.missing, "feeds.liquidation_feed") {
		t.Errorf("missing = %v, want it to name feeds.liquidation_feed", health.diff.missing)
	}
	if len(health.diff.added) != 0 {
		t.Errorf("added = %v, want none: every live field exists in the fixture", health.diff.added)
	}

	// An endpoint whose body still matches must stay silent, otherwise the
	// report is noise and nobody will read it.
	unchanged := byName["long-short-ratio"]
	if unchanged.err != nil {
		t.Fatalf("long-short-ratio probe failed: %v", unchanged.err)
	}
	if unchanged.breaking() || len(unchanged.diff.added) != 0 {
		t.Errorf("an unchanged endpoint was reported as drifted: %+v", unchanged.diff)
	}

	// A venue going dark is routine and must not be reported as drift. This is
	// what dynamicMaps exists for: without it, the report would fail every time
	// a venue stopped contributing — a daily event here, not a contract break.
	orderflow := byName["orderflow"]
	if orderflow.err != nil {
		t.Fatalf("orderflow probe failed: %v", orderflow.err)
	}
	if orderflow.breaking() {
		t.Errorf("orderflow drifted only by venue membership, which is not a break: %+v", orderflow.diff)
	}
}

// TestDriftReportIgnoresVenueMembership pins the dynamicMaps behaviour directly,
// so the opt-in report stays usable when an upstream build rotates a venue.
func TestDriftReportIgnoresVenueMembership(t *testing.T) {
	dynamic := map[string]bool{"cumulative_cvd_by_venue": true}

	// A realistic record wrapper: top-level containers are told apart from
	// records only when a sibling of a different kind is present.
	recorded := map[string]string{}
	flattenShape(map[string]any{
		"symbol":                  "BTC",
		"cumulative_cvd_by_venue": map[string]any{"hyperliquid": 1.0, "binance": 2.0},
	}, "", recorded, dynamic)

	live := map[string]string{}
	flattenShape(map[string]any{
		"symbol":                  "BTC",
		"cumulative_cvd_by_venue": map[string]any{"hyperliquid": 5.0},
	}, "", live, dynamic)

	if diff := diffShapes(recorded, live); len(diff.missing) > 0 || len(diff.typeChanged) > 0 {
		t.Errorf("venue membership change was reported as breaking: %+v", diff)
	}
}

// decodeShape reads a fixture into a generic value and flattens it.
func decodeShape(fixture string, dynamic map[string]bool) (map[string]string, error) {
	var value any
	if err := json.Unmarshal([]byte(fixture), &value); err != nil {
		return nil, err
	}
	out := map[string]string{}
	flattenShape(value, "", out, dynamic)
	return out, nil
}

// shapeDiff is a classified comparison of two flattened shapes.
type shapeDiff struct {
	missing     []string // field path -> "expected kind"
	typeChanged []string // field path -> "expected kind, got kind"
	added       []string // field path -> "observed kind"
}

// diffShapes classifies every difference between the recorded and live shapes.
func diffShapes(recorded, live map[string]string) shapeDiff {
	var diff shapeDiff

	for path, want := range recorded {
		got, ok := live[path]
		switch {
		case !ok:
			diff.missing = append(diff.missing, fmt.Sprintf("%s (%s)", path, want))
		case got != want:
			diff.typeChanged = append(diff.typeChanged, fmt.Sprintf("%s (%s, got %s)", path, want, got))
		}
	}
	for path, got := range live {
		if _, ok := recorded[path]; !ok {
			diff.added = append(diff.added, fmt.Sprintf("%s (%s)", path, got))
		}
	}

	sort.Strings(diff.missing)
	sort.Strings(diff.typeChanged)
	sort.Strings(diff.added)
	return diff
}

// reportFinding prints one endpoint's differences in a form that can be pasted
// into an issue or a release note.
func reportFinding(t *testing.T, finding driftFinding) {
	t.Helper()

	name := finding.probe.name
	diff := finding.diff

	if !finding.breaking() && len(diff.added) == 0 {
		t.Logf("[%s] %s: unchanged", name, finding.probe.path)
		return
	}

	t.Logf("[%s] %s: %d missing, %d type-changed, %d added",
		name, finding.probe.path, len(diff.missing), len(diff.typeChanged), len(diff.added))

	for _, line := range diff.missing {
		t.Logf("[%s]   MISSING      %s", name, line)
	}
	for _, line := range diff.typeChanged {
		t.Logf("[%s]   TYPE CHANGED %s", name, line)
	}
	for _, line := range diff.added {
		t.Logf("[%s]   ADDED        %s", name, line)
	}
}

// flattenShape records the JSON kind found at every path in a value.
//
// Two rules decide which maps collapse into a single "<path>.*" entry:
//
//  1. Container-valued maps collapse automatically. A map whose non-null values
//     are all objects (or all arrays) is a *container* in this API — venues,
//     timeframes, symbols — not a record of the endpoint's own fields. Its keys
//     are open-ended, and a new venue or timeframe is not a contract break.
//  2. Scalars keyed by an open-ended dimension must be declared by the probe via
//     dynamicMaps. They cannot be told apart from a fixed record by shape alone:
//     `cumulative_cvd_by_venue` (open-ended) and `feeds` (fixed) are both flat
//     string-or-number maps. Guessing here would either make every dark venue
//     look like a break, or silently stop reporting a removed feed's name. So
//     the ambiguity is resolved once, explicitly, at the probe.
//
// Everything else is reported field by field. In particular a flat all-string
// object is NOT collapsed — an earlier version did collapse those, which turned
// the health payload into a single "*" and made the report useless for the one
// endpoint a user looks at first.
//
// Known limits.
//
//   - A wrapper holding exactly one sub-object (`{"venues": {...}}`) is itself
//     container-shaped, so it collapses and the name `venues` disappears from the
//     report. Renaming it would go unnoticed. This is accepted because the
//     alternative — collapsing only multi-key maps — makes the report asymmetric
//     between a fixture with one symbol and a live response with several, which
//     produces a false "MISSING" on every run. In practice endpoint roots are
//     multi-field records, so they never hit this.
//   - If a container's every entry is null upstream (every venue dark), the
//     recorded side still describes its shape while the live side collapses to
//     "null", which reads as drift. That is arguably correct: an all-null
//     container means the sidecar is not working.
func flattenShape(value any, path string, out map[string]string, dynamic map[string]bool) {
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			out[path] = "empty-object"
			return
		}
		if dynamic[path] || containerValued(typed) {
			flattenShape(representativeValue(typed, dynamic), joinShapePath(path, "*"), out, dynamic)
			return
		}
		for _, key := range sortedKeysOf(typed) {
			flattenShape(typed[key], joinShapePath(path, key), out, dynamic)
		}
	case []any:
		if len(typed) == 0 {
			out[path] = "empty-array"
			return
		}
		// Arrays are homogeneous on every endpoint here, so one element
		// describes the element shape.
		flattenShape(typed[0], path+"[]", out, dynamic)
	default:
		out[path] = jsonKindOf(value)
	}
}

// containerValued reports whether every non-null value of a map is a sub-object
// or an array of the same kind, which is the signature of a container keyed by
// an open-ended dimension.
//
// Nulls are skipped rather than treated as a kind: upstream serialises inf and
// nan as null, so a field going null is a value change, not a shape change.
func containerValued(m map[string]any) bool {
	kind := ""
	for _, key := range sortedKeysOf(m) {
		if m[key] == nil {
			continue
		}
		current := jsonKindOf(m[key])
		if current != "object" && current != "array" {
			return false
		}
		if kind == "" {
			kind = current
			continue
		}
		if current != kind {
			return false
		}
	}
	// All-null has no shape to describe.
	return kind != ""
}

// representativeValue picks the richest entry of a collapsed map, so the report
// describes the fullest variant rather than whichever key sorts first. An
// empty-but-present entry (a venue that contributed nothing) must not win.
func representativeValue(m map[string]any, dynamic map[string]bool) any {
	var best any
	bestSize := -1
	for _, key := range sortedKeysOf(m) {
		if m[key] == nil {
			continue
		}
		if size := shapeSize(m[key], dynamic); size > bestSize {
			best, bestSize = m[key], size
		}
	}
	return best
}

// shapeSize counts distinct paths in a value's shape, used to rank candidates.
func shapeSize(value any, dynamic map[string]bool) int {
	seen := map[string]string{}
	flattenShape(value, "", seen, dynamic)
	return len(seen)
}

// jsonKindOf names the JSON kind of a decoded value.
func jsonKindOf(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case float64, json.Number:
		return "number"
	case string:
		return "string"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return fmt.Sprintf("%T", value)
	}
}

// sortedKeysOf returns a map's keys in a stable order.
//
// Go randomises map iteration, and this report has to be reproducible: a
// representative chosen at random would make consecutive runs disagree.
func sortedKeysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// joinShapePath appends a path segment without emitting a leading dot at the
// root, so a top-level collapsed map is "*", not ".*".
func joinShapePath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// containsSubstring reports whether any element contains needle.
func containsSubstring(lines []string, needle string) bool {
	for _, line := range lines {
		if strings.Contains(line, needle) {
			return true
		}
	}
	return false
}

// TestFlattenShapeCollapsesContainersOnly is the rule that makes the report
// readable: container maps collapse, records do not.
func TestFlattenShapeCollapsesContainersOnly(t *testing.T) {
	value := map[string]any{
		"symbol": "BTC",
		"venues": map[string]any{
			"hyperliquid": map[string]any{"trades": 1.0},
			"binance":     map[string]any{"trades": 2.0, "extra": true},
		},
		// A flat scalar object is a record of named fields, not a container.
		// Collapsing this is what previously erased the whole health payload.
		"feeds": map[string]any{"liquidation_feed": "ok", "orderflow_engine": "partial"},
		// Scalars keyed by venue: declared dynamic by the caller.
		"by_venue": map[string]any{"hyperliquid": 1.0, "binance": 2.0},
	}

	shape := map[string]string{}
	flattenShape(value, "", shape, map[string]bool{"by_venue": true})

	want := map[string]string{
		"symbol":                 "string",
		"venues.*.trades":        "number",
		"venues.*.extra":         "bool",
		"feeds.liquidation_feed": "string",
		"feeds.orderflow_engine": "string",
		"by_venue.*":             "number",
	}
	for path, kind := range want {
		if shape[path] != kind {
			t.Errorf("shape[%q] = %q, want %q (full shape: %v)", path, shape[path], kind, shape)
		}
	}
	// Venue names must not leak into the shape: that is the whole point.
	for path := range shape {
		if strings.Contains(path, "binance") || strings.Contains(path, "hyperliquid") {
			t.Errorf("shape %q should have collapsed the venue name", path)
		}
	}
	if _, ok := shape["feeds.*"]; ok {
		t.Errorf("a flat record was collapsed: %v", shape)
	}
}

// TestFlattenShapeTreatsNullAsAValueNotAKind guards the inf/nan serialisation
// upstream performs: a field going null must not change how its parent is
// classified.
func TestFlattenShapeTreatsNullAsAValueNotAKind(t *testing.T) {
	// The wrapper is a realistic record (two fields of different kinds). A bare
	// single-key map is itself container-shaped and would collapse, which would
	// test the wrapper rather than the null handling.
	shape := map[string]string{}
	flattenShape(map[string]any{
		"symbol": "BTC",
		"venues": map[string]any{
			"hyperliquid": map[string]any{"trades": 1.0},
			"binance":     nil,
		},
	}, "", shape, nil)

	if shape["venues.*.trades"] != "number" {
		t.Errorf("a null sibling broke the container: %v", shape)
	}
}

// TestFlattenShapePrefersTheRichestEntry guards the representative choice: a
// venue that reported nothing must not mask one that reported everything.
func TestFlattenShapePrefersTheRichestEntry(t *testing.T) {
	// The map is passed directly so the paths stay readable: wrapping it would
	// add collapse levels of its own and hide what is being asserted.
	value := map[string]any{
		"aapl_empty": map[string]any{},
		"silent":     nil,
		"zebra_full": map[string]any{"status": "ok", "sockets": 4.0},
	}

	shape := map[string]string{}
	flattenShape(value, "", shape, nil)

	if shape["*.status"] != "string" || shape["*.sockets"] != "number" {
		t.Errorf("expected the populated entry to describe the shape, got %v", shape)
	}
	if _, empty := shape["*"]; empty {
		t.Errorf("an empty or null entry won the representative pick: %v", shape)
	}
}

// TestDiffShapesClassifiesChanges pins the three buckets the report prints, and
// makes sure a kind change is never mistaken for an addition.
func TestDiffShapesClassifiesChanges(t *testing.T) {
	recorded := map[string]string{
		"status":     "string",
		"venues.*.n": "number",
		"gone":       "string",
	}
	live := map[string]string{
		"status":     "string",
		"venues.*.n": "string", // loosened type
		"fresh":      "bool",
	}

	diff := diffShapes(recorded, live)

	if len(diff.missing) != 1 || !strings.Contains(diff.missing[0], "gone") {
		t.Errorf("missing = %v, want just the removed field", diff.missing)
	}
	if len(diff.typeChanged) != 1 || !strings.Contains(diff.typeChanged[0], "venues.*.n") {
		t.Errorf("typeChanged = %v, want just the loosened field", diff.typeChanged)
	}
	if len(diff.added) != 1 || !strings.Contains(diff.added[0], "fresh") {
		t.Errorf("added = %v, want just the new field", diff.added)
	}
}

// --- The pin --------------------------------------------------------------

// upstreamEnvPath is relative to this package, which is the working directory
// `go test` uses. It points at the file deploy/hyperdata/sidecar.sh sources, so
// the pin has exactly one home.
const upstreamEnvPath = "../../deploy/hyperdata/upstream.env"

// parseUpstreamEnv reads the KEY="value" pairs out of the sidecar pin file.
//
// The file is shell-sourced by deploy/hyperdata/sidecar.sh, so this only handles
// the subset that file actually uses: blank lines, comments, and quoted or bare
// assignments. Anything past that belongs in the script, not duplicated here.
func parseUpstreamEnv(raw string) map[string]string {
	values := map[string]string{}

	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, found := strings.Cut(trimmed, "=")
		if !found {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}

	return values
}

// TestUpstreamPinIsStillMeaningful keeps the pin from rotting into decoration.
//
// A pin that nothing reads drifts silently: someone upgrades the sidecar, the
// file still names the old commit, and the next person trusts it. These
// assertions make it load-bearing by tying it to the constant and the recorded
// fixture the adapter actually uses.
func TestUpstreamPinIsStillMeaningful(t *testing.T) {
	raw, err := os.ReadFile(upstreamEnvPath)
	if err != nil {
		t.Fatalf("cannot read the upstream pin at %s: %v", upstreamEnvPath, err)
	}
	pin := parseUpstreamEnv(string(raw))

	// Upstream publishes no tags, so nothing but a commit is reproducible.
	revision := pin["HYPERDATA_REVISION"]
	if len(revision) != 40 || !isLowerHex(revision) {
		t.Errorf("HYPERDATA_REVISION = %q, want a full 40-character lowercase commit SHA", revision)
	}
	if !strings.Contains(pin["HYPERDATA_REPO"], "HyperData-Terminal") {
		t.Errorf("HYPERDATA_REPO = %q does not look like the sidecar's repository", pin["HYPERDATA_REPO"])
	}

	testedMajor, err := strconv.Atoi(pin["HYPERDATA_TESTED_MAJOR"])
	if err != nil {
		t.Fatalf("HYPERDATA_TESTED_MAJOR = %q is not an integer: %v", pin["HYPERDATA_TESTED_MAJOR"], err)
	}

	// The number that must never drift away from the code.
	if testedMajor != hyperDataTestedMajor {
		t.Errorf("upstream.env says HYPERDATA_TESTED_MAJOR=%d but hyperDataTestedMajor=%d; update both together",
			testedMajor, hyperDataTestedMajor)
	}

	// And the recorded fixture has to describe the same major, otherwise the
	// contract test is asserting against a different generation than the pin
	// claims, which is worse than having no pin at all.
	fixtureVersion := fixtureHealthVersion(t)
	major, ok := majorVersion(fixtureVersion)
	if !ok {
		t.Fatalf("the recorded health fixture version %q is not a dotted version", fixtureVersion)
	}
	if major != testedMajor {
		t.Errorf("the recorded health fixture reports %s (major %d), but the pin claims major %d; "+
			"refresh the fixtures when moving the pin", fixtureVersion, major, testedMajor)
	}
}

// fixtureHealthVersion reads the version out of the recorded health body, so the
// fixture cannot be quietly left behind when the pin moves.
func fixtureHealthVersion(t *testing.T) string {
	t.Helper()

	var health struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(fixtureHealth), &health); err != nil {
		t.Fatalf("the recorded health fixture is not valid JSON: %v", err)
	}
	if health.Version == "" {
		t.Fatal("the recorded health fixture carries no version, so it cannot anchor the pin")
	}
	return health.Version
}

// isLowerHex reports whether s is non-empty lowercase hexadecimal.
func isLowerHex(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return s != ""
}

// TestParseUpstreamEnvIgnoresCommentsAndQuotes guards the pin parser against the
// file it actually reads, since a mis-parse would silently disable every
// assertion above.
func TestParseUpstreamEnvIgnoresCommentsAndQuotes(t *testing.T) {
	raw := "# a comment\n" +
		"\n" +
		"HYPERDATA_REPO=\"https://example.invalid/repo.git\"\n" +
		"HYPERDATA_REVISION='abc123'\n" +
		"HYPERDATA_PORT=8420\n" +
		"# HYPERDATA_TESTED_MAJOR=\"99\" is commented out and must not be read\n"

	values := parseUpstreamEnv(raw)

	if values["HYPERDATA_REPO"] != "https://example.invalid/repo.git" {
		t.Errorf("double-quoted value = %q, want the quotes stripped", values["HYPERDATA_REPO"])
	}
	if values["HYPERDATA_REVISION"] != "abc123" {
		t.Errorf("single-quoted value = %q, want the quotes stripped", values["HYPERDATA_REVISION"])
	}
	if values["HYPERDATA_PORT"] != "8420" {
		t.Errorf("bare value = %q, want it read as-is", values["HYPERDATA_PORT"])
	}
	if _, present := values["HYPERDATA_TESTED_MAJOR"]; present {
		t.Error("a commented-out assignment was parsed, which would defeat the pin assertions")
	}
}
