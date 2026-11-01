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

  const origins = [
    { value: 'worktree', label: i18n.t('origin.worktree') },
    { value: 'staged', label: i18n.t('origin.staged') },
    { value: 'base', label: i18n.t('origin.base') },
  ] as const

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

  function progressPercent(): number {
    if (progress?.phase === 'collected') return 50
    if (progress?.phase === 'analyzed') return 100
    return 0
  }

  function submit(event: SubmitEvent): void {
    event.preventDefault()
    onrun()
  }

  const canRun = $derived(
    !running && root.trim().length > 0 && (origin !== 'base' || base.trim().length > 0),
  )
</script>

<section class="panel panel-pad fade-in" aria-labelledby="audit-title">
  <div class="mb-5">
    <h2 id="audit-title" class="panel-title">{i18n.t('audit.title')}</h2>
    <p class="panel-sub mt-1">{i18n.t('audit.idleHint')}</p>
  </div>

  <form class="grid gap-5 md:grid-cols-2" onsubmit={submit} autocomplete="off">
    <!-- Repository selector -->
    <div class="field md:col-span-2">
      <label class="field-label" for="repository">{i18n.t('audit.repository')}</label>
      <div class="field-row">
        <input
          id="repository"
          class="input input-mono"
          type="text"
          spellcheck="false"
          placeholder={i18n.t('audit.repositoryPlaceholder')}
          bind:value={root}
          disabled={running}
          aria-describedby="repository-hint"
        />
        <button
          type="button"
          class="btn btn-ghost btn-sm"
          onclick={onchoose}
          disabled={running || !bridgeAvailable}
          aria-label={i18n.t('audit.chooseRepositoryTitle')}
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-6l-2-2H5a2 2 0 0 0-2 2z"></path>
          </svg>
          <span class="hidden sm:inline">{i18n.t('audit.chooseRepository')}</span>
        </button>
      </div>
      <p id="repository-hint" class="field-hint">{i18n.t('audit.repositoryPlaceholder')}</p>
    </div>

    <!-- Origin selector -->
    <div class="field">
      <label class="field-label" for="origin">{i18n.t('audit.origin')}</label>
      <div class="select-wrap">
        <select
          id="origin"
          class="select"
          bind:value={origin}
          disabled={running}
          onchange={() => base = ''}
        >
          {#each origins as opt (opt.value)}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>
        <svg class="select-chevron" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
        </svg>
      </div>
      <p class="field-hint">{i18n.t(`origin.${origin}Desc`)}</p>
    </div>

    <!-- Base revision (only for 'base' origin) -->
    {#if origin === 'base'}
      <div class="field md:col-span-2">
        <label class="field-label" for="base">{i18n.t('audit.base')}</label>
        <input
          id="base"
          class="input input-mono"
          type="text"
          spellcheck="false"
          placeholder={i18n.t('audit.basePlaceholder')}
          bind:value={base}
          disabled={running}
        />
        <p class="field-hint">{i18n.t('audit.basePlaceholder')}</p>
      </div>
    {/if}

    <!-- Action buttons -->
    <div class="flex flex-wrap items-center gap-3 md:col-span-2">
      <button
        type="submit"
        class="btn btn-primary flex-1 sm:flex-none min-w-[160px]"
        disabled={!canRun || running}
      >
        {#if running}
          <span class="spinner"></span>
          {i18n.t('audit.running')}
        {:else}
          {i18n.t('audit.run')}
        {/if}
      </button>
      {#if running}
        <button type="button" class="btn btn-ghost btn-sm" onclick={oncancel}>
          {i18n.t('audit.cancel')}
        </button>
      {/if}
    </div>
  </form>

  {#if !bridgeAvailable}
    <div class="callout callout-warning mt-5" role="status">
      <svg class="mt-0.5 h-4 w-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
      </svg>
      <span>{i18n.t('audit.error.bridge')}</span>
    </div>
  {/if}

  {#if running}
    <div class="mt-5 flex flex-col gap-2">
      <div class="flex items-center gap-3">
        <span class="spinner"></span>
        <span class="text-sm font-medium text-text-primary">{progressText()}</span>
      </div>
      <div class="meter">
        <div class="meter-fill" style="width: {progressPercent()}%"></div>
      </div>
    </div>
  {/if}

  {#if error}
    <div class="callout callout-error mt-5" role="alert">
      <svg class="mt-0.5 h-4 w-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
      </svg>
      <div class="min-w-0 flex-1">
        <p class="font-medium">{i18n.t('audit.error.title')}</p>
        <p class="mt-1 text-sm">{error}</p>
      </div>
    </div>
  {/if}
</section>
