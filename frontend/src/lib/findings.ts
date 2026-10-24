// The pure helpers of the findings list.
//
// The module holds no state and no presentation: it filters, sorts and
// locates the evidence of a finding in the change set. Keeping it separate
// makes the rules of the list readable and testable by inspection.

import type { domain } from './wailsjs/go/models'

export type SortKey = 'path' | 'severity' | 'rule' | 'epistemic' | 'language'
export type SortDirection = 'asc' | 'desc'

export interface Filters {
  rule: string
  severity: string
  epistemic: string
  language: string
  path: string
  sort: SortKey
  direction: SortDirection
}

export const defaultFilters: Filters = {
  rule: '',
  severity: '',
  epistemic: '',
  language: '',
  path: '',
  sort: 'path',
  direction: 'asc',
}

const severityRank: Record<string, number> = {
  info: 0,
  warning: 1,
  error: 2,
}

// locationText renders the path and the line of a location.
export function locationText(location?: domain.Location): string {
  if (!location || !location.path) {
    return ''
  }
  if (!location.start_line || location.start_line <= 0) {
    return location.path
  }
  if (location.end_line && location.end_line !== location.start_line) {
    return `${location.path}:${location.start_line}-${location.end_line}`
  }
  return `${location.path}:${location.start_line}`
}

// findingKey returns a stable key for a finding of the list.
export function findingKey(finding: domain.Finding, index: number): string {
  return [
    finding.rule_id,
    finding.location?.path ?? '',
    finding.location?.start_line ?? 0,
    index,
  ].join(':')
}

// uniqueValues returns the sorted unique values of a field of the findings.
export function uniqueValues(
  findings: domain.Finding[],
  select: (finding: domain.Finding) => string | undefined,
): string[] {
  const values = new Set<string>()
  for (const finding of findings) {
    const value = select(finding)
    if (value) {
      values.add(value)
    }
  }
  return [...values].sort((left, right) => left.localeCompare(right))
}

// filterAndSort applies the filters and the order of the list.
export function filterAndSort(
  findings: domain.Finding[],
  filters: Filters,
): domain.Finding[] {
  const path = filters.path.trim().toLowerCase()
  const filtered = findings.filter((finding) => {
    if (filters.rule && finding.rule_id !== filters.rule) {
      return false
    }
    if (filters.severity && finding.severity !== filters.severity) {
      return false
    }
    if (filters.epistemic && finding.epistemic_status !== filters.epistemic) {
      return false
    }
    if (filters.language && finding.language !== filters.language) {
      return false
    }
    if (path && !(finding.location?.path ?? '').toLowerCase().includes(path)) {
      return false
    }
    return true
  })

  const direction = filters.direction === 'desc' ? -1 : 1
  return filtered.sort((left, right) => {
    const order = compare(left, right, filters.sort)
    if (order !== 0) {
      return order * direction
    }
    return locationText(left.location).localeCompare(
      locationText(right.location),
    )
  })
}

// compare orders two findings by the selected key.
function compare(
  left: domain.Finding,
  right: domain.Finding,
  key: SortKey,
): number {
  switch (key) {
    case 'severity':
      return (
        (severityRank[left.severity] ?? -1) - (severityRank[right.severity] ?? -1)
      )
    case 'rule':
      return left.rule_id.localeCompare(right.rule_id)
    case 'epistemic':
      return left.epistemic_status.localeCompare(right.epistemic_status)
    case 'language':
      return (left.language ?? '').localeCompare(right.language ?? '')
    case 'path':
    default:
      return locationText(left.location).localeCompare(
        locationText(right.location),
      )
  }
}

// hunkFor returns the hunk of the change set that the evidence of a finding
// references.
export function hunkFor(
  finding: domain.Finding,
  changes?: domain.ChangeSet | null,
): domain.Hunk | null {
  if (!changes || !finding.evidence) {
    return null
  }
  const evidence = finding.evidence.find((item) => item.kind === 'diff_hunk')
  if (!evidence) {
    return null
  }
  const rest = evidence.ref.startsWith('hunk:')
    ? evidence.ref.slice('hunk:'.length)
    : evidence.ref
  const separator = rest.lastIndexOf(':')
  if (separator < 0) {
    return null
  }
  const path = rest.slice(0, separator)
  const index = Number(rest.slice(separator + 1))
  const file = changes.files?.find((candidate) => candidate.path === path)
  return file?.hunks?.find((candidate) => candidate.index === index) ?? null
}

// hunkLinePrefix returns the prefix of a line of a hunk.
export function hunkLinePrefix(kind: string): string {
  switch (kind) {
    case 'added':
      return '+'
    case 'removed':
      return '-'
    default:
      return ' '
  }
}
