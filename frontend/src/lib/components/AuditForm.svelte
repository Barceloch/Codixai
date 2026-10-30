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
    { value: 'worktree', label: i18n.t('origin.worktree'), description: 'All uncommitted changes in the working tree' },
    { value: 'staged', label: i18n.t('origin.staged'), description: 'Only changes added to the staging area' },
    { value: 'base', label: i18n.t('origin.base'), description: 'Compare against a specific base revision' },
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
    <div class="card-body gap-6">
      <div>
        <h2 id="audit-title" class="card-title text-lg text-primary-color">{i18n.t('audit.title')}</h2>
        <p class="text-sm text-text-secondary mt-1">{i18n.t('audit.idleHint')}</p>
      </div>

      <form class="grid gap-4 md:grid-cols-2" onsubmit={submit} autocomplete="off">
        <!-- Repository selector -->
        <div class="grid gap-2 md:col-span-2">
          <label class="label text-xs font-medium text-text-secondary" for="repository">
            {i18n.t('audit.repository')}
          </label>
          <div class="repository-input">
            <input
              id="repository"
              class="input input-bordered w-full bg-surface/50"
              type="text"
              spellcheck="false"
              placeholder={i18n.t('audit.repositoryPlaceholder')}
              bind:value={root}
              disabled={running}
              aria-describedby="repository-hint"
            />
            <button
              type="button"
              class="btn-glass btn-sm flex items-center gap-2"
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
          <p id="repository-hint" class="text-xs text-text-muted">{i18n.t('audit.repositoryPlaceholder')}</p>
        </div>

        <!-- Origin selector -->
        <div class="flex flex-col gap-2">
          <label class="label text-xs font-medium text-text-secondary" for="origin">
            {i18n.t('audit.origin')}
          </label>
          <div class="relative">
            <select
              id="origin"
              class="select select-bordered w-full bg-surface/50 appearance-none"
              bind:value={origin}
              disabled={running}
              onchange={() => base = ''}
            >
              {#each origins as opt (opt.value)}
                <option value={opt.value}>{opt.label}</option>
              {/each}
            </select>
            <div class="absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none">
              <svg class="w-4 h-4 text-text-secondary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
              </svg>
            </div>
          </div>
          <p class="text-xs text-text-muted">{i18n.t(`origin.${origin}Desc`)}</p>
        </div>

        <!-- Base revision (only for 'base' origin) -->
        {#if origin === 'base'}
          <div class="flex flex-col gap-2 md:col-span-2">
            <label class="label text-xs font-medium text-text-secondary" for="base">
              {i18n.t('audit.base')}
            </label>
            <input
              id="base"
              class="input input-bordered w-full bg-surface/50"
              type="text"
              spellcheck="false"
              placeholder={i18n.t('audit.basePlaceholder')}
              bind:value={base}
              disabled={running}
            />
            <p class="text-xs text-text-muted">{i18n.t('audit.basePlaceholder')}</p>
          </div>
        {/if}

        <!-- Action buttons -->
        <div class="flex flex-wrap items-center gap-3 md:col-span-2">
          <button
            type="submit"
            class="btn-primary-glass flex-1 sm:flex-none min-w-[160px] {running ? 'opacity-60 cursor-not-allowed' : ''}"
            disabled={!canRun || running}
          >
            {#if running}
              <span class="flex items-center justify-center gap-2">
                <span class="spinner-glass w-4 h-4"></span>
                {i18n.t('audit.running')}
              </span>
            {:else}
              {i18n.t('audit.run')}
            {/if}
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
  <div class="alert alert-warning glass-surface mt-4" role="status">
    <span>{i18n.t('audit.error.bridge')}</span>
  </div>
{/if}

{#if running}
  <div class="mt-4 animate-fade-in">
    <div class="flex items-center gap-3 mb-2">
      <div class="loading-spinner w-5 h-5"></div>
      <span class="text-sm font-medium text-primary-color">{progressText()}</span>
    </div>
    <progress class="progress progress-glass w-full" max="100" value={progress?.phase === 'collected' ? 50 : progress?.phase === 'analyzed' ? 100 : 0}></progress>
  </div>
{/if}

{#if error}
  <div class="alert-error-glass mt-4 animate-slide-up" role="alert">
    <div class="flex items-start gap-3">
      <svg class="w-5 h-5 mt-0.5 flex-shrink-0 text-error" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
      </svg>
      <div class="flex-1">
        <p class="font-medium text-error">{i18n.t('audit.error.title')}</p>
        <p class="text-sm text-error/90 mt-1">{error}</p>
      </div>
    </div>
  </div>
{/if}