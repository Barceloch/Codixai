<script lang="ts">
  // The verification screen.
  //
  // The component is self contained: it owns the request, the run in
  // progress and the document that the adapter returned, so the shell only
  // places it in the tab. It never reinterprets a claim: the state of every
  // result is the one the use case decided, and the document is the one the
  // adapter sent in the done event.
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

  // The states that the verification document can produce. The label of each
  // one lives in the catalog: the document carries the identifier, never a
  // translation of it.
  const states: Exclude<StateFilter, ''>[] = [
    'verified',
    'contradicted',
    'unresolved',
  ]

  let bridgeAvailable = $state(isBridgeAvailable())
  let root = $state('')
  let base = $state('')
  let claimsPath = $state('')
  let running = $state(false)
  let progress = $state<ui.VerifyProgress | null>(null)
  let view = $state<ui.VerifyView | null>(null)
  let error = $state('')
  let notice = $state('')
  let runID = $state('')
  let selectedKey = $state('')
  let filters = $state<{
    state: StateFilter
    path: string
    sort: SortKey
    direction: SortDirection
  }>({ state: '', path: '', sort: 'id', direction: 'asc' })

  // rows applies the filters and the order to the results of the document.
  // The key of a row is the position in the whole document, so the selection
  // survives a change of the filters.
  const rows = $derived.by<Row[]>(() => {
    const results = view?.document?.results ?? []
    const indexed = results.map((result, index) => ({
      key: `${result.claim.id}:${index}`,
      result,
    }))
    const path = filters.path.trim().toLowerCase()
    const filtered = indexed.filter(({ result }) => {
      if (filters.state && result.status !== filters.state) {
        return false
      }
      if (path && !(result.claim.paths ?? []).some((item) => item.toLowerCase().includes(path))) {
        return false
      }
      return true
    })
    const direction = filters.direction === 'desc' ? -1 : 1
    return filtered.sort((left, right) => {
      const order =
        filters.sort === 'state'
          ? left.result.status.localeCompare(right.result.status)
          : left.result.claim.id.localeCompare(right.result.claim.id)
      if (order !== 0) {
        return order * direction
      }
      return left.result.claim.id.localeCompare(right.result.claim.id)
    })
  })

  const selected = $derived(rows.find((row) => row.key === selectedKey) ?? null)

  const canRun = $derived(
    !running &&
      root.trim().length > 0 &&
      base.trim().length > 0 &&
      claimsPath.trim().length > 0,
  )

  // newRunID identifies a run before it starts, so the progress events, the
  // cancellation and the export refer to the same run.
  function newRunID(): string {
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
      return `run-${crypto.randomUUID()}`
    }
    return `run-${Date.now()}`
  }

  // describeError turns a rejected promise into a message for the person.
  function describeError(cause: unknown): string {
    if (cause instanceof Error && cause.message) {
      return cause.message
    }
    if (typeof cause === 'string' && cause.trim()) {
      return cause
    }
    return i18n.t('verify.error.generic')
  }

  // progressText translates the phase of the run. The adapter only reports
  // the phases it can observe; the rest is the generic wording of the run.
  function progressText(): string {
    switch (progress?.phase) {
      case 'validating':
        return i18n.t('verify.progress.validating')
      case 'collecting':
        return i18n.t('verify.progress.collecting')
      case 'done':
        return i18n.t('verify.progress.done')
      case 'cancelled':
        return i18n.t('verify.progress.cancelled')
      default:
        return i18n.t('verify.running')
    }
  }

  // stateLabel translates the state of a result.
  function stateLabel(status: string): string {
    switch (status) {
      case 'verified':
        return i18n.t('verify.stateVerified')
      case 'contradicted':
        return i18n.t('verify.stateContradicted')
      case 'unresolved':
        return i18n.t('verify.stateUnresolved')
      default:
        return status
    }
  }

  // badgeClass picks the badge of the state. A state without a known class
  // keeps the neutral badge, so the table never hides an unexpected value.
  function badgeClass(status: string): string {
    switch (status) {
      case 'verified':
        return 'badge-success-glass'
      case 'contradicted':
        return 'badge-error-glass'
      case 'unresolved':
        return 'badge-warning-glass'
      default:
        return 'badge-glass'
    }
  }

  async function chooseRoot(): Promise<void> {
    error = ''
    try {
      const path = await ChooseRepository(i18n.t('verify.chooseRepositoryTitle'))
      if (path) {
        root = path
      }
    } catch (cause) {
      error = describeError(cause)
    }
  }

  async function chooseClaims(): Promise<void> {
    error = ''
    try {
      const path = await ChooseFile(i18n.t('verify.chooseClaimsTitle'), ['.json'])
      if (path) {
        claimsPath = path
      }
    } catch (cause) {
      error = describeError(cause)
    }
  }

  // run starts the asynchronous verification and waits for its events. The
  // identifier is generated once and used by the request, the call and the
  // cancellation, so the events of the run are the ones the screen folds.
  async function run(): Promise<void> {
    if (running) {
      return
    }
    if (!root.trim()) {
      error = i18n.t('verify.error.noRoot')
      return
    }
    if (!base.trim()) {
      error = i18n.t('verify.error.noBase')
      return
    }
    if (!claimsPath.trim()) {
      error = i18n.t('verify.error.noClaims')
      return
    }

    error = ''
    notice = ''
    progress = null
    view = null
    selectedKey = ''
    filters = { state: '', path: '', sort: 'id', direction: 'asc' }
    runID = newRunID()
    running = true

    try {
      const request = new ui.VerifyRequest({
        root: root.trim(),
        base: base.trim(),
        claims_path: claimsPath.trim(),
        run_id: runID,
      })
      await VerifyAsync(request, runID)
    } catch (cause) {
      running = false
      progress = null
      error = describeError(cause)
    }
  }

  // cancel asks the adapter to stop the run.
  async function cancel(): Promise<void> {
    if (!runID) {
      return
    }
    try {
      await CancelAudit(runID)
    } catch (cause) {
      error = describeError(cause)
    }
  }

  // exportReport writes the finished verification document through the
  // native dialog. The verification has no SARIF rendering yet, so the
  // screen only offers the JSON document.
  async function exportReport(): Promise<void> {
    if (!view) {
      return
    }
    error = ''
    notice = ''
    try {
      const path = await Export(
        new ui.ExportRequest({
          run_id: view.run_id,
          format: 'json',
          suggested_name: 'codixai-verify.json',
          title: i18n.t('export.saveTitle'),
        }),
      )
      notice = path
        ? i18n.t('export.saved', { path })
        : i18n.t('export.cancelled')
    } catch (cause) {
      error = describeError(cause)
    }
  }

  function selectRow(row: Row): void {
    selectedKey = row.key
  }

  function clearFilters(): void {
    filters = { state: '', path: '', sort: 'id', direction: 'asc' }
  }

  // applyEvent folds one event of the run into the state of the screen.
  function applyEvent(raw: ui.VerifyEvent): void {
    const event = ui.VerifyEvent.createFrom(raw)
    if (!event || event.run_id !== runID) {
      return
    }
    if (event.kind === 'progress' && event.progress) {
      progress = event.progress
      return
    }
    if (event.kind === 'done' && event.view) {
      running = false
      progress = null
      view = event.view
      selectedKey = ''
      return
    }
    if (event.kind === 'cancelled') {
      running = false
      progress = null
      return
    }
    if (event.kind === 'error') {
      running = false
      progress = null
      error = event.error || i18n.t('verify.error.generic')
    }
  }

  // prefill fills the repository field with the root of the window.
  async function prefill(): Promise<void> {
    try {
      const path = await Root()
      if (path && !root) {
        root = path
      }
    } catch {
      // The field stays empty and the person types the path.
    }
  }

  function submit(event: SubmitEvent): void {
    event.preventDefault()
    void run()
  }

  onMount(() => {
    bridgeAvailable = isBridgeAvailable()
    if (!bridgeAvailable) {
      return
    }
    const stopVerify = onVerifyEvent(applyEvent)
    void prefill()
    return stopVerify
  })
