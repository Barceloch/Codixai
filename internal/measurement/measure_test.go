package measurement

import (
	"encoding/json"
	"testing"
	"time"
)

func intPtr(v int) *int       { return &v }
func int64Ptr(v int64) *int64 { return &v }
func boolPtr(v bool) *bool    { return &v }

func validRun(scenario string) RunRecord {
	iv := int64(500)
	ov := int64(200)
	tv := int64(700)
	files := 12
	read := 5
	calls := 34
	completed := true
	correct := true
	dur := int64(45000)
	found := 10
	known := 12
	return RunRecord{
		SchemaVersion:   SchemaVersion,
		RunID:           "run-1",
		Scenario:        scenario,
		TaskID:          "investigate_symbol",
		TaskDescription: "investigate",
		Repository:      Repository{Root: "/tmp/r", Revision: "abc123", State: "clean"},
		Model:           Model{Provider: "p", Model: "m"},
		Metrics: Metrics{
			InputTokens:    TokenMetric{Value: &iv, Source: TokenSourceProviderReported},
			OutputTokens:   TokenMetric{Value: &ov, Source: TokenSourceProviderReported},
			TotalTokens:    TokenMetric{Value: &tv, Source: TokenSourceProviderReported},
			FilesInspected: &files,
			FilesReadFull:  &read,
			ToolCalls:      &calls,
			TaskCompleted:  &completed,
			TaskCorrect:    &correct,
			DurationMs:     &dur,
		},
		Completeness: Completeness{ReferencesFound: &found, ReferencesKnown: &known},
		RecordedAt:   time.Now().UTC().Format(time.RFC3339),
	}
}

