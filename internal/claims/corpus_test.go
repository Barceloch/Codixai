package claims

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/integrations/git"
)

// corpusRoot is the directory of the corpus of the claims.
const corpusRoot = "../../testdata/claims"

// expectedCase is the expected result of one case of the corpus.
type expectedCase struct {
	// Base is the base revision of the comparison of the case.
	Base string `json:"base,omitempty"`
	// Head is the head revision of the comparison of the case.
	Head string `json:"head,omitempty"`
	// ExpectedStatuses holds the expected status of every claim of the
	// case, indexed by the identifier of the claim.
	ExpectedStatuses map[string]string `json:"expected_statuses"`
}

// alwaysUnresolvedTypes holds the types of the catalog that can never be
// verified nor contradicted: the decision 0007 states that they are always
// unresolved.
var alwaysUnresolvedTypes = map[string]bool{
	domain.ClaimTypeAPIUnchanged:      true,
	domain.ClaimTypeBehaviorPreserved: true,
}

func TestClaimCorpus(t *testing.T) {
	entries, err := os.ReadDir(corpusRoot)
	if err != nil {
		t.Fatalf("cannot read the corpus: %v", err)
	}
	states := map[string]map[string]bool{}
	cases := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cases++
		runCorpusCase(t, filepath.Join(corpusRoot, entry.Name()), states)
	}
	if cases == 0 {
		t.Fatal("the corpus holds no case")
	}
	requireCorpusCoverage(t, states)
}

// runCorpusCase runs one case of the corpus and compares the results with
// the expected ones.
func runCorpusCase(t *testing.T, dir string, states map[string]map[string]bool) {
	t.Helper()
	name := filepath.Base(dir)

	claimsPath := filepath.Join(dir, "claims.json")
	data, err := os.ReadFile(claimsPath)
	if err != nil {
		t.Fatalf("%s: cannot read the claim file: %v", name, err)
	}
	file, err := Parse(data, claimsPath)
	if err != nil {
		t.Fatalf("%s: the claim file is not valid: %v", name, err)
	}

	expectedPath := filepath.Join(dir, "expected.json")
	expectedData, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("%s: cannot read the expected result: %v", name, err)
	}
	var expected expectedCase
	if err := json.Unmarshal(expectedData, &expected); err != nil {
		t.Fatalf("%s: the expected result is not valid: %v", name, err)
	}

	diffPath := filepath.Join(dir, "change.diff")
	diff, err := os.Open(diffPath)
	if err != nil {
		t.Fatalf("%s: cannot read the diff: %v", name, err)
	}
	defer diff.Close()
	changes, err := git.ParseUnifiedDiff(diff)
	if err != nil {
		t.Fatalf("%s: the diff is not valid: %v", name, err)
	}
	if root := filepath.Join(dir, "repo"); directoryExists(root) {
		changes.Root = root
	}
	changes.Base = expected.Base
	changes.Head = expected.Head

	results, err := NewEvaluator().Evaluate(context.Background(), changes, file, expected.Head)
	if err != nil {
		t.Fatalf("%s: the evaluation failed: %v", name, err)
	}
	if len(results) != len(expected.ExpectedStatuses) {
		t.Fatalf("%s: results: got %d, want %d", name, len(results), len(expected.ExpectedStatuses))
	}
	for index, result := range results {
		if index > 0 && results[index-1].Claim.ID >= result.Claim.ID {
			t.Errorf("%s: the results are not sorted by the identifier of the claim", name)
		}
		want, ok := expected.ExpectedStatuses[result.Claim.ID]
		if !ok {
			t.Errorf("%s: the result of the claim %q is not expected", name, result.Claim.ID)
			continue
		}
		if result.Status != want {
			t.Errorf("%s: the claim %q: got the status %q, want %q (message: %s)",
				name, result.Claim.ID, result.Status, want, result.Message)
		}
		if states[result.Claim.Type] == nil {
			states[result.Claim.Type] = map[string]bool{}
		}
		states[result.Claim.Type][result.Status] = true
	}
}

// requireCorpusCoverage checks that the corpus covers every type of the
// catalog with every state the type can reach.
func requireCorpusCoverage(t *testing.T, states map[string]map[string]bool) {
	t.Helper()
	for _, name := range Types() {
		observed := states[name]
		if len(observed) == 0 {
			t.Errorf("the corpus holds no case for the type %q", name)
			continue
		}
		if !observed[domain.ClaimUnresolved] {
			t.Errorf("the corpus holds no unresolved case for the type %q", name)
		}
		if alwaysUnresolvedTypes[name] {
			if observed[domain.ClaimVerified] || observed[domain.ClaimContradicted] {
				t.Errorf("the type %q can only be unresolved, but the corpus holds another state", name)
			}
			continue
		}
		if !observed[domain.ClaimVerified] {
			t.Errorf("the corpus holds no verified case for the type %q", name)
		}
		if !observed[domain.ClaimContradicted] {
			t.Errorf("the corpus holds no contradicted case for the type %q", name)
		}
	}
}

// directoryExists reports whether a path is an existing directory.
func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
