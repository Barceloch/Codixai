package policies

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// FileName is the name of the configuration file of a repository.
	FileName = ".codixai.yml"
	// AltFileName is the alternative name accepted for the configuration file.
	AltFileName = ".codixai.yaml"
	// DefaultBaseline is the default path of the baseline file.
	DefaultBaseline = ".codixai/baseline.json"
)

// Fail on values: the severity that makes a command fail.
const (
	FailOnError   = "error"
	FailOnWarning = "warning"
	FailOnNone    = "none"
)

// Output formats.
const (
	FormatText = "text"
	FormatJSON = "json"
)

// Rule severities. The values match the severities of the domain package.
const (
	SeverityInfo    = "info"
	SeverityWarning = "warning"
	SeverityError   = "error"
)

// SupportedLanguages lists the languages that can be selected for the
// analysis.
var SupportedLanguages = []string{"go", "python", "php", "javascript", "typescript"}

// DefaultExclude lists the paths that are never analyzed deeply.
var DefaultExclude = []string{
	"vendor/**",
	"node_modules/**",
	"dist/**",
	"build/**",
	".git/**",
	"coverage/**",
}

// ProjectConfig holds the identity of the analyzed project.
type ProjectConfig struct {
	// Name is the human readable name of the project.
	Name string `json:"name,omitempty"`
}

// AnalysisConfig holds the options of the analysis.
type AnalysisConfig struct {
	// Languages are the languages to analyze. An empty list means that no
	// language specific rule runs.
	Languages []string `json:"languages,omitempty"`
	// Exclude holds the glob patterns of the paths that are not analyzed.
	Exclude []string `json:"exclude,omitempty"`
	// RespectGitignore tells whether the .gitignore rules of the repository
	// are honored.
	RespectGitignore bool `json:"respect_gitignore"`
}

// RuleConfig holds the configuration of a single rule.
type RuleConfig struct {
	// Enabled tells whether the rule runs. A nil value means that the rule
	// keeps the default of the engine, which is enabled.
	Enabled *bool `json:"enabled,omitempty"`
	// Severity overrides the severity declared by the rule.
	Severity string `json:"severity,omitempty"`
}

// IsEnabled reports whether the rule runs.
func (r RuleConfig) IsEnabled() bool {
	return r.Enabled == nil || *r.Enabled
}

// Suppression is an explicit, justified and scoped exception.
type Suppression struct {
	// Rule is the identifier of the suppressed rule.
	Rule string `json:"rule"`
	// Path is the glob pattern of the suppressed paths. An empty value
	// suppresses the rule everywhere.
	Path string `json:"path,omitempty"`
	// Reason explains why the finding is suppressed.
	Reason string `json:"reason"`
}

// AuditConfig holds the options of the audit.
type AuditConfig struct {
	// FailOn is the severity that makes the audit fail.
	FailOn string `json:"fail_on"`
	// Baseline is the path of the baseline file.
	Baseline string `json:"baseline"`
	// Rules holds the per rule overrides.
	Rules map[string]RuleConfig `json:"rules,omitempty"`
	// Suppressions holds the explicit suppressions.
	Suppressions []Suppression `json:"suppressions,omitempty"`
}

// OutputConfig holds the options of the report.
type OutputConfig struct {
	// Format is the format of the report: text or json.
	Format string `json:"format"`
}

// Config is the validated configuration of the analyzed repository.
type Config struct {
	// Version is the version of the configuration format.
	Version int `json:"version"`
	// Project holds the identity of the project.
	Project ProjectConfig `json:"project"`
	// Analysis holds the options of the analysis.
	Analysis AnalysisConfig `json:"analysis"`
	// Audit holds the options of the audit.
	Audit AuditConfig `json:"audit"`
	// Output holds the options of the report.
	Output OutputConfig `json:"output"`
	// Source is the path of the file the configuration was loaded from. It
	// is empty when the defaults are used.
	Source string `json:"source,omitempty"`

	// matchers holds the compiled exclude patterns.
	matchers []*globMatcher
}

