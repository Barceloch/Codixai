//go:build ui

package cli

import (
	"context"
	"fmt"

	"github.com/barceloch/codixai/internal/analysis"
	"github.com/barceloch/codixai/internal/application"
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
			"worktree or a base revision), it runs the rules and it renders the\n" +
			"versioned document, with the findings, their evidence and their\n" +
			"limitations.\n" +
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
		Title: "Codixai",
		Root:  root,
		Audit: auditRunner(),
	})
}

// auditRunner builds the read-only audit use case for the window.
//
// The window sends the repository it holds and the origin the person chose,
// so every run resolves the root and the origin again: the arguments of the
// frontend are validated exactly like the arguments of the command line.
func auditRunner() desktopui.AuditFunc {
	return func(ctx context.Context, target string, worktree bool, base string) (*application.AuditResult, error) {
		root, err := resolveRoot(target)
		if err != nil {
			return nil, err
		}
		if err := checkGitRepository(root); err != nil {
			return nil, err
		}
		origin, baseRef, err := resolveOrigin(base, false, worktree, "audit")
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
			Source:   source,
			Analyzer: analyzer,
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
