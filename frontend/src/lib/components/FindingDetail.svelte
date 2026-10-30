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
</script>

<aside class="card-glass-glow glass-surface animate-fade-in" aria-labelledby="detail-title">
  <div class="card-body gap-4">
    <h2 id="detail-title" class="card-title text-lg text-primary-color">{i18n.t('detail.title')}</h2>

    {#if !finding}
      <div class="empty-state py-8">
        <svg class="empty-state-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5.586a1 1 0 0 1 .707.293l5.414 5.414a1 1 0 0 1 .293.707V19a2 2 0 0 1-2 2z"></path>
        </svg>
        <p class="empty-state-title">{i18n.t('detail.empty')}</p>
      </div>
    {:else}
      <div class="flex flex-wrap items-center gap-2">
        <span class="badge badge-primary-glass">
          {i18n.t(`severity.${finding.severity}`)}
        </span>
        <span class="badge badge-glass text-xs text-text-secondary">
          {i18n.t(`epistemic.${finding.epistemic_status}`)}
        </span>
        {#if finding.language}
          <span class="badge badge-secondary text-xs">{finding.language}</span>
        {/if}
        <span class="font-mono text-sm font-semibold text-primary-color">{finding.rule_id}</span>
        <span class="text-xs text-text-secondary">v{finding.rule_version}</span>
      </div>

      <div class="space-y-4 border-t border-border-dark pt-4">
        <div>
          <h3 class="text-xs font-medium text-text-secondary uppercase tracking-wide">{i18n.t('detail.location')}</h3>
          <p class="break-anywhere text-sm text-text-secondary mt-1 font-mono">{locationText(finding.location)}</p>
        </div>

        <div>
          <h3 class="text-xs font-medium text-text-secondary uppercase tracking-wide">{i18n.t('detail.message')}</h3>
          <p class="break-anywhere text-sm text-text-secondary mt-1">{finding.message}</p>
        </div>

        {#if finding.evidence && finding.evidence.length > 0}
          <div>
            <h3 class="text-xs font-medium text-text-secondary uppercase tracking-wide">{i18n.t('detail.evidence')}</h3>
            <ul class="mt-2 space-y-2">
              {#each finding.evidence as evidence (evidence.kind + evidence.ref)}
                <li class="rounded-md bg-surface/50 border border-border p-3 text-xs text-text-secondary">
                  <div class="flex items-start gap-2">
                    <span class="font-semibold text-primary-color min-w-[80px]">{evidence.kind}</span>
                    <div class="flex-1 min-w-0">
                      <span class="font-mono break-anywhere">{evidence.ref}</span>
                      {#if evidence.detail}
                        <div class="mt-1 text-text-secondary/80">{evidence.detail}</div>
                      {/if}
                    </div>
                  </div>
                </li>
              {/each}
            </ul>
          </div>
        {/if}

        {#if hunk}
          <div>
            <h3 class="text-xs font-medium text-text-secondary uppercase tracking-wide">{i18n.t('detail.hunk')}</h3>
            <pre class="pre-glass mt-2 font-mono text-xs overflow-x-auto"><code>
{#each hunk.lines as line, i (i)}
  <span class="{lineClass(line.kind)}">{line.kind === 'added' ? '+' : line.kind === 'removed' ? '-' : ' '} {line.content}</span>
{/each}
            </code></pre>
          </div>
        {/if}

        <div>
          <h3 class="text-xs font-medium text-text-secondary uppercase tracking-wide">{i18n.t('detail.support')}</h3>
          <dl class="mt-2 grid gap-1 text-sm">
            <div class="flex gap-2">
              <dt class="text-text-secondary">{i18n.t('detail.supportLevel')}:</dt>
              <dd class="text-text-secondary">{i18n.t(`support.${finding.support?.level ?? 'none'}`)}</dd>
            </div>
            {#if finding.support?.method}
              <div class="flex gap-2">
                <dt class="text-text-secondary">{i18n.t('detail.method')}:</dt>
                <dd class="break-anywhere text-text-secondary">{finding.support.method}</dd>
              </div>
            {/if}
          </dl>
        </div>

        <div>
          <h3 class="text-xs font-medium text-text-secondary uppercase tracking-wide">{i18n.t('detail.limitations')}</h3>
          {#if finding.support?.limitations && finding.support.limitations.length > 0}
            <ul class="mt-2 list-disc text-xs text-text-secondary space-y-1 pl-4">
              {#each finding.support.limitations as limitation (limitation)}
                <li class="break-anywhere">{limitation}</li>
              {/each}
            </ul>
          {:else}
            <p class="mt-2 text-sm text-text-secondary/70">{i18n.t('detail.none')}</p>
          {/if}
        </div>

        {#if finding.recommendation}
          <div>
            <h3 class="text-xs font-medium text-text-secondary uppercase tracking-wide">{i18n.t('detail.recommendation')}</h3>
            <p class="break-anywhere text-sm text-text-secondary mt-1">{finding.recommendation}</p>
          </div>
        {/if}

        {#if finding.revision?.base || finding.revision?.head}
          <div class="border-t border-border-dark pt-4">
            <h3 class="text-xs font-medium text-text-secondary uppercase tracking-wide">{i18n.t('detail.revision')}</h3>
            <p class="break-anywhere font-mono text-xs text-text-secondary mt-1">
              {#if finding.revision.base}{i18n.t('detail.base')}: {finding.revision.base}{/if}
              {#if finding.revision.base && finding.revision.head} · {/if}
              {#if finding.revision.head}{i18n.t('detail.head')}: {finding.revision.head}{/if}
            </p>
          </div>
        {/if}
      </div>
    {/if}
  </div>
</aside>