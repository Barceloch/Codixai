//go:build ui

// Package ui implements the Wails v2 adapter of Codixai.
//
// The package is built only with the ui tag: the default build of the tool
// neither compiles it nor reaches Wails, the webview or the embedded
// assets. It holds the window of the spike, the binding of the read-only
// audit use case and the static frontend of ui/assets, which is embedded
// with go:embed.
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
package ui

import (
	"context"
	"embed"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AutoQuitEnv names the environment variable that closes the window after
// the given duration.
//
// It exists for the measurement of the startup time of the ui variant: the
// spike starts the binary, lets the window reach its first frame and quits
// without a person closing it. It is empty by default and it never changes
// the behavior of the window when it is not set.
const AutoQuitEnv = "CODIXAI_UI_AUTOQUIT"

// assets holds the static frontend of the spike.
//
// The frontend is plain HTML and JavaScript on purpose: the spike measures
// the packaging of the binary, not a frontend toolchain. The production
// interface of P7-06 does not reuse these assets.
//
//go:embed all:assets
var assets embed.FS

// AuditRequest is the request that the window sends to the audit use case.
type AuditRequest struct {
	// Root is the path of the repository to audit.
	Root string `json:"root"`
	// Worktree audits the changes that are not committed yet.
	Worktree bool `json:"worktree"`
	// Base is the revision to compare the repository with. It is empty
	// when the origin is the worktree.
	Base string `json:"base"`
}

// AuditView is the result that the window renders.
//
// The document is the same versioned document that the command line
// writes with --format json: the interface never reinterprets a finding,
// it renders what the use case decided.
type AuditView struct {
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
	// Document is the analysis document produced by the use case.
	Document *domain.AnalysisRun `json:"document"`
}

// AuditFunc runs the read-only audit use case.
//
// The command that starts the window composes the providers, so that the
// adapter does not know how the changes are collected or how the rules are
// applied.
type AuditFunc func(ctx context.Context, root string, worktree bool, base string) (*application.AuditResult, error)

// Options configures the window.
type Options struct {
	// Title is the title of the window.
	Title string
	// Root is the repository that the window shows when it opens.
	Root string
	// Audit runs the audit use case.
	Audit AuditFunc
}

// App is the struct bound to the frontend.
//
// Its exported methods are the whole surface of the window: the frontend
// cannot reach anything else.
type App struct {
	ctx   context.Context
	audit AuditFunc
	root  string
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
	app := &App{audit: opts.Audit, root: opts.Root}

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
	root := strings.TrimSpace(request.Root)
	if root == "" {
		return nil, fmt.Errorf("the audit needs the path of a repository: write it in the repository field")
	}
	base := strings.TrimSpace(request.Base)
	if request.Worktree && base != "" {
		return nil, fmt.Errorf("the audit takes one origin: choose the worktree or the base revision %q, not both", base)
	}
	if !request.Worktree && base == "" {
		return nil, fmt.Errorf("the audit needs an origin: choose the worktree, or write a base revision such as 'main'")
	}

	// The runtime context exists once OnStartup has fired. A call that
	// arrives earlier (a reload of the page during the start) runs with a
	// background context instead of a nil one.
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := a.audit(ctx, root, request.Worktree, base)
	if err != nil {
		return nil, err
	}
	if result == nil || result.Run == nil {
		return nil, fmt.Errorf("the audit did not produce a document: run 'codixai audit --diff --worktree' in the terminal to see the error of the analysis")
	}

	origin := string(domain.OriginWorktree)
	if base != "" {
		origin = string(domain.OriginBase)
	}
	return &AuditView{
		SchemaVersion:   result.Run.SchemaVersion,
		ToolVersion:     result.Run.ToolVersion,
		Root:            root,
		Origin:          origin,
		Base:            base,
		ChangedFiles:    result.ChangedFiles,
		FailOn:          result.FailOn,
		Failed:          result.Failed,
		FailingFindings: result.FailingFindings,
		Document:        result.Run,
	}, nil
}
