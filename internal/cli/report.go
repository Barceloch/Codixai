package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"codixai/internal/application"
	"codixai/internal/cli/ui"
	"codixai/internal/domain"
)

// writeAuditReport writes the human readable report of an audit.
//
// The report never hides a limitation and never claims a capability that
// the analysis does not have: the severity, the epistemic status and the
// support level of every finding are always present in their textual form.
func writeAuditReport(w io.Writer, r *ui.Renderer, result *application.AuditResult, root string) {
	run := result.Run

	fmt.Fprintln(w, r.Title("Codixai audit"))
	writeField(w, r, "  ", "origin", string(domain.ChangeOrigin(run.Origin)))
	writeField(w, r, "  ", "revision", describeRevision(run.Revision))
	writeField(w, r, "  ", "root", displayPath(root))
	writeField(w, r, "  ", "files", fmt.Sprintf("%d changed", result.ChangedFiles))
	writeField(w, r, "  ", "generated at", run.GeneratedAt)
	writeField(w, r, "  ", "tool", fmt.Sprintf("codixai %s (schema %s)", run.ToolVersion, run.SchemaVersion))
	writeField(w, r, "  ", "policy", fmt.Sprintf("fail_on %s", result.FailOn))

	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s (%d)\n", r.Title("Findings"), len(run.Findings))
	if len(run.Findings) == 0 {
		fmt.Fprintln(w, "  No finding. The absence of findings does not prove that the change is correct.")
	}
	for _, finding := range run.Findings {
		writeFinding(w, r, finding)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, r.Title("Summary"))
	writeField(w, r, "  ", "total", fmt.Sprintf("%d", run.Summary.Total))
	writeField(w, r, "  ", "by severity", describeCounts(run.Summary.BySeverity))
	writeField(w, r, "  ", "by rule", describeCounts(run.Summary.ByRule))
	writeField(w, r, "  ", "by language", describeCounts(run.Summary.ByLanguage))
	writeField(w, r, "  ", "by epistemic status", describeCounts(run.Summary.ByEpistemicStatus))

	if len(run.Limitations) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, r.Title("Limitations"))
		for _, limitation := range run.Limitations {
			fmt.Fprintf(w, "  - %s\n", limitation)
		}
	}
}

// writeFinding writes the block of one finding.
func writeFinding(w io.Writer, r *ui.Renderer, finding domain.Finding) {
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s  %s %s  %s\n",
		r.Severity(finding.Severity),
		r.Bold(finding.RuleID),
		r.Dim("v"+finding.RuleVersion),
		r.Dim(string(finding.EpistemicStatus)),
	)
	writeField(w, r, "    ", "path", describeLocation(finding.Location))
	writeField(w, r, "    ", "message", finding.Message)
	if len(finding.Evidence) > 0 {
		writeField(w, r, "    ", "evidence", describeEvidence(finding.Evidence))
	}
	writeField(w, r, "    ", "support", describeSupport(finding.Support))
	if len(finding.Support.Limitations) > 0 {
		writeField(w, r, "    ", "limitations", "")
		for _, limitation := range finding.Support.Limitations {
			fmt.Fprintf(w, "      - %s\n", limitation)
		}
	}
	if finding.Recommendation != "" {
		writeField(w, r, "    ", "recommendation", finding.Recommendation)
	}
}

// writeField writes a label and its value with an aligned layout.
func writeField(w io.Writer, r *ui.Renderer, indent, label, value string) {
	if value == "" {
		fmt.Fprintf(w, "%s%s\n", indent, r.Dim(label+":"))
		return
	}
	fmt.Fprintf(w, "%s%-18s %s\n", indent, r.Dim(label+":"), value)
}

// describeLocation renders the location of a finding.
func describeLocation(location domain.Location) string {
	if location.StartLine <= 0 {
		return location.Path
	}
	if location.EndLine > 0 && location.EndLine != location.StartLine {
		return fmt.Sprintf("%s:%d-%d", location.Path, location.StartLine, location.EndLine)
	}
	return fmt.Sprintf("%s:%d", location.Path, location.StartLine)
}

// describeRevision renders the compared endpoints of the diff.
func describeRevision(revision domain.Revision) string {
	parts := make([]string, 0, 2)
	if revision.Base != "" {
		parts = append(parts, "base "+revision.Base)
	}
	if revision.Head != "" {
		parts = append(parts, "head "+revision.Head)
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, ", ")
}

// describeEvidence renders the evidence of a finding.
func describeEvidence(evidence []domain.Evidence) string {
	parts := make([]string, 0, len(evidence))
	for _, item := range evidence {
		if item.Detail != "" {
			parts = append(parts, fmt.Sprintf("%s %s (%s)", item.Kind, item.Ref, item.Detail))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %s", item.Kind, item.Ref))
	}
	return strings.Join(parts, ", ")
}

// describeSupport renders the support level of a finding.
func describeSupport(support domain.Support) string {
	level := ui.SupportLabel(support.Level)
	if support.Method == "" {
		return level
	}
	return fmt.Sprintf("%s (%s)", level, support.Method)
}

// describeCounts renders a map of counters with a stable order.
func describeCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s %d", key, counts[key]))
	}
	return strings.Join(parts, ", ")
}

// writeRuleCatalog writes the rule catalog with its support matrix.
func writeRuleCatalog(w io.Writer, r *ui.Renderer, rules []domain.Rule) {
	languages := ruleLanguages()

	fmt.Fprintln(w, r.Title("Rule catalog"))
	fmt.Fprintln(w)

	headers := append([]string{"RULE", "SEVERITY"}, languages...)
	table := ui.NewTable(headers...)
	for _, rule := range rules {
		cells := []string{rule.ID, string(rule.Severity)}
		for _, language := range languages {
			cells = append(cells, ui.SupportLabel(rule.Languages[language]))
		}
		table.Append(cells...)
	}
	table.Render(w, r)

	for _, rule := range rules {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s %s\n", r.Bold(rule.ID), r.Dim("v"+rule.Version))
		fmt.Fprintf(w, "  %s\n", rule.Description)
		if rule.Method != "" {
			fmt.Fprintf(w, "  %s %s\n", r.Dim("method:"), rule.Method)
		}
		if rule.EvidenceKind != "" {
			fmt.Fprintf(w, "  %s %s\n", r.Dim("evidence:"), rule.EvidenceKind)
		}
		for _, limitation := range rule.Limitations {
			fmt.Fprintf(w, "  %s %s\n", r.Dim("limitation:"), limitation)
		}
	}
}
