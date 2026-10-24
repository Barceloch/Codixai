//go:build !ui

package integration

import (
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/cli"
)

// TestUIConfigurationIsNotAvailableInTheDefaultBuild checks that the
// CLI only build answers the ui command with a usage error that explains
// how to obtain the variant with the interface.
//
// The test belongs to the default variant: the build with the ui tag
// replaces the command with the one that opens the window, so the file is
// excluded from that build.
func TestUIConfigurationIsNotAvailableInTheDefaultBuild(t *testing.T) {
	code, stdout, stderr := execute(t, "ui")

	if code != cli.ExitInvalidArgs {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitInvalidArgs, stderr)
	}
	if stdout != "" {
		t.Errorf("the standard output is not empty:\n%s", stdout)
	}
	for _, want := range []string{"ui", "-tags", "production"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("the error does not mention %q:\n%s", want, stderr)
		}
	}
}

// TestUIDefaultBuildRejectsArguments checks that the ui command of the
// CLI only build rejects positional arguments as an invalid invocation.
func TestUIDefaultBuildRejectsArguments(t *testing.T) {
	code, _, stderr := execute(t, "ui", "extra")

	if code != cli.ExitInvalidArgs {
		t.Fatalf("the exit code is %d, want %d: %s", code, cli.ExitInvalidArgs, stderr)
	}
}
