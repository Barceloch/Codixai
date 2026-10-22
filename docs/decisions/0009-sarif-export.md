# 0009 — The SARIF export

- Status: **Accepted**
- Date: 2026-10-22

## Context

The specification lists the SARIF export as a deliverable of Phase 3: a
document that a compatible platform can import, with consistent rules and
locations.

SARIF is a format with a versioned schema of its own. The decision has to
fix the version, the mapping of the concepts of Codixai onto the concepts
of SARIF, and the place where the stable fingerprint of the finding lives,
so that a platform that supports the suppression of a known finding can use
it.

The environment of the project has no network access, so the official JSON
schema of SARIF cannot be fetched and the document cannot be validated
against it. That limitation is stated instead of hidden.

## Decision

### The version and the shape

The export is **SARIF 2.1.0**. The document holds one run, whose tool is
the driver `codixai`, with the version of the tool and the rules of the
catalog.

```json
{
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "codixai",
          "version": "0.1.0",
          "informationUri": "https://github.com/Barceloch/Codixai",
          "rules": [
            {
              "id": "CDX-SUP-001",
              "name": "NewAnalyzerSuppression",
              "shortDescription": { "text": "..." },
              "fullDescription": { "text": "..." },
              "defaultConfiguration": { "level": "warning" },
              "properties": {
                "support": { "level": "syntactic", "method": "...", "limitations": ["..."] }
              }
            }
          ]
        }
      },
      "results": [
        {
          "ruleId": "CDX-SUP-001",
          "ruleIndex": 0,
          "level": "warning",
          "message": { "text": "..." },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "internal/payment/service.go" },
                "region": { "startLine": 41, "endLine": 41 }
              }
            }
          ],
          "partialFingerprints": { "codixaiFingerprint/v1": "3f2a91c0d4b7e815" },
          "properties": {
            "epistemic_status": "observed",
            "support_level": "syntactic",
            "rule_version": "1"
          }
        }
      ],
      "properties": { "suppressedByBaseline": 3 }
    }
  ]
}
```

### The mapping

| Codixai | SARIF | Note |
| -------- | ----- | ---- |
| `schema_version` of the audit | `version` of the document | Always `2.1.0`. |
| Tool name and version | `tool.driver` | The name is `codixai`. |
| Rule identifier | `rules[].id` and `results[].ruleId` | The stable `CDX-<AREA>-<NNN>` identifier. |
| Description of the rule | `rules[].shortDescription` and `fullDescription` | The text of the catalog, in English. |
| Default severity of the rule | `rules[].defaultConfiguration.level` | `error`, `warning` or `note`. |
| Severity of a finding | `results[].level` | `error` stays `error`, `warning` stays `warning`, `info` becomes `note`. |
| Message of a finding | `results[].message.text` | The same text that the text output shows. |
| Path and line range | `results[].locations[].physicalLocation` | The URI is the repository relative path, with forward slashes. |
| `epistemic_status` | `results[].properties.epistemic_status` | A SARIF property, because SARIF has no equivalent. |
| Support level and limitations | `results[].properties.support_level` and `rules[].properties.support` | The honest depth of the analysis travels with the result. |
| Stable fingerprint | `results[].partialFingerprints["codixaiFingerprint/v1"]` | The fingerprint of the decision 0008. SARIF defines `partialFingerprints` for exactly this purpose. |
| Findings known by the baseline | `runs[].properties.suppressedByBaseline` | The count. The known findings are not results: they are not new. |

The document is deterministic: the rules are sorted by their identifier and
the results are sorted by path, line and rule, exactly like the findings of
the audit.

### The command

```bash
codixai report --format sarif --base main
codixai report --format json --base main
```

- The command holds the same origins as the audit: `--base`, `--staged` and
  `--worktree`, one of them required.
- `--format sarif` writes the SARIF document and `--format json` writes the
  document of the audit. The two documents describe the same run.
- The document goes to the standard output and nothing else does. The
  diagnostics and the count of the findings known by the baseline go to the
  standard error.
- The exit code is the exit code of the audit: `3` when a new finding
  reaches the threshold.

### The validation

The document is validated by the tests of the project against the structure
that the specification of SARIF 2.1.0 requires: the version, the run, the
driver, the rules, the results, the locations and the fingerprints. The
document is **not** validated against the official JSON schema of SARIF,
because the environment has no network access to fetch it. The limitation is
declared in the documentation of the export.

## Consequences

Positive:

- A platform that imports SARIF receives the rules, the locations, the
  severity and the honest epistemic status of every new finding.
- The fingerprint travels in `partialFingerprints`, so a platform that
  supports the suppression of a known finding can use the same identity
  that the baseline uses.
- The document is deterministic and it is the same run that the audit
  reports.

Negative:

- The document is not validated against the official schema in this
  environment. The validation has to be added when the network is
  available, or in a pipeline that can fetch the schema.
- The epistemic status and the support level travel as properties, because
  SARIF has no field for them. A consumer that does not read the properties
  loses that information.

## Alternatives considered

**Export SARIF 2.0.0.**
Rejected. It is the older version and it does not carry the properties that
the findings need.

**Write a custom JSON document instead.**
Rejected. The whole point of the SARIF export is that a platform can import
it without a custom adapter.

**Put the fingerprint in the message of the result.**
Rejected. The message is for the human. SARIF has a field designed for the
identity of a result, and it is `partialFingerprints`.

**Include the findings known by the baseline as results.**
Rejected. A platform that fails on the results would fail on the debt that
the baseline exists to hide. The count travels as a property of the run.
