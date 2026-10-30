//go:build ui

// Package ui implements the Wails v2 adapter of Codixai.
//
// The package is built only with the ui tag: the default build of the tool
// neither compiles it nor reaches Wails, the webview or the embedded
// frontend. It holds the window, the bound methods that the Svelte frontend
// calls, and the embed of the compiled frontend in frontend/dist.
//
// The adapter only translates. It validates what arrives from the window,
// because the frontend is untrusted input, and it delegates every decision
// to the use cases of the application layer: it holds no analysis rule, no
// policy and no persistence. The composition of the providers stays in the
// command that starts the window.
//
// The build tags of the variant are the ones of Wails v2: the window is
// compiled with "ui production" (or "ui dev"), and the build without the
// Wails tags reports how to add them instead of failing inside the toolkit.
// The "ui bindings" combination exists only to generate the TypeScript
// bindings with the Wails generator; it never opens a window.
package ui

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/barceloch/codixai/internal/index"
	"github.com/barceloch/codixai/internal/visualization"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AutoQuitEnv names the environment variable that closes the window after
// the given duration.
//
// It exists for the measurement of the startup time of the ui variant: the
// window starts, reaches its first frame and quits without a person closing
// it. It is empty by default and it never changes the behavior of the window
// when it is not set.
const AutoQuitEnv = "CODIXAI_UI_AUTOQUIT"

// AuditEventName is the event that carries the progress of an asynchronous
// audit and its outcome.
const AuditEventName = "audit:event"

// VerifyEventName is the event that carries the progress of an asynchronous
// verification and its outcome.
const VerifyEventName = "verify:event"

// The phases of an asynchronous audit.
//
// They are the real steps of the use case, not a timer: the change source
// reports when the changes were collected, the analyzer when the rules were
// applied, and the adapter when the document is ready. The policy and the
// baseline of the repository run inside the use case after the analyzer
// returns, so no phase claims to observe them.
const (
	// PhaseValidating means the adapter is validating the request.
	PhaseValidating = "validating"
	// PhaseCollecting means the change source is collecting the diff.
	PhaseCollecting = "collecting"
	// PhaseCollected means the change source returned the change set.
	PhaseCollected = "collected"
	// PhaseAnalyzing means the rule catalog is being applied.
	PhaseAnalyzing = "analyzing"
	// PhaseAnalyzed means the rule catalog returned the findings.
	PhaseAnalyzed = "analyzed"
	// PhaseDone means the document is ready.
	PhaseDone = "done"
	// PhaseCancelled means the person cancelled the run.
	PhaseCancelled = "cancelled"
)

// The kinds of an audit event.
const (
	// EventProgress carries the progress of a running audit.
	EventProgress = "progress"
	// EventDone carries the finished document.
	EventDone = "done"
	// EventError carries the failure of the audit.
	EventError = "error"
	// EventCancelled reports that the run was cancelled.
	EventCancelled = "cancelled"
)

// AuditRequest is the request that the window sends to the audit use case.
type AuditRequest struct {
	// Root is the path of the repository to audit.
	Root string `json:"root"`
	// Origin selects the changes: worktree, staged or base.
	//
	// The field is the preferred one. The Worktree and Staged fields are
	// kept for the callers of the first prototype and are accepted only
	// when Origin is empty.
	Origin string `json:"origin,omitempty"`
	// Worktree audits the changes that are not committed yet.
	Worktree bool `json:"worktree,omitempty"`
	// Staged audits the staging area.
	Staged bool `json:"staged,omitempty"`
	// Base is the revision to compare the repository with. It is empty
	// when the origin is the worktree or the staging area.
	Base string `json:"base,omitempty"`
	// RunID identifies the run. The window generates it so that the
	// progress events and the export refer to the same run. When it is
	// empty the adapter generates one.
	RunID string `json:"run_id,omitempty"`
}

// AuditProgress is one step of an asynchronous audit.
type AuditProgress struct {
	// RunID identifies the run that produced the progress.
	RunID string `json:"run_id"`
	// Phase is one of the phases of the audit.
	Phase string `json:"phase"`
	// ChangedFiles counts the files of the change set, once collected.
	ChangedFiles int `json:"changed_files,omitempty"`
	// Findings counts the findings of the analysis, once analyzed.
	Findings int `json:"findings,omitempty"`
}

