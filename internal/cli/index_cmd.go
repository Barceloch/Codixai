package cli

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/integrations/git"
)

// indexDiffBase holds the --base flag value for index diff command.
var indexDiffBase string
var indexForce bool
var indexPath string

// newIndexCommand creates the index subcommand.
func newIndexCommand(env *Env) *Command {
	cmd := &Command{
		Use:   "index",
		Short: "Structural index commands for the repository.",
	}
	cmd.AddCommand(&Command{
		Use:   "build",
		Short: "Build the structural index of the repository.",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&env.Root, "root", "", "Repository root (defaults to current directory)")
			fs.BoolVar(&indexForce, "force", false, "Force a full rebuild of the index")
			fs.StringVar(&indexPath, "index-path", "", "Path to the index file (defaults to .codixai/index.db)")
		},
		RunE: func(ctx context.Context, args []string) error {
			return runIndexBuild(ctx, env)
		},
	})
	cmd.AddCommand(&Command{
		Use:   "status",
		Short: "Show index status.",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&env.Root, "root", "", "Repository root (defaults to current directory)")
			fs.StringVar(&indexPath, "index-path", "", "Path to the index file (defaults to .codixai/index.db)")
		},
		RunE: func(ctx context.Context, args []string) error {
			return runIndexStatus(ctx, env)
		},
	})
	cmd.AddCommand(&Command{
		Use:   "overview",
		Short: "Show index overview.",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&env.Root, "root", "", "Repository root (defaults to current directory)")
			fs.StringVar(&indexPath, "index-path", "", "Path to the index file (defaults to .codixai/index.db)")
		},
		RunE: func(ctx context.Context, args []string) error {
			return runIndexOverview(ctx, env)
		},
	})
	cmd.AddCommand(&Command{
		Use:   "find [query]",
		Short: "Find symbols in the index.",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&env.Root, "root", "", "Repository root (defaults to current directory)")
			fs.StringVar(&indexPath, "index-path", "", "Path to the index file (defaults to .codixai/index.db)")
		},
		RunE: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return Usagef("the find command requires a query")
			}
			return runIndexFind(ctx, env, args[0])
		},
	})
	cmd.AddCommand(&Command{
		Use:   "refs [query]",
		Short: "Find references of a symbol.",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&env.Root, "root", "", "Repository root (defaults to current directory)")
			fs.StringVar(&indexPath, "index-path", "", "Path to the index file (defaults to .codixai/index.db)")
		},
		RunE: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return Usagef("the refs command requires a symbol name")
			}
			return runIndexRefs(ctx, env, args[0])
		},
	})
	cmd.AddCommand(&Command{
		Use:   "deps",
		Short: "List declared dependencies from the index.",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&env.Root, "root", "", "Repository root (defaults to current directory)")
			fs.StringVar(&indexPath, "index-path", "", "Path to the index file (defaults to .codixai/index.db)")
		},
		RunE: func(ctx context.Context, args []string) error {
			return runIndexDeps(ctx, env)
		},
	})
	cmd.AddCommand(&Command{
		Use:   "diff --base <ref>",
		Short: "Compare blobs between base and HEAD without checkout.",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&env.Root, "root", "", "Repository root (defaults to current directory)")
			fs.StringVar(&indexDiffBase, "base", "", "Base revision to compare with HEAD (required)")
		},
		RunE: func(ctx context.Context, args []string) error {
			return runIndexDiff(ctx, env)
		},
	})
	return cmd
}

// runIndexBuild builds the structural index of the repository.
func runIndexBuild(ctx context.Context, env *Env) error {
	root, err := resolveRoot(env.Root)
	if err != nil {
		return err
	}

	result, err := application.RunIndex(ctx, application.IndexRequest{
		Root:            root,
		IndexPath:       indexPath,
		ForceRebuild:    indexForce,
		ToolVersion:     Version,
		AnalyzerVersion: Version,
	})
	if err != nil {
		return err
	}

	if env.JSON() {
		doc := map[string]any{
			"schema_version":     domain.SchemaVersion,
			"tool_version":       Version,
			"root":               result.Root,
			"index_path":         result.IndexPath,
			"files_indexed":      result.FilesIndexed,
			"symbols_count":      result.SymbolsCount,
			"dependencies_count": result.DependenciesCount,
			"duration_ms":        result.Duration.Milliseconds(),
		}
		return writeJSONDocument(env.Stdout, doc, "index")
	}

	fmt.Fprintf(env.Stdout, "Index built successfully\n")
	fmt.Fprintf(env.Stdout, "  root: %s\n", result.Root)
	fmt.Fprintf(env.Stdout, "  index: %s\n", result.IndexPath)
	fmt.Fprintf(env.Stdout, "  files indexed: %d\n", result.FilesIndexed)
	fmt.Fprintf(env.Stdout, "  symbols: %d\n", result.SymbolsCount)
	fmt.Fprintf(env.Stdout, "  dependencies: %d\n", result.DependenciesCount)
	fmt.Fprintf(env.Stdout, "  duration: %s\n", result.Duration)
	return nil
}

