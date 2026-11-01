<script lang="ts">
  import type { domain } from '../wailsjs/go/models'
  import { i18n } from '../i18n.svelte'
  import { hunkFor, locationText } from '../findings'

  let {
    finding,
    changes,
  }: {
    finding: domain.Finding | null
    changes?: domain.ChangeSet | null
  } = $props()

  const hunk = $derived(finding ? hunkFor(finding, changes) : null)

  function lineClass(kind: string): string {
    if (kind === 'added') {
      return 'text-success'
    }
    if (kind === 'removed') {
      return 'text-error'
    }
    return 'text-text-secondary'
  }

  function linePrefix(kind: string): string {
    if (kind === 'added') {
      return '+'
    }
    if (kind === 'removed') {
      return '-'
    }
    return ' '
  }

  function severityClass(severity: string): string {
    switch (severity) {
      case 'error': return 'chip chip-error'
      case 'warning': return 'chip chip-warning'
      default: return 'chip chip-info'
    }
  }
</script>

<aside class="panel panel-pad fade-in" aria-labelledby="detail-title">
  <div class="panel-head">
    <h2 id="detail-title" class="panel-title">{i18n.t('detail.title')}</h2>
  </div>

  {#if !finding}
    <div class="empty">
      <svg class="empty-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5.586a1 1 0 0 1 .707.293l5.414 5.414a1 1 0 0 1 .293.707V19a2 2 0 0 1-2 2z"></path>
      </svg>
      <p class="empty-title">{i18n.t('detail.empty')}</p>
    </div>
  {:else}
    <div class="flex flex-wrap items-center gap-2">
      <span class={severityClass(finding.severity)}>
        {i18n.t(`severity.${finding.severity}`)}
      </span>
      <span class="chip">
        {i18n.t(`epistemic.${finding.epistemic_status}`)}
      </span>
      {#if finding.language}
        <span class="chip chip-accent">{finding.language}</span>
      {/if}
      <span class="cell-strong" style="font-size: 13px">{finding.rule_id}</span>
      <span class="cell-muted">v{finding.rule_version}</span>
    </div>

    <div class="mt-4 space-y-5 border-t border-border pt-4">
      <div>
        <h3 class="eyebrow">{i18n.t('detail.location')}</h3>
        <p class="mt-1 break-anywhere font-mono text-sm text-text-secondary">{locationText(finding.location)}</p>
      </div>

      <div>
        <h3 class="eyebrow">{i18n.t('detail.message')}</h3>
        <p class="mt-1 break-anywhere text-sm text-text-secondary">{finding.message}</p>
      </div>

      {#if finding.evidence && finding.evidence.length > 0}
        <div>
          <h3 class="eyebrow">{i18n.t('detail.evidence')}</h3>
          <ul class="mt-2 space-y-2">
            {#each finding.evidence as evidence (evidence.kind + evidence.ref)}
              <li class="callout">
                <span class="chip chip-accent min-w-[80px] justify-center">{evidence.kind}</span>
                <div class="min-w-0 flex-1">
                  <p class="cell-mono break-anywhere">{evidence.ref}</p>
                  {#if evidence.detail}
                    <p class="mt-1 text-xs text-text-muted">{evidence.detail}</p>
                  {/if}
                </div>
              </li>
            {/each}
          </ul>
        </div>
      {/if}

      {#if hunk}
        <div>
          <h3 class="eyebrow">{i18n.t('detail.hunk')}</h3>
          <pre class="code-block mt-2"><code>{#each hunk.lines as line, i (i)}<span class={lineClass(line.kind)}>{linePrefix(line.kind)} {line.content}</span>
{/each}</code></pre>
        </div>
      {/if}

      <div>
        <h3 class="eyebrow">{i18n.t('detail.support')}</h3>
        <dl class="mt-2 grid gap-1">
          <div class="flex gap-2 text-sm">
            <dt class="text-text-muted">{i18n.t('detail.supportLevel')}:</dt>
            <dd class="text-text-secondary">{i18n.t(`support.${finding.support?.level ?? 'none'}`)}</dd>
          </div>
          {#if finding.support?.method}
            <div class="flex gap-2 text-sm">
              <dt class="text-text-muted">{i18n.t('detail.method')}:</dt>
              <dd class="break-anywhere text-text-secondary">{finding.support.method}</dd>
            </div>
          {/if}
        </dl>
      </div>

      <div>
        <h3 class="eyebrow">{i18n.t('detail.limitations')}</h3>
        {#if finding.support?.limitations && finding.support.limitations.length > 0}
          <ul class="mt-2 list-disc space-y-1 pl-4 text-xs text-text-secondary">
            {#each finding.support.limitations as limitation (limitation)}
              <li class="break-anywhere">{limitation}</li>
            {/each}
          </ul>
        {:else}
          <p class="mt-2 text-sm text-text-secondary">{i18n.t('detail.none')}</p>
        {/if}
      </div>

      {#if finding.recommendation}
        <div>
          <h3 class="eyebrow">{i18n.t('detail.recommendation')}</h3>
          <p class="mt-1 break-anywhere text-sm text-text-secondary">{finding.recommendation}</p>
        </div>
      {/if}

      {#if finding.revision?.base || finding.revision?.head}
        <div class="border-t border-border pt-4">
          <h3 class="eyebrow">{i18n.t('detail.revision')}</h3>
          <p class="mt-1 break-anywhere font-mono text-xs text-text-secondary">
            {#if finding.revision.base}{i18n.t('detail.base')}: {finding.revision.base}{/if}
            {#if finding.revision.base && finding.revision.head} · {/if}
            {#if finding.revision.head}{i18n.t('detail.head')}: {finding.revision.head}{/if}
          </p>
        </div>
      {/if}
    </div>
  {/if}
</aside>