// AuditEvent is the payload of the audit:event event.
//
// The window receives one event per progress step and one final event with
// the document, the error or the cancellation. The frontend never polls the
// adapter: the events carry the whole state.
type AuditEvent struct {
	// RunID identifies the run.
	RunID string `json:"run_id"`
	// Kind is progress, done, error or cancelled.
	Kind string `json:"kind"`
	// Progress carries the step when Kind is progress.
	Progress *AuditProgress `json:"progress,omitempty"`
	// View carries the document when Kind is done.
	View *AuditView `json:"view,omitempty"`
	// Error explains the failure when Kind is error.
	Error string `json:"error,omitempty"`
}

// AuditView is the result that the window renders.
//
// The Document field is the same versioned document that the command line
// writes with --format json: the interface never reinterprets a finding, it
// renders what the use case decided. The other fields are the context that
// the screen needs to present the document (the origin, the threshold, the
// counts of the policy and the change set with the hunks of the evidence);
// they are not part of the document and never change its bytes.
type AuditView struct {
	// RunID identifies the run that produced the view.
	RunID string `json:"run_id"`
	// SchemaVersion is the version of the schema of the document.
	SchemaVersion string `json:"schema_version"`
	// ToolVersion is the version of the tool that produced the document.
	ToolVersion string `json:"tool_version"`
	// Root is the audited repository.
	Root string `json:"root"`
	// Origin names the origin of the audited changes.
	Origin string `json:"origin"`
	// Base is the base revision, when the origin is a base.
	Base string `json:"base,omitempty"`
	// ChangedFiles counts the files that the analysis considered.
	ChangedFiles int `json:"changed_files"`
	// FailOn is the severity that makes an audit fail.
	FailOn string `json:"fail_on"`
	// Failed states whether a finding reaches the threshold.
	Failed bool `json:"failed"`
	// FailingFindings counts the findings that reach the threshold.
	FailingFindings int `json:"failing_findings"`
	// KnownFindings counts the findings that the baseline already knows.
	KnownFindings int `json:"known_findings"`
	// SuppressedFindings counts the findings that the configuration
	// suppresses.
	SuppressedFindings int `json:"suppressed_findings"`
	// ExcludedFiles holds the paths that the configuration excludes.
	ExcludedFiles []string `json:"excluded_files,omitempty"`
	// Document is the analysis document produced by the use case.
	Document *domain.AnalysisRun `json:"document"`
	// Changes is the change set that the analysis inspected. It holds the
	// hunks that the evidence of a finding references.
	Changes *domain.ChangeSet `json:"changes,omitempty"`
}

// AuditRunRequest is the request that the adapter sends to the audit
// composition root.
//
// It is not bound to the frontend: it carries the already validated origin
// and the callback that reports the real phases of the use case.
type AuditRunRequest struct {
	// Root is the absolute path of the repository.
	Root string
	// Origin is the selected origin.
	Origin domain.ChangeOrigin
	// Base is the base revision, when the origin is base.
	Base string
	// Progress reports the phases of the run. It may be nil.
	Progress func(AuditProgress)
}

// AuditFunc runs the read-only audit use case.
//
// The command that starts the window composes the providers, so that the
// adapter does not know how the changes are collected or how the rules are
// applied.
type AuditFunc func(ctx context.Context, request AuditRunRequest) (*application.AuditResult, error)

// Options configures the window.
type Options struct {
	// Title is the title of the window.
	Title string
	// Root is the repository that the window shows when it opens.
	Root string
	// Audit runs the audit use case.
	Audit AuditFunc
	// Verify runs the verification use case.
	Verify VerifyFunc
}

// VerifyFunc runs the read-only verification use case.
//
// The command that starts the window composes the providers, so that the
// adapter does not know how the changes are collected or how the claims are
// evaluated.
type VerifyFunc func(ctx context.Context, request VerifyRequest) (*application.VerifyResult, error)

// VerifyRequest is the request that the window sends to the verification
// use case.
//
// Every field is serializable, so the frontend builds the request as a plain
// JSON object: the progress of the run travels through the verify:event
// event and never through a callback of the request.
type VerifyRequest struct {
	// Root is the absolute path of the repository.
	Root string `json:"root"`
	// Base is the base revision of the comparison, as the user wrote it.
	Base string `json:"base"`
	// ClaimsPath is the path of the claim file to verify.
	ClaimsPath string `json:"claims_path"`
	// RunID identifies the run. The window generates it so that the
	// progress events and the export refer to the same run. When it is
	// empty the adapter generates one.
	RunID string `json:"run_id,omitempty"`
}

