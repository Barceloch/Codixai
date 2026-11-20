package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/barceloch/codixai/internal/measurement"
)

// measureBaseline and measureCodixai hold the --baseline / --codixai flags.
var measureBaseline string
var measureCodixai string

// newMeasureCommand builds the measure subcommand.
func newMeasureCommand(env *Env) *Command {
	cmd := &Command{
		Use:   "measure",
		Short: "Record and compare measurement runs for ACO Phase 0.",
		Long: "Measure is the ACO Phase 0 measurement harness. It validates run records\n" +
			"produced by an operator and compares a baseline run against a run that used\n" +
			"Codixai. Token counts are carried with an explicit provenance so that an\n" +
			"estimate is never presented as a provider-reported measurement.\n" +
			"\n" +
			"The tool never calls a model provider, never transmits data and never\n" +
			"executes analyzed code.",
	}
	cmd.AddCommand(&Command{
		Use:   "validate <file>",
		Short: "Validate a measurement run record.",
		Flags: func(fs *flag.FlagSet) {},
		RunE: func(ctx context.Context, args []string) error {
			if len(args) != 1 {
				return Usagef("validate requires exactly one file argument")
			}
			return runMeasureValidate(ctx, env, args[0])
		},
	})
	cmd.AddCommand(&Command{
		Use:   "compare",
		Short: "Compare a baseline run against a Codixai run.",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&measureBaseline, "baseline", "", "Path to the baseline run record (required)")
			fs.StringVar(&measureCodixai, "codixai", "", "Path to the Codixai run record (required)")
		},
		RunE: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return Usagef("compare does not accept positional arguments")
			}
			return runMeasureCompare(ctx, env)
		},
	})
	cmd.AddCommand(&Command{
		Use:   "tasks",
		Short: "List the registered measurement tasks.",
		Flags: func(fs *flag.FlagSet) {},
		RunE: func(ctx context.Context, args []string) error {
			if len(args) > 0 {
				return Usagef("tasks does not accept arguments")
			}
			return runMeasureTasks(ctx, env)
		},
	})
	return cmd
}

func readRunRecord(path string) (measurement.RunRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return measurement.RunRecord{}, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var r measurement.RunRecord
	if err := json.Unmarshal(data, &r); err != nil {
		return measurement.RunRecord{}, fmt.Errorf("cannot parse %s: %w", path, err)
	}
	return r, nil
}

func runMeasureValidate(ctx context.Context, env *Env, path string) error {
	r, err := readRunRecord(path)
	if err != nil {
		return sanitizeError(err, "")
	}
	if err := measurement.ValidateRunRecord(r); err != nil {
		fmt.Fprintf(env.Stderr, "codixai: invalid run record: %v\n", err)
		return err
	}
	if env.JSON() {
		doc := map[string]any{
			"schema_version": measurement.SchemaVersion,
			"tool_version":   Version,
			"file":           path,
			"valid":          true,
		}
		return writeJSONDocument(env.Stdout, doc, "measure validate")
	}
	fmt.Fprintf(env.Stdout, "valid run record: %s\n", path)
	return nil
}

func runMeasureCompare(ctx context.Context, env *Env) error {
	if measureBaseline == "" {
		return Usagef("--baseline is required")
	}
	if measureCodixai == "" {
		return Usagef("--codixai is required")
	}
	b, err := readRunRecord(measureBaseline)
	if err != nil {
		return sanitizeError(err, "")
	}
	c, err := readRunRecord(measureCodixai)
	if err != nil {
		return sanitizeError(err, "")
	}
	report, err := measurement.Compare(b, c, Version)
	if err != nil {
		return sanitizeError(err, "")
	}
	if env.JSON() {
		return writeJSONDocument(env.Stdout, report, "measure compare")
	}
	// Text summary.
	fmt.Fprintf(env.Stdout, "Task: %s\n", report.TaskID)
	fmt.Fprintf(env.Stdout, "Comparable: %v\n", report.Comparability.Comparable)
	if len(report.Comparability.Reasons) > 0 {
		for _, r := range report.Comparability.Reasons {
			fmt.Fprintf(env.Stdout, "  reason: %s\n", r)
		}
	}
	if report.Savings != nil && report.Savings.TotalTokens != nil {
		fmt.Fprintf(env.Stdout, "Total-token savings: %.2f%%\n", *report.Savings.TotalTokens)
	} else {
		fmt.Fprintf(env.Stdout, "Total-token savings: not computable\n")
	}
	fmt.Fprintf(env.Stdout, "Interpretation: %s\n", report.Interpretation)
	return nil
}

func runMeasureTasks(ctx context.Context, env *Env) error {
	reg, err := measurement.LoadTasks()
	if err != nil {
		return sanitizeError(err, "")
	}
	if env.JSON() {
		doc := map[string]any{
			"schema_version": measurement.SchemaVersion,
			"tool_version":   Version,
			"tasks":          reg.Tasks,
		}
		return writeJSONDocument(env.Stdout, doc, "measure tasks")
	}
	for _, t := range reg.Tasks {
		fmt.Fprintf(env.Stdout, "%s (%s): %s\n", t.ID, t.Kind, t.Description)
	}
	return nil
}
