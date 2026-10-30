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
</script>

<section class="section-glass card-glass glass-surface" aria-labelledby="findings-title">
  <div class="card-body gap-4">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h2 id="findings-title" class="card-title text-lg text-primary-color">
        {i18n.t('findings.title')}
        <span class="badge badge-glass text-xs text-primary-color/70 ml-2">
          {i18n.t('findings.count', { count: findings.length })}
        </span>
      </h2>
      <button type="button" class="btn-glass btn-xs" onclick={clear}>
        {i18n.t('filters.clear')}
      </button>
    </div>

    <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-6" aria-label={i18n.t('filters.title')}>
      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium text-text-secondary">{i18n.t('filters.rule')}</span>
        <select class="select select-bordered select-sm bg-surface/50" bind:value={filters.rule}>
          <option value="">{i18n.t('filters.all')}</option>
          {#each rules as rule (rule)}
            <option value={rule}>{rule}</option>
          {/each}
        </select>
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium text-text-secondary">{i18n.t('filters.severity')}</span>
        <select class="select select-bordered select-sm bg-surface/50" bind:value={filters.severity}>
          <option value="">{i18n.t('filters.all')}</option>
          {#each ['info', 'warning', 'error'] as severity (severity)}
            <option value={severity}>{i18n.t(`severity.${severity}`)}</option>
          {/each}
        </select>
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium text-text-secondary">{i18n.t('filters.epistemic')}</span>
        <select class="select select-bordered select-sm bg-surface/50" bind:value={filters.epistemic}>
          <option value="">{i18n.t('filters.all')}</option>
          {#each ['observed', 'inferred', 'unknown'] as status (status)}
            <option value={status}>{i18n.t(`epistemic.${status}`)}</option>
          {/each}
        </select>
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium text-text-secondary">{i18n.t('filters.language')}</span>
        <select class="select select-bordered select-sm bg-surface/50" bind:value={filters.language}>
          <option value="">{i18n.t('filters.all')}</option>
          {#each languages as language (language)}
            <option value={language}>{language}</option>
          {/each}
        </select>
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium text-text-secondary">{i18n.t('filters.path')}</span>
        <input
          class="input input-bordered input-sm bg-surface/50"
          type="text"
          spellcheck="false"
          placeholder={i18n.t('filters.pathPlaceholder')}
          bind:value={filters.path}
        />
      </label>

      <div class="flex flex-col gap-1">
        <span class="text-xs font-medium text-text-secondary">{i18n.t('filters.sort')}</span>
        <div class="join">
          <select class="select select-bordered select-sm join-item bg-surface/50" bind:value={filters.sort}>
            {#each sortKeys as key (key)}
              <option value={key}>{i18n.t(`filters.sort${key.charAt(0).toUpperCase()}${key.slice(1)}`)}</option>
            {/each}
          </select>
          <select class="select select-bordered select-sm join-item bg-surface/50" bind:value={filters.direction} aria-label={i18n.t('filters.sort')}>
            {#each directions as direction (direction)}
              <option value={direction}>
                {direction === 'asc' ? i18n.t('filters.ascending') : i18n.t('filters.descending')}
              </option>
            {/each}
          </select>
        </div>
      </div>
    </div>

    {#if findings.length === 0}
      <div class="empty-state animate-fade-in">
        <svg class="empty-state-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          {#if allFindings.length > 0}
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3.055 11H5a2 2 0 0 1 2 2v1a2 2 0 0 0 2 2 2 2 0 0 1 2 2v2.945M8 3.935V5.5A2.5 2.5 0 0 0 10.5 8h.5a2 2 0 0 1 2 2 2 2 0 1 0 4 0 2 2 0 0 1 2-2h1.064M15 20.488V18a2 2 0 0 1 2-2h3.064M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0z"></path>
          {:else}
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5.586a1 1 0 0 1 .707.293l5.414 5.414a1 1 0 0 1 .293.707V19a2 2 0 0 1-2 2z"></path>
          {/if}
        </svg>
        {#if allFindings.length > 0}
          <p class="empty-state-title">{i18n.t('findings.noResults')}</p>
          <p class="empty-state-description">{i18n.t('findings.noResultsHint')}</p>
        {:else}
          <p class="empty-state-title">{i18n.t('findings.empty')}</p>
          <p class="empty-state-description">{i18n.t('findings.emptyHint')}</p>
        {/if}
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="table-glass">
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
                class="cursor-pointer hover:bg-primary/5 {selectedKey === findingKey(finding, index) ? 'selected' : ''}"
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
                  <span class="badge badge-glass-sm text-primary-color">
                    {i18n.t(`severity.${finding.severity}`)}
                  </span>
                </td>
                <td class="font-mono text-xs whitespace-nowrap">{finding.rule_id}</td>
                <td class="whitespace-nowrap text-xs text-text-secondary">
                  {i18n.t(`epistemic.${finding.epistemic_status}`)}
                </td>
                <td class="whitespace-nowrap text-xs text-text-secondary">{finding.language || '—'}</td>
                <td class="break-anywhere font-mono text-xs text-text-secondary">
                  {locationText(finding.location)}
                </td>
                <td class="font-mono max-w-xs text-text-secondary">
                  <span class="line-clamp-2">{finding.message}</span>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</section>