// VerifyProgress reports one phase of a verification.
type VerifyProgress struct {
	// RunID identifies the verification run.
	RunID string `json:"run_id"`
	// Phase is the current phase of the run.
	Phase string `json:"phase"`
	// Total is the number of claims when the evaluation finishes.
	Total int `json:"total,omitempty"`
	// Contradicted counts the contradicted claims when the evaluation finishes.
	Contradicted int `json:"contradicted,omitempty"`
	// Unresolved counts the unresolved claims when the evaluation finishes.
	Unresolved int `json:"unresolved,omitempty"`
	// Error carries the failure message when the phase is error.
	Error string `json:"error,omitempty"`
}

// VerifyView is the result of a verification that the window renders.
//
// The Document field is the same versioned document that the command line
// writes with --format json: the interface never reinterprets a claim, it
// renders the state that the use case decided. The other fields are the
// context that the screen needs to present the document; they are not part
// of the document and never change its bytes.
type VerifyView struct {
	// RunID identifies the run that produced the view.
	RunID string `json:"run_id"`
	// Root is the verified repository.
	Root string `json:"root"`
	// Base is the base revision of the comparison.
	Base string `json:"base"`
	// Failed states whether a claim reaches the failure threshold.
	Failed bool `json:"failed"`
	// Contradicted counts the contradicted claims.
	Contradicted int `json:"contradicted"`
	// Unresolved counts the unresolved claims.
	Unresolved int `json:"unresolved"`
	// Document is the verification document produced by the use case.
	Document *application.VerificationDocument `json:"document"`
}

// VerifyEvent is the payload of the verify:event event.
//
// The window receives one event per progress step and one final event with
// the document, the error or the cancellation. The frontend never polls the
// adapter: the events carry the whole state.
type VerifyEvent struct {
	// RunID identifies the run.
	RunID string `json:"run_id"`
	// Kind is progress, done, error or cancelled.
	Kind string `json:"kind"`
	// Progress carries the step when Kind is progress.
	Progress *VerifyProgress `json:"progress,omitempty"`
	// View carries the document when Kind is done.
	View *VerifyView `json:"view,omitempty"`
	// Error explains the failure when Kind is error.
	Error string `json:"error,omitempty"`
}

// VerifyPhase are the phases of a verification, in order.
const (
	// VerifyPhaseValidating means the adapter is validating the request.
	VerifyPhaseValidating = "validating"
	// VerifyPhaseCollecting means the changes are being collected and the
	// claim file is being evaluated.
	VerifyPhaseCollecting = "collecting"
	// VerifyPhaseCollected means the changes were collected. The adapter
	// does not observe it today: the use case exposes no hook for it.
	VerifyPhaseCollected = "collected"
	// VerifyPhaseEvaluating means the claims are being evaluated. The
	// adapter does not observe it today: the use case exposes no hook for
	// it.
	VerifyPhaseEvaluating = "evaluating"
	// VerifyPhaseEvaluated means the claims were evaluated. The adapter
	// does not observe it today: the use case exposes no hook for it.
	VerifyPhaseEvaluated = "evaluated"
	// VerifyPhaseDone means the document is ready.
	VerifyPhaseDone = "done"
	// VerifyPhaseError means the verification failed.
	VerifyPhaseError = "error"
	// VerifyPhaseCancelled means the person cancelled the run.
	VerifyPhaseCancelled = "cancelled"
)

// auditJob is the state of one asynchronous audit or verification.
type auditJob struct {
	// cancel stops the run.
	cancel context.CancelFunc
	// root is the audited repository, kept for the export guard.
	root string
	// result is the finished document. It is nil until the run ends.
	result any
}

// App is the struct bound to the frontend.
//
// Its exported methods are the whole surface of the window: the frontend
// cannot reach anything else.
type App struct {
	ctx    context.Context
	audit  AuditFunc
	verify VerifyFunc
	root   string

	// emitEvent replaces the event emitter of the window. It is nil by
	// default, and the adapter then emits through the runtime of Wails.
	// The tests inject it to observe the events without a window.
	emitEvent func(name string, data any)

	mu   sync.Mutex
	jobs map[string]*auditJob
}

// NewApp returns the bound struct of the window.
//
// It exists so that the tests can exercise the adapter without opening a
// window. Run builds the same struct from its options.
func NewApp(audit AuditFunc, verify VerifyFunc, root string) *App {
	return &App{audit: audit, verify: verify, root: root, jobs: map[string]*auditJob{}}
}