// runIndexStatus shows the status of the structural index.
func runIndexStatus(ctx context.Context, env *Env) error {
	root, err := resolveRoot(env.Root)
	if err != nil {
		return err
	}

	result, err := application.RunIndexStatus(ctx, application.IndexStatusRequest{
		Root:      root,
		IndexPath: indexPath,
	})
	if err != nil {
		return err
	}

	if env.JSON() {
		doc := map[string]any{
			"schema_version": domain.SchemaVersion,
			"tool_version":   Version,
			"root":           result.RootPath,
			"status":         "indexed",
			"files_count":    result.FileCount,
			"symbols_count":  result.SymbolCount,
			"deps_count":     result.DepCount,
			"last_indexed":   result.LastIndexed,
		}
		return writeJSONDocument(env.Stdout, doc, "index status")
	}

	fmt.Fprintf(env.Stdout, "Index status\n")
	fmt.Fprintf(env.Stdout, "  root: %s\n", result.RootPath)
	fmt.Fprintf(env.Stdout, "  status: ready\n")
	fmt.Fprintf(env.Stdout, "  files: %d\n", result.FileCount)
	fmt.Fprintf(env.Stdout, "  symbols: %d\n", result.SymbolCount)
	fmt.Fprintf(env.Stdout, "  dependencies: %d\n", result.DepCount)
	if result.LastIndexed > 0 {
		fmt.Fprintf(env.Stdout, "  last indexed: %d\n", result.LastIndexed)
	}
	return nil
}

// runIndexOverview shows an overview of the structural index.
func runIndexOverview(ctx context.Context, env *Env) error {
	root, err := resolveRoot(env.Root)
	if err != nil {
		return err
	}

	// For overview, we need more detailed info. Use the application layer to get symbols.
	findResult, err := application.RunIndexFind(ctx, application.IndexFindRequest{
		Root:      root,
		IndexPath: indexPath,
		Query:     "",
	})
	if err != nil {
		return err
	}

	byLang := make(map[string]int)
	byKind := make(map[string]int)
	for _, sym := range findResult {
		byLang[sym.Language]++
		byKind[sym.Kind]++
	}

	if env.JSON() {
		doc := map[string]any{
			"schema_version": domain.SchemaVersion,
			"tool_version":   Version,
			"root":           root,
			"overview": map[string]any{
				"total_symbols": len(findResult),
				"by_language":   byLang,
				"by_kind":       byKind,
			},
		}
		return writeJSONDocument(env.Stdout, doc, "index overview")
	}

	fmt.Fprintf(env.Stdout, "Index overview\n")
	fmt.Fprintf(env.Stdout, "  root: %s\n", root)
	fmt.Fprintf(env.Stdout, "  total symbols: %d\n", len(findResult))
	fmt.Fprintf(env.Stdout, "  by language:\n")
	for lang, count := range byLang {
		fmt.Fprintf(env.Stdout, "    %s: %d\n", lang, count)
	}
	fmt.Fprintf(env.Stdout, "  by kind:\n")
	for kind, count := range byKind {
		fmt.Fprintf(env.Stdout, "    %s: %d\n", kind, count)
	}
	return nil
}

// runIndexFind finds symbols matching a pattern.
func runIndexFind(ctx context.Context, env *Env, query string) error {
	if strings.TrimSpace(query) == "" {
		return Usagef("query is required")
	}

	root, err := resolveRoot(env.Root)
	if err != nil {
		return err
	}

	results, err := application.RunIndexFind(ctx, application.IndexFindRequest{
		Root:      root,
		IndexPath: indexPath,
		Query:     query,
	})
	if err != nil {
		return err
	}

	if env.JSON() {
		out := make([]map[string]any, len(results))
		for i, sym := range results {
			out[i] = map[string]any{
				"name":     sym.Name,
				"kind":     sym.Kind,
				"file":     sym.File,
				"line":     sym.Line,
				"language": sym.Language,
			}
		}
		doc := map[string]any{
			"schema_version": domain.SchemaVersion,
			"tool_version":   Version,
			"root":           root,
			"query":          query,
			"results":        out,
		}
		return writeJSONDocument(env.Stdout, doc, "index find")
	}

	fmt.Fprintf(env.Stdout, "Find results for query: %s\n", query)
	for _, sym := range results {
		fmt.Fprintf(env.Stdout, "  %s %s:%d\n", sym.Name, sym.File, sym.Line)
	}
	return nil
}

