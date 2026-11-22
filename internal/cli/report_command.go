package cli

import (
	"context"
	"flag"
	"strings"

	"github.com/barceloch/codixai/internal/analysis"
	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/integrations/git"
	"github.com/barceloch/codixai/internal/policies"
	"github.com/barceloch/codixai/internal/report"
)

// reportOptions holds the flags of the report command.
type reportOptions struct {
	base      string
	staged    bool
	worktree  bool
	directory string
}

// newReportCommand builds the report command.
//
// The command is the machine facing entry point of the tool. It runs the same
// audit use case as the audit command and writes either the SARIF document of
// the run or the document of the audit, so a platform receives the findings
// of exactly the same run that the audit reports.
func newReportCommand(env *Env) *Command {
	options := &reportOptions{}
	command := &Command{
		Use:   "report",
		Short: "Write the machine readable document of an audit.",
		Long: "Write the machine readable document of an audit.\n" +
			"\n" +
			"The command is the machine facing entry point of the tool: it is meant for\n" +
			"the platforms and the continuous integration jobs that import the findings\n" +
			"of a repository instead of reading a report.\n" +
			"\n" +
			"--format sarif writes the SARIF 2.1.0 document of the run, so a compatible\n" +
			"platform can import the rules, the locations, the severity and the stable\n" +
			"fingerprint of every new finding. The SARIF document is the default format\n" +
			"of the command. The export is fixed by the decision 0009, in\n" +
			"docs/decisions/0009-sarif-export.md.\n" +
			"\n" +
			"--format json writes the document of the audit, the same document, byte for\n" +
			"byte, that 'codixai audit --format json' writes for the same changes.\n" +
			"\n" +
			"Exactly one origin has to be selected:\n" +
			"  --base compares the repository with a revision\n" +
			"  --staged reads the staging area\n" +
			"  --worktree reads the changes that are not committed yet\n" +
			"  --directory analyzes a directory without requiring a Git repository\n" +
			"\n" +
			"The command applies the same policy and the same baseline as the audit. A\n" +
			"finding that the baseline knows is not a result of the document and does\n" +
			"not reach the failure threshold; its count is a diagnostic of the standard\n" +
			"error. The standard output only ever holds the requested document.\n" +
			"\n" +
			"The exit code is the exit code of the audit: 3 when a new finding reaches\n" +
			"the configured failure threshold, 1 for an execution error and 2 for\n" +
			"invalid arguments.\n" +
			"\n" +
			"The SARIF document is not validated against the official JSON schema of the\n" +
			"format: the tool has no network access and it never fetches it. The\n" +
			"document is validated against the structure that the decision 0009 fixes.\n" +
			"\n" +
			"The command reads the repository or directory: it never modifies it.",
		Example: "codixai report --format sarif --base main\n" +
			"codixai report --format sarif --worktree\n" +
			"codixai report --format json --base main\n" +
			"codixai report --format sarif --directory ./mi-proyecto\n" +
			"codixai report --format json --directory ./mi-proyecto",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&options.base, "base", "", "Base revision to compare the repository with HEAD.")
			fs.BoolVar(&options.staged, "staged", false, "Report the findings of the staging area.")
			fs.BoolVar(&options.worktree, "worktree", false, "Report the findings of the changes that are not committed yet.")
			fs.StringVar(&options.directory, "directory", "", "Analyze the current content of a directory without requiring a Git repository.")
		},
	}
	command.RunE = func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return Usagef("the report command does not accept arguments, got %q", args[0])
		}
		return runReport(ctx, env, options)
	}
	return command
}

// runReport runs the report command.
//
// The command audits the changes with the same use case as the audit command
// and renders the outcome in the requested format. The failure threshold and
// the diagnostics behave exactly like the ones of the audit, so a pipeline
// can replace one command with the other.
func runReport(ctx context.Context, env *Env, options *reportOptions) error {
	format, err := resolveReportFormat(env)
	if err != nil {
		return err
	}

	// Validate mutually exclusive origin flags.
	originCount := 0
	if strings.TrimSpace(options.base) != "" {
		originCount++
	}
	if options.staged {
		originCount++
	}
	if options.worktree {
		originCount++
	}
	if strings.TrimSpace(options.directory) != "" {
		originCount++
	}
	if originCount == 0 {
		return Usagef("no origin selected: use --base <ref>, --staged, --worktree or --directory <path>")
	}
	if originCount > 1 {
		return Usagef("several origins selected: use only one of --base <ref>, --staged, --worktree or --directory <path>")
	}

	// Directory mode: analyze a directory without Git.
	if strings.TrimSpace(options.directory) != "" {
		return runDirectoryReport(ctx, env, format, options.directory)
	}

	// Git modes: existing behavior.
	origin, base, err := resolveOrigin(options.base, options.staged, options.worktree, "report")
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
	})
	if err != nil {
		return sanitizeError(err, root)
	}

	return writeReportOutput(env, format, result.Run, result, root)
}