// Run opens the window and blocks until it is closed.
//
// It must be called from the main goroutine: the toolkit of the window
// owns the main thread. The context is checked before the window opens;
// the lifecycle of the window itself belongs to the toolkit.
//
// Wails v2 compiles its application layer only with the production (or
// dev) build tag, so the binary of the variant is built with "ui
// production". The build with the ui tag alone still compiles and reports
// the missing tags when the window is opened.
func Run(ctx context.Context, opts Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if opts.Audit == nil {
		return fmt.Errorf("the window needs the audit use case: pass it in ui.Options.Audit")
	}
	app := NewApp(opts.Audit, opts.Verify, opts.Root)

	title := opts.Title
	if title == "" {
		title = "Codixai"
	}

	err := runWindow(app, title)
	if err != nil {
		return fmt.Errorf("cannot open the window: %w", err)
	}
	return nil
}

// startup stores the context of the runtime and schedules the automatic
// quit of the measurement mode.
//
// Every call of the v2 runtime needs this context, so it is captured before
// the window asks for anything.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	value := strings.TrimSpace(os.Getenv(AutoQuitEnv))
	if value == "" {
		return
	}
	after, err := time.ParseDuration(value)
	if err != nil || after <= 0 {
		fmt.Fprintf(os.Stderr, "codixai: %s=%q is not a positive duration (for example 1s): the window stays open\n", AutoQuitEnv, value)
		return
	}
	time.AfterFunc(after, func() { runtime.Quit(a.ctx) })
}

// Root returns the repository that the window shows when it opens.
func (a *App) Root() string { return a.root }

// Audit runs the read-only audit use case and returns its document.
//
// The window is untrusted input: an empty root or two origins at once are
// rejected with an error that tells the person what to choose. The path is
// resolved by the use case like the command line does, and it never escapes
// the selected root.
func (a *App) Audit(request AuditRequest) (*AuditView, error) {
	origin, base, err := resolveRequestOrigin(request)
	if err != nil {
		return nil, err
	}
	root := strings.TrimSpace(request.Root)
	if root == "" {
		return nil, fmt.Errorf("the audit needs the path of a repository: write it in the repository field")
	}

	runID := strings.TrimSpace(request.RunID)
	if runID == "" {
		runID = newRunID()
	}

	result, err := a.audit(a.runtimeContext(), AuditRunRequest{Root: root, Origin: origin, Base: base})
	if err != nil {
		return nil, err
	}
	if result == nil || result.Run == nil {
		return nil, fmt.Errorf("the audit did not produce a document: run 'codixai audit --diff --worktree' in the terminal to see the error of the analysis")
	}
	a.storeResult(runID, root, result)
	return a.view(runID, root, origin, base, result), nil
}

// AuditAsync starts an audit and returns its first progress event.
//
// The run happens in its own goroutine and reports every phase through the
// audit:event event. The method returns immediately, so the window stays
// responsive and can cancel the run with CancelAudit.
func (a *App) AuditAsync(request AuditRequest, runID string) (*AuditEvent, error) {
	origin, base, err := resolveRequestOrigin(request)
	if err != nil {
		return nil, err
	}
	root := strings.TrimSpace(request.Root)
	if root == "" {
		return nil, fmt.Errorf("the audit needs the path of a repository: write it in the repository field")
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		runID = strings.TrimSpace(request.RunID)
	}
	if runID == "" {
		runID = newRunID()
	}

	ctx, cancel := context.WithCancel(a.runtimeContext())
	a.mu.Lock()
	if a.jobs == nil {
		a.jobs = map[string]*auditJob{}
	}
	a.pruneFinishedLocked(runID)
	if _, exists := a.jobs[runID]; exists {
		a.mu.Unlock()
		cancel()
		return nil, fmt.Errorf("the run %q is already in progress: wait for it or cancel it", runID)
	}
	a.jobs[runID] = &auditJob{cancel: cancel, root: root}
	a.mu.Unlock()

	progress := &AuditProgress{RunID: runID, Phase: PhaseValidating}
	go a.runAsync(ctx, runID, root, origin, base)
	return &AuditEvent{RunID: runID, Kind: EventProgress, Progress: progress}, nil
}