// runIndexRefs finds references of a symbol.
func runIndexRefs(ctx context.Context, env *Env, symbolName string) error {
	if strings.TrimSpace(symbolName) == "" {
		return Usagef("symbol name is required")
	}

	root, err := resolveRoot(env.Root)
	if err != nil {
		return err
	}

	results, err := application.RunIndexRefs(ctx, application.IndexRefsRequest{
		Root:      root,
		IndexPath: indexPath,
		Symbol:    symbolName,
	})
	if err != nil {
		return err
	}

	if env.JSON() {
		out := make([]map[string]any, len(results))
		for i, sym := range results {
			out[i] = map[string]any{
				"name":     sym.Name,
				"kind":     sym.Kind,
				"file":     sym.File,
				"line":     sym.Line,
				"language": sym.Language,
			}
		}
		doc := map[string]any{
			"schema_version": domain.SchemaVersion,
			"tool_version":   Version,
			"root":           root,
			"symbol":         symbolName,
			"references":     out,
		}
		return writeJSONDocument(env.Stdout, doc, "index refs")
	}

	fmt.Fprintf(env.Stdout, "References for symbol: %s\n", symbolName)
	for _, sym := range results {
		fmt.Fprintf(env.Stdout, "  %s %s:%d\n", sym.Name, sym.File, sym.Line)
	}
	return nil
}

// runIndexDeps lists declared dependencies from the index.
func runIndexDeps(ctx context.Context, env *Env) error {
	root, err := resolveRoot(env.Root)
	if err != nil {
		return err
	}

	results, err := application.RunIndexDeps(ctx, application.IndexDepsRequest{
		Root:      root,
		IndexPath: indexPath,
	})
	if err != nil {
		return err
	}

	if env.JSON() {
		out := make([]map[string]any, len(results))
		for i, dep := range results {
			out[i] = map[string]any{
				"path":     dep.Path,
				"file":     dep.File,
				"line":     dep.Line,
				"language": dep.Language,
			}
		}
		doc := map[string]any{
			"schema_version": domain.SchemaVersion,
			"tool_version":   Version,
			"root":           root,
			"dependencies":   out,
		}
		return writeJSONDocument(env.Stdout, doc, "index deps")
	}

	fmt.Fprintf(env.Stdout, "Declared dependencies\n")
	fmt.Fprintf(env.Stdout, "  root: %s\n", root)
	fmt.Fprintf(env.Stdout, "  total: %d\n", len(results))
	for _, dep := range results {
		fmt.Fprintf(env.Stdout, "  %s %s:%d\n", dep.Path, dep.File, dep.Line)
	}
	return nil
}

// runIndexDiff compares blobs between base and HEAD without checkout.
func runIndexDiff(ctx context.Context, env *Env) error {
	root, err := resolveRoot(env.Root)
	if err != nil {
		return err
	}

	if err := checkGitRepository(root); err != nil {
		return err
	}

	base := indexDiffBase
	if base == "" {
		return Usagef("the --base flag is required for index diff")
	}

	// Use git diff to compare blobs without checkout
	// This leverages the existing git integration
	source, err := git.New(root)
	if err != nil {
		return err
	}

	changes, err := source.Changes(ctx, domain.OriginBase, base)
	if err != nil {
		return err
	}

	if env.JSON() {
		out := make([]map[string]any, len(changes.Files))
		for i, file := range changes.Files {
			added, removed := 0, 0
			for _, hunk := range file.Hunks {
				for _, line := range hunk.Lines {
					if line.Kind == domain.LineAdded {
						added++
					} else if line.Kind == domain.LineRemoved {
						removed++
					}
				}
			}
			out[i] = map[string]any{
				"path":     file.Path,
				"old_path": file.OldPath,
				"status":   file.Status,
				"language": file.Language,
				"added":    added,
				"removed":  removed,
			}
		}
		doc := map[string]any{
			"schema_version": domain.SchemaVersion,
			"tool_version":   Version,
			"root":           root,
			"base":           base,
			"head":           changes.Head,
			"files":          out,
		}
		return writeJSONDocument(env.Stdout, doc, "index diff")
	}

	fmt.Fprintf(env.Stdout, "Blob comparison (base: %s, head: %s)\n", base, changes.Head)
	for _, file := range changes.Files {
		added, removed := 0, 0
		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				if line.Kind == domain.LineAdded {
					added++
				} else if line.Kind == domain.LineRemoved {
					removed++
				}
			}
		}
		fmt.Fprintf(env.Stdout, "  %s (%s) +%d -%d\n", file.Path, file.Status, added, removed)
	}
	return nil
}
