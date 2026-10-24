<script lang="ts">
  // The composition root of the audit screen.
  //
  // The component owns the state of the window: the request, the run in
  // progress, the document that the adapter returned and the selection of
  // the list. It calls the bound methods through the bridge and never
  // reinterprets a finding: the document is the one the use case produced.
  import { onMount } from 'svelte'
  import Header from './lib/components/Header.svelte'
  import AuditForm from './lib/components/AuditForm.svelte'
  import SummaryPanel from './lib/components/SummaryPanel.svelte'
  import FindingsPanel from './lib/components/FindingsPanel.svelte'
  import FindingDetail from './lib/components/FindingDetail.svelte'
  import { i18n } from './lib/i18n.svelte'
  import {
    AuditAsync,
    CancelAudit,
    ChooseRepository,
    Export,
    Root,
    isBridgeAvailable,
    onAuditEvent,
  } from './lib/bridge'
  import { ui } from './lib/wailsjs/go/models'
  import type { domain } from './lib/wailsjs/go/models'
  import {
    defaultFilters,
    filterAndSort,
    findingKey,
    type Filters,
  } from './lib/findings'

  let bridgeAvailable = $state(isBridgeAvailable())
  let root = $state('')
  let origin = $state('worktree')
  let base = $state('')
  let running = $state(false)
  let progress = $state<ui.AuditProgress | null>(null)
  let view = $state<ui.AuditView | null>(null)
  let error = $state('')
  let notice = $state('')
  let runID = $state('')
  let filters = $state<Filters>({ ...defaultFilters })
  let selectedKey = $state('')

  const findings = $derived(view?.document?.findings ?? [])
  const visible = $derived(filterAndSort(findings, filters))
  const selected = $derived(
    visible.find(
      (finding, index) => findingKey(finding, index) === selectedKey,
    ) ?? null,
  )

  // newRunID identifies a run before it starts, so the progress events and
  // the export refer to the same run.
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
    return i18n.t('audit.error.generic')
  }

  // choose asks the window for a directory and fills the repository field.
  async function choose(): Promise<void> {
    error = ''
    try {
      const path = await ChooseRepository(
        i18n.t('audit.chooseRepositoryTitle'),
      )
      if (path) {
        root = path
      }
    } catch (cause) {
      error = describeError(cause)
    }
  }

  // run starts the asynchronous audit and waits for its events.
  async function run(): Promise<void> {
    if (running) {
      return
    }
    error = ''
    notice = ''
    progress = null
    view = null
    selectedKey = ''
    filters = { ...defaultFilters }
    runID = newRunID()
    running = true
    try {
      const request = new ui.AuditRequest({
        root: root.trim(),
        origin,
        base: origin === 'base' ? base.trim() : '',
        run_id: runID,
      })
      await AuditAsync(request, runID)
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

  // exportReport writes the finished document through the native dialog.
  async function exportReport(format: 'json' | 'sarif'): Promise<void> {
    if (!view) {
      return
    }
    error = ''
    notice = ''
    try {
      const path = await Export(
        new ui.ExportRequest({
          run_id: view.run_id,
          format,
          suggested_name: `codixai-audit.${format}`,
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

  // selectFinding remembers the row that the person chose.
  function selectFinding(finding: domain.Finding, index: number): void {
    selectedKey = findingKey(finding, index)
  }

  // applyEvent folds one event of the run into the state of the screen.
  function applyEvent(event: ui.AuditEvent): void {
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
      filters = { ...defaultFilters }
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
      error = event.error || i18n.t('audit.error.generic')
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

  onMount(() => {
    bridgeAvailable = isBridgeAvailable()
    if (!bridgeAvailable) {
      return
    }
    const stop = onAuditEvent(applyEvent)
    void prefill()
    return stop
  })
</script>

<svelte:head>
  <title>{i18n.t('app.title')}</title>
</svelte:head>

<div class="flex min-h-screen flex-col">
  <Header />

  <main class="mx-auto flex w-full max-w-7xl flex-1 flex-col gap-4 px-4 py-6">
    <AuditForm
      bind:root
      bind:origin
      bind:base
      {running}
      {progress}
      {error}
      {bridgeAvailable}
      onrun={run}
      oncancel={cancel}
      onchoose={choose}
    />

    {#if notice}
      <div class="alert alert-success" role="status">
        <span class="break-anywhere">{notice}</span>
      </div>
    {/if}

    {#if view}
      <SummaryPanel {view} />

      <div class="flex flex-wrap items-center justify-between gap-2">
        <h2 class="text-lg font-semibold">{i18n.t('export.title')}</h2>
        <div class="flex flex-wrap gap-2">
          <button
            type="button"
            class="btn btn-outline btn-sm"
            onclick={() => exportReport('json')}
          >
            {i18n.t('export.json')}
          </button>
          <button
            type="button"
            class="btn btn-outline btn-sm"
            onclick={() => exportReport('sarif')}
          >
            {i18n.t('export.sarif')}
          </button>
        </div>
      </div>

      <div class="grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <FindingsPanel
          bind:filters
          findings={visible}
          allFindings={findings}
          {selectedKey}
          onselect={selectFinding}
        />
        <FindingDetail finding={selected} changes={view.changes} />
      </div>
    {:else if !running}
      <div class="rounded-box border border-dashed border-base-300 bg-base-100 p-8 text-center">
        <p class="font-medium">{i18n.t('audit.idle')}</p>
        <p class="text-sm text-base-content/70">{i18n.t('audit.idleHint')}</p>
      </div>
    {/if}
  </main>

  <footer class="border-t border-base-300 bg-base-100 px-4 py-3">
    <p class="mx-auto max-w-7xl text-xs text-base-content/60">
      {i18n.t('footer.note')}
    </p>
  </footer>
</div>