// runAsync runs the audit and reports its phases and its outcome.
func (a *App) runAsync(ctx context.Context, runID, root string, origin domain.ChangeOrigin, base string) {
	report := func(progress AuditProgress) {
		progress.RunID = runID
		a.emit(AuditEvent{RunID: runID, Kind: EventProgress, Progress: &progress})
	}

	result, err := a.audit(ctx, AuditRunRequest{
		Root:   root,
		Origin: origin,
		Base:   base,
		Progress: func(progress AuditProgress) {
			report(progress)
		},
	})
	if err != nil {
		a.removeJob(runID)
		if ctx.Err() != nil {
			a.emit(AuditEvent{
				RunID:    runID,
				Kind:     EventCancelled,
				Progress: &AuditProgress{RunID: runID, Phase: PhaseCancelled},
			})
			return
		}
		a.emit(AuditEvent{RunID: runID, Kind: EventError, Error: err.Error()})
		return
	}
	if result == nil || result.Run == nil {
		a.removeJob(runID)
		a.emit(AuditEvent{
			RunID: runID,
			Kind:  EventError,
			Error: "the audit did not produce a document: run 'codixai audit --diff --worktree' in the terminal to see the error of the analysis",
		})
		return
	}

	a.mu.Lock()
	if job := a.jobs[runID]; job != nil {
		job.cancel = nil
		job.result = result
	}
	a.mu.Unlock()

	view := a.view(runID, root, origin, base, result)
	a.emit(AuditEvent{
		RunID: runID,
		Kind:  EventDone,
		View:  view,
		Progress: &AuditProgress{
			RunID:        runID,
			Phase:        PhaseDone,
			ChangedFiles: view.ChangedFiles,
			Findings:     len(view.Document.Findings),
		},
	})
}

// CancelAudit cancels a running audit.
//
// The cancellation is cooperative: the use case checks the context between
// the files, so a run that is finishing may still deliver its document.
func (a *App) CancelAudit(runID string) error {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return fmt.Errorf("the cancellation needs the identifier of the run")
	}
	a.mu.Lock()
	job := a.jobs[runID]
	a.mu.Unlock()
	if job == nil || job.cancel == nil {
		return fmt.Errorf("the run %q is not in progress: start an audit before cancelling it", runID)
	}
	job.cancel()
	return nil
}

// VerifyAsync starts a verification of a claim file against the repository
// changes and returns its first progress step.
//
// The run happens in its own goroutine and reports its phases and its outcome
// through the verify:event event, so the window stays responsive and can
// cancel the run with CancelAudit. The request carries the root repository,
// the base revision and the claims file; the adapter validates them before
// the use case runs. The verification never modifies the analyzed
// repository: the manifests and the reports of the repository are read as
// data. Cancellation is cooperative: the context is checked between steps.
func (a *App) VerifyAsync(request VerifyRequest, runID string) (*VerifyEvent, error) {
	root := strings.TrimSpace(request.Root)
	if root == "" {
		return nil, fmt.Errorf("the verification needs the root of the repository: write it in the repository field")
	}
	base := strings.TrimSpace(request.Base)
	if base == "" {
		return nil, fmt.Errorf("the verification needs a base revision: write one such as 'main' in the base field")
	}
	claimsPath := strings.TrimSpace(request.ClaimsPath)
	if claimsPath == "" {
		return nil, fmt.Errorf("the verification needs the path of the claim file: load it in the claims file field")
	}
	if a.verify == nil {
		return nil, fmt.Errorf("the window has no verification use case: start it with 'codixai ui'")
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		runID = strings.TrimSpace(request.RunID)
	}
	if runID == "" {
		runID = newRunID()
	}

	ctx, cancel := context.WithCancel(a.runtimeContext())
	a.mu.Lock()
	if a.jobs == nil {
		a.jobs = map[string]*auditJob{}
	}
	a.pruneFinishedLocked(runID)
	if _, exists := a.jobs[runID]; exists {
		a.mu.Unlock()
		cancel()
		return nil, fmt.Errorf("the verification %q is already in progress: wait for it or cancel it", runID)
	}
	a.jobs[runID] = &auditJob{cancel: cancel, root: root}
	a.mu.Unlock()

	progress := &VerifyProgress{RunID: runID, Phase: VerifyPhaseValidating}
	go a.runVerifyAsync(ctx, runID, root, base, claimsPath)
	return &VerifyEvent{RunID: runID, Kind: EventProgress, Progress: progress}, nil
}

