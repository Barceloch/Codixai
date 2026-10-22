package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/claims"
	"github.com/barceloch/codixai/internal/integrations/git"
)

// verifyOptions holds the flags of the verify command.
type verifyOptions struct {
	base   string
	claims string
}

// newVerifyCommand builds the verify command.
func newVerifyCommand(env *Env) *Command {
	options := &verifyOptions{}
	command := &Command{
		Use:   "verify",
		Short: "Verify the claims of an agent against the repository.",
		Long: "Verify the claims of an agent against the state of the repository.\n" +
			"\n" +
			"The command reads a claim file, compares its claims with the changes\n" +
			"between the base revision and the head of the repository and reports one\n" +
			"state per claim: verified, contradicted or unresolved. An unresolved\n" +
			"claim means that the evidence was not enough to decide, never that the\n" +
			"claim is true.\n" +
			"\n" +
			"The claim file is a JSON document with a versioned format:\n" +
			"\n" +
			"  {\n" +
			"    \"schema_version\": \"0.1\",\n" +
			"    \"base\": \"main\",\n" +
			"    \"claims\": [\n" +
			"      { \"id\": \"c1\", \"type\": \"tests_added\", \"paths\": [\"internal/payment/**\"] },\n" +
			"      { \"id\": \"c2\", \"type\": \"files_unchanged\", \"paths\": [\"internal/billing/**\"] },\n" +
			"      { \"id\": \"c3\", \"type\": \"tests_passed\", \"report\": \"reports/go-test.xml\" }\n" +
			"    ]\n" +
			"  }\n" +
			"\n" +
			"The base of the file is optional. When it is present it has to match the\n" +
			"base of the comparison, because the claims would be about another\n" +
			"revision. The types of the catalog are:\n" +
			"\n" +
			"  " + strings.Join(claims.Types(), ", ") + "\n" +
			"\n" +
			"The format of the claim file, the catalog of the types and the linking of\n" +
			"an external test report are fixed by the decision 0007, in\n" +
			"docs/decisions/0007-claim-format-and-report-linking.md.\n" +
			"\n" +
			"The command reads the repository and the claim file: it never modifies the\n" +
			"repository, it never runs the tests of the repository and it never\n" +
			"accesses the network.",
		Example: "codixai verify --base main --claims claims.json\n" +
			"codixai verify --base main --claims claims.json --format json",
		Flags: func(fs *flag.FlagSet) {
			fs.StringVar(&options.base, "base", "", "Base revision to compare the repository with HEAD. Required.")
			fs.StringVar(&options.claims, "claims", "", "Path of the claim file to verify. Required.")
		},
	}
	command.RunE = func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return Usagef("the verify command does not accept arguments, got %q", args[0])
		}
		return runVerify(ctx, env, options)
	}
	return command
}

// runVerify runs the verify command.
func runVerify(ctx context.Context, env *Env, options *verifyOptions) error {
	if strings.TrimSpace(options.base) == "" {
		return Usagef("the flag --base is required: use --base <ref> to choose the revision the claims are about")
	}
	if strings.TrimSpace(options.claims) == "" {
		return Usagef("the flag --claims is required: use --claims <file> to choose the claim file to verify")
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

	result, err := application.RunVerify(ctx, application.VerifyRequest{
		Root:        root,
		Base:        options.base,
		ClaimsPath:  options.claims,
		Source:      source,
		Loader:      claims.Loader{},
		Evaluator:   claims.NewEvaluator(claims.WithRoot(root), claims.WithBase(options.base)),
		ToolVersion: Version,
	})
	if err != nil {
		return sanitizeError(err, root)
	}

	if env.JSON() {
		encoder := json.NewEncoder(env.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result.Document); err != nil {
			return fmt.Errorf("cannot write the verification document: %w", err)
		}
	} else {
		writeVerifyReport(env.Stdout, env.Renderer(), result)
	}

	if result.Failed {
		return &VerificationError{
			Count:      result.Contradicted,
			Unresolved: result.Unresolved,
			Threshold:  verificationThreshold(result),
		}
	}
	return nil
}

// verificationThreshold describes the rule that made the verification fail.
func verificationThreshold(result *application.VerifyResult) string {
	if result.Contradicted > 0 {
		return "a contradicted claim"
	}
	return "an unresolved claim (verify.fail_on_unresolved is true)"
}
