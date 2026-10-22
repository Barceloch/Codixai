package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/barceloch/codixai/internal/analysis"
	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/baseline"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/integrations/git"
	"github.com/barceloch/codixai/internal/policies"
)

// baselineOptions holds the flags of the baseline create command.
type baselineOptions struct {
	base     string
	staged   bool
	worktree bool
	out      string
	force    bool
}

// newBaselineCommand builds the baseline command and its subcommands.
func newBaselineCommand(env *Env) *Command {
	command := &Command{
		Use:   "baseline",
		Short: "Manage the baseline of the findings of the repository.",
		Long: "Manage the baseline of the findings of the repository.\n" +
			"\n" +
			"A baseline is a versioned file that lives in the analyzed repository and\n" +
			"that records the findings the repository already has. It lets a pipeline\n" +
			"fail on the new findings without turning the existing debt into constant\n" +
			"noise.\n" +
			"\n" +
			"The baseline is meant to be committed with the repository: it is shared\n" +
			"through the review of a change, like the code itself. When the baseline\n" +
			"changes, the change has to be reviewed, because an entry that disappears\n" +
			"is a finding that is not reported anymore.\n" +
			"\n" +
			"The identity of a finding is its fingerprint: the rule, the path and a\n" +
			"normalized context, never the line number. The baseline therefore\n" +
			"survives the movement of the code.",
		Example: "codixai baseline create --base main\n" +
			"codixai baseline create --worktree",
	}
	command.AddCommand(newBaselineCreateCommand(env))
	return command
}

// newBaselineCreateCommand builds the baseline create command.
func newBaselineCreateCommand(env *Env) *Command {
	options := &baselineOptions{}
	command := &Command{
		Use:   "create",
		Short: "Create the baseline of the findings of the repository.",
		Long: "Create the baseline of the findings of the repository.\n" +
			"\n" +
			"Exactly one origin has to be selected: --base compares the repository with\n" +
			"a revision, --staged records the staging area and --worktree records the\n" +
			"changes that are not committed yet.\n" +
			"\n" +
			"The baseline is written to the path of the audit.baseline key of the\n" +
			"configuration, unless --out names another one. A relative path is\n" +
			"resolved against the root of the repository.\n" +
			"\n" +
			"The command never overwrites an existing baseline without --force, and it\n" +
			"says what it would overwrite.\n" +
			"\n" +
			"The baseline that is being created is never applied to the run that\n" +
			"creates it: the command records every finding of the compared range.\n" +
			"\n" +
			"The command writes one file of the repository and it never modifies\n" +
			"anything else.",
		Example: "codixai baseline create --base main\n" +
			"codixai baseline create --worktree\n" +
			"codixai baseline create --base main --out .codixai/baseline.json --force",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&options.base, "base", "", "Base revision to compare the repository with HEAD.")
			fs.BoolVar(&options.staged, "staged", false, "Record the findings of the staging area.")
			fs.BoolVar(&options.worktree, "worktree", false, "Record the findings of the changes that are not committed yet.")
			fs.StringVar(&options.out, "out", "", "Path of the baseline file to write. Defaults to the audit.baseline key of the configuration.")
			fs.BoolVar(&options.force, "force", false, "Replace an existing baseline file.")
		},
	}
	command.RunE = func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return Usagef("the baseline create command does not accept arguments, got %q", args[0])
		}
		return runBaselineCreate(ctx, env, options)
	}
	return command
}

// runBaselineCreate creates the baseline of the findings of a repository.
//
// The command collects the changes of the selected origin, audits them
// without applying any baseline and writes the findings of the audit into a
// versioned document of the repository.
func runBaselineCreate(ctx context.Context, env *Env, options *baselineOptions) error {
	origin, base, err := resolveOrigin(options.base, options.staged, options.worktree, "record in the baseline")
	if err != nil {
		return err
	}
	root, err := resolveRoot(env.Root)
	if err != nil {
		return err
	}
	if err := checkGitRepository(root); err != nil {
		return err
	}
	config, err := policies.Load(root)
	if err != nil {
		return err
	}

	target := strings.TrimSpace(options.out)
	if target == "" {
		target = config.Audit.Baseline
	}
	path, err := baseline.Resolve(root, target)
	if err != nil {
		return err
	}
	name := displayPath(path)

	previous, exists, err := readExistingBaseline(path)
	if err != nil && !options.force {
		return err
	}
	if err != nil {
		// A forced replacement does not depend on the previous document: a
		// baseline that cannot be read records nothing that can be
		// identified, so the replacement counts every finding as new.
		previous, exists = baseline.New(), true
	}
	if exists && !options.force {
		return fmt.Errorf("the baseline %s already exists: it records %d finding(s) and it would be replaced. "+
			"Use --force to replace it", name, len(previous.Findings))
	}

	source, err := git.New(root)
	if err != nil {
		return sanitizeError(err, root)
	}
	analyzer := analysis.New(analysis.WithToolVersion(Version))

	result, err := application.RunAudit(ctx, application.AuditRequest{
		Root:     root,
		Origin:   origin,
		Base:     base,
		Source:   source,
		Analyzer: analyzer,
		Config:   config,
		// The baseline that is being created is never applied to the run
		// that creates it.
		SkipBaseline: true,
	})
	if err != nil {
		return sanitizeError(err, root)
	}

	entries := baseline.Entries(result.Run.Findings, result.Changes)
	file := &baseline.File{
		SchemaVersion: baseline.SchemaVersion,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		ToolVersion:   Version,
		Findings:      entries,
	}
	if err := baseline.Save(path, file); err != nil {
		return err
	}

	newFindings := 0
	for _, entry := range entries {
		if !previous.Contains(entry.Fingerprint) {
			newFindings++
		}
	}
	return reportBaseline(env, name, len(entries), newFindings)
}

// readExistingBaseline reads the baseline that is already at a path.
//
// A missing file is not an error: the repository has no baseline yet.
func readExistingBaseline(path string) (*baseline.File, bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return baseline.New(), false, nil
		}
		return nil, false, fmt.Errorf("cannot read the baseline %s: %w", displayPath(path), err)
	}
	file, err := baseline.Load(path)
	if err != nil {
		return nil, true, err
	}
	return file, true, nil
}

// reportBaseline reports the result of the baseline create command.
//
// The JSON format writes a document to the standard output and keeps the
// message for the standard error, so that the standard output only ever
// holds the document that was requested.
func reportBaseline(env *Env, path string, findings, newFindings int) error {
	message := fmt.Sprintf("%s created: %d finding(s) recorded, %d new", path, findings, newFindings)
	if env.JSON() {
		document := map[string]any{
			"schema_version": domain.SchemaVersion,
			"tool_version":   Version,
			"path":           path,
			"findings":       findings,
			"new_findings":   newFindings,
			"created":        true,
			"message":        message,
		}
		encoder := json.NewEncoder(env.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(document); err != nil {
			return fmt.Errorf("cannot write the baseline document: %w", err)
		}
		fmt.Fprintf(env.Stderr, "codixai: %s\n", message)
		return nil
	}
	renderer := env.Renderer()
	fmt.Fprintf(env.Stdout, "%s %s\n", renderer.Bold(path), renderer.Dim(fmt.Sprintf("created: %d finding(s) recorded, %d new", findings, newFindings)))
	return nil
}