// resolveReportFormat returns the format of the document that the command
// writes.
//
// The SARIF document is the default of the command, because the command
// exists for the machines that import it. The tool's default format is not
// used as the default of the command, so an explicit text format is reported
// instead of being silently replaced: the report command has no text
// document.
func resolveReportFormat(env *Env) (string, error) {
	format := strings.ToLower(strings.TrimSpace(env.Format))
	if !env.FormatSet {
		return report.Format, nil
	}
	switch format {
	case report.Format, policies.FormatJSON:
		return format, nil
	default:
		return "", Usagef("the report command writes %s or %s, not %q: use 'codixai audit' for the text report",
			report.Format, policies.FormatJSON, env.Format)
	}
}

// severityRank orders the severities.
//
// The ranks are categorical: they express an order, not a measurement.
func severityRank(severity domain.Severity) int {
	switch severity {
	case domain.SeverityInfo:
		return 1
	case domain.SeverityWarning:
		return 2
	case domain.SeverityError:
		return 3
	default:
		return 0
	}
}

// thresholdRank returns the rank of a failure threshold.
//
// The threshold none never reaches a finding.
func thresholdRank(failOn string) int {
	switch strings.ToLower(strings.TrimSpace(failOn)) {
	case policies.FailOnError:
		return severityRank(domain.SeverityError)
	case policies.FailOnWarning:
		return severityRank(domain.SeverityWarning)
	case policies.FailOnNone:
		return 0
	default:
		return severityRank(domain.SeverityError)
	}
}

// runDirectoryReport runs a directory analysis and writes the report.
func runDirectoryReport(ctx context.Context, env *Env, format, directory string) error {
	root, err := resolveRoot(directory)
	if err != nil {
		return err
	}

	analyzer := analysis.NewDirectoryAnalyzer(analysis.WithToolVersion(Version))

	result, err := application.RunDirectoryAudit(ctx, application.DirectoryAuditRequest{
		Root:     root,
		Analyzer: analyzer,
	})
	if err != nil {
		return sanitizeError(err, root)
	}

	// Load configuration to determine failure threshold.
	config, err := policies.Load(root)
	if err != nil {
		return err
	}
	failOn := config.Audit.FailOn
	if failOn == "" {
		failOn = policies.FailOnError
	}

	// For directory analysis, write diagnostics and check failure threshold.
	writeDirectoryAuditDiagnostics(env, result, root)

	failed := false
	count := 0
	if result.Run.Summary.BySeverity != nil {
		threshold := thresholdRank(failOn)
		for severity, c := range result.Run.Summary.BySeverity {
			if severityRank(domain.Severity(severity)) >= threshold && threshold > 0 {
				failed = true
				count += c
			}
		}
	}

	// Write the document first, then return FindingsError if applicable.
	if err := writeReportOutput(env, format, result.Run, nil, root); err != nil {
		return err
	}

	if failed {
		return &FindingsError{Count: count, Threshold: failOn}
	}

	return nil
}

// writeReportOutput writes the report in the requested format.
func writeReportOutput(env *Env, format string, run *domain.AnalysisRun, auditResult *application.AuditResult, root string) error {
	switch format {
	case report.Format:
		var document report.SarifDocument
		if auditResult != nil {
			// Git mode: use SARIF with baseline fingerprinting.
			document = report.SARIF(auditResult, report.SARIFOptions{ToolVersion: Version})
		} else {
			// Directory mode: use SARIFDirectory without baseline fingerprinting.
			document = report.SARIFDirectory(run, report.SARIFOptions{ToolVersion: Version})
		}
		if err := writeJSONDocument(env.Stdout, document, "SARIF document"); err != nil {
			return err
		}
	default:
		if err := writeJSONDocument(env.Stdout, run, "analysis document"); err != nil {
			return err
		}
	}

	// Write diagnostics for Git mode.
	if auditResult != nil {
		writeAuditDiagnostics(env, auditResult)
		if auditResult.Failed {
			return &FindingsError{Count: auditResult.FailingFindings, Threshold: auditResult.FailOn}
		}
	}
	return nil
}
