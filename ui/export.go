//go:build ui

package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/barceloch/codixai/internal/application"
	"github.com/barceloch/codixai/internal/report"
)

// EncodeExport returns the bytes of the report of a finished run.
//
// The JSON format is the same document, byte for byte, that
// "codixai audit --format json" writes for the same changes: the same
// encoder and the same indentation. The SARIF format is the same document
// that "codixai report --format sarif" writes. The parity is checked by
// tests/integration/ui_parity_test.go.
//
// The function is exported so that the parity test exercises the exact
// encoder of the export and not a copy of it.
func EncodeExport(result any, format string) ([]byte, error) {
	switch r := result.(type) {
	case *application.AuditResult:
		if r == nil || r.Run == nil {
			return nil, fmt.Errorf("the audit did not produce a document to export: run the audit first")
		}
		return encodeAuditExport(r, format)
	case *application.VerifyResult:
		if r == nil || r.Document == nil {
			return nil, fmt.Errorf("the verification did not produce a document to export: run the verification first")
		}
		return encodeVerifyExport(r, format)
	default:
		return nil, fmt.Errorf("unknown export result type %T", result)
	}
}

func encodeAuditExport(result *application.AuditResult, format string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		return encodeJSON(result.Run, "JSON report")
	case report.Format:
		document := report.SARIF(result, report.SARIFOptions{ToolVersion: result.Run.ToolVersion})
		return encodeJSON(document, "SARIF report")
	default:
		return nil, fmt.Errorf("unknown export format %q: choose json or sarif", format)
	}
}

func encodeVerifyExport(result *application.VerifyResult, format string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		return encodeJSON(result.Document, "JSON report")
	case report.Format:
		// TODO: SARIF for verification is not yet implemented
		return nil, fmt.Errorf("SARIF export for verification is not yet implemented")
	default:
		return nil, fmt.Errorf("unknown export format %q: choose json or sarif", format)
	}
}

// encodeJSON writes a document with the layout of the CLI.
//
// It mirrors writeJSONDocument of internal/cli: the interface and the
// command line have to produce the same bytes for the same value, so the
// encoder is kept identical on purpose.
func encodeJSON(value any, name string) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("cannot write the %s: %w", name, err)
	}
	return buffer.Bytes(), nil
}
