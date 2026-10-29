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

<aside class="card-glow glass-surface" aria-labelledby="detail-title">
  <div class="card-body gap-4">
    <h2 id="detail-title" class="card-title text-lg text-primary-color">{i18n.t('detail.title')}</h2>

    {#if !finding}
      <p class="text-sm text-base-content/70 text-primary-color/60">{i18n.t('detail.empty')}</p>
    {:else}
      <div class="flex flex-wrap items-center gap-2">
        <span class="badge badge-primary-glass me-2">
          {i18n.t(`severity.${finding.severity}`)}
        </span>
        <span class="badge badge-outline text-xs text-primary-color/70">
          {i18n.t(`epistemic.${finding.epistemic_status}`)}
        </span>
        {#if finding.language}
          <span class="badge badge-ghost text-xs text-primary-color/70">{finding.language}</span>
        {/if}
        <span class="font-mono text-sm font-semibold text-primary-color">{finding.rule_id}</span>
        <span class="text-xs text-base-content/60 text-primary-color/60">v{finding.rule_version}</span>
      </div>

      <div>
        <h3 class="text-xs font-semibold uppercase text-base-content/60 tracking-wide text-primary-color">{i18n.t('detail.location')}</h3>
        <p class="break-anywhere text-sm text-primary-color/70">{locationText(finding.location)}</p>
      </div>

      <div>
        <h3 class="text-xs font-semibold uppercase text-base-content/60 tracking-wide text-primary-color">{i18n.t('detail.message')}</h3>
        <p class="break-anywhere text-sm text-primary-color/70">{finding.message}</p>
      </div>

      {#if finding.evidence && finding.evidence.length > 0}
        <div>
          <h3 class="text-xs font-semibold uppercase text-base-content/60 text-primary-color">{i18n.t('detail.evidence')}</h3>
          <ul class="mt-1 space-y-1">
            {#each finding.evidence as evidence, index}
              <li class="rounded-box bg-base-200 px-3 py-2 text-xs text-primary-color/80">
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
        <h3 class="text-xs font-semibold uppercase text-base-content/60 text-primary-color">{i18n.t('detail.support')}</h3>
        <dl class="mt-1 grid gap-1 text-sm">
          <div class="flex gap-2 text-xs text-primary-color/60">
            <dt>{i18n.t('detail.supportLevel')}:</dt>
            <dd>{i18n.t(`support.${finding.support?.level ?? 'none'}`)}</dd>
          </div>
          {#if finding.support?.method}
            <div class="flex gap-2 text-xs text-primary-color/60">
              <dt>{i18n.t('detail.method')}:</dt>
              <dd class="break-anywhere text-primary-color/60">{finding.support.method}</dd>
            </div>
          {/if}
        </dl>
      </div>

      <div>
        <h3 class="text-xs font-semibold uppercase text-base-content/60 text-primary-color">{i18n.t('detail.limitations')}</h3>
        {#if finding.support?.limitations && finding.support.limitations.length > 0}
          <ul class="mt-1 list-disc text-xs text-primary-color/70 space-y-1">
            {#each finding.support.limitations as limitation}
              <li class="break-anywhere text-primary-color/70">{limitation}</li>
            {/each}
          {:else}
            <p class="mt-1 text-sm text-base-content/70 text-primary-color/60">{i18n.t('detail.none')}</p>
          {/if}
        </div>

        {#if finding.recommendation}
          <div>
            <h3 class="text-xs font-semibold uppercase text-base-content/60 text-primary-color">{i18n.t('detail.recommendation')}</h3>
            <p class="break-anywhere text-sm text-primary-color/70">{finding.recommendation}</p>
          </div>
        {/if}

        {#if finding.revision?.base || finding.revision?.head}
          <div>
            <h3 class="text-xs font-semibold uppercase text-base-content/60 text-primary-color">{i18n.t('detail.revision')}</h3>
            <p class="break-anywhere font-mono text-xs text-primary-color/70">
              {#if finding.revision.base}{i18n.t('detail.base')}: {finding.revision.base}{/if}
              {#if finding.revision.base && finding.revision.head} · {/if}
              {#if finding.revision.head}{i18n.t('detail.head')}: {finding.revision.head}{/if}
            </p>
          </div>
        {/if}
      </div>
    {:else}
      <p class="text-sm text-base-content/70 text-primary-color/60">{i18n.t('detail.empty')}</p>
    {/if}
</aside>