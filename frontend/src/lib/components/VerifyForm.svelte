<script lang="ts">
  import { onMount } from 'svelte'
  import { i18n } from '../i18n.svelte'
  import {
    CancelAudit,
    ChooseFile,
    ChooseRepository,
    Export,
    Root,
    VerifyAsync,
    isBridgeAvailable,
    onVerifyEvent,
  } from '../bridge'
  import { ui } from '../wailsjs/go/models'
  import type { domain } from '../wailsjs/go/models'

  type StateFilter = '' | 'verified' | 'contradicted' | 'unresolved'
  type SortKey = 'id' | 'state'
  type SortDirection = 'asc' | 'desc'

  interface Row {
    key: string
    result: domain.ClaimResult
  }

  const states: Exclude<StateFilter, ''>[] = ['verified', 'contradicted', 'unresolved']

  let bridgeAvailable = $state(isBridgeAvailable())
  let root = $state('')
  let base = $state('')
  let claimsPath = $state('')
  let running = $state(false)
  let progress = $state<ui.VerifyProgress | null>(null)
  let view = $state<ui.VerifyView | null>(null)
  let error = $state('')
  let runID = $state('')
  let selectedKey = $state('')
  let filters = $state<{
    state: StateFilter
    path: string
    sort: SortKey
    direction: SortDirection
  }>({ state: '', path: '', sort: 'id', direction: 'asc' })

  const rows = $derived.by<Row[]>(() => {
    const results = view?.document?.results ?? []
    const indexed = results.map((result, index) => ({
      key: `${result.claim.id}:${index}`,
      result,
    }))
    const path = filters.path.trim().toLowerCase()
    const filtered = indexed.filter(({ result }) => {
      if (filters.state && result.status !== filters.state) return false
      if (path && !(result.claim.paths ?? []).some((item) => item.toLowerCase().includes(path))) return false
      return true
    })
    const direction = filters.direction === 'desc' ? -1 : 1
    return filtered.sort((left, right) => {
      const order = filters.sort === 'state'
          ? left.result.status.localeCompare(right.result.status)
          : left.result.claim.id.localeCompare(right.result.claim.id)
      return order * direction
    })
  })

  const selected = $derived(rows.find((row) => row.key === selectedKey) ?? null)

  const canRun = $derived(!running && root.trim().length > 0 && base.trim().length > 0 && claimsPath.trim().length > 0)

  function newRunID(): string {
    return `run-${crypto.randomUUID()}`
  }

  function describeError(cause: unknown): string {
    return (cause instanceof Error && cause.message) ? cause.message : (typeof cause === 'string' ? cause : i18n.t('verify.error.generic'))
  }

  function progressText(): string {
    switch (progress?.phase) {
      case 'validating': return i18n.t('verify.progress.validating')
      case 'collecting': return i18n.t('verify.progress.collecting')
      case 'done': return i18n.t('verify.progress.done')
      case 'cancelled': return i18n.t('verify.progress.cancelled')
      default: return i18n.t('verify.running')
    }
  }

  function progressPercent(): number {
    switch (progress?.phase) {
      case 'validating': return 25
      case 'collecting': return 65
      case 'done': return 100
      default: return 0
    }
  }

  function stateLabel(status: string): string {
    return i18n.t(`verify.state${status.charAt(0).toUpperCase()}${status.slice(1)}`)
  }

  function chipClass(status: string): string {
    switch (status) {
      case 'verified': return 'chip chip-success'
      case 'contradicted': return 'chip chip-error'
      case 'unresolved': return 'chip chip-warning'
      default: return 'chip'
    }
  }

  async function chooseRoot(): Promise<void> {
    try {
      const path = await ChooseRepository(i18n.t('verify.chooseRepositoryTitle'))
      if (path) root = path
    } catch (cause) { error = describeError(cause) }
  }

  async function chooseClaims(): Promise<void> {
    try {
      const path = await ChooseFile(i18n.t('verify.chooseClaimsTitle'), ['.json'])
      if (path) claimsPath = path
    } catch (cause) { error = describeError(cause) }
  }

  async function run(): Promise<void> {
    if (running || !canRun) return
    error = ''; progress = null; view = null; selectedKey = '';
    runID = newRunID()
    running = true
    try {
      await VerifyAsync(new ui.VerifyRequest({ root: root.trim(), base: base.trim(), claims_path: claimsPath.trim(), run_id: runID }), runID)
    } catch (cause) { running = false; error = describeError(cause) }
  }

  async function cancel(): Promise<void> {
    if (!runID) return
    try { await CancelAudit(runID) } catch (cause) { error = describeError(cause) }
  }

  async function exportReport(): Promise<void> {
    if (!view) return
    try {
      await Export(new ui.ExportRequest({ run_id: view.run_id, format: 'json', suggested_name: 'codixai-verify.json', title: i18n.t('export.saveTitle') }))
    } catch (cause) { error = describeError(cause) }
  }

  function selectRow(row: Row): void { selectedKey = row.key }

  function clearFilters(): void { filters = { state: '', path: '', sort: 'id', direction: 'asc' } }

  function applyEvent(raw: ui.VerifyEvent): void {
    const event = ui.VerifyEvent.createFrom(raw)
    if (!event || event.run_id !== runID) return
    if (event.kind === 'progress' && event.progress) progress = event.progress
    else if (event.kind === 'done' && event.view) { running = false; view = event.view }
    else if (event.kind === 'cancelled') { running = false; progress = null }
    else if (event.kind === 'error') { running = false; error = event.error || i18n.t('verify.error.generic') }
  }

  onMount(() => {
    bridgeAvailable = isBridgeAvailable()
    if (!bridgeAvailable) return
    const stopVerify = onVerifyEvent(applyEvent)
    Root().then(path => { if (path && !root) root = path })
    return stopVerify
  })
