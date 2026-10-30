//go:build ui

package cli

import (
	"context"
	"fmt"

	"github.com/barceloch/codixai/internal/analysis"
	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/claims"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/integrations/git"
	desktopui "github.com/barceloch/codixai/ui"
)

// newUICommand builds the ui command of the variant with the interface.
//
// The command is the composition root of the window: it resolves the
// repository, it wires the read-only providers of the audit with the same
// use case as the audit command, and it hands them to the Wails adapter.
// The window itself decides nothing: it renders the versioned document of
// the use case.
func newUICommand(env *Env) *Command {
	command := &Command{
		Use:   "ui",
		Short: "Open the graphical interface and audit a repository.",
		Long: "Open the graphical interface of Codixai.\n" +
			"\n" +
			"The window audits the changes of a repository through the same\n" +
			"read-only use case as 'codixai audit': it selects the origin (the\n" +
			"worktree, the staging area or a base revision), it runs the rules and\n" +
			"it renders the versioned document, with the findings, their evidence\n" +
			"and their limitations.\n" +
			"\n" +
			"The command reads the repository: it never modifies it, it never\n" +
			"executes its code and it never accesses the network.",
		Example: "codixai ui\n" +
			"codixai ui --root ../another-repository",
	}
	command.RunE = func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return Usagef("the ui command does not accept arguments, got %q", args[0])
		}
		return runUI(ctx, env)
	}
	return command
}

// runUI opens the window with the initial repository of the invocation.
func runUI(ctx context.Context, env *Env) error {
	root, err := resolveRoot(env.Root)
	if err != nil {
		return err
	}
	if err := checkGitRepository(root); err != nil {
		return err
	}
	return desktopui.Run(ctx, desktopui.Options{
		Title:  "Codixai",
		Root:   root,
		Audit:  auditRunner(),
		Verify: verifyRunner(),
	})
}

// UIAuditFunc returns the read-only audit use case that the window calls.
//
// It exists so that the parity test runs the exact composition of the
// window and not a copy of it.
func UIAuditFunc() desktopui.AuditFunc {
	return auditRunner()
}

// auditRunner builds the read-only audit use case for the window.
//
// The window sends the repository it holds and the origin the person chose,
// so every run resolves the root and the origin again: the arguments of the
// frontend are validated exactly like the arguments of the command line.
func auditRunner() desktopui.AuditFunc {
	return func(ctx context.Context, request desktopui.AuditRunRequest) (*application.AuditResult, error) {
		root, err := resolveRoot(request.Root)
		if err != nil {
			return nil, err
		}
		if err := checkGitRepository(root); err != nil {
			return nil, err
		}
		origin, baseRef, err := resolveOrigin(
			request.Base,
			request.Origin == domain.OriginStaged,
			request.Origin == domain.OriginWorktree,
			"audit",
		)
		if err != nil {
			return nil, err
		}
		source, err := git.New(root)
		if err != nil {
			return nil, sanitizeError(err, root)
		}
		analyzer := analysis.New(analysis.WithToolVersion(Version))
		result, err := application.RunAudit(ctx, application.AuditRequest{
			Root:     root,
			Origin:   origin,
			Base:     baseRef,
			Source:   progressSource{next: source, progress: request.Progress},
			Analyzer: progressAnalyzer{next: analyzer, progress: request.Progress},
		})
		if err != nil {
			return nil, sanitizeError(err, root)
		}
		if result == nil {
			return nil, fmt.Errorf("the audit did not produce a result for %s", displayPath(root))
		}
		return result, nil
	}
}

// progressSource reports the collection of the changes.
//
// It wraps the Git source of the composition root, so the phases that the
// window shows are the phases that the use case really runs.
type progressSource struct {
	next     application.ChangeSource
	progress func(desktopui.AuditProgress)
}

// Changes collects the changes and reports the phase before and after.
func (s progressSource) Changes(ctx context.Context, origin domain.ChangeOrigin, base string) (*domain.ChangeSet, error) {
	s.report(desktopui.AuditProgress{Phase: desktopui.PhaseCollecting})
	changes, err := s.next.Changes(ctx, origin, base)
	if err != nil {
		return nil, err
	}
	s.report(desktopui.AuditProgress{Phase: desktopui.PhaseCollected, ChangedFiles: len(changes.Files)})
	return changes, nil
}

// Head returns the current HEAD revision of the repository.
func (s progressSource) Head(ctx context.Context) (string, error) {
	return s.next.Head(ctx)
}

// report sends a phase when the caller asked for progress.
func (s progressSource) report(progress desktopui.AuditProgress) {
	if s.progress != nil {
		s.progress(progress)
	}
}

// progressAnalyzer reports the application of the rule catalog.
type progressAnalyzer struct {
	next     application.DiffAnalyzer
	progress func(desktopui.AuditProgress)
}

// Analyze applies the rules and reports the phase before and after.
func (a progressAnalyzer) Analyze(ctx context.Context, changes *domain.ChangeSet) (*domain.AnalysisRun, error) {
	a.report(desktopui.AuditProgress{Phase: desktopui.PhaseAnalyzing, ChangedFiles: len(changes.Files)})
	run, err := a.next.Analyze(ctx, changes)
	if err != nil {
		return nil, err
	}
	a.report(desktopui.AuditProgress{
		Phase:        desktopui.PhaseAnalyzed,
		ChangedFiles: len(changes.Files),
		Findings:     len(run.Findings),
	})
	return run, nil
}

// Rules returns the rules exposed by the engine.
func (a progressAnalyzer) Rules() []domain.Rule {
	return a.next.Rules()
}

// report sends a phase when the caller asked for progress.
func (a progressAnalyzer) report(progress desktopui.AuditProgress) {
	if a.progress != nil {
		a.progress(progress)
	}
}

// verifyRunner builds the read-only verification use case for the window.
//
// The window sends the repository it holds, the base revision and the claim
// file the person chose, so every run resolves the root again and wires the
// same providers as 'codixai verify': the arguments of the frontend are
// validated exactly like the arguments of the command line.
func verifyRunner() desktopui.VerifyFunc {
	return func(ctx context.Context, request desktopui.VerifyRequest) (*application.VerifyResult, error) {
		root, err := resolveRoot(request.Root)
		if err != nil {
			return nil, err
		}
		if err := checkGitRepository(root); err != nil {
			return nil, err
		}
		source, err := git.New(root)
		if err != nil {
			return nil, sanitizeError(err, root)
		}
		result, err := application.RunVerify(ctx, application.VerifyRequest{
			Root:        root,
			Base:        request.Base,
			ClaimsPath:  request.ClaimsPath,
			Source:      source,
			Loader:      claims.Loader{},
			Evaluator:   claims.NewEvaluator(claims.WithRoot(root), claims.WithBase(request.Base)),
			ToolVersion: Version,
		})
		if err != nil {
			return nil, sanitizeError(err, root)
		}
		if result == nil {
			return nil, fmt.Errorf("the verification did not produce a result for %s", displayPath(root))
		}
		return result, nil
	}
}
