//go:build ui

package ui

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/domain"
)

// fakeAudit records the request of the window and returns a fixed result.
type fakeAudit struct {
	root     string
	origin   domain.ChangeOrigin
	base     string
	calls    int
	block    bool
	started  chan struct{}
	finished chan struct{}
}

// run implements AuditFunc.
func (f *fakeAudit) run(ctx context.Context, request AuditRunRequest) (*application.AuditResult, error) {
	f.calls++
	f.root = request.Root
	f.origin = request.Origin
	f.base = request.Base
	if f.started != nil {
		close(f.started)
	}
	if f.block {
		<-ctx.Done()
		if f.finished != nil {
			close(f.finished)
		}
		return nil, ctx.Err()
	}
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
	app := NewApp(fake.run, nil, "")

	_, err := app.Audit(AuditRequest{Root: "   ", Origin: "worktree"})
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
	app := NewApp(fake.run, nil, "")

	_, err := app.Audit(AuditRequest{Root: "/tmp/repository", Origin: "worktree", Base: "main"})
	if err == nil {
		t.Fatal("the audit accepted two origins")
	}
	if !strings.Contains(err.Error(), "base revision") {
		t.Errorf("the error does not explain the origin: %v", err)
	}
}

// TestAuditRequiresAnOrigin checks that a request without an origin is
// rejected.
func TestAuditRequiresAnOrigin(t *testing.T) {
	fake := &fakeAudit{}
	app := NewApp(fake.run, nil, "")

	_, err := app.Audit(AuditRequest{Root: "/tmp/repository"})
	if err == nil {
		t.Fatal("the audit accepted a request without an origin")
	}
	if !strings.Contains(err.Error(), "worktree") {
		t.Errorf("the error does not name the worktree: %v", err)
	}
}

// TestAuditRejectsAnUnknownOrigin checks that the origin is one of the
// three origins of the use case.
func TestAuditRejectsAnUnknownOrigin(t *testing.T) {
	fake := &fakeAudit{}
	app := NewApp(fake.run, nil, "")

	_, err := app.Audit(AuditRequest{Root: "/tmp/repository", Origin: "index"})
	if err == nil {
		t.Fatal("the audit accepted an unknown origin")
	}
	if !strings.Contains(err.Error(), "unknown origin") {
		t.Errorf("the error does not name the origin: %v", err)
	}
}