</script>

<section class="panel panel-pad fade-in" aria-labelledby="verify-title">
  <div class="mb-5">
    <h2 id="verify-title" class="panel-title">{i18n.t('verify.title')}</h2>
    <p class="panel-sub mt-1">{i18n.t('verify.idleHint')}</p>
  </div>

  <form class="grid gap-5 md:grid-cols-2" onsubmit={(e) => { e.preventDefault(); run() }} autocomplete="off">
    <div class="field md:col-span-2">
      <label class="field-label" for="verify-repository">{i18n.t('verify.repository')}</label>
      <div class="field-row">
        <input id="verify-repository" class="input input-mono" type="text" spellcheck="false" bind:value={root} disabled={running} placeholder={i18n.t('verify.repositoryPlaceholder')} />
        <button type="button" class="btn btn-ghost btn-sm" onclick={chooseRoot} disabled={running || !bridgeAvailable}>{i18n.t('verify.chooseRepository')}</button>
      </div>
    </div>

    <div class="field">
      <label class="field-label" for="verify-base">{i18n.t('verify.base')}</label>
      <input id="verify-base" class="input input-mono" type="text" spellcheck="false" bind:value={base} disabled={running} placeholder={i18n.t('verify.basePlaceholder')} />
    </div>

    <div class="field">
      <label class="field-label" for="verify-claims">{i18n.t('verify.claimsFile')}</label>
      <div class="field-row">
        <input id="verify-claims" class="input input-mono" type="text" spellcheck="false" bind:value={claimsPath} disabled={running} placeholder={i18n.t('verify.claimsPlaceholder')} />
        <button type="button" class="btn btn-ghost btn-sm" onclick={chooseClaims} disabled={running || !bridgeAvailable}>{i18n.t('verify.chooseClaims')}</button>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-3 md:col-span-2 mt-2">
      <button type="submit" class="btn btn-primary flex-1 sm:flex-none sm:min-w-[180px]" disabled={!canRun || running}>
        {running ? i18n.t('verify.running') : i18n.t('verify.run')}
      </button>
      {#if running}<button type="button" class="btn btn-ghost btn-sm" onclick={cancel}>{i18n.t('verify.cancel')}</button>{/if}
    </div>
  </form>

  {#if running}
    <div class="mt-5 flex flex-col gap-2">
      <div class="flex items-center gap-3">
        <span class="spinner"></span>
        <span class="text-sm font-medium text-text-primary">{progressText()}</span>
      </div>
      <div class="meter">
        <div class="meter-fill" style="width: {progressPercent()}%"></div>
      </div>
    </div>
  {/if}

  {#if error}
    <div class="callout callout-error mt-5" role="alert">
      <svg class="mt-0.5 h-4 w-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
      </svg>
      <div class="min-w-0 flex-1">
        <p class="font-medium">{i18n.t('verify.error.title')}</p>
        <p class="mt-1 text-sm">{error}</p>
      </div>
    </div>
  {/if}
</section>

{#if view}
  <section class="panel panel-pad fade-in" aria-labelledby="verify-results-title">
    <div class="panel-head">
      <h2 id="verify-results-title" class="panel-title">{i18n.t('verify.results')}</h2>
      <button type="button" class="btn btn-ghost btn-sm" onclick={exportReport}>{i18n.t('export.json')}</button>
    </div>

    <!-- Filters -->
    <div class="grid gap-3 sm:grid-cols-4">
      <div class="field">
        <label class="field-label" for="verify-state-filter">{i18n.t('verify.state')}</label>
        <div class="select-wrap">
          <select id="verify-state-filter" class="select select-sm" bind:value={filters.state}>
            <option value="">{i18n.t('filters.all')}</option>
            {#each states as state (state)}
              <option value={state}>{i18n.t(`verify.state${state.charAt(0).toUpperCase()}${state.slice(1)}`)}</option>
            {/each}
          </select>
          <svg class="select-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
          </svg>
        </div>
      </div>
      <div class="field sm:col-span-2">
        <label class="field-label" for="verify-path-filter">{i18n.t('filters.path')}</label>
        <input id="verify-path-filter" class="input input-sm" type="text" bind:value={filters.path} placeholder={i18n.t('filters.pathPlaceholder')} />
      </div>
      <div class="flex items-end">
        <button type="button" class="btn btn-quiet btn-sm w-full" onclick={clearFilters}>{i18n.t('filters.clear')}</button>
      </div>
    </div>

    {#if rows.length === 0}
      <div class="empty mt-4">
        <p class="empty-title">{i18n.t('verify.resultsEmpty')}</p>
      </div>
    {:else}
      <div class="scroll-x mt-4">
        <table class="data-table">
          <thead>
            <tr>
              <th>{i18n.t('verify.claimId')}</th>
              <th>{i18n.t('verify.state')}</th>
              <th>{i18n.t('verify.claim')}</th>
              <th>{i18n.t('verify.paths')}</th>
            </tr>
          </thead>
          <tbody>
            {#each rows as row (row.key)}
              <tr
                class:is-selected={selectedKey === row.key}
                tabindex="0"
                onclick={() => selectRow(row)}
                onkeydown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault()
                    selectRow(row)
                  }
                }}
              >
                <td class="cell-strong">{row.result.claim.id}</td>
                <td><span class={chipClass(row.result.status)}>{stateLabel(row.result.status)}</span></td>
                <td class="cell-muted">{row.result.claim.type}</td>
                <td class="cell-mono break-anywhere">{row.result.claim.paths?.join(', ') ?? '—'}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}

    {#if selected}
      <div class="mt-5 flex flex-col gap-4 border-t border-border pt-4">
        <h3 class="eyebrow">{i18n.t('verify.detail')}</h3>
        <div>
          <p class="eyebrow">{i18n.t('verify.claim')}</p>
          <p class="mt-1 text-sm text-text-primary">{selected.result.claim.type}</p>
        </div>
        <div>
          <p class="eyebrow">{i18n.t('verify.message')}</p>
          <p class="mt-1 break-anywhere text-sm text-text-primary">{selected.result.message}</p>
        </div>
      </div>
    {/if}
  </section>
{/if}
