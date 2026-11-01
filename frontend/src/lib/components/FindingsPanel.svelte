<script lang="ts">
  import type { domain } from '../wailsjs/go/models'
  import { i18n } from '../i18n.svelte'
  import {
    findingKey,
    locationText,
    uniqueValues,
    type Filters,
    type SortDirection,
    type SortKey,
  } from '../findings'

  let {
    findings,
    allFindings,
    filters = $bindable(),
    selectedKey,
    onselect,
  }: {
    findings: domain.Finding[]
    allFindings: domain.Finding[]
    filters: Filters
    selectedKey: string
    onselect: (finding: domain.Finding, index: number) => void
  } = $props()

  const rules = $derived(
    uniqueValues(allFindings, (finding) => finding.rule_id),
  )
  const languages = $derived(
    uniqueValues(allFindings, (finding) => finding.language),
  )

  const sortKeys: SortKey[] = [
    'path',
    'severity',
    'rule',
    'epistemic',
    'language',
  ]
  const directions: SortDirection[] = ['asc', 'desc']

  function clear(): void {
    filters.rule = ''
    filters.severity = ''
    filters.epistemic = ''
    filters.language = ''
    filters.path = ''
    filters.sort = 'path'
    filters.direction = 'asc'
  }

  function select(finding: domain.Finding, index: number): void {
    onselect(finding, index)
  }

  function severityClass(severity: string): string {
    switch (severity) {
      case 'error': return 'chip chip-error'
      case 'warning': return 'chip chip-warning'
      default: return 'chip chip-info'
    }
  }
</script>