// TestAuditReturnsTheDocumentOfTheUseCase checks that the adapter passes the
// chosen origin to the use case and renders its result without reinterpreting
// it.
func TestAuditReturnsTheDocumentOfTheUseCase(t *testing.T) {
	fake := &fakeAudit{}
	app := NewApp(fake.run, nil, "")

	view, err := app.Audit(AuditRequest{Root: "/tmp/repository", Origin: "worktree", RunID: "run-test"})
	if err != nil {
		t.Fatalf("the audit failed: %v", err)
	}
	if fake.calls != 1 || fake.root != "/tmp/repository" || fake.origin != domain.OriginWorktree || fake.base != "" {
		t.Fatalf("the use case received root=%q origin=%q base=%q", fake.root, fake.origin, fake.base)
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
	if view.RunID != "run-test" {
		t.Errorf("the view carries the run identifier %q", view.RunID)
	}
}

// TestAuditPassesTheBaseRevision checks the base origin of the window.
func TestAuditPassesTheBaseRevision(t *testing.T) {
	fake := &fakeAudit{}
	app := NewApp(fake.run, nil, "")

	view, err := app.Audit(AuditRequest{Root: "/tmp/repository", Origin: "base", Base: "main"})
	if err != nil {
		t.Fatalf("the audit failed: %v", err)
	}
	if fake.origin != domain.OriginBase || fake.base != "main" {
		t.Fatalf("the use case received origin=%q base=%q", fake.origin, fake.base)
	}
	if view.Origin != string(domain.OriginBase) || view.Base != "main" {
		t.Errorf("the view carries origin=%q base=%q", view.Origin, view.Base)
	}
}

// TestAuditAcceptsTheStagedOrigin checks the third origin of the audit.
func TestAuditAcceptsTheStagedOrigin(t *testing.T) {
	fake := &fakeAudit{}
	app := NewApp(fake.run, nil, "")

	view, err := app.Audit(AuditRequest{Root: "/tmp/repository", Origin: "staged"})
	if err != nil {
		t.Fatalf("the audit failed: %v", err)
	}
	if fake.origin != domain.OriginStaged || fake.base != "" {
		t.Fatalf("the use case received origin=%q base=%q", fake.origin, fake.base)
	}
	if view.Origin != string(domain.OriginStaged) {
		t.Errorf("the view carries origin=%q", view.Origin)
	}
}

// TestAuditAcceptsTheLegacyOriginShape checks that the first prototype's
// request still works.
func TestAuditAcceptsTheLegacyOriginShape(t *testing.T) {
	fake := &fakeAudit{}
	app := NewApp(fake.run, nil, "")

	if _, err := app.Audit(AuditRequest{Root: "/tmp/repository", Worktree: true}); err != nil {
		t.Fatalf("the legacy worktree request failed: %v", err)
	}
	if fake.origin != domain.OriginWorktree {
		t.Fatalf("the use case received origin=%q", fake.origin)
	}
}

// TestAuditAsyncCancels checks that a running audit is cancellable.
func TestAuditAsyncCancels(t *testing.T) {
	fake := &fakeAudit{block: true, started: make(chan struct{}), finished: make(chan struct{})}
	app := NewApp(fake.run, nil, "")

	if _, err := app.AuditAsync(AuditRequest{Root: "/tmp/repository", Origin: "worktree"}, "run-cancel"); err != nil {
		t.Fatalf("the asynchronous audit did not start: %v", err)
	}
	select {
	case <-fake.started:
	case <-time.After(2 * time.Second):
		t.Fatal("the use case did not start")
	}
	if err := app.CancelAudit("run-cancel"); err != nil {
		t.Fatalf("the cancellation failed: %v", err)
	}
	select {
	case <-fake.finished:
	case <-time.After(2 * time.Second):
		t.Fatal("the use case did not observe the cancellation")
	}

	// The goroutine forgets the job after the use case returns. Poll the
	// cancellation until it reports that the run is no longer in progress.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := app.CancelAudit("run-cancel"); err != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("the cancellation of a finished run was accepted")
}

// TestAuditAsyncForgetsTheFinishedRuns checks that a new run drops the
// results of the previous finished runs, so the window does not grow without
// bound.
func TestAuditAsyncForgetsTheFinishedRuns(t *testing.T) {
	fake := &fakeAudit{}
	app := NewApp(fake.run, nil, "")

	if _, err := app.AuditAsync(AuditRequest{Root: "/tmp/repository", Origin: "worktree"}, "run-one"); err != nil {
		t.Fatalf("the first audit did not start: %v", err)
	}
	waitForResult(t, app, "run-one")

	if _, err := app.AuditAsync(AuditRequest{Root: "/tmp/repository", Origin: "worktree"}, "run-two"); err != nil {
		t.Fatalf("the second audit did not start: %v", err)
	}
	app.mu.Lock()
	_, kept := app.jobs["run-one"]
	count := len(app.jobs)
	app.mu.Unlock()
	if kept {
		t.Error("the finished run is still held after a new run started")
	}
	if count != 1 {
		t.Errorf("the window holds %d runs, want 1", count)
	}
}

// waitForResult waits until the asynchronous run stores its document.
func waitForResult(t *testing.T, app *App, runID string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		app.mu.Lock()
		job := app.jobs[runID]
		done := job != nil && job.result != nil
		app.mu.Unlock()
		if done {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the run %q did not finish", runID)
}

// fakeVerify records the request of the window and returns a fixed result.
type fakeVerify struct {
	root       string
	base       string
	claimsPath string
	calls      int
	block      bool
	err        error
	started    chan struct{}
	finished   chan struct{}
}

// run implements VerifyFunc.
func (f *fakeVerify) run(ctx context.Context, request VerifyRequest) (*application.VerifyResult, error) {
	f.calls++
	f.root = request.Root
	f.base = request.Base
	f.claimsPath = request.ClaimsPath
	if f.started != nil {
		close(f.started)
	}
	if f.block {
		<-ctx.Done()
		if f.finished != nil {
			close(f.finished)
		}
		return nil, ctx.Err()
	}
	if f.err != nil {
		return nil, f.err
	}
	return &application.VerifyResult{
		Document: &application.VerificationDocument{
			SchemaVersion: "0.1",
			ToolVersion:   "test",
			Base:          "main",
			Summary: application.VerificationSummary{
				Total:    3,
				ByStatus: map[string]int{"verified": 1, "contradicted": 1, "unresolved": 1},
			},
		},
		Contradicted: 1,
		Unresolved:   1,
	}, nil
}

// captureVerifyEvents installs an emitter that records the events of the
// verification, so the tests observe them without a window.
func captureVerifyEvents(t *testing.T, app *App) <-chan VerifyEvent {
	t.Helper()
	events := make(chan VerifyEvent, 16)
	app.emitEvent = func(name string, data any) {
		if name != VerifyEventName {
			t.Errorf("the adapter emitted the event %q, want %q", name, VerifyEventName)
			return
		}
		event, ok := data.(VerifyEvent)
		if !ok {
			t.Errorf("the event %q carries %T, want ui.VerifyEvent", name, data)
			return
		}
		events <- event
	}
	return events
}

// waitForVerifyEvent waits for the event of the given kind.
func waitForVerifyEvent(t *testing.T, events <-chan VerifyEvent, kind string) VerifyEvent {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case event := <-events:
			if event.Kind == kind {
				return event
			}
		case <-deadline:
			t.Fatalf("the adapter did not emit the event %q", kind)
			return VerifyEvent{}
		}
	}
}