func TestValidateRunRecord_Valid(t *testing.T) {
	r := validRun("baseline")
	if err := ValidateRunRecord(r); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidateRunRecord_MissingFields(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*RunRecord)
	}{
		{"schema_version", func(r *RunRecord) { r.SchemaVersion = "" }},
		{"run_id", func(r *RunRecord) { r.RunID = "" }},
		{"task_id", func(r *RunRecord) { r.TaskID = "" }},
		{"provider", func(r *RunRecord) { r.Model.Provider = "" }},
		{"model", func(r *RunRecord) { r.Model.Model = "" }},
		{"recorded_at", func(r *RunRecord) { r.RecordedAt = "" }},
		{"scenario", func(r *RunRecord) { r.Scenario = "foo" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := validRun("baseline")
			c.mutate(&r)
			if err := ValidateRunRecord(r); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidateRunRecord_TokenProvenance(t *testing.T) {
	// value present but source unknown -> error
	r := validRun("baseline")
	r.Metrics.InputTokens = TokenMetric{Value: int64Ptr(5), Source: TokenSourceUnknown}
	if err := ValidateRunRecord(r); err == nil {
		t.Fatal("expected error: value with unknown source")
	}
	// no value but source reported -> error
	r = validRun("baseline")
	r.Metrics.InputTokens = TokenMetric{Value: nil, Source: TokenSourceProviderReported}
	if err := ValidateRunRecord(r); err == nil {
		t.Fatal("expected error: no value with reported source")
	}
	// unknown source with nil value -> ok
	r = validRun("baseline")
	r.Metrics.InputTokens = TokenMetric{Value: nil, Source: TokenSourceUnknown}
	if err := ValidateRunRecord(r); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	// negative value -> error
	r = validRun("baseline")
	neg := int64(-1)
	r.Metrics.TotalTokens = TokenMetric{Value: &neg, Source: TokenSourceProviderReported}
	if err := ValidateRunRecord(r); err == nil {
		t.Fatal("expected error: negative token")
	}
}

func TestValidateRunRecord_NegativeCounts(t *testing.T) {
	r := validRun("baseline")
	neg := -1
	r.Metrics.FilesInspected = &neg
	if err := ValidateRunRecord(r); err == nil {
		t.Fatal("expected error")
	}
}

func TestCompare_ComparableWithSavings(t *testing.T) {
	b := validRun("baseline")
	c := validRun("codixai")
	c.RunID = "run-2"
	c.Metrics.TotalTokens = TokenMetric{Value: int64Ptr(490), Source: TokenSourceProviderReported}
	c.Metrics.InputTokens = TokenMetric{Value: int64Ptr(400), Source: TokenSourceProviderReported}
	c.Metrics.OutputTokens = TokenMetric{Value: int64Ptr(100), Source: TokenSourceProviderReported}

	rep, err := Compare(b, c, "0.1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rep.Comparability.Comparable {
		t.Fatalf("expected comparable, got reasons %v", rep.Comparability.Reasons)
	}
	if rep.Savings == nil || rep.Savings.TotalTokens == nil {
		t.Fatal("expected savings")
	}
	if *rep.Savings.TotalTokens < 29.999 || *rep.Savings.TotalTokens > 30.001 {
		t.Fatalf("expected 30%% savings, got %v", *rep.Savings.TotalTokens)
	}
	if rep.Interpretation == "" {
		t.Fatal("expected interpretation")
	}
}

func TestCompare_NotComparable_TaskID(t *testing.T) {
	b := validRun("baseline")
	c := validRun("codixai")
	c.TaskID = "other_task"
	rep, err := Compare(b, c, "0.1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.Comparability.Comparable {
		t.Fatal("expected not comparable")
	}
	if len(rep.Comparability.Reasons) == 0 {
		t.Fatal("expected reasons")
	}
	if rep.Savings != nil {
		t.Fatal("expected no savings when not comparable")
	}
}

func TestCompare_NotComparable_Revision(t *testing.T) {
	b := validRun("baseline")
	c := validRun("codixai")
	c.Repository.Revision = "different"
	rep, _ := Compare(b, c, "0.1.0")
	if rep.Comparability.Comparable {
		t.Fatal("expected not comparable")
	}
}

func TestCompare_NotComparable_Model(t *testing.T) {
	b := validRun("baseline")
	c := validRun("codixai")
	c.Model.Model = "other"
	rep, _ := Compare(b, c, "0.1.0")
	if rep.Comparability.Comparable {
		t.Fatal("expected not comparable")
	}
}

func TestCompare_ScenarioLabels(t *testing.T) {
	b := validRun("baseline")
	c := validRun("codixai")
	b.Scenario = "codixai"
	rep, _ := Compare(b, c, "0.1.0")
	if rep.Comparability.Comparable {
		t.Fatal("expected not comparable due to scenario")
	}
}

func TestCompare_UnknownTokens_NoSavings(t *testing.T) {
	b := validRun("baseline")
	c := validRun("codixai")
	b.Metrics.TotalTokens = TokenMetric{Value: nil, Source: TokenSourceUnknown}
	rep, err := Compare(b, c, "0.1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rep.Comparability.Comparable {
		t.Fatal("expected comparable")
	}
	if rep.Savings != nil {
		t.Fatal("expected no savings when tokens unknown")
	}
}

func TestCompare_EstimateNotReportedAsSavings(t *testing.T) {
	b := validRun("baseline")
	c := validRun("codixai")
	b.Metrics.TotalTokens = TokenMetric{Value: int64Ptr(1000), Source: TokenSourceTokenizerEstimate}
	c.Metrics.TotalTokens = TokenMetric{Value: int64Ptr(500), Source: TokenSourceProviderReported}
	rep, _ := Compare(b, c, "0.1.0")
	if rep.Savings != nil {
		t.Fatal("estimate must not produce savings")
	}
}

func TestCompare_ValidationFailure(t *testing.T) {
	b := validRun("baseline")
	c := validRun("codixai")
	b.RunID = ""
	_, err := Compare(b, c, "0.1.0")
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCompare_Deterministic(t *testing.T) {
	b := validRun("baseline")
	c := validRun("codixai")
	r1, _ := Compare(b, c, "0.1.0")
	r2, _ := Compare(b, c, "0.1.0")
	j1, _ := json.Marshal(r1)
	j2, _ := json.Marshal(r2)
	if string(j1) != string(j2) {
		t.Fatal("comparison not deterministic")
	}
}

func TestEstimateTokensFromBytes(t *testing.T) {
	if EstimateTokensFromBytes(0) != 0 {
		t.Fatal("expected 0")
	}
	if EstimateTokensFromBytes(4) != 1 {
		t.Fatal("expected 1")
	}
	if EstimateTokensFromBytes(100) != 25 {
		t.Fatal("expected 25")
	}
	if EstimateTokensFromBytes(-5) != 0 {
		t.Fatal("expected 0 for negative")
	}
}

func TestLoadTasks(t *testing.T) {
	reg, err := LoadTasks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version mismatch: %s", reg.SchemaVersion)
	}
	if len(reg.Tasks) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(reg.Tasks))
	}
	ids := map[string]bool{}
	for _, tk := range reg.Tasks {
		if tk.ID == "" || tk.Kind == "" || tk.Description == "" {
			t.Fatalf("incomplete task: %+v", tk)
		}
		if ids[tk.ID] {
			t.Fatalf("duplicate task id: %s", tk.ID)
		}
		ids[tk.ID] = true
	}
}
