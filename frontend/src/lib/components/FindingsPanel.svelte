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
        <span class="badge badge-glass text-xs text-primary-color/70">
          {i18n.t('findings.count', { count: findings.length })}
        </span>
      </h2>
      <button type="button" class="btn-glass btn-xs" onclick={clear}>
        {i18n.t('filters.clear')}
      </button>
    </div>

    <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-6" aria-label={i18n.t('filters.title')}>
      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium text-base-content/60 text-primary-color/60">{i18n.t('filters.rule')}</span>
        <select class="select select-bordered select-sm" bind:value={filters.rule}>
          <option value="">{i18n.t('filters.all')}</option>
          {#each rules as rule}
            <option value={rule}>{rule}</option>
          {/each}
        </select>
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium text-base-content/60 text-primary-color/60">{i18n.t('filters.severity')}</span>
        <select class="select select-bordered select-sm" bind:value={filters.severity}>
          <option value="">{i18n.t('filters.all')}</option>
          {#each ['info', 'warning', 'error'] as severity}
            <option value={severity}>{i18n.t(`severity.${severity}`)}</option>
          {/each}
        </select>
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium text-base-content/60 text-primary-color/60">{i18n.t('filters.epistemic')}</span>
        <select class="select select-bordered select-sm" bind:value={filters.epistemic}>
          <option value="">{i18n.t('filters.all')}</option>
          {#each ['observed', 'inferred', 'unknown'] as status}
            <option value={status}>{i18n.t(`epistemic.${status}`)}</option>
          {/each}
        </select>
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium text-base-content/60 text-primary-color/60">{i18n.t('filters.language')}</span>
        <select class="select select-bordered select-sm" bind:value={filters.language}>
          <option value="">{i18n.t('filters.all')}</option>
          {#each languages as language}
            <option value={language}>{language}</option>
          {/each}
        </select>
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium text-base-content/60 text-primary-color/60">{i18n.t('filters.path')}</span>
        <input
          class="input input-bordered input-sm"
          type="text"
          spellcheck="false"
          placeholder={i18n.t('filters.pathPlaceholder')}
          bind:value={filters.path}
        />
      </label>

      <div class="flex flex-col gap-1">
        <span class="text-xs font-medium text-base-content/60 text-primary-color/60">{i18n.t('filters.sort')}</span>
        <div class="join">
          <select class="select select-bordered select-sm join-item" bind:value={filters.sort}>
            {#each sortKeys as key}
              <option value={key}>{i18n.t(`filters.sort${key.charAt(0).toUpperCase()}${key.slice(1)}`)}</option>
            {/each}
          </select>
          <select class="select select-bordered select-sm join-item" bind:value={filters.direction} aria-label={i18n.t('filters.sort')}>
            {#each directions as direction}
              <option value={direction}>
                {direction === 'asc' ? i18n.t('filters.ascending') : i18n.t('filters.descending')}
              </option>
            {/each}
          </select>
        </div>
      </div>
    </div>

    {#if findings.length === 0}
      <div class="rounded-box border border-dashed border-base-300 p-6 text-center text-base-content/60">
        {#if allFindings.length > 0}
          <p class="font-medium">{i18n.t('findings.noResults')}</p>
          <p class="text-sm text-base-content/70 text-primary-color/60">{i18n.t('findings.noResultsHint')}</p>
        {:else}
          <p class="font-medium text-base-content/60">{i18n.t('findings.empty')}</p>
          <p class="text-sm text-base-content/70 text-primary-color/60">{i18n.t('findings.emptyHint')}</p>
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
                class="cursor-pointer hover:bg-base-200 {selectedKey === findingKey(finding, index) ? 'bg-base-200' : ''}"
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
                <td class="whitespace-nowrap text-xs text-primary-color/70">
                  {i18n.t(`epistemic.${finding.epistemic_status}`)}
                </td>
                <td class="whitespace-nowrap text-xs text-primary-color/70">{finding.language || '—'}</td>
                <td class="break-anywhere font-mono text-xs text-primary-color/70">
                  {locationText(finding.location)}
                </td>
                <td class="font-mono max-w-xs text-primary-color/70">
                  <span class="line-clamp-2 text-primary-color/80">{finding.message}</span>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</section>