// TestVerifyRequestIsSerializable checks that the window can send the request
// as JSON: no field of the request is a Go callback.
func TestVerifyRequestIsSerializable(t *testing.T) {
	data, err := json.Marshal(VerifyRequest{Root: "/tmp/repository", Base: "main", ClaimsPath: "/tmp/claims.json", RunID: "run-verify"})
	if err != nil {
		t.Fatalf("the verification request is not serializable: %v", err)
	}
	for _, field := range []string{`"root"`, `"base"`, `"claims_path"`, `"run_id"`} {
		if !strings.Contains(string(data), field) {
			t.Errorf("the request does not carry the field %s: %s", field, data)
		}
	}
}

// TestVerifyAsyncRejectsAMissingRoot checks that the adapter validates the
// repository before the use case runs.
func TestVerifyAsyncRejectsAMissingRoot(t *testing.T) {
	fake := &fakeVerify{}
	app := NewApp((&fakeAudit{}).run, fake.run, "")

	_, err := app.VerifyAsync(VerifyRequest{Root: "   ", Base: "main", ClaimsPath: "/tmp/claims.json"}, "run-verify")
	if err == nil {
		t.Fatal("the verification accepted an empty root")
	}
	if !strings.Contains(err.Error(), "repository field") {
		t.Errorf("the error does not name the field: %v", err)
	}
	if fake.calls != 0 {
		t.Errorf("the use case ran %d time(s) for an invalid request", fake.calls)
	}
}

// TestVerifyAsyncRejectsAMissingBase checks that the verification names the
// missing base revision.
func TestVerifyAsyncRejectsAMissingBase(t *testing.T) {
	fake := &fakeVerify{}
	app := NewApp((&fakeAudit{}).run, fake.run, "")

	_, err := app.VerifyAsync(VerifyRequest{Root: "/tmp/repository", ClaimsPath: "/tmp/claims.json"}, "run-verify")
	if err == nil {
		t.Fatal("the verification accepted an empty base")
	}
	if !strings.Contains(err.Error(), "base revision") {
		t.Errorf("the error does not name the base revision: %v", err)
	}
	if fake.calls != 0 {
		t.Errorf("the use case ran %d time(s) for an invalid request", fake.calls)
	}
}

