package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Barceloch/Codixai/internal/analysis"
	"github.com/Barceloch/Codixai/internal/application"
	"github.com/Barceloch/Codixai/internal/domain"
	"github.com/Barceloch/Codixai/internal/integrations/git"
	"github.com/Barceloch/Codixai/internal/languages/registry"
	"github.com/Barceloch/Codixai/internal/policies"
)

// NewRootCommand builds the command tree of the CLI.
//
// The whole tree is defined here: the flags of every command and its
// handler live in the same constructor, so that the interface of the tool
// can be read in one place.
func NewRootCommand(env *Env) *Command {
	root := &Command{
		Use:   "codixai",
		Short: "Verify the changes of a repository against the rule catalog.",
		Long: "Codixai is a local first verifier of the work done in a repository.\n" +
			"It reads the changes of a Git repository, applies a deterministic rule\n" +
			"catalog to them and reports what it observed, what it inferred and what\n" +
			"it could not check.\n" +
			"\n" +
			"Codixai never executes the code of the analyzed repository, never invokes\n" +
			"a package manager and never accesses the network.",
		Example: "codixai audit --diff --worktree\n" +
			"codixai audit --diff --base main\n" +
			"codixai init\n" +
			"codixai version",
		PersistentFlags: []string{"root", "format", "no-color", "plain", "version"},
	}
	root.Flags = func(fs *flag.FlagSet) {
		fs.StringVar(&env.Root, "root", "", "Root of the analyzed repository. Defaults to the current directory.")
		fs.StringVar(&env.Format, "format", policies.FormatText, "Output format of the commands that produce a document: text or json.")
		fs.BoolVar(&env.NoColor, "no-color", false, "Disable the color output.")
		fs.BoolVar(&env.Plain, "plain", false, "Print the output without decoration.")
		fs.BoolVar(&env.ShowVersion, "version", false, "Print the version of the tool and exit.")
	}
	root.RunE = func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return Usagef("unknown command %q", args[0])
		}
		if env.ShowVersion {
			return printVersion(ctx)
		}
		root.printHelp(ctx)
		return nil
	}

	root.AddCommand(newAuditCommand(env))
	root.AddCommand(newInitCommand(env))
	root.AddCommand(newVersionCommand(env))
	root.AddCommand(newHelpCommand(root))
	return root
}

// newHelpCommand builds the help command.
func newHelpCommand(root *Command) *Command {
	return &Command{
		Use:   "help [command]",
		Short: "Print the help of a command.",
		Long:  "Print the help of the command, or the help of the tool when no command is given.",
		RunE: func(ctx context.Context, args []string) error {
			env := envFrom(ctx)
			if len(args) == 0 {
				root.Help(env.Stdout, env.Renderer())
				return nil
			}
			command := root.lookup(args[0])
			if command == nil {
				return Usagef("unknown command %q", args[0])
			}
			command.Help(env.Stdout, env.Renderer())
			return nil
		},
	}
}

// newVersionCommand builds the version command.
func newVersionCommand(env *Env) *Command {
	return &Command{
		Use:   "version",
		Short: "Print the version of the tool.",
		Long:  "Print the version of the tool and the version of the schema of its output.",
		RunE: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return Usagef("the version command does not accept arguments, got %q", args[0])
			}
			return printVersion(ctx)
		},
	}
}

// printVersion writes the version of the tool.
func printVersion(ctx context.Context) error {
	env := envFrom(ctx)
	if env.JSON() {
		document := map[string]string{
			"schema_version": domain.SchemaVersion,
			"tool_version":   Version,
		}
		encoder := json.NewEncoder(env.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(document); err != nil {
			return fmt.Errorf("cannot write the version document: %w", err)
		}
		return nil
	}
	renderer := env.Renderer()
	fmt.Fprintf(env.Stdout, "%s %s\n", renderer.Title("codixai"), renderer.Bold(Version))
	fmt.Fprintf(env.Stdout, "%s %s\n", renderer.Dim("schema version:"), domain.SchemaVersion)
	return nil
}

// newInitCommand builds the init command.
func newInitCommand(env *Env) *Command {
	var force bool
	command := &Command{
		Use:   "init",
		Short: "Create the configuration file of the repository.",
		Long: "Create a minimal configuration file in the root of the repository.\n" +
			"\n" +
			"The command never overwrites an existing configuration file: it reports\n" +
			"that the file already exists and leaves it untouched. The --force flag\n" +
			"replaces it, and it only acts when it is given explicitly.",
		Example: "codixai init\n" +
			"codixai init --force",
		Flags: func(fs *flag.FlagSet) {
			fs.BoolVar(&force, "force", false, "Replace an existing configuration file.")
		},
	}
	command.RunE = func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return Usagef("the init command does not accept arguments, got %q", args[0])
		}
		return runInit(ctx, env, force)
	}
	return command
}

