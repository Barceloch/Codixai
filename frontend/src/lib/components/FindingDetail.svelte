<script lang="ts">
  import type { domain } from '../wailsjs/go/models'
  import { i18n } from '../i18n.svelte'
  import { hunkFor, hunkLinePrefix, locationText } from '../findings'

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
    return 'text-base-content/70'
  }
</script>

<aside class="card border border-base-300 bg-base-100 shadow-sm" aria-labelledby="detail-title">
  <div class="card-body gap-4">
    <h2 id="detail-title" class="card-title text-lg">{i18n.t('detail.title')}</h2>

    {#if !finding}
      <p class="text-sm text-base-content/70">{i18n.t('detail.empty')}</p>
    {:else}
      <div class="flex flex-wrap items-center gap-2">
        <span class="badge {finding.severity === 'error' ? 'badge-error' : finding.severity === 'warning' ? 'badge-warning' : 'badge-info'}">
          {i18n.t(`severity.${finding.severity}`)}
        </span>
        <span class="badge badge-outline">{i18n.t(`epistemic.${finding.epistemic_status}`)}</span>
        {#if finding.language}
          <span class="badge badge-ghost">{finding.language}</span>
        {/if}
        <span class="font-mono text-sm font-semibold">{finding.rule_id}</span>
        <span class="text-xs text-base-content/60">v{finding.rule_version}</span>
      </div>

      <div>
        <h3 class="text-xs font-semibold uppercase tracking-wide text-base-content/60">
          {i18n.t('detail.location')}
        </h3>
        <p class="break-anywhere font-mono text-sm">{locationText(finding.location)}</p>
      </div>

      <div>
        <h3 class="text-xs font-semibold uppercase tracking-wide text-base-content/60">
          {i18n.t('detail.message')}
        </h3>
        <p class="break-anywhere text-sm">{finding.message}</p>
      </div>

      {#if finding.evidence && finding.evidence.length > 0}
        <div>
          <h3 class="text-xs font-semibold uppercase tracking-wide text-base-content/60">
            {i18n.t('detail.evidence')}
          </h3>
          <ul class="mt-1 space-y-1">
            {#each finding.evidence as evidence, index (index)}
              <li class="rounded-box bg-base-200 px-3 py-2 text-xs">
                <span class="font-semibold">{evidence.kind}</span>
                <span class="break-anywhere font-mono text-base-content/80">{evidence.ref}</span>
                {#if evidence.detail}
                  <span class="block text-base-content/70">{evidence.detail}</span>
                {/if}
              </li>
            {/each}
          </ul>
        </div>
      {/if}

      <div>
        <h3 class="text-xs font-semibold uppercase tracking-wide text-base-content/60">
          {i18n.t('detail.hunk')}
        </h3>
        {#if hunk}
          <div class="mt-1 overflow-x-auto rounded-box bg-base-300 p-3">
            <p class="mb-2 font-mono text-xs text-base-content/70">
              {hunk.header || `@@ -${hunk.old_start},${hunk.old_lines} +${hunk.new_start},${hunk.new_lines} @@`}
            </p>
            <pre class="text-xs leading-5"><code>{#each hunk.lines as line, index (index)}<span class={lineClass(line.kind)}>{line.new_line ?? line.old_line ?? ''} {hunkLinePrefix(line.kind)}{line.content}
</span>{/each}</code></pre>
          </div>
        {:else}
          <p class="mt-1 text-sm text-base-content/70">{i18n.t('detail.hunkEmpty')}</p>
        {/if}
      </div>

      <div>
        <h3 class="text-xs font-semibold uppercase tracking-wide text-base-content/60">
          {i18n.t('detail.support')}
        </h3>
        <dl class="mt-1 grid gap-1 text-sm">
          <div class="flex gap-2">
            <dt class="text-base-content/60">{i18n.t('detail.supportLevel')}:</dt>
            <dd>{i18n.t(`support.${finding.support?.level ?? 'none'}`)}</dd>
          </div>
          {#if finding.support?.method}
            <div class="flex gap-2">
              <dt class="text-base-content/60">{i18n.t('detail.method')}:</dt>
              <dd class="break-anywhere">{finding.support.method}</dd>
            </div>
          {/if}
        </dl>
      </div>

      <div>
        <h3 class="text-xs font-semibold uppercase tracking-wide text-base-content/60">
          {i18n.t('detail.limitations')}
        </h3>
        {#if finding.support?.limitations && finding.support.limitations.length > 0}
          <ul class="mt-1 list-inside list-disc space-y-1 text-sm">
            {#each finding.support.limitations as limitation, index (index)}
              <li class="break-anywhere">{limitation}</li>
            {/each}
          </ul>
        {:else}
          <p class="mt-1 text-sm text-base-content/70">{i18n.t('detail.none')}</p>
        {/if}
      </div>

      {#if finding.recommendation}
        <div>
          <h3 class="text-xs font-semibold uppercase tracking-wide text-base-content/60">
            {i18n.t('detail.recommendation')}
          </h3>
          <p class="break-anywhere text-sm">{finding.recommendation}</p>
        </div>
      {/if}

      {#if finding.revision?.base || finding.revision?.head}
        <div>
          <h3 class="text-xs font-semibold uppercase tracking-wide text-base-content/60">
            {i18n.t('detail.revision')}
          </h3>
          <p class="break-anywhere font-mono text-xs">
            {#if finding.revision.base}{i18n.t('detail.base')}: {finding.revision.base}{/if}
            {#if finding.revision.base && finding.revision.head} · {/if}
            {#if finding.revision.head}{i18n.t('detail.head')}: {finding.revision.head}{/if}
          </p>
        </div>
      {/if}
    {/if}
  </div>
</aside>