<section class="panel panel-pad" aria-labelledby="findings-title">
  <div class="panel-head">
    <h2 id="findings-title" class="panel-title">
      {i18n.t('findings.title')}
      <span class="chip ml-2">{i18n.t('findings.count', { count: findings.length })}</span>
    </h2>
    <button type="button" class="btn btn-quiet btn-sm" onclick={clear}>
      {i18n.t('filters.clear')}
    </button>
  </div>

  <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3" aria-label={i18n.t('filters.title')}>
    <div class="field">
      <label class="field-label" for="filter-rule">{i18n.t('filters.rule')}</label>
      <div class="select-wrap">
        <select id="filter-rule" class="select select-sm" bind:value={filters.rule}>
          <option value="">{i18n.t('filters.all')}</option>
          {#each rules as rule (rule)}
            <option value={rule}>{rule}</option>
          {/each}
        </select>
        <svg class="select-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
        </svg>
      </div>
    </div>

    <div class="field">
      <label class="field-label" for="filter-severity">{i18n.t('filters.severity')}</label>
      <div class="select-wrap">
        <select id="filter-severity" class="select select-sm" bind:value={filters.severity}>
          <option value="">{i18n.t('filters.all')}</option>
          {#each ['info', 'warning', 'error'] as severity (severity)}
            <option value={severity}>{i18n.t(`severity.${severity}`)}</option>
          {/each}
        </select>
        <svg class="select-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
        </svg>
      </div>
    </div>

    <div class="field">
      <label class="field-label" for="filter-epistemic">{i18n.t('filters.epistemic')}</label>
      <div class="select-wrap">
        <select id="filter-epistemic" class="select select-sm" bind:value={filters.epistemic}>
          <option value="">{i18n.t('filters.all')}</option>
          {#each ['observed', 'inferred', 'unknown'] as status (status)}
            <option value={status}>{i18n.t(`epistemic.${status}`)}</option>
          {/each}
        </select>
        <svg class="select-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
        </svg>
      </div>
    </div>

    <div class="field">
      <label class="field-label" for="filter-language">{i18n.t('filters.language')}</label>
      <div class="select-wrap">
        <select id="filter-language" class="select select-sm" bind:value={filters.language}>
          <option value="">{i18n.t('filters.all')}</option>
          {#each languages as language (language)}
            <option value={language}>{language}</option>
          {/each}
        </select>
        <svg class="select-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
        </svg>
      </div>
    </div>

    <div class="field">
      <label class="field-label" for="filter-path">{i18n.t('filters.path')}</label>
      <input
        id="filter-path"
        class="input input-sm"
        type="text"
        spellcheck="false"
        placeholder={i18n.t('filters.pathPlaceholder')}
        bind:value={filters.path}
      />
    </div>

    <div class="field">
      <span class="field-label">{i18n.t('filters.sort')}</span>
      <div class="flex gap-2">
        <div class="select-wrap flex-1">
          <select class="select select-sm" bind:value={filters.sort}>
            {#each sortKeys as key (key)}
              <option value={key}>{i18n.t(`filters.sort${key.charAt(0).toUpperCase()}${key.slice(1)}`)}</option>
            {/each}
          </select>
          <svg class="select-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
          </svg>
        </div>
        <div class="select-wrap flex-1">
          <select class="select select-sm" bind:value={filters.direction} aria-label={i18n.t('filters.sort')}>
            {#each directions as direction (direction)}
              <option value={direction}>
                {direction === 'asc' ? i18n.t('filters.ascending') : i18n.t('filters.descending')}
              </option>
            {/each}
          </select>
          <svg class="select-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
          </svg>
        </div>
      </div>
    </div>
  </div>

  {#if findings.length === 0}
    <div class="empty fade-in">
      <svg class="empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        {#if allFindings.length > 0}
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3.055 11H5a2 2 0 0 1 2 2v1a2 2 0 0 0 2 2 2 2 0 0 1 2 2v2.945M8 3.935V5.5A2.5 2.5 0 0 0 10.5 8h.5a2 2 0 0 1 2 2 2 2 0 1 0 4 0 2 2 0 0 1 2-2h1.064M15 20.488V18a2 2 0 0 1 2-2h3.064M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0z"></path>
        {:else}
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5.586a1 1 0 0 1 .707.293l5.414 5.414a1 1 0 0 1 .293.707V19a2 2 0 0 1-2 2z"></path>
        {/if}
      </svg>
      {#if allFindings.length > 0}
        <p class="empty-title">{i18n.t('findings.noResults')}</p>
        <p class="empty-text">{i18n.t('findings.noResultsHint')}</p>
      {:else}
        <p class="empty-title">{i18n.t('findings.empty')}</p>
        <p class="empty-text">{i18n.t('findings.emptyHint')}</p>
      {/if}
    </div>
  {:else}
    <div class="scroll-x mt-4">
      <table class="data-table">
        <thead>
          <tr>
            <th>{i18n.t('table.severity')}</th>
            <th>{i18n.t('table.rule')}</th>
            <th>{i18n.t('table.epistemic')}</th>
            <th>{i18n.t('table.language')}</th>
            <th>{i18n.t('table.location')}</th>
            <th>{i18n.t('table.message')}</th>
          </tr>
        </thead>
        <tbody>
          {#each findings as finding, index (findingKey(finding, index))}
            <tr
              class:is-selected={selectedKey === findingKey(finding, index)}
              tabindex="0"
              role="button"
              aria-pressed={selectedKey === findingKey(finding, index)}
              onclick={() => select(finding, index)}
              onkeydown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault()
                  select(finding, index)
                }
              }}
            >
              <td>
                <span class={severityClass(finding.severity)}>
                  {i18n.t(`severity.${finding.severity}`)}
                </span>
              </td>
              <td class="cell-strong whitespace-nowrap">{finding.rule_id}</td>
              <td class="cell-muted whitespace-nowrap">
                {i18n.t(`epistemic.${finding.epistemic_status}`)}
              </td>
              <td class="cell-muted whitespace-nowrap">{finding.language || '—'}</td>
              <td class="cell-mono break-anywhere">
                {locationText(finding.location)}
              </td>
              <td class="cell-muted">
                <span class="clamp-2">{finding.message}</span>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</section>
