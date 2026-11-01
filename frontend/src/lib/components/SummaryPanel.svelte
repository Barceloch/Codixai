<script lang="ts">
  import type { ui } from '../wailsjs/go/models'
  import { i18n } from '../i18n.svelte'

  let { view }: { view: ui.AuditView } = $props()

  const summary = $derived(view.document?.summary)
  const findings = $derived(view.document?.findings ?? [])
  const excluded = $derived(view.excluded_files ?? [])

  function severityClass(severity: string): string {
    switch (severity) {
      case 'error': return 'chip chip-error'
      case 'warning': return 'chip chip-warning'
      default: return 'chip chip-info'
    }
  }
</script>

<section class="panel panel-pad slide-up" aria-labelledby="summary-title">
  <div class="panel-head">
    <h2 id="summary-title" class="panel-title">{i18n.t('summary.title')}</h2>
    {#if view.failed}
      <span class="chip chip-error">{i18n.t('summary.failed')}</span>
    {:else}
      <span class="chip chip-success">{i18n.t('summary.passed')}</span>
    {/if}
  </div>

  <dl class="grid gap-x-8 gap-y-5 sm:grid-cols-2 lg:grid-cols-4">
    <div>
      <dt class="eyebrow">{i18n.t('summary.root')}</dt>
      <dd class="mt-1 break-anywhere text-sm text-text-primary">{view.root}</dd>
    </div>
    <div>
      <dt class="eyebrow">{i18n.t('summary.origin')}</dt>
      <dd class="mt-1 text-sm text-text-primary">
        {i18n.t(`origin.${view.origin}`)}
        {#if view.base}<span class="ml-1 text-xs text-text-muted">({view.base})</span>{/if}
      </dd>
    </div>
    <div>
      <dt class="eyebrow">{i18n.t('summary.changedFiles')}</dt>
      <dd class="mt-1 text-sm text-text-primary">{view.changed_files}</dd>
    </div>
    <div>
      <dt class="eyebrow">{i18n.t('summary.findings')}</dt>
      <dd class="mt-1 text-sm text-text-primary">{summary?.total ?? findings.length}</dd>
    </div>
    <div>
      <dt class="eyebrow">{i18n.t('summary.failOn')}</dt>
      <dd class="mt-1 text-sm text-text-primary">{view.fail_on}</dd>
    </div>
    <div>
      <dt class="eyebrow">{i18n.t('summary.known')}</dt>
      <dd class="mt-1 text-sm text-text-primary">{view.known_findings}</dd>
    </div>
    <div>
      <dt class="eyebrow">{i18n.t('summary.suppressed')}</dt>
      <dd class="mt-1 text-sm text-text-primary">{view.suppressed_findings}</dd>
    </div>
    <div>
      <dt class="eyebrow">{i18n.t('summary.excluded')}</dt>
      <dd class="mt-1 text-sm text-text-primary">{excluded.length}</dd>
    </div>
  </dl>

  <div class="mt-6 flex flex-wrap gap-2" aria-label={i18n.t('summary.findings')}>
    {#each Object.entries(summary?.by_severity ?? {}) as [severity, count] (severity)}
      <span class={severityClass(severity)}>{i18n.t(`severity.${severity}`)}: {count}</span>
    {/each}
    {#each Object.entries(summary?.by_epistemic_status ?? {}) as [status, count] (status)}
      <span class="chip">{i18n.t(`epistemic.${status}`)}: {count}</span>
    {/each}
    {#each Object.entries(summary?.by_language ?? {}) as [language, count] (language)}
      <span class="chip chip-accent">{language}: {count}</span>
    {/each}
  </div>

  {#if excluded.length > 0}
    <details class="mt-4 group">
      <summary class="eyebrow flex cursor-pointer list-none items-center gap-2">
        {i18n.t('summary.excluded')} ({excluded.length})
        <svg class="h-3.5 w-3.5 transition-transform group-open:rotate-90" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"></path>
        </svg>
      </summary>
      <ul class="mt-3 list-inside list-disc space-y-1">
        {#each excluded as path (path)}
          <li class="break-anywhere font-mono text-xs text-text-muted">{path}</li>
        {/each}
      </ul>
    </details>
  {/if}
</section>