// runVerifyAsync runs the verification and reports its phases and its
// outcome through the verify:event event.
//
// Only the phases that the adapter can observe are reported: the collection
// before the use case runs, because the adapter hands over the whole call,
// and the end, because the use case returns the document. The use case
// exposes no hook for the collection or the evaluation of the claims, so the
// adapter does not claim to observe them.
func (a *App) runVerifyAsync(ctx context.Context, runID, root, base, claimsPath string) {
	a.emitVerify(VerifyEvent{
		RunID:    runID,
		Kind:     EventProgress,
		Progress: &VerifyProgress{RunID: runID, Phase: VerifyPhaseCollecting},
	})

	result, err := a.verify(ctx, VerifyRequest{Root: root, Base: base, ClaimsPath: claimsPath, RunID: runID})
	if err != nil {
		a.removeJob(runID)
		if ctx.Err() != nil {
			a.emitVerify(VerifyEvent{
				RunID:    runID,
				Kind:     EventCancelled,
				Progress: &VerifyProgress{RunID: runID, Phase: VerifyPhaseCancelled},
			})
			return
		}
		a.emitVerify(VerifyEvent{RunID: runID, Kind: EventError, Error: err.Error()})
		return
	}
	if result == nil || result.Document == nil {
		a.removeJob(runID)
		a.emitVerify(VerifyEvent{
			RunID: runID,
			Kind:  EventError,
			Error: "the verification did not produce a document: run 'codixai verify --base main --claims claims.json' in the terminal to see the error of the analysis",
		})
		return
	}

	a.mu.Lock()
	if job := a.jobs[runID]; job != nil {
		job.cancel = nil
		job.result = result
	}
	a.mu.Unlock()

	view := &VerifyView{
		RunID:        runID,
		Root:         root,
		Base:         base,
		Failed:       result.Failed,
		Contradicted: result.Contradicted,
		Unresolved:   result.Unresolved,
		Document:     result.Document,
	}
	a.emitVerify(VerifyEvent{
		RunID: runID,
		Kind:  EventDone,
		View:  view,
		Progress: &VerifyProgress{
			RunID:        runID,
			Phase:        VerifyPhaseDone,
			Total:        result.Document.Summary.Total,
			Contradicted: result.Contradicted,
			Unresolved:   result.Unresolved,
		},
	})
}

// Verify runs a verification synchronously and returns its result.
//
// It is the same use case as VerifyAsync, with the same validation of the
// request, and it emits no event: the caller already holds the result.
func (a *App) Verify(request VerifyRequest) (*application.VerifyResult, error) {
	root := strings.TrimSpace(request.Root)
	if root == "" {
		return nil, fmt.Errorf("the verification needs the root of the repository: write it in the repository field")
	}
	base := strings.TrimSpace(request.Base)
	if base == "" {
		return nil, fmt.Errorf("the verification needs a base revision: write one such as 'main' in the base field")
	}
	claimsPath := strings.TrimSpace(request.ClaimsPath)
	if claimsPath == "" {
		return nil, fmt.Errorf("the verification needs the path of the claim file: load it in the claims file field")
	}
	if a.verify == nil {
		return nil, fmt.Errorf("the window has no verification use case: open it with 'codixai ui'")
	}
	runID := strings.TrimSpace(request.RunID)
	if runID == "" {
		runID = newRunID()
	}
	result, err := a.verify(a.runtimeContext(), VerifyRequest{Root: root, Base: base, ClaimsPath: claimsPath, RunID: runID})
	if err != nil {
		return nil, err
	}
	if result == nil || result.Document == nil {
		return nil, fmt.Errorf("the verification did not produce a document: run 'codixai verify --base main --claims claims.json' in the terminal to see the error of the analysis")
	}
	a.storeVerifyResult(runID, root, result)
	return result, nil
}

// Visualize performs a structural visualization of the repository.
// It returns symbols and dependencies found in the index.
func (a *App) Visualize(root string) (*visualization.VisualizationResult, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("the visualization needs the root of the repository: write it in the repository field")
	}

	idx := index.New()
	if err := idx.Build(a.runtimeContext(), root); err != nil {
		return nil, fmt.Errorf("failed to build index: %w", err)
	}

	return visualization.Visualize(idx), nil
}

// ChooseRepository opens the native directory dialog and returns the path
// that the person chose, or an empty string when the dialog is cancelled.
//
// The title comes from the frontend so that the dialog speaks the language
// of the interface.
func (a *App) ChooseRepository(title string) (string, error) {
	ctx := a.ctx
	if ctx == nil {
		return "", fmt.Errorf("the window is not ready yet: wait for it to open and try again")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Select the repository to audit"
	}
	return runtime.OpenDirectoryDialog(ctx, runtime.OpenDialogOptions{Title: title})
}

// ChooseFile opens the native file dialog and returns the path that the
// person chose, or an empty string when the dialog is cancelled.
//
// The title and the extensions come from the frontend so that the dialog
// speaks the language of the interface and only offers the files that the
// screen can read. An extension is written with its dot, for example
// ".json"; the dialog shows them as "*.json" and accepts any of them.
func (a *App) ChooseFile(title string, extensions []string) (string, error) {
	ctx := a.ctx
	if ctx == nil {
		return "", fmt.Errorf("the window is not ready yet: wait for it to open and try again")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Select a file"
	}

	var filters []runtime.FileFilter
	if filter, ok := fileFilter(extensions); ok {
		filters = []runtime.FileFilter{filter}
	}
	return runtime.OpenFileDialog(ctx, runtime.OpenDialogOptions{Title: title, Filters: filters})
}