// TestVerifyAsyncRejectsAMissingClaimsFile checks that the verification names
// the missing claim file.
func TestVerifyAsyncRejectsAMissingClaimsFile(t *testing.T) {
	fake := &fakeVerify{}
	app := NewApp((&fakeAudit{}).run, fake.run, "")

	_, err := app.VerifyAsync(VerifyRequest{Root: "/tmp/repository", Base: "main"}, "run-verify")
	if err == nil {
		t.Fatal("the verification accepted an empty claim file")
	}
	if !strings.Contains(err.Error(), "claims file field") {
		t.Errorf("the error does not name the claims file field: %v", err)
	}
	if fake.calls != 0 {
		t.Errorf("the use case ran %d time(s) for an invalid request", fake.calls)
	}
}

// TestVerifyAsyncNeedsAUseCase checks that a window without the verification
// use case reports it instead of crashing in the goroutine of the run.
func TestVerifyAsyncNeedsAUseCase(t *testing.T) {
	app := NewApp((&fakeAudit{}).run, nil, "")

	_, err := app.VerifyAsync(VerifyRequest{Root: "/tmp/repository", Base: "main", ClaimsPath: "/tmp/claims.json"}, "run-verify")
	if err == nil {
		t.Fatal("the verification started without a use case")
	}
	if !strings.Contains(err.Error(), "no verification use case") {
		t.Errorf("the error does not explain the missing use case: %v", err)
	}
}

// TestVerifyAsyncEmitsTheDocument checks that the asynchronous verification
// reports its document through the verify:event event.
func TestVerifyAsyncEmitsTheDocument(t *testing.T) {
	fake := &fakeVerify{}
	app := NewApp((&fakeAudit{}).run, fake.run, "")
	events := captureVerifyEvents(t, app)

	first, err := app.VerifyAsync(VerifyRequest{Root: "/tmp/repository", Base: "main", ClaimsPath: "/tmp/claims.json"}, "run-verify")
	if err != nil {
		t.Fatalf("the asynchronous verification did not start: %v", err)
	}
	if first == nil || first.RunID != "run-verify" || first.Progress == nil || first.Progress.Phase != VerifyPhaseValidating {
		t.Fatalf("the first event is %+v", first)
	}

	event := waitForVerifyEvent(t, events, EventDone)
	if event.Kind != EventDone {
		t.Fatalf("the event kind is %q", event.Kind)
	}
	if event.View == nil {
		t.Fatal("the done event does not carry the view")
	}
	if event.View.RunID != "run-verify" {
		t.Errorf("the view carries the run identifier %q", event.View.RunID)
	}
	if event.View.Document == nil {
		t.Error("the view does not carry the verification document")
	}
	if event.View.Contradicted != 1 || event.View.Unresolved != 1 {
		t.Errorf("the view carries contradicted=%d unresolved=%d", event.View.Contradicted, event.View.Unresolved)
	}
	if event.Progress == nil || event.Progress.Phase != VerifyPhaseDone {
		t.Fatalf("the done event does not carry the finished progress: %+v", event.Progress)
	}
	if event.Progress.Total != 3 || event.Progress.Contradicted != 1 || event.Progress.Unresolved != 1 {
		t.Errorf("the finished progress carries total=%d contradicted=%d unresolved=%d",
			event.Progress.Total, event.Progress.Contradicted, event.Progress.Unresolved)
	}
	if fake.root != "/tmp/repository" || fake.base != "main" || fake.claimsPath != "/tmp/claims.json" {
		t.Errorf("the use case received root=%q base=%q claims=%q", fake.root, fake.base, fake.claimsPath)
	}
	waitForResult(t, app, "run-verify")
}

// TestVerifyAsyncEmitsTheError checks that a failed verification reports its
// error through the event.
func TestVerifyAsyncEmitsTheError(t *testing.T) {
	fake := &fakeVerify{err: errors.New("cannot read the claim file: choose an existing file")}
	app := NewApp((&fakeAudit{}).run, fake.run, "")
	events := captureVerifyEvents(t, app)

	if _, err := app.VerifyAsync(VerifyRequest{Root: "/tmp/repository", Base: "main", ClaimsPath: "/tmp/claims.json"}, "run-verify"); err != nil {
		t.Fatalf("the asynchronous verification did not start: %v", err)
	}

	event := waitForVerifyEvent(t, events, EventError)
	if !strings.Contains(event.Error, "choose an existing file") {
		t.Errorf("the error event carries %q", event.Error)
	}
}

