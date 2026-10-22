package claims

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/barceloch/codixai/internal/domain"
)

// The formats of the external test reports that the package reads.
const (
	// FormatJUnitXML is the JUnit XML format.
	FormatJUnitXML = "junit-xml"
	// FormatGoTestJSON is the JSON stream of "go test -json".
	FormatGoTestJSON = "go-test-json"
)

// maxReportLineSize bounds the size of a single line of a report. A longer
// line is rejected instead of exhausting the memory of the process.
const maxReportLineSize = 4 * 1024 * 1024

// report holds the outcome of an external test report.
//
// The report is always external evidence: it was produced by another tool
// and Codixai did not run the tests of the repository.
type report struct {
	// Format is the identifier of the format of the report.
	Format string
	// Tests is the number of tests the report counts.
	Tests int
	// Failures is the number of failed tests.
	Failures int
	// Errors is the number of errors: the error elements of a JUnit XML
	// document and the failed packages of a "go test -json" stream.
	Errors int
	// Revision is the revision the report declares, when it declares one.
	Revision string
}

// success reports whether the report holds no failure and no error.
func (r report) success() bool {
	return r.Failures == 0 && r.Errors == 0
}

// parseReport reads a report of the given path.
//
// The format is decided by the extension of the path: ".xml" is a JUnit XML
// document and ".json" is the JSON stream of "go test -json". A report in
// another format is rejected: the package never guesses.
func parseReport(path string, data []byte) (report, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".xml":
		return parseJUnitXML(data)
	case ".json":
		return parseGoTestJSON(data)
	default:
		return report{}, fmt.Errorf("the format of the report %q is not supported: only JUnit XML (\".xml\") and the JSON stream of \"go test -json\" (\".json\") are read", path)
	}
}

// junitSuite holds the counters of one suite of a JUnit XML document.
type junitSuite struct {
	tests         int
	failures      int
	errors        int
	attrTests     int
	attrFailures  int
	attrErrors    int
	hasAttrTests  bool
	hasAttrFail   bool
	hasAttrErrors bool
}

// parseJUnitXML reads the counters and the revision of a JUnit XML document.
//
// The counters of the child elements are authoritative. The attributes of a
// suite are only used when the suite holds no child element, because the two
// kinds of counter describe the same thing.
func parseJUnitXML(data []byte) (report, error) {
	result := report{Format: FormatJUnitXML}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var root junitSuite
	var suites []junitSuite
	var current *junitSuite
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return report{}, fmt.Errorf("the report is not a valid JUnit XML document: %w", err)
		}
		switch element := token.(type) {
		case xml.StartElement:
			switch element.Name.Local {
			case "testsuites":
				readJUnitAttributes(&root, element)
			case "testsuite":
				suite := junitSuite{}
				readJUnitAttributes(&suite, element)
				current = &suite
			case "testcase":
				if current != nil {
					current.tests++
				}
			case "failure":
				if current != nil {
					current.failures++
				}
			case "error":
				if current != nil {
					current.errors++
				}
			case "property":
				name := xmlAttribute(element, "name")
				value := xmlAttribute(element, "value")
				if (name == "revision" || name == "commit") && value != "" && result.Revision == "" {
					result.Revision = value
				}
			}
		case xml.EndElement:
			if element.Name.Local == "testsuite" && current != nil {
				suites = append(suites, *current)
				current = nil
			}
		}
	}
	if len(suites) == 0 {
		// The document holds a single suite, or the counters are only
		// declared by the attributes of the root element.
		result.Tests += root.countTests()
		result.Failures += root.countFailures()
		result.Errors += root.countErrors()
		return result, nil
	}
	for _, suite := range suites {
		result.Tests += suite.countTests()
		result.Failures += suite.countFailures()
		result.Errors += suite.countErrors()
	}
	return result, nil
}