// ExportRequest describes the report to write.
type ExportRequest struct {
	// RunID identifies the run whose document is exported.
	RunID string `json:"run_id"`
	// Format is json or sarif.
	Format string `json:"format"`
	// SuggestedName is the name that the save dialog proposes.
	SuggestedName string `json:"suggested_name,omitempty"`
	// Title is the title of the save dialog, in the language of the
	// interface.
	Title string `json:"title,omitempty"`
}

// Export writes the document of a finished run to the path that the person
// chooses in the native save dialog.
//
// The path is the only place the adapter writes to. A path inside the
// audited repository is rejected: the tool never writes into the repository
// that it reads.
func (a *App) Export(request ExportRequest) (string, error) {
	result, root, err := a.result(request.RunID)
	if err != nil {
		return "", err
	}
	data, err := EncodeExport(result, request.Format)
	if err != nil {
		return "", err
	}

	ctx := a.ctx
	if ctx == nil {
		return "", fmt.Errorf("the window is not ready yet: wait for it to open and try again")
	}

	format := strings.ToLower(strings.TrimSpace(request.Format))
	if format == "" {
		format = "json"
	}
	extension := exportExtension(format)
	suggested := strings.TrimSpace(request.SuggestedName)
	if suggested == "" {
		suggested = "codixai-audit" + extension
	}
	if !strings.HasSuffix(strings.ToLower(suggested), extension) {
		suggested += extension
	}
	title := strings.TrimSpace(request.Title)
	if title == "" {
		title = "Save the audit report"
	}

	path, err := runtime.SaveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: suggested,
		Filters: []runtime.FileFilter{{
			DisplayName: strings.ToUpper(strings.TrimPrefix(extension, ".")),
			Pattern:     "*" + extension,
		}},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	if err := ensureOutsideRoot(path, root); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("cannot write the report to %s: %w", path, err)
	}
	return path, nil
}

// runtimeContext returns the context of the runtime, or a background
// context when a call arrives before OnStartup.
func (a *App) runtimeContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// storeResult keeps the document of a synchronous run for its export.
func (a *App) storeResult(runID, root string, result *application.AuditResult) {
	a.storeValue(runID, root, result)
}

// storeVerifyResult keeps the document of a synchronous verification for its
// export.
func (a *App) storeVerifyResult(runID, root string, result *application.VerifyResult) {
	a.storeValue(runID, root, result)
}

// storeValue keeps the document of a finished run for its export.
func (a *App) storeValue(runID, root string, result any) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.jobs == nil {
		a.jobs = map[string]*auditJob{}
	}
	a.pruneFinishedLocked(runID)
	a.jobs[runID] = &auditJob{root: root, result: result}
}

// pruneFinishedLocked forgets the finished runs except the one to keep.
//
// A window can run many audits in one session. Each result holds the
// document and the change set, so keeping every finished run would grow
// without bound. The screen only exports the run it shows, so the older
// finished runs are dropped when a new one starts.
func (a *App) pruneFinishedLocked(keep string) {
	for id, job := range a.jobs {
		if id == keep || job == nil {
			continue
		}
		if job.cancel == nil && job.result != nil {
			delete(a.jobs, id)
		}
	}
}

// removeJob forgets a run that did not produce a document.
func (a *App) removeJob(runID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.jobs, runID)
}

// result returns the document of a finished run.
func (a *App) result(runID string) (any, string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	job := a.jobs[strings.TrimSpace(runID)]
	if job == nil || job.result == nil {
		return nil, "", fmt.Errorf("the run %q has no result to export: run the audit first", runID)
	}
	return job.result, job.root, nil
}

// emit sends an audit event to the window.
func (a *App) emit(event AuditEvent) {
	a.sendEvent(AuditEventName, event)
}

// emitVerify sends a verification event to the window.
func (a *App) emitVerify(event VerifyEvent) {
	a.sendEvent(VerifyEventName, event)
}

// sendEvent hands an event to the window.
//
// The injected emitter wins, so a test observes the events without a window.
// Without it the event goes to the runtime of Wails, and only when the
// runtime already gave its context: a run started before OnStartup has no
// window to deliver to.
func (a *App) sendEvent(name string, data any) {
	if a.emitEvent != nil {
		a.emitEvent(name, data)
		return
	}
	ctx := a.ctx
	if ctx == nil {
		return
	}
	runtime.EventsEmit(ctx, name, data)
}