// Defaults returns the configuration used when the repository has no
// configuration file.
func Defaults() *Config {
	config := &Config{
		Version: 1,
		Analysis: AnalysisConfig{
			Languages:        append([]string(nil), SupportedLanguages...),
			Exclude:          append([]string(nil), DefaultExclude...),
			RespectGitignore: true,
		},
		Audit: AuditConfig{
			FailOn:       FailOnError,
			Baseline:     DefaultBaseline,
			Rules:        map[string]RuleConfig{},
			Suppressions: []Suppression{},
		},
		Output: OutputConfig{Format: FormatText},
	}
	config.compile()
	return config
}

// Load reads the configuration of the repository located at root.
//
// A missing configuration file is not an error: the defaults are returned. A
// malformed file is an error whose message names the file and the offending
// line.
func Load(root string) (*Config, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("the repository root is required")
	}
	path := filepath.Join(root, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("cannot read %s: %w", path, err)
		}
		alternative := filepath.Join(root, AltFileName)
		data, err = os.ReadFile(alternative)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("cannot read %s: %w", alternative, err)
			}
			return Defaults(), nil
		}
		path = alternative
	}
	config, err := Parse(data, path)
	if err != nil {
		return nil, err
	}
	config.Source = path
	return config, nil
}

// Parse validates a configuration document. The path is only used to build
// the error messages.
func Parse(data []byte, path string) (*Config, error) {
	config := Defaults()
	document, err := parseYAML(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if document == nil || document.kind == yamlNull {
		return config, nil
	}
	if document.kind != yamlMapping {
		return nil, fmt.Errorf("%s: line %d: the configuration must be a mapping of keys", path, document.line)
	}
	loader := &configLoader{path: path}

	if err := loader.loadVersion(document, config); err != nil {
		return nil, err
	}
	if err := loader.loadProject(document, config); err != nil {
		return nil, err
	}
	if err := loader.loadAnalysis(document, config); err != nil {
		return nil, err
	}
	if err := loader.loadAudit(document, config); err != nil {
		return nil, err
	}
	if err := loader.loadOutput(document, config); err != nil {
		return nil, err
	}
	if err := config.compile(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return config, nil
}

// configLoader validates a document and fills a configuration.
type configLoader struct {
	path string
}

// errorf builds an error that names the file and the offending line.
func (l *configLoader) errorf(node *yamlNode, format string, args ...any) error {
	line := 0
	if node != nil {
		line = node.line
	}
	return fmt.Errorf("%s: line %d: %s", l.path, line, fmt.Sprintf(format, args...))
}

// loadVersion reads the version of the configuration format.
func (l *configLoader) loadVersion(document *yamlNode, config *Config) error {
	node := document.get("version")
	if node == nil || node.kind == yamlNull {
		return nil
	}
	if node.kind != yamlScalar {
		return l.errorf(node, "version must be a number")
	}
	version, err := strconv.Atoi(strings.TrimSpace(node.value))
	if err != nil {
		return l.errorf(node, "version must be a number, got %q", node.value)
	}
	if version != 1 {
		return l.errorf(node, "unsupported configuration version %d, the supported version is 1", version)
	}
	config.Version = version
	return nil
}

// loadProject reads the identity of the project.
func (l *configLoader) loadProject(document *yamlNode, config *Config) error {
	node := document.get("project")
	if node == nil || node.kind == yamlNull {
		return nil
	}
	if node.kind != yamlMapping {
		return l.errorf(node, "project must be a mapping of keys")
	}
	name := node.get("name")
	if name == nil || name.kind == yamlNull {
		return nil
	}
	if name.kind != yamlScalar {
		return l.errorf(name, "project.name must be a text value")
	}
	config.Project.Name = strings.TrimSpace(name.value)
	return nil
}

// loadAnalysis reads the options of the analysis.
func (l *configLoader) loadAnalysis(document *yamlNode, config *Config) error {
	node := document.get("analysis")
	if node == nil || node.kind == yamlNull {
		return nil
	}
	if node.kind != yamlMapping {
		return l.errorf(node, "analysis must be a mapping of keys")
	}

	if languages := node.get("languages"); languages != nil && languages.kind != yamlNull {
		values, err := l.stringList(languages, "analysis.languages")
		if err != nil {
			return err
		}
		for _, language := range values {
			if !isSupportedLanguage(language) {
				return l.errorf(languages, "unsupported language %q, supported languages are %s", language, strings.Join(SupportedLanguages, ", "))
			}
		}
		config.Analysis.Languages = values
	}

	if exclude := node.get("exclude"); exclude != nil && exclude.kind != yamlNull {
		values, err := l.stringList(exclude, "analysis.exclude")
		if err != nil {
			return err
		}
		for _, pattern := range values {
			if _, err := newGlobMatcher(pattern); err != nil {
				return l.errorf(exclude, "invalid exclude pattern %q: %v", pattern, err)
			}
		}
		config.Analysis.Exclude = values
	}

	if respect := node.get("respect_gitignore"); respect != nil && respect.kind != yamlNull {
		if respect.kind != yamlScalar {
			return l.errorf(respect, "analysis.respect_gitignore must be a boolean")
		}
		value, ok := parseBool(respect.value)
		if !ok {
			return l.errorf(respect, "analysis.respect_gitignore must be a boolean, got %q", respect.value)
		}
		config.Analysis.RespectGitignore = value
	}
	return nil
}

// loadAudit reads the options of the audit.
func (l *configLoader) loadAudit(document *yamlNode, config *Config) error {
	node := document.get("audit")
	if node == nil || node.kind == yamlNull {
		return nil
	}
	if node.kind != yamlMapping {
		return l.errorf(node, "audit must be a mapping of keys")
	}

	if failOn := node.get("fail_on"); failOn != nil && failOn.kind != yamlNull {
		if failOn.kind != yamlScalar {
			return l.errorf(failOn, "audit.fail_on must be a text value")
		}
		value := strings.ToLower(strings.TrimSpace(failOn.value))
		switch value {
		case FailOnError, FailOnWarning, FailOnNone:
			config.Audit.FailOn = value
		default:
			return l.errorf(failOn, "invalid audit.fail_on %q, expected one of error, warning, none", failOn.value)
		}
	}

	if baseline := node.get("baseline"); baseline != nil && baseline.kind != yamlNull {
		if baseline.kind != yamlScalar {
			return l.errorf(baseline, "audit.baseline must be a text value")
		}
		value := strings.TrimSpace(baseline.value)
		if value == "" {
			return l.errorf(baseline, "audit.baseline must not be empty")
		}
		config.Audit.Baseline = value
	}

	if rules := node.get("rules"); rules != nil && rules.kind != yamlNull {
		if rules.kind != yamlMapping {
			return l.errorf(rules, "audit.rules must be a mapping of rule identifiers")
		}
		for _, key := range rules.keys {
			rule := rules.items[key]
			if rule.kind != yamlMapping {
				return l.errorf(rule, "the configuration of the rule %q must be a mapping of keys", key)
			}
			override := RuleConfig{}
			if enabled := rule.get("enabled"); enabled != nil && enabled.kind != yamlNull {
				if enabled.kind != yamlScalar {
					return l.errorf(enabled, "audit.rules.%s.enabled must be a boolean", key)
				}
				value, ok := parseBool(enabled.value)
				if !ok {
					return l.errorf(enabled, "audit.rules.%s.enabled must be a boolean, got %q", key, enabled.value)
				}
				override.Enabled = &value
			}
			if severity := rule.get("severity"); severity != nil && severity.kind != yamlNull {
				if severity.kind != yamlScalar {
					return l.errorf(severity, "audit.rules.%s.severity must be a text value", key)
				}
				value := strings.ToLower(strings.TrimSpace(severity.value))
				switch value {
				case SeverityError, SeverityWarning, SeverityInfo:
					override.Severity = value
				default:
					return l.errorf(severity, "invalid severity %q for the rule %q, expected one of error, warning, info", severity.value, key)
				}
			}
			config.Audit.Rules[key] = override
		}
	}

	if suppressions := node.get("suppressions"); suppressions != nil && suppressions.kind != yamlNull {
		if suppressions.kind != yamlSequence {
			return l.errorf(suppressions, "audit.suppressions must be a list")
		}
		values := make([]Suppression, 0, len(suppressions.list))
		for _, item := range suppressions.list {
			if item.kind != yamlMapping {
				return l.errorf(item, "every entry of audit.suppressions must be a mapping with the rule, path and reason keys")
			}
			rule := item.get("rule")
			if rule == nil || rule.kind != yamlScalar || strings.TrimSpace(rule.value) == "" {
				return l.errorf(item, "every entry of audit.suppressions must declare a rule")
			}
			reason := item.get("reason")
			if reason == nil || reason.kind != yamlScalar || strings.TrimSpace(reason.value) == "" {
				return l.errorf(item, "the suppression of the rule %q must declare a reason", rule.value)
			}
			suppression := Suppression{
				Rule:   strings.TrimSpace(rule.value),
				Reason: strings.TrimSpace(reason.value),
			}
			if path := item.get("path"); path != nil && path.kind != yamlNull {
				if path.kind != yamlScalar {
					return l.errorf(path, "the path of a suppression must be a text value")
				}
				value := strings.TrimSpace(path.value)
				if value != "" {
					if _, err := newGlobMatcher(value); err != nil {
						return l.errorf(path, "invalid path pattern %q: %v", value, err)
					}
				}
				suppression.Path = value
			}
			values = append(values, suppression)
		}
		config.Audit.Suppressions = values
	}
	return nil
}

// loadOutput reads the options of the report.
func (l *configLoader) loadOutput(document *yamlNode, config *Config) error {
	node := document.get("output")
	if node == nil || node.kind == yamlNull {
		return nil
	}
	if node.kind != yamlMapping {
		return l.errorf(node, "output must be a mapping of keys")
	}
	format := node.get("format")
	if format == nil || format.kind == yamlNull {
		return nil
	}
	if format.kind != yamlScalar {
		return l.errorf(format, "output.format must be a text value")
	}
	value := strings.ToLower(strings.TrimSpace(format.value))
	switch value {
	case FormatText, FormatJSON:
		config.Output.Format = value
	default:
		return l.errorf(format, "invalid output.format %q, expected one of text, json", format.value)
	}
	return nil
}

// stringList reads a list of text values.
func (l *configLoader) stringList(node *yamlNode, field string) ([]string, error) {
	if node.kind != yamlSequence {
		return nil, l.errorf(node, "%s must be a list", field)
	}
	values := make([]string, 0, len(node.list))
	for _, item := range node.list {
		if item.kind != yamlScalar {
			return nil, l.errorf(item, "%s must be a list of text values", field)
		}
		value := strings.TrimSpace(item.value)
		if value == "" {
			return nil, l.errorf(item, "%s must not contain empty values", field)
		}
		values = append(values, value)
	}
	return values, nil
}

// compile builds the compiled exclude patterns of the configuration.
func (c *Config) compile() error {
	matchers := make([]*globMatcher, 0, len(c.Analysis.Exclude))
	for _, pattern := range c.Analysis.Exclude {
		matcher, err := newGlobMatcher(pattern)
		if err != nil {
			return fmt.Errorf("invalid exclude pattern %q: %w", pattern, err)
		}
		matchers = append(matchers, matcher)
	}
	c.matchers = matchers
	return nil
}

// Excluded reports whether a repository relative path is excluded from the
// analysis by the configured patterns.
func (c *Config) Excluded(path string) bool {
	for _, matcher := range c.matchers {
		if matcher.matches(path) {
			return true
		}
	}
	return false
}

// Rule returns the configuration of a rule and whether the rule was
// configured.
func (c *Config) Rule(id string) (RuleConfig, bool) {
	rule, ok := c.Audit.Rules[id]
	return rule, ok
}

// IsSuppressed reports whether a finding of a rule in a path is suppressed.
// A suppression without a path suppresses the rule everywhere.
func (c *Config) IsSuppressed(rule, path string) bool {
	for _, suppression := range c.Audit.Suppressions {
		if suppression.Rule != rule {
			continue
		}
		if suppression.Path == "" {
			return true
		}
		matched, err := MatchGlob(suppression.Path, path)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// isSupportedLanguage reports whether a language can be analyzed.
func isSupportedLanguage(language string) bool {
	for _, supported := range SupportedLanguages {
		if supported == language {
			return true
		}
	}
	return false
}

// parseBool parses the boolean scalars of the configuration.
func parseBool(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "yes", "on", "1":
		return true, true
	case "false", "no", "off", "0":
		return false, true
	}
	return false, false
}
