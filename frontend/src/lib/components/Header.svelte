<script lang="ts">
  import { i18n, type Locale } from '../i18n.svelte'
  import { theme, type ThemePreference } from '../theme.svelte'

  function changeLocale(event: Event): void {
    const value = (event.currentTarget as HTMLSelectElement).value
    if (value === 'en' || value === 'es') {
      i18n.setLocale(value as Locale)
    }
  }

  function changeTheme(event: Event): void {
    const value = (event.currentTarget as HTMLSelectElement).value
    if (value === 'system' || value === 'light' || value === 'dark') {
      theme.setPreference(value as ThemePreference)
    }
  }
</script>

<header class="glass-surface border-b border border-base-300 bg-base-100/80 backdrop-blur-lg">
  <div class="mx-auto flex max-w-7xl flex-col gap-3 px-4 py-3 lg:flex-row lg:items-center lg:justify-between">
    <div>
      <div class="flex items-center gap-2">
        <h1 class="text-2xl font-bold tracking-tight text-primary-color">{i18n.t('app.title')}</h1>
        <span class="badge badge-glass badge-sm text-primary-color">{i18n.t('app.localOnly')}</span>
      </div>
      <p class="text-sm text-base-content/70 text-primary-color/60">{i18n.t('app.subtitle')}</p>
    </div>

    <div class="flex flex-wrap items-end gap-3">
      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium uppercase text-base-content/60">Lenguaje</span>
        <select
          class="select select-bordered select-sm"
          value={i18n.locale}
          onchange={changeLocale}
          aria-label={i18n.t('header.language')}
        >
          <option value="en">English</option>
          <option value="es">Español</option>
        </select>
      </label>

      <label class="flex flex-col gap-1">
        <span class="text-xs font-medium uppercase text-base-content/60">Tema</span>
        <select
          class="select select-bordered select-sm"
          value={theme.preference}
          onchange={changeTheme}
          aria-label={i18n.t('header.theme')}
        >
          <option value="system">{i18n.t('header.themeSystem')}</option>
          <option value="light">{i18n.t('header.themeLight')}</option>
          <option value="dark">{i18n.t('header.themeDark')}</option>
        </select>
      </label>
    </div>
  </div>
</header>