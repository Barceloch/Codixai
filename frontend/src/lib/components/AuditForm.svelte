<script lang="ts">
  import type { ui } from '../wailsjs/go/models'
  import { i18n } from '../i18n.svelte'

  let {
    root = $bindable(''),
    origin = $bindable('worktree'),
    base = $bindable(''),
    running,
    progress,
    error,
    bridgeAvailable,
    onrun,
    oncancel,
    onchoose,
  }: {
    root: string
    origin: string
    base: string
    running: boolean
    progress: ui.AuditProgress | null
    error: string
    bridgeAvailable: boolean
    onrun: () => void
    oncancel: () => void
    onchoose: () => void
  } = $props()

  const origins = ['worktree', 'staged', 'base'] as const

  function progressText(): string {
    if (!progress) {
      return i18n.t('audit.progress.validating')
    }
    const key = `audit.progress.${progress.phase}`
    if (progress.phase === 'collected') {
      return i18n.t(key, { count: progress.changed_files ?? 0 })
    }
    if (progress.phase === 'analyzed') {
      return i18n.t(key, { count: progress.findings ?? 0 })
    }
    return i18n.t(key)
  }

  function submit(event: SubmitEvent): void {
    event.preventDefault()
    onrun()
  }

  const canRun = $derived(
    !running && root.trim().length > 0 && (origin !== 'base' || base.trim().length > 0),
  )
</script>

<section class="card border border-base-300 bg-base-100 shadow-sm" aria-labelledby="audit-title">
  <div class="card-body gap-4">
    <h2 id="audit-title" class="card-title text-lg">{i18n.t('audit.title')}</h2>

    <form class="grid gap-4 md:grid-cols-2" onsubmit={submit} autocomplete="off">
      <div class="flex flex-col gap-1 md:col-span-2">
        <label class="text-sm font-medium" for="repository">{i18n.t('audit.repository')}</label>
        <div class="join w-full">
          <input
            id="repository"
            class="input input-bordered join-item w-full"
            type="text"
            spellcheck="false"
            placeholder={i18n.t('audit.repositoryPlaceholder')}
            bind:value={root}
            disabled={running}
          />
          <button
            type="button"
            class="btn join-item"
            onclick={onchoose}
            disabled={running || !bridgeAvailable}
          >
            {i18n.t('audit.chooseRepository')}
          </button>
        </div>
      </div>

      <div class="flex flex-col gap-1">
        <label class="text-sm font-medium" for="origin">{i18n.t('audit.origin')}</label>
        <select
          id="origin"
          class="select select-bordered w-full"
          bind:value={origin}
          disabled={running}
        >
          {#each origins as value (value)}
            <option value={value}>{i18n.t(`origin.${value}`)}</option>
          {/each}
        </select>
      </div>

      {#if origin === 'base'}
        <div class="flex flex-col gap-1">
          <label class="text-sm font-medium" for="base">{i18n.t('audit.base')}</label>
          <input
            id="base"
            class="input input-bordered w-full"
            type="text"
            spellcheck="false"
            placeholder={i18n.t('audit.basePlaceholder')}
            bind:value={base}
            disabled={running}
          />
        </div>
      {/if}

      <div class="flex flex-wrap items-center gap-2 md:col-span-2">
        <button type="submit" class="btn btn-primary" disabled={!canRun}>
          {running ? i18n.t('audit.running') : i18n.t('audit.run')}
        </button>
        {#if running}
          <button type="button" class="btn btn-ghost" onclick={oncancel}>
            {i18n.t('audit.cancel')}
          </button>
        {/if}
      </div>
    </form>

    {#if !bridgeAvailable}
      <div class="alert alert-warning" role="status">
        <span>{i18n.t('audit.error.bridge')}</span>
      </div>
    {/if}

    {#if running}
      <div class="alert" role="status" aria-live="polite">
        <span class="loading loading-spinner loading-sm"></span>
        <div>
          <p class="font-medium">{progressText()}</p>
          {#if progress && progress.changed_files}
            <p class="text-xs text-base-content/70">
              {i18n.t('summary.changedFiles')}: {progress.changed_files}
            </p>
          {/if}
        </div>
      </div>
    {/if}

    {#if error}
      <div class="alert alert-error" role="alert">
        <div>
          <p class="font-semibold">{i18n.t('audit.error.title')}</p>
          <p class="break-anywhere text-sm">{error}</p>
        </div>
      </div>
    {/if}
  </div>
</section>