// runInit creates the configuration file of the repository.
func runInit(ctx context.Context, env *Env, force bool) error {
	root, err := resolveRoot(env.Root)
	if err != nil {
		return err
	}
	path := filepath.Join(root, policies.FileName)
	document := defaultConfigDocument(filepath.Base(root))

	// The generated document is validated before it is written, so that a
	// broken generator never leaves an unusable configuration behind.
	if _, err := policies.Parse([]byte(document), policies.FileName); err != nil {
		return fmt.Errorf("the generated configuration is invalid, the file was not written: %w", err)
	}

	exists := true
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory, the configuration file was not written", policies.FileName)
		}
	} else if os.IsNotExist(err) {
		exists = false
	} else {
		return fmt.Errorf("cannot read %s: %w", policies.FileName, err)
	}

	if exists && !force {
		return reportInit(env, "already exists, it was not modified. Use --force to replace it.", false)
	}
	if err := os.WriteFile(path, []byte(document), 0o644); err != nil {
		return fmt.Errorf("cannot write %s: %w", policies.FileName, err)
	}
	return reportInit(env, "created.", true)
}

// reportInit reports the result of the init command.
//
// The JSON format writes a document to the standard output and keeps the
// message for the standard error, so that the standard output only ever
// holds the document that was requested.
func reportInit(env *Env, status string, created bool) error {
	name := policies.FileName
	message := fmt.Sprintf("%s %s", name, status)
	if env.JSON() {
		document := map[string]any{
			"schema_version": domain.SchemaVersion,
			"tool_version":   Version,
			"path":           name,
			"created":        created,
			"message":        message,
		}
		encoder := json.NewEncoder(env.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(document); err != nil {
			return fmt.Errorf("cannot write the init document: %w", err)
		}
		fmt.Fprintf(env.Stderr, "codixai: %s\n", message)
		return nil
	}
	renderer := env.Renderer()
	fmt.Fprintf(env.Stdout, "%s %s\n", renderer.Bold(name), renderer.Dim(status))
	return nil
}

// defaultConfigDocument returns the minimal configuration of a repository.
//
// The document only holds the keys that the configuration loader of the
// policies package understands.
func defaultConfigDocument(projectName string) string {
	var builder strings.Builder
	builder.WriteString("# Configuration of Codixai for the analyzed repository.\n")
	builder.WriteString("# Every key is optional: the values below are the documented defaults.\n")
	builder.WriteString("version: 1\n")
	builder.WriteString("\n")
	builder.WriteString("project:\n")
	builder.WriteString(fmt.Sprintf("  name: %s\n", sanitizeName(projectName)))
	builder.WriteString("\n")
	builder.WriteString("analysis:\n")
	builder.WriteString("  languages:\n")
	for _, language := range policies.SupportedLanguages {
		builder.WriteString(fmt.Sprintf("    - %s\n", language))
	}
	builder.WriteString("  exclude:\n")
	for _, pattern := range policies.DefaultExclude {
		builder.WriteString(fmt.Sprintf("    - %s\n", pattern))
	}
	builder.WriteString("  respect_gitignore: true\n")
	builder.WriteString("\n")
	builder.WriteString("audit:\n")
	builder.WriteString(fmt.Sprintf("  fail_on: %s\n", policies.FailOnError))
	builder.WriteString(fmt.Sprintf("  baseline: %s\n", policies.DefaultBaseline))
	builder.WriteString("  rules:\n")
	builder.WriteString("  suppressions: []\n")
	builder.WriteString("\n")
	builder.WriteString("output:\n")
	builder.WriteString(fmt.Sprintf("  format: %s\n", policies.FormatText))
	return builder.String()
}

// sanitizeName turns the name of a directory into a plain project name.
func sanitizeName(name string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-', r == '_':
			return r
		default:
			return '-'
		}
	}, name)
	cleaned = strings.Trim(cleaned, "-")
	if cleaned == "" {
		return "codixai-project"
	}
	return cleaned
}

// auditOptions holds the flags of the audit command.
type auditOptions struct {
	diff     bool
	base     string
	staged   bool
	worktree bool
	rules    bool
}

