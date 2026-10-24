<script lang="ts">
  import type { ui } from '../wailsjs/go/models'
  import { i18n } from '../i18n.svelte'

  let { view }: { view: ui.AuditView } = $props()

  const summary = $derived(view.document?.summary)
  const findings = $derived(view.document?.findings ?? [])
  const excluded = $derived(view.excluded_files ?? [])
</script>

<section class="card border border-base-300 bg-base-100 shadow-sm" aria-labelledby="summary-title">
  <div class="card-body gap-3">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h2 id="summary-title" class="card-title text-lg">{i18n.t('summary.title')}</h2>
      {#if view.failed}
        <span class="badge badge-error">{i18n.t('summary.failed')}</span>
      {:else}
        <span class="badge badge-success">{i18n.t('summary.passed')}</span>
      {/if}
    </div>

    <dl class="grid gap-x-6 gap-y-2 text-sm sm:grid-cols-2 lg:grid-cols-3">
      <div>
        <dt class="text-base-content/60">{i18n.t('summary.root')}</dt>
        <dd class="break-anywhere font-medium">{view.root}</dd>
      </div>
      <div>
        <dt class="text-base-content/60">{i18n.t('summary.origin')}</dt>
        <dd class="font-medium">
          {i18n.t(`origin.${view.origin}`)}
          {#if view.base}<span class="text-base-content/70">({view.base})</span>{/if}
        </dd>
      </div>
      <div>
        <dt class="text-base-content/60">{i18n.t('summary.changedFiles')}</dt>
        <dd class="font-medium">{view.changed_files}</dd>
      </div>
      <div>
        <dt class="text-base-content/60">{i18n.t('summary.findings')}</dt>
        <dd class="font-medium">{summary?.total ?? findings.length}</dd>
      </div>
      <div>
        <dt class="text-base-content/60">{i18n.t('summary.failOn')}</dt>
        <dd class="font-medium">{view.fail_on}</dd>
      </div>
      <div>
        <dt class="text-base-content/60">{i18n.t('summary.known')}</dt>
        <dd class="font-medium">{view.known_findings}</dd>
      </div>
      <div>
        <dt class="text-base-content/60">{i18n.t('summary.suppressed')}</dt>
        <dd class="font-medium">{view.suppressed_findings}</dd>
      </div>
      <div>
        <dt class="text-base-content/60">{i18n.t('summary.excluded')}</dt>
        <dd class="font-medium">{excluded.length}</dd>
      </div>
    </dl>

    <div class="flex flex-wrap gap-2" aria-label={i18n.t('summary.findings')}>
      {#each Object.entries(summary?.by_severity ?? {}) as [severity, count] (severity)}
        <span class="badge badge-outline">{i18n.t(`severity.${severity}`)}: {count}</span>
      {/each}
      {#each Object.entries(summary?.by_epistemic_status ?? {}) as [status, count] (status)}
        <span class="badge badge-outline">{i18n.t(`epistemic.${status}`)}: {count}</span>
      {/each}
      {#each Object.entries(summary?.by_language ?? {}) as [language, count] (language)}
        <span class="badge badge-ghost">{language}: {count}</span>
      {/each}
    </div>

    {#if excluded.length > 0}
      <details class="text-sm">
        <summary class="cursor-pointer text-base-content/70">
          {i18n.t('summary.excluded')} ({excluded.length})
        </summary>
        <ul class="mt-2 list-inside list-disc break-anywhere">
          {#each excluded as path (path)}
            <li>{path}</li>
          {/each}
        </ul>
      </details>
    {/if}
  </div>
</section>