// view builds the renderable result of a run.
func (a *App) view(runID, root string, origin domain.ChangeOrigin, base string, result *application.AuditResult) *AuditView {
	return &AuditView{
		RunID:              runID,
		SchemaVersion:      result.Run.SchemaVersion,
		ToolVersion:        result.Run.ToolVersion,
		Root:               root,
		Origin:             string(origin),
		Base:               base,
		ChangedFiles:       result.ChangedFiles,
		FailOn:             result.FailOn,
		Failed:             result.Failed,
		FailingFindings:    result.FailingFindings,
		KnownFindings:      result.KnownFindings,
		SuppressedFindings: len(result.Suppressed),
		ExcludedFiles:      result.ExcludedFiles,
		Document:           result.Run,
		Changes:            result.Changes,
	}
}

// resolveRequestOrigin validates the origin of a request.
//
// The Origin field is the preferred one. The Worktree, Staged and Base
// fields are the legacy shape of the first prototype and are accepted only
// when Origin is empty, so a caller never selects two origins at once.
func resolveRequestOrigin(request AuditRequest) (domain.ChangeOrigin, string, error) {
	origin := strings.ToLower(strings.TrimSpace(request.Origin))
	base := strings.TrimSpace(request.Base)

	if origin != "" {
		if request.Worktree || request.Staged {
			return "", "", fmt.Errorf("the audit takes one origin: choose %q or the legacy worktree/staged flag, not both", origin)
		}
		switch origin {
		case string(domain.OriginWorktree):
			if base != "" {
				return "", "", fmt.Errorf("the worktree origin does not take a base revision: remove %q", base)
			}
			return domain.OriginWorktree, "", nil
		case string(domain.OriginStaged):
			if base != "" {
				return "", "", fmt.Errorf("the staged origin does not take a base revision: remove %q", base)
			}
			return domain.OriginStaged, "", nil
		case string(domain.OriginBase):
			if base == "" {
				return "", "", fmt.Errorf("the base origin needs a revision: write one such as 'main'")
			}
			return domain.OriginBase, base, nil
		default:
			return "", "", fmt.Errorf("unknown origin %q: choose worktree, staged or base", origin)
		}
	}

	selected := 0
	if request.Worktree {
		selected++
	}
	if request.Staged {
		selected++
	}
	if base != "" {
		selected++
	}
	switch {
	case selected == 0:
		return "", "", fmt.Errorf("the audit needs an origin: choose the worktree, the staging area or a base revision such as 'main'")
	case selected > 1:
		return "", "", fmt.Errorf("the audit takes one origin: choose the worktree, the staging area or the base revision %q, not several", base)
	}
	if request.Worktree {
		return domain.OriginWorktree, "", nil
	}
	if request.Staged {
		return domain.OriginStaged, "", nil
	}
	return domain.OriginBase, base, nil
}

// exportExtension returns the extension of a format.
func exportExtension(format string) string {
	if strings.EqualFold(strings.TrimSpace(format), "sarif") {
		return ".sarif"
	}
	return ".json"
}

// fileFilter builds the filter of the file dialog from a list of extensions.
//
// An extension is written with its dot, for example ".json"; the dialog shows
// them as "*.json" and accepts any of them. The second result is false when
// the list holds no usable extension, and the dialog then offers every file.
func fileFilter(extensions []string) (runtime.FileFilter, bool) {
	patterns := make([]string, 0, len(extensions))
	names := make([]string, 0, len(extensions))
	for _, extension := range extensions {
		extension = strings.TrimSpace(extension)
		if extension == "" {
			continue
		}
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		patterns = append(patterns, "*"+extension)
		names = append(names, strings.ToUpper(strings.TrimPrefix(extension, ".")))
	}
	if len(patterns) == 0 {
		return runtime.FileFilter{}, false
	}
	return runtime.FileFilter{
		DisplayName: strings.Join(names, ", "),
		Pattern:     strings.Join(patterns, ";"),
	}, true
}

// ensureOutsideRoot rejects a report path inside the audited repository.
func ensureOutsideRoot(path, root string) error {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("cannot resolve the report path %q: %w", path, err)
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("cannot resolve the audited repository %q: %w", root, err)
	}
	relative, err := filepath.Rel(absoluteRoot, absolutePath)
	if err != nil {
		return nil
	}
	if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("the report cannot be written inside the audited repository %s: choose a path outside it", root)
	}
	return nil
}

// newRunID returns an identifier for a run.
func newRunID() string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("run-%d", time.Now().UnixNano())
	}
	return "run-" + hex.EncodeToString(buffer)
}
