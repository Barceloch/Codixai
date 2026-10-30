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

<section class="section-glass glass-surface" aria-labelledby="audit-title">
  <div class="card-glass card-glass-elevated">
    <div class="card-body gap-4">
      <h2 id="audit-title" class="card-title text-lg text-primary-color">{i18n.t('audit.title')}</h2>

      <form class="grid gap-4 md:grid-cols-2" onsubmit={submit} autocomplete="off">
        <div class="grid gap-1 md:col-span-2">
          <label class="label text-xs text-primary-color/60" for="repository">{i18n.t('audit.repository')}</label>
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
              class="btn-glass btn-sm"
              onclick={onchoose}
              disabled={running || !bridgeAvailable}
            >
              {i18n.t('audit.chooseRepository')}
            </button>
          </div>
        </div>

        <div class="flex flex-col gap-1">
          <label class="label text-xs text-primary-color/60" for="origin">{i18n.t('audit.origin')}</label>
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
            <label class="label text-xs text-primary-color/60" for="base">{i18n.t('audit.base')}</label>
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
          <button
            type="submit"
            class="btn btn-primary-glass w-full disabled={!canRun}"
            >
              {running ? i18n.t('audit.running') : i18n.t('audit.run')}
            </button>
            {#if running}
              <button type="button" class="btn-glass btn-sm" onclick={oncancel}>
                {i18n.t('audit.cancel')}
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
    <p class="text-sm text-base-content/70 text-primary-color/60">{progress?.phase}</p>
  </div>
{/if}

{#if error}
  <div class="alert alert-error-glass" role="alert">
    <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-4 w-4" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m-2 2l-2 2m2 2l2 2m-2 2l-2 2m2 2l2 2" /></svg>
    <span class="text-primary-color">{error}</span>
  </div>
{/if}