</script>

<section class="section-glass glass-surface" aria-labelledby="verify-title">
  <div class="card-glass card-glass-elevated">
    <div class="card-body gap-4">
      <h2 id="verify-title" class="card-title text-lg text-primary-color">
        {i18n.t('verify.title')}
      </h2>

      <form class="grid gap-4 md:grid-cols-2" onsubmit={submit} autocomplete="off">
        <div class="grid gap-1 md:col-span-2">
          <label class="label text-xs text-primary-color/60" for="verify-repository">
            {i18n.t('verify.repository')}
          </label>
          <div class="join w-full">
            <input
              id="verify-repository"
              class="input input-bordered join-item w-full"
              type="text"
              spellcheck="false"
              placeholder={i18n.t('verify.repositoryPlaceholder')}
              bind:value={root}
              disabled={running}
            />
            <button
              type="button"
              class="btn-glass btn-sm"
              onclick={chooseRoot}
              disabled={running || !bridgeAvailable}
            >
              {i18n.t('verify.chooseRepository')}
            </button>
          </div>
        </div>

        <div class="flex flex-col gap-1">
          <label class="label text-xs text-primary-color/60" for="verify-base">
            {i18n.t('verify.base')}
          </label>
          <input
            id="verify-base"
            class="input input-bordered w-full"
            type="text"
            spellcheck="false"
            placeholder={i18n.t('verify.basePlaceholder')}
            bind:value={base}
            disabled={running}
          />
        </div>

        <div class="flex flex-col gap-1">
          <label class="label text-xs text-primary-color/60" for="verify-claims">
            {i18n.t('verify.claimsFile')}
          </label>
          <div class="join w-full">
            <input
              id="verify-claims"
              class="input input-bordered join-item w-full"
              type="text"
              spellcheck="false"
              placeholder={i18n.t('verify.claimsPlaceholder')}
              bind:value={claimsPath}
              disabled={running}
            />
            <button
              type="button"
              class="btn-glass btn-sm"
              onclick={chooseClaims}
              disabled={running || !bridgeAvailable}
            >
              {i18n.t('verify.chooseClaims')}
            </button>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-2 md:col-span-2">
          <button
            type="submit"
            class="btn btn-primary-glass w-full disabled={!canRun}"
          >
            {running ? i18n.t('verify.running') : i18n.t('verify.run')}
          </button>
          {#if running}
            <button type="button" class="btn-glass btn-sm" onclick={cancel}>
              {i18n.t('verify.cancel')}
            </button>
          {/if}
        </div>
      </form>
    </div>
  </div>
</section>

{#if !bridgeAvailable}
  <div class="alert alert-warning glass-surface" role="status">
    <span>{i18n.t('audit.error.bridge')}</span>
  </div>
{/if}

{#if running}
  <div class="flex flex-col gap-1">
    <progress class="progress progress-glass" max="100"></progress>
    <p class="text-sm text-primary-color/60">{progressText()}</p>
  </div>
{/if}

{#if error}
  <div class="alert alert-error-glass" role="alert">
    <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-4 w-4" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m-2 2l-2 2m-2 2l-2 2m2 2l-2 2m2 2l2 2m-2 2l-2 2m2 2l2 2" /></svg>
    <span class="text-primary-color">{error}</span>
  </div>
{/if}

{#if notice}
  <div class="alert alert-success-glass" role="status">
    <span class="break-anywhere text-primary-color">{notice}</span>
  </div>
{/if}

{#if view}
  <section class="section-glass card-glass glass-surface" aria-labelledby="verify-results-title">
    <div class="card-body gap-4">
      <div class="flex flex-wrap items-center justify-between gap-2">
        <h2 id="verify-results-title" class="card-title text-lg text-primary-color">
          {i18n.t('verify.results')}
        </h2>
        <button type="button" class="btn-glass btn-sm" onclick={exportReport}>
          {i18n.t('export.json')}
        </button>
      </div>

      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <label class="flex flex-col gap-1">
          <span class="text-xs font-medium text-primary-color/60">{i18n.t('verify.state')}</span>
          <select class="select select-bordered select-sm" bind:value={filters.state}>
            <option value="">{i18n.t('filters.all')}</option>
            {#each states as state (state)}
              <option value={state}>{i18n.t(`verify.state${state.charAt(0).toUpperCase()}${state.slice(1)}`)}</option>
            {/each}
          </select>
        </label>

        <label class="flex flex-col gap-1">
          <span class="text-xs font-medium text-primary-color/60">{i18n.t('filters.path')}</span>
          <input
            class="input input-bordered input-sm"
            type="text"
            spellcheck="false"
            placeholder={i18n.t('filters.pathPlaceholder')}
            bind:value={filters.path}
          />
        </label>

        <div class="flex flex-col gap-1">
          <span class="text-xs font-medium text-primary-color/60">{i18n.t('filters.sort')}</span>
          <div class="join">
            <select class="select select-bordered select-sm join-item" bind:value={filters.sort}>
              <option value="id">{i18n.t('verify.claimId')}</option>
              <option value="state">{i18n.t('verify.state')}</option>
            </select>
            <select
              class="select select-bordered select-sm join-item"
              bind:value={filters.direction}
              aria-label={i18n.t('filters.sort')}
            >
              <option value="asc">{i18n.t('filters.ascending')}</option>
              <option value="desc">{i18n.t('filters.descending')}</option>
            </select>
          </div>
        </div>

        <div class="flex items-end">
          <button type="button" class="btn-glass btn-sm" onclick={clearFilters}>
            {i18n.t('filters.clear')}
          </button>
        </div>
      </div>

      {#if rows.length === 0}
        <div class="rounded-box border border-dashed border-base-300 p-6 text-center text-primary-color/60">
          <p class="font-medium">{i18n.t('verify.resultsEmpty')}</p>
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
                <tr
                  class="cursor-pointer hover:bg-base-200 {selectedKey === row.key ? 'selected' : ''}"
                  tabindex="0"
                  role="button"
                  aria-pressed={selectedKey === row.key}
                  onclick={() => selectRow(row)}
                  onkeydown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault()
                      selectRow(row)
                    }
                  }}
                >
                  <td class="font-mono text-xs whitespace-nowrap">{row.result.claim.id}</td>
                  <td>
                    <span class="badge badge-sm {badgeClass(row.result.status)}">
                      {stateLabel(row.result.status)}
                    </span>
                  </td>
                  <td class="whitespace-nowrap text-xs text-primary-color/70">{row.result.claim.type}</td>
                  <td class="break-anywhere font-mono text-xs text-primary-color/70">
                    {(row.result.claim.paths ?? []).join(', ') || '—'}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}

      {#if selected}
        <div class="divider"></div>
        <div class="prose prose-sm max-w-none text-primary-color">
          <h3 class="text-sm font-semibold">{i18n.t('verify.detail')}</h3>
          <p><strong>{i18n.t('verify.claim')}:</strong> {selected.result.claim.type}</p>
          <p><strong>{i18n.t('verify.paths')}:</strong> {(selected.result.claim.paths ?? []).join(', ') || '—'}</p>
          <p><strong>{i18n.t('verify.message')}:</strong> {selected.result.message}</p>

          {#if selected.result.evidence?.length}
            <h4 class="text-sm font-semibold">{i18n.t('verify.evidence')}</h4>
            <table class="table-glass">
              <thead>
                <tr>
                  <th>{i18n.t('detail.evidenceKind')}</th>
                  <th>{i18n.t('detail.evidenceRef')}</th>
                </tr>
              </thead>
              <tbody>
                {#each selected.result.evidence as item, index (index)}
                  <tr>
                    <td class="font-mono text-xs">{item.kind}</td>
                    <td class="break-anywhere font-mono text-xs">
                      {item.ref}{#if item.detail} — {item.detail}{/if}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          {/if}

          {#if selected.result.limitations?.length}
            <h4 class="text-sm font-semibold">{i18n.t('verify.limitations')}</h4>
            <ul>
              {#each selected.result.limitations as limitation, index (index)}
                <li>{limitation}</li>
              {/each}
            </ul>
          {/if}
        </div>
      {/if}
    </div>
  </section>
{:else if !running}
  <div class="rounded-box border border-dashed border-base-300 bg-base-100 p-8 text-center">
    <p class="font-medium">{i18n.t('verify.idle')}</p>
    <p class="text-sm text-base-content/70">{i18n.t('verify.idleHint')}</p>
  </div>
{/if}