// newAuditCommand builds the audit command.
func newAuditCommand(env *Env) *Command {
	options := &auditOptions{}
	command := &Command{
		Use:   "audit",
		Short: "Audit the changes of a repository against the rule catalog.",
		Long: "Audit the changes of a repository against the rule catalog.\n" +
			"\n" +
			"The command reads the changes of the repository, applies the rules that\n" +
			"support the language of every changed file and reports the findings with\n" +
			"their evidence, their support level and their limitations.\n" +
			"\n" +
			"Exactly one origin has to be selected: --base compares the repository with\n" +
			"a revision, --staged audits the staging area and --worktree audits the\n" +
			"changes that are not committed yet.\n" +
			"\n" +
			"The command reads the repository: it never modifies it.",
		Example: "codixai audit --diff --worktree\n" +
			"codixai audit --diff --staged\n" +
			"codixai audit --diff --base main\n" +
			"codixai audit --rules",
		Flags: func(fs *flag.FlagSet) {
			fs.BoolVar(&options.diff, "diff", false, "Audit the changes of the repository. Required in this milestone.")
			fs.StringVar(&options.base, "base", "", "Base revision to compare the repository with HEAD.")
			fs.BoolVar(&options.staged, "staged", false, "Audit the changes of the staging area.")
			fs.BoolVar(&options.worktree, "worktree", false, "Audit the changes that are not committed yet.")
			fs.BoolVar(&options.rules, "rules", false, "Print the rule catalog and exit.")
		},
	}
	command.RunE = func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return Usagef("the audit command does not accept arguments, got %q", args[0])
		}
		return runAudit(ctx, env, options)
	}
	return command
}

// runAudit runs the audit command.
func runAudit(ctx context.Context, env *Env, options *auditOptions) error {
	if options.rules {
		return printRuleCatalog(ctx, env)
	}
	if !options.diff {
		return Usagef("the flag --diff is required: this milestone audits changes only, " +
			"use --diff with --base, --staged or --worktree")
	}
	origin, base, err := resolveOrigin(options)
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

	if env.JSON() {
		encoder := json.NewEncoder(env.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result.Run); err != nil {
			return fmt.Errorf("cannot write the analysis document: %w", err)
		}
	} else {
		writeAuditReport(env.Stdout, env.Renderer(), result, root)
	}

	// The diagnostics never reach the standard output: the standard output
	// only holds the document that was requested.
	if len(result.ExcludedFiles) > 0 {
		fmt.Fprintf(env.Stderr, "codixai: %d file(s) excluded by the configuration: %s\n",
			len(result.ExcludedFiles), strings.Join(result.ExcludedFiles, ", "))
	}
	if len(result.Suppressed) > 0 {
		fmt.Fprintf(env.Stderr, "codixai: %d finding(s) suppressed by the configuration\n", len(result.Suppressed))
	}
	if result.Failed {
		return &FindingsError{Count: result.FailingFindings, Threshold: result.FailOn}
	}
	return nil
}

// resolveOrigin returns the origin of the changes and the base revision.
func resolveOrigin(options *auditOptions) (domain.ChangeOrigin, string, error) {
	selected := 0
	if strings.TrimSpace(options.base) != "" {
		selected++
	}
	if options.staged {
		selected++
	}
	if options.worktree {
		selected++
	}
	switch {
	case selected == 0:
		return "", "", Usagef("no origin selected: use --base <ref>, --staged or --worktree to choose the changes to audit")
	case selected > 1:
		return "", "", Usagef("several origins selected: use only one of --base <ref>, --staged or --worktree")
	}
	if strings.TrimSpace(options.base) != "" {
		return domain.OriginBase, options.base, nil
	}
	if options.staged {
		return domain.OriginStaged, "", nil
	}
	return domain.OriginWorktree, "", nil
}

// printRuleCatalog writes the rule catalog with its support matrix.
func printRuleCatalog(ctx context.Context, env *Env) error {
	rules := analysis.New().Rules()
	if env.JSON() {
		document := map[string]any{
			"schema_version": domain.SchemaVersion,
			"tool_version":   Version,
			"rules":          rules,
		}
		encoder := json.NewEncoder(env.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(document); err != nil {
			return fmt.Errorf("cannot write the rule catalog: %w", err)
		}
		return nil
	}
	writeRuleCatalog(env.Stdout, env.Renderer(), rules)
	return nil
}

// ruleLanguages returns the languages of the support matrix, in a stable
// order.
func ruleLanguages() []string {
	return registry.Languages()
}