// TestVerifyAsyncEmitsTheCancellation checks that a cancelled verification
// reports the cancellation instead of an error.
func TestVerifyAsyncEmitsTheCancellation(t *testing.T) {
	fake := &fakeVerify{block: true, started: make(chan struct{}), finished: make(chan struct{})}
	app := NewApp((&fakeAudit{}).run, fake.run, "")
	events := captureVerifyEvents(t, app)

	if _, err := app.VerifyAsync(VerifyRequest{Root: "/tmp/repository", Base: "main", ClaimsPath: "/tmp/claims.json"}, "run-verify"); err != nil {
		t.Fatalf("the asynchronous verification did not start: %v", err)
	}
	select {
	case <-fake.started:
	case <-time.After(2 * time.Second):
		t.Fatal("the use case did not start")
	}
	if err := app.CancelAudit("run-verify"); err != nil {
		t.Fatalf("the cancellation failed: %v", err)
	}
	select {
	case <-fake.finished:
	case <-time.After(2 * time.Second):
		t.Fatal("the use case did not observe the cancellation")
	}

	event := waitForVerifyEvent(t, events, EventCancelled)
	if event.Progress == nil || event.Progress.Phase != VerifyPhaseCancelled {
		t.Fatalf("the cancelled event does not carry the cancelled phase: %+v", event.Progress)
	}
}

// TestChooseFileNeedsAReadyWindow checks that the file dialog reports the
// missing window instead of panicking.
func TestChooseFileNeedsAReadyWindow(t *testing.T) {
	app := NewApp((&fakeAudit{}).run, (&fakeVerify{}).run, "")

	path, err := app.ChooseFile("", []string{".json"})
	if err == nil {
		t.Fatalf("the dialog returned %q without a window", path)
	}
	if !strings.Contains(err.Error(), "the window is not ready yet") {
		t.Errorf("the error does not explain how to recover: %v", err)
	}
}

// TestEncodeExportJSON checks that the export writes the document with the
// layout of the command line.
func TestEncodeExportJSON(t *testing.T) {
	result := &application.AuditResult{
		Run: &domain.AnalysisRun{
			SchemaVersion: "0.1",
			ToolVersion:   "test",
			Origin:        "worktree",
			Findings:      []domain.Finding{},
		},
		FailOn: "error",
	}

	data, err := EncodeExport(result, "json")
	if err != nil {
		t.Fatalf("the JSON export failed: %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("the export is not a JSON document:\n%s", data)
	}
	if !strings.Contains(string(data), "\n  \"schema_version\": \"0.1\"") {
		t.Errorf("the export does not use the two space indentation of the CLI:\n%s", data)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Error("the export does not end with a newline")
	}
}

// TestEncodeExportRejectsAnUnknownFormat checks the validation of the
// format.
func TestEncodeExportRejectsAnUnknownFormat(t *testing.T) {
	result := &application.AuditResult{Run: &domain.AnalysisRun{SchemaVersion: "0.1"}}

	if _, err := EncodeExport(result, "html"); err == nil {
		t.Fatal("the export accepted an unknown format")
	}
}

// TestEnsureOutsideRoot checks that the export never writes inside the
// audited repository.
func TestEnsureOutsideRoot(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "report.json")
	outside := filepath.Join(t.TempDir(), "report.json")

	if err := ensureOutsideRoot(inside, root); err == nil {
		t.Error("the export accepted a path inside the audited repository")
	}
	if err := ensureOutsideRoot(outside, root); err != nil {
		t.Errorf("the export rejected a path outside the repository: %v", err)
	}
}

// TestExportWithoutAFinishedRun checks that the export names the missing
// run.
func TestExportWithoutAFinishedRun(t *testing.T) {
	app := NewApp((&fakeAudit{}).run, nil, "")

	if _, err := app.Export(ExportRequest{RunID: "missing", Format: "json"}); err == nil {
		t.Fatal("the export accepted an unknown run")
	}
}

// TestEmbeddedFrontendIsNotEmpty checks that the embed of the compiled
// frontend is reachable from the package.
func TestEmbeddedFrontendIsNotEmpty(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "frontend", "dist")); err != nil {
		t.Fatalf("the frontend dist directory is not reachable: %v", err)
	}
}
