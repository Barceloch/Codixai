package claims

import (
	"strings"
	"testing"

	"github.com/barceloch/codixai/internal/domain"
)

const junitSuccess = `<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="codixai" tests="2" failures="0" errors="0">
  <testsuite name="internal/payment" tests="2" failures="0" errors="0">
    <properties>
      <property name="revision" value="9f2c1ab"/>
    </properties>
    <testcase name="TestCharge" classname="internal/payment"/>
    <testcase name="TestRefund" classname="internal/payment"/>
  </testsuite>
</testsuites>
`

const junitFailure = `<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="codixai" tests="2" failures="1" errors="0">
  <testsuite name="internal/payment" tests="2" failures="1" errors="0">
    <properties>
      <property name="commit" value="9f2c1ab"/>
    </properties>
    <testcase name="TestCharge" classname="internal/payment"/>
    <testcase name="TestRefund" classname="internal/payment">
      <failure message="boom">unexpected value</failure>
    </testcase>
  </testsuite>
</testsuites>
`

const goTestSuccess = `{"Time":"2026-10-20T10:00:00Z","Action":"run","Package":"internal/payment","Test":"TestCharge"}
{"Time":"2026-10-20T10:00:01Z","Action":"pass","Package":"internal/payment","Test":"TestCharge","Elapsed":0.01}
{"Time":"2026-10-20T10:00:02Z","Action":"run","Package":"internal/payment","Test":"TestRefund"}
{"Time":"2026-10-20T10:00:03Z","Action":"pass","Package":"internal/payment","Test":"TestRefund","Elapsed":0.02}
{"Time":"2026-10-20T10:00:04Z","Action":"pass","Package":"internal/payment","Elapsed":0.03}
`

const goTestFailure = `{"Time":"2026-10-20T10:00:00Z","Action":"run","Package":"internal/payment","Test":"TestCharge","revision":"9f2c1ab"}
{"Time":"2026-10-20T10:00:01Z","Action":"fail","Package":"internal/payment","Test":"TestCharge","Elapsed":0.01}
{"Time":"2026-10-20T10:00:02Z","Action":"fail","Package":"internal/payment","Elapsed":0.02}
`

func TestParseJUnitXML(t *testing.T) {
	parsed, err := parseReport("reports/junit.xml", []byte(junitSuccess))
	if err != nil {
		t.Fatalf("parseReport returned an error: %v", err)
	}
	if parsed.Format != FormatJUnitXML {
		t.Errorf("format: got %q, want %q", parsed.Format, FormatJUnitXML)
	}
	if parsed.Tests != 2 {
		t.Errorf("tests: got %d, want 2", parsed.Tests)
	}
	if parsed.Failures != 0 || parsed.Errors != 0 {
		t.Errorf("failures: got %d, errors: got %d, want 0 and 0", parsed.Failures, parsed.Errors)
	}
	if parsed.Revision != "9f2c1ab" {
		t.Errorf("revision: got %q, want %q", parsed.Revision, "9f2c1ab")
	}
	if !parsed.success() {
		t.Error("expected the report to be a success")
	}
}

func TestParseJUnitXMLCountsTheFailures(t *testing.T) {
	parsed, err := parseReport("reports/junit.xml", []byte(junitFailure))
	if err != nil {
		t.Fatalf("parseReport returned an error: %v", err)
	}
	if parsed.Tests != 2 {
		t.Errorf("tests: got %d, want 2", parsed.Tests)
	}
	if parsed.Failures != 1 {
		t.Errorf("failures: got %d, want 1", parsed.Failures)
	}
	if parsed.success() {
		t.Error("expected the report not to be a success")
	}
	if parsed.Revision != "9f2c1ab" {
		t.Errorf("revision: got %q, want %q", parsed.Revision, "9f2c1ab")
	}
}

func TestParseJUnitXMLUsesTheAttributesOfAnEmptySuite(t *testing.T) {
	document := `<testsuite name="internal/payment" tests="3" failures="1" errors="2"/>`
	parsed, err := parseReport("reports/junit.xml", []byte(document))
	if err != nil {
		t.Fatalf("parseReport returned an error: %v", err)
	}
	if parsed.Tests != 3 || parsed.Failures != 1 || parsed.Errors != 2 {
		t.Errorf("counters: got %d tests, %d failures and %d errors", parsed.Tests, parsed.Failures, parsed.Errors)
	}
}

func TestParseJUnitXMLRejectsAMalformedDocument(t *testing.T) {
	if _, err := parseReport("reports/junit.xml", []byte("<testsuites>")); err == nil {
		t.Fatal("expected an error for a malformed document")
	}
}