// readJUnitAttributes reads the counters declared by the attributes of a
// suite element.
func readJUnitAttributes(suite *junitSuite, element xml.StartElement) {
	for _, attribute := range element.Attr {
		switch attribute.Name.Local {
		case "tests":
			if value, err := strconv.Atoi(strings.TrimSpace(attribute.Value)); err == nil {
				suite.attrTests = value
				suite.hasAttrTests = true
			}
		case "failures":
			if value, err := strconv.Atoi(strings.TrimSpace(attribute.Value)); err == nil {
				suite.attrFailures = value
				suite.hasAttrFail = true
			}
		case "errors":
			if value, err := strconv.Atoi(strings.TrimSpace(attribute.Value)); err == nil {
				suite.attrErrors = value
				suite.hasAttrErrors = true
			}
		}
	}
}

// countTests returns the number of tests of a suite.
func (s junitSuite) countTests() int {
	if s.tests == 0 && s.hasAttrTests {
		return s.attrTests
	}
	return s.tests
}

// countFailures returns the number of failures of a suite.
func (s junitSuite) countFailures() int {
	if s.failures == 0 && s.hasAttrFail {
		return s.attrFailures
	}
	return s.failures
}

// countErrors returns the number of errors of a suite.
func (s junitSuite) countErrors() int {
	if s.errors == 0 && s.hasAttrErrors {
		return s.attrErrors
	}
	return s.errors
}

// xmlAttribute returns the value of an attribute of an element.
func xmlAttribute(element xml.StartElement, name string) string {
	for _, attribute := range element.Attr {
		if attribute.Name.Local == name {
			return attribute.Value
		}
	}
	return ""
}

// goTestEvent is one event of the JSON stream of "go test -json".
type goTestEvent struct {
	// Action is the event: "run", "pass", "fail" or "skip".
	Action string `json:"Action"`
	// Package is the package of the event.
	Package string `json:"Package"`
	// Test is the test of the event, empty for a package event.
	Test string `json:"Test"`
	// Revision is the revision the event declares, when it declares one.
	Revision string `json:"revision"`
	// RevisionUpper is the same field written with the capital of the
	// other keys of the stream.
	RevisionUpper string `json:"Revision"`
}

// parseGoTestJSON reads the counters and the revision of the JSON stream of
// "go test -json".
func parseGoTestJSON(data []byte) (report, error) {
	result := report{Format: FormatGoTestJSON}
	tests := map[string]bool{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), maxReportLineSize)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var event goTestEvent
		if err := json.Unmarshal([]byte(text), &event); err != nil {
			return report{}, fmt.Errorf("the report is not a valid \"go test -json\" stream: the line %d is not a JSON object", line)
		}
		revision := event.Revision
		if revision == "" {
			revision = event.RevisionUpper
		}
		if revision != "" && result.Revision == "" {
			result.Revision = revision
		}
		switch event.Action {
		case "fail":
			if event.Test != "" {
				result.Failures++
				tests[event.Test] = true
				continue
			}
			result.Errors++
		case "pass", "skip":
			if event.Test != "" {
				tests[event.Test] = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return report{}, fmt.Errorf("the report cannot be read: %w", err)
	}
	result.Tests = len(tests)
	return result, nil
}

// linkReport decides whether a report can be linked to the revision under
// analysis.
//
// A report is linked when the claim declares a report revision that is the
// head of the compared range, or when the report itself declares that
// revision. An unlinked report leaves the claim unresolved.
func linkReport(claim domain.Claim, parsed report, head string) (bool, string) {
	if claim.ReportRevision != "" && head != "" && claim.ReportRevision == head {
		return true, fmt.Sprintf("the claim declares the report revision %q, which is the head of the compared range", head)
	}
	if parsed.Revision != "" && head != "" && parsed.Revision == head {
		return true, fmt.Sprintf("the report declares the revision %q, which is the head of the compared range", head)
	}
	switch {
	case claim.ReportRevision != "" && head != "":
		return false, fmt.Sprintf("the claim declares the report revision %q but the head of the compared range is %q", claim.ReportRevision, head)
	case parsed.Revision != "" && head != "":
		return false, fmt.Sprintf("the report declares the revision %q but the head of the compared range is %q", parsed.Revision, head)
	case head == "":
		return false, "the head of the compared range is unknown, so the report cannot be linked to a revision"
	default:
		return false, fmt.Sprintf("neither the claim nor the report declares a revision, so the report cannot be linked to %q", head)
	}
}
