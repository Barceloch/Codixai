//go:build ui

package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/domain"
)

// fakeAudit records the request of the window and returns a fixed result.
type fakeAudit struct {
	root     string
	worktree bool
	base     string
	calls    int
}

// run implements AuditFunc.
func (f *fakeAudit) run(_ context.Context, root string, worktree bool, base string) (*application.AuditResult, error) {
	f.calls++
	f.root = root
	f.worktree = worktree
	f.base = base
	return &application.AuditResult{
		Run: &domain.AnalysisRun{
			SchemaVersion: "0.1",
			ToolVersion:   "test",
			Findings:      nil,
		},
		FailOn:          "error",
		Failed:          true,
		FailingFindings: 2,
		ChangedFiles:    3,
	}, nil
}

// TestAuditRejectsAnEmptyRoot checks that the adapter validates the root
// before the use case runs.
func TestAuditRejectsAnEmptyRoot(t *testing.T) {
	fake := &fakeAudit{}
	app := &App{audit: fake.run}

	_, err := app.Audit(AuditRequest{Root: "   ", Worktree: true})
	if err == nil {
		t.Fatal("the audit accepted an empty root")
	}
	if !strings.Contains(err.Error(), "repository field") {
		t.Errorf("the error does not name the field: %v", err)
	}
	if fake.calls != 0 {
		t.Errorf("the use case ran %d time(s) for an invalid request", fake.calls)
	}
}

// TestAuditRejectsTwoOrigins checks that the worktree and a base revision
// are not accepted at the same time.
func TestAuditRejectsTwoOrigins(t *testing.T) {
	fake := &fakeAudit{}
	app := &App{audit: fake.run}

	_, err := app.Audit(AuditRequest{Root: "/tmp/repository", Worktree: true, Base: "main"})
	if err == nil {
		t.Fatal("the audit accepted two origins")
	}
	if !strings.Contains(err.Error(), "one origin") {
		t.Errorf("the error does not explain the origin: %v", err)
	}
}

// TestAuditRequiresAnOrigin checks that a request without an origin is
// rejected.
func TestAuditRequiresAnOrigin(t *testing.T) {
	fake := &fakeAudit{}
	app := &App{audit: fake.run}

	_, err := app.Audit(AuditRequest{Root: "/tmp/repository"})
	if err == nil {
		t.Fatal("the audit accepted a request without an origin")
	}
	if !strings.Contains(err.Error(), "worktree") {
		t.Errorf("the error does not name the worktree: %v", err)
	}
}

// TestAuditReturnsTheDocumentOfTheUseCase checks that the adapter passes the
// chosen origin to the use case and renders its result without reinterpreting
// it.
func TestAuditReturnsTheDocumentOfTheUseCase(t *testing.T) {
	fake := &fakeAudit{}
	app := &App{audit: fake.run}

	view, err := app.Audit(AuditRequest{Root: "/tmp/repository", Worktree: true})
	if err != nil {
		t.Fatalf("the audit failed: %v", err)
	}
	if fake.calls != 1 || fake.root != "/tmp/repository" || !fake.worktree || fake.base != "" {
		t.Fatalf("the use case received root=%q worktree=%v base=%q", fake.root, fake.worktree, fake.base)
	}
	if view.Origin != string(domain.OriginWorktree) {
		t.Errorf("the origin is %q, want %q", view.Origin, domain.OriginWorktree)
	}
	if view.SchemaVersion != "0.1" || view.FailingFindings != 2 || view.ChangedFiles != 3 {
		t.Errorf("the view does not carry the result of the use case: %+v", view)
	}
	if view.Document == nil {
		t.Error("the view does not carry the versioned document")
	}
}

// TestAuditPassesTheBaseRevision checks the other origin of the window.
func TestAuditPassesTheBaseRevision(t *testing.T) {
	fake := &fakeAudit{}
	app := &App{audit: fake.run}

	view, err := app.Audit(AuditRequest{Root: "/tmp/repository", Base: "main"})
	if err != nil {
		t.Fatalf("the audit failed: %v", err)
	}
	if fake.worktree || fake.base != "main" {
		t.Fatalf("the use case received worktree=%v base=%q", fake.worktree, fake.base)
	}
	if view.Origin != string(domain.OriginBase) || view.Base != "main" {
		t.Errorf("the view carries origin=%q base=%q", view.Origin, view.Base)
	}
}
