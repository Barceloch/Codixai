<script lang="ts">
  import { onMount } from 'svelte'
  import Header from './lib/components/Header.svelte'
  import AuditForm from './lib/components/AuditForm.svelte'
  import VerifyForm from './lib/components/VerifyForm.svelte'
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
  let activeTab = $state<'audit' | 'verify'>('audit')
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

  function newRunID(): string {
    if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
      return `run-${crypto.randomUUID()}`
    }
    return `run-${Date.now()}`
  }

  function describeError(cause: unknown): string {
    if (cause instanceof Error && cause.message) return cause.message
    if (typeof cause === 'string' && cause.trim()) return cause
    return i18n.t('audit.error.generic')
  }

  async function choose(): Promise<void> {
    error = ''
    try {
      const path = await ChooseRepository(i18n.t('audit.chooseRepositoryTitle'))
      if (path) root = path
    } catch (cause) { error = describeError(cause) }
  }

  async function run(): Promise<void> {
    if (running) return
    error = ''; notice = ''; progress = null; view = null; selectedKey = ''; filters = { ...defaultFilters }
    runID = newRunID()
    running = true
    try {
      await AuditAsync(new ui.AuditRequest({ root: root.trim(), origin, base: origin === 'base' ? base.trim() : '', run_id: runID }), runID)
    } catch (cause) { running = false; progress = null; error = describeError(cause) }
  }

  async function cancel(): Promise<void> {
    if (!runID) return
    try { await CancelAudit(runID) } catch (cause) { error = describeError(cause) }
  }

  async function exportReport(format: 'json' | 'sarif'): Promise<void> {
    if (!view) return
    error = ''; notice = ''
    try {
      const path = await Export(new ui.ExportRequest({ run_id: view.run_id, format, suggested_name: `codixai-${activeTab}.${format}`, title: i18n.t('export.saveTitle') }))
      notice = path ? i18n.t('export.saved', { path }) : i18n.t('export.cancelled')
    } catch (cause) { error = describeError(cause) }
  }

  function selectFinding(finding: domain.Finding, index: number): void {
    selectedKey = findingKey(finding, index)
  }

  function applyEvent(event: ui.AuditEvent): void {
    if (!event || event.run_id !== runID) return
    if (event.kind === 'progress' && event.progress) { progress = event.progress; return }
    if (event.kind === 'done' && event.view) { running = false; progress = null; view = event.view; selectedKey = ''; filters = { ...defaultFilters }; return }
    if (event.kind === 'cancelled') { running = false; progress = null; return }
    if (event.kind === 'error') { running = false; progress = null; error = event.error || i18n.t('audit.error.generic') }
  }

  async function prefill(): Promise<void> {
    try {
      const path = await Root()
      if (path && !root) root = path
    } catch { /* empty */ }
  }

  onMount(() => {
    bridgeAvailable = isBridgeAvailable()
    if (!bridgeAvailable) return
    const stopAudit = onAuditEvent(applyEvent)
    void prefill()
    return stopAudit
  })
</script>

<svelte:head>
  <title>{i18n.t('app.title')}</title>
</svelte:head>

<div class="flex min-h-screen flex-col bg-background text-text-primary">
  <Header />

  <main class="mx-auto flex w-full max-w-7xl flex-1 flex-col gap-4 px-4 py-6">
    <!-- Tabs -->
    <div class="tabs-glass" role="tablist" aria-label="Main tabs">
      <button
        role="tab"
        class:tab-active={activeTab === 'audit'}
        aria-selected={activeTab === 'audit'}
        onclick={() => { activeTab = 'audit'; view = null; error = ''; notice = ''; }}
      >
        {i18n.t('tab.audit')}
      </button>
      <button
        role="tab"
        class:tab-active={activeTab === 'verify'}
        aria-selected={activeTab === 'verify'}
        onclick={() => { activeTab = 'verify'; view = null; error = ''; notice = ''; }}
      >
        {i18n.t('tab.verify')}
      </button>
    </div>

    {#if activeTab === 'audit'}
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
        <div class="alert-success-glass animate-fade-in" role="status">
          <span class="break-anywhere">{notice}</span>
        </div>
      {/if}

      {#if view}
        <div class="grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)] animate-fade-in">
          <SummaryPanel {view} />

          <div class="flex flex-wrap items-center justify-between gap-2">
            <h2 class="text-lg font-semibold text-primary-color">{i18n.t('export.title')}</h2>
            <div class="flex flex-wrap gap-2">
              <button type="button" class="btn-outline-glass btn-sm" onclick={() => exportReport('json')}>
                {i18n.t('export.json')}
              </button>
              <button type="button" class="btn-outline-glass btn-sm" onclick={() => exportReport('sarif')}>
                {i18n.t('export.sarif')}
              </button>
            </div>
          </div>

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
        <div class="empty-state card-glass animate-fade-in">
          <svg class="empty-state-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5.586a1 1 0 0 1 .707.293l5.414 5.414a1 1 0 0 1 .293.707V19a2 2 0 0 1-2 2z"></path>
          </svg>
          <p class="empty-state-title">{i18n.t('audit.idle')}</p>
          <p class="empty-state-description">{i18n.t('audit.idleHint')}</p>
        </div>
      {/if}
    {/if}

    {#if activeTab === 'verify'}
      <VerifyForm />
    {/if}
  </main>

  <footer class="border-t border-border-dark px-4 py-3">
    <p class="mx-auto max-w-7xl text-xs text-text-secondary text-center">
      {i18n.t('footer.note')}
    </p>
  </footer>
</div>