func TestParseGoTestJSON(t *testing.T) {
	parsed, err := parseReport("reports/go-test.json", []byte(goTestSuccess))
	if err != nil {
		t.Fatalf("parseReport returned an error: %v", err)
	}
	if parsed.Format != FormatGoTestJSON {
		t.Errorf("format: got %q, want %q", parsed.Format, FormatGoTestJSON)
	}
	if parsed.Tests != 2 {
		t.Errorf("tests: got %d, want 2", parsed.Tests)
	}
	if parsed.Failures != 0 || parsed.Errors != 0 {
		t.Errorf("failures: got %d, errors: got %d, want 0 and 0", parsed.Failures, parsed.Errors)
	}
	if parsed.Revision != "" {
		t.Errorf("revision: got %q, want an empty revision", parsed.Revision)
	}
	if !parsed.success() {
		t.Error("expected the report to be a success")
	}
}

func TestParseGoTestJSONCountsTheFailures(t *testing.T) {
	parsed, err := parseReport("reports/go-test.json", []byte(goTestFailure))
	if err != nil {
		t.Fatalf("parseReport returned an error: %v", err)
	}
	if parsed.Failures != 1 {
		t.Errorf("failures: got %d, want 1", parsed.Failures)
	}
	if parsed.Errors != 1 {
		t.Errorf("errors: got %d, want 1", parsed.Errors)
	}
	if parsed.Revision != "9f2c1ab" {
		t.Errorf("revision: got %q, want %q", parsed.Revision, "9f2c1ab")
	}
	if parsed.success() {
		t.Error("expected the report not to be a success")
	}
}

func TestParseGoTestJSONRejectsAMalformedStream(t *testing.T) {
	_, err := parseReport("reports/go-test.json", []byte("not json\n"))
	if err == nil {
		t.Fatal("expected an error for a malformed stream")
	}
	if !strings.Contains(err.Error(), "go test -json") {
		t.Errorf("unexpected message: %v", err)
	}
}

func TestParseReportRejectsAnUnknownFormat(t *testing.T) {
	_, err := parseReport("reports/report.txt", []byte("everything is fine"))
	if err == nil {
		t.Fatal("expected an error for an unknown format")
	}
	if !strings.Contains(err.Error(), "is not supported") {
		t.Errorf("unexpected message: %v", err)
	}
}

func TestLinkReportWithTheRevisionOfTheClaim(t *testing.T) {
	claim := domain.Claim{Report: "reports/go-test.json", ReportRevision: "9f2c1ab"}
	linked, reason := linkReport(claim, report{}, "9f2c1ab")
	if !linked {
		t.Fatalf("expected the report to be linked: %s", reason)
	}
}

func TestLinkReportWithTheRevisionOfTheReport(t *testing.T) {
	claim := domain.Claim{Report: "reports/go-test.json"}
	linked, reason := linkReport(claim, report{Revision: "9f2c1ab"}, "9f2c1ab")
	if !linked {
		t.Fatalf("expected the report to be linked: %s", reason)
	}
}

func TestLinkReportRejectsAnotherRevisionOfTheClaim(t *testing.T) {
	claim := domain.Claim{Report: "reports/go-test.json", ReportRevision: "1111111"}
	linked, reason := linkReport(claim, report{}, "9f2c1ab")
	if linked {
		t.Fatal("expected the report not to be linked")
	}
	if !strings.Contains(reason, "1111111") || !strings.Contains(reason, "9f2c1ab") {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestLinkReportRejectsAnotherRevisionOfTheReport(t *testing.T) {
	claim := domain.Claim{Report: "reports/go-test.json"}
	linked, reason := linkReport(claim, report{Revision: "1111111"}, "9f2c1ab")
	if linked {
		t.Fatal("expected the report not to be linked")
	}
	if !strings.Contains(reason, "1111111") {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestLinkReportWithoutAnyRevision(t *testing.T) {
	claim := domain.Claim{Report: "reports/go-test.json"}
	linked, reason := linkReport(claim, report{}, "9f2c1ab")
	if linked {
		t.Fatal("expected the report not to be linked")
	}
	if !strings.Contains(reason, "neither the claim nor the report declares a revision") {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestLinkReportWithoutAHead(t *testing.T) {
	claim := domain.Claim{Report: "reports/go-test.json"}
	linked, reason := linkReport(claim, report{Revision: "9f2c1ab"}, "")
	if linked {
		t.Fatal("expected the report not to be linked")
	}
	if !strings.Contains(reason, "head of the compared range is unknown") {
		t.Errorf("unexpected reason: %s", reason)
	}
}
