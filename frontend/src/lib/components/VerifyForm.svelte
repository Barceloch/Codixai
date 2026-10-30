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

  function stateLabel(status: string): string {
    return i18n.t(`verify.state${status.charAt(0).toUpperCase()}${status.slice(1)}`)
  }

  function badgeClass(status: string): string {
    switch (status) {
      case 'verified': return 'badge-success-glass'
      case 'contradicted': return 'badge-error-glass'
      case 'unresolved': return 'badge-warning-glass'
      default: return 'badge-glass'
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

<section class="section-glass glass-surface animate-fade-in" aria-labelledby="verify-title">
  <div class="card-glass card-glass-elevated">
    <div class="card-body gap-6">
      <div>
        <h2 id="verify-title" class="card-title text-lg text-primary-color">{i18n.t('verify.title')}</h2>
        <p class="text-sm text-text-secondary mt-1">{i18n.t('verify.idleHint')}</p>
      </div>

      <form class="grid gap-4 md:grid-cols-2" onsubmit={(e) => { e.preventDefault(); run() }} autocomplete="off">
        <div class="grid gap-2 md:col-span-2">
          <label class="label text-xs font-medium text-text-secondary" for="verify-repository">{i18n.t('verify.repository')}</label>
          <div class="repository-input">
            <input id="verify-repository" class="input input-bordered w-full bg-surface/50" type="text" bind:value={root} disabled={running} placeholder={i18n.t('verify.repositoryPlaceholder')} />
            <button type="button" class="btn-glass btn-sm" onclick={chooseRoot} disabled={running || !bridgeAvailable}>{i18n.t('verify.chooseRepository')}</button>
          </div>
        </div>

        <div class="flex flex-col gap-2">
          <label class="label text-xs font-medium text-text-secondary" for="verify-base">{i18n.t('verify.base')}</label>
          <input id="verify-base" class="input input-bordered w-full bg-surface/50" type="text" bind:value={base} disabled={running} placeholder={i18n.t('verify.basePlaceholder')} />
        </div>

        <div class="flex flex-col gap-2">
          <label class="label text-xs font-medium text-text-secondary" for="verify-claims">{i18n.t('verify.claimsFile')}</label>
          <div class="repository-input">
            <input id="verify-claims" class="input input-bordered w-full bg-surface/50" type="text" bind:value={claimsPath} disabled={running} placeholder={i18n.t('verify.claimsPlaceholder')} />
            <button type="button" class="btn-glass btn-sm" onclick={chooseClaims} disabled={running || !bridgeAvailable}>{i18n.t('verify.chooseClaims')}</button>
          </div>
        </div>

        <div class="flex items-center gap-3 md:col-span-2 mt-2">
          <button type="submit" class="btn-primary-glass flex-1" disabled={!canRun || running}>
            {running ? i18n.t('verify.running') : i18n.t('verify.run')}
          </button>
          {#if running}<button type="button" class="btn-glass btn-sm" onclick={cancel}>{i18n.t('verify.cancel')}</button>{/if}
        </div>
      </form>
    </div>
  </div>
</section>

{#if running}
  <div class="mt-4 animate-fade-in">
    <div class="flex items-center gap-3 mb-2">
      <div class="loading-spinner w-5 h-5"></div>
      <span class="text-sm font-medium text-primary-color">{progressText()}</span>
    </div>
    <progress class="progress progress-glass w-full" max="100"></progress>
  </div>
{/if}

{#if error}
  <div class="alert-error-glass mt-4 animate-slide-up" role="alert">
    <p class="font-medium">{i18n.t('verify.error.title')}</p>
    <p class="text-sm">{error}</p>
  </div>
{/if}

{#if view}
  <section class="section-glass card-glass glass-surface animate-fade-in mt-6" aria-labelledby="verify-results-title">
    <div class="card-body gap-4">
      <div class="flex items-center justify-between">
        <h2 id="verify-results-title" class="card-title text-lg text-primary-color">{i18n.t('verify.results')}</h2>
        <button type="button" class="btn-glass btn-sm" onclick={exportReport}>{i18n.t('export.json')}</button>
      </div>
      
      <!-- Filters -->
      <div class="grid gap-3 sm:grid-cols-4">
        <label class="flex flex-col gap-1">
          <span class="text-xs font-medium text-text-secondary">{i18n.t('verify.state')}</span>
          <select class="select select-bordered select-sm bg-surface/50" bind:value={filters.state}>
            <option value="">{i18n.t('filters.all')}</option>
            {#each states as state (state)}
              <option value={state}>{i18n.t(`verify.state${state.charAt(0).toUpperCase()}${state.slice(1)}`)}</option>
            {/each}
          </select>
        </label>
        <label class="flex flex-col gap-1 sm:col-span-2">
          <span class="text-xs font-medium text-text-secondary">{i18n.t('filters.path')}</span>
          <input class="input input-bordered input-sm bg-surface/50" type="text" bind:value={filters.path} placeholder={i18n.t('filters.pathPlaceholder')} />
        </label>
        <div class="flex items-end">
          <button type="button" class="btn-glass btn-sm w-full" onclick={clearFilters}>{i18n.t('filters.clear')}</button>
        </div>
      </div>

      {#if rows.length === 0}
        <div class="empty-state">
          <p class="empty-state-title">{i18n.t('verify.resultsEmpty')}</p>
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="table-glass">
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
                <tr class="cursor-pointer hover:bg-primary/5 {selectedKey === row.key ? 'selected' : ''}" onclick={() => selectRow(row)}>
                  <td class="font-mono text-xs">{row.result.claim.id}</td>
                  <td><span class="badge badge-sm {badgeClass(row.result.status)}">{stateLabel(row.result.status)}</span></td>
                  <td class="text-xs">{row.result.claim.type}</td>
                  <td class="break-anywhere font-mono text-xs">{row.result.claim.paths?.join(', ') ?? '—'}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}

      {#if selected}
        <div class="border-t border-border-dark pt-4 prose prose-sm max-w-none text-text-secondary">
          <h3 class="text-primary-color font-medium">{i18n.t('verify.detail')}</h3>
          <p><strong>{i18n.t('verify.claim')}:</strong> {selected.result.claim.type}</p>
          <p><strong>{i18n.t('verify.message')}:</strong> {selected.result.message}</p>
        </div>
      {/if}
    </div>
  </section>
{/if}