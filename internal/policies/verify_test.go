package policies

import (
	"strings"
	"testing"
)

func TestVerifyFailOnUnresolvedDefaultsToFalse(t *testing.T) {
	config, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if config.Verify.FailOnUnresolved {
		t.Error("expected verify.fail_on_unresolved to be false by default")
	}
}

func TestVerifyFailOnUnresolvedIsRead(t *testing.T) {
	dir := writeConfig(t, `version: 1

verify:
  fail_on_unresolved: true
`)
	config, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if !config.Verify.FailOnUnresolved {
		t.Error("expected verify.fail_on_unresolved to be true")
	}
}

func TestVerifyFailOnUnresolvedAcceptsFalse(t *testing.T) {
	dir := writeConfig(t, `version: 1

verify:
  fail_on_unresolved: false
`)
	config, err := Load(dir)
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}
	if config.Verify.FailOnUnresolved {
		t.Error("expected verify.fail_on_unresolved to be false")
	}
}

func TestVerifyFailOnUnresolvedRejectsNonBoolean(t *testing.T) {
	dir := writeConfig(t, `version: 1

verify:
  fail_on_unresolved: maybe
`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected an error for a non boolean value")
	}
	if !strings.Contains(err.Error(), "verify.fail_on_unresolved must be a boolean") {
		t.Errorf("unexpected message: %v", err)
	}
}

func TestVerifyMustBeAMapping(t *testing.T) {
	dir := writeConfig(t, `version: 1

verify: true
`)
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected an error when verify is not a mapping")
	}
	if !strings.Contains(err.Error(), "verify must be a mapping of keys") {
		t.Errorf("unexpected message: %v", err)
	}
}
