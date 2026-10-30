<script lang="ts">
  import { i18n, type Locale } from '../i18n.svelte'
  import { theme, type ThemePreference } from '../theme.svelte'

  // The header holds the two switches of the window. They are icon only: the
  // flags identify the language and the sun/moon identifies the theme, so no
  // label is needed and the switch is deducible by itself.

  const locales: { value: Locale; flag: string; title: string }[] = [
    { value: 'en', flag: '🇬🇧', title: 'English' },
    { value: 'es', flag: '🇪🇸', title: 'Español' },
  ]

  function setLocale(locale: Locale): void {
    i18n.setLocale(locale)
  }

  const themes: { value: ThemePreference; icon: string; title: string }[] = [
    { value: 'light', icon: '☀️', title: 'Tema claro' },
    { value: 'dark', icon: '🌙', title: 'Tema oscuro' },
  ]

  function setTheme(preference: ThemePreference): void {
    theme.setPreference(preference)
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

    <div class="flex flex-wrap items-center gap-2">
      <!-- Language switch: flags only -->
      <div class="join join-horizontal">
        {#each locales as loc}
          <button
            class="btn btn-sm btn-ghost join-item"
            class:btn-active={i18n.locale === loc.value}
            onclick={() => setLocale(loc.value)}
            title={loc.title}
            aria-label={loc.title}
          >
            <span class="text-base leading-none">{loc.flag}</span>
          </button>
        {/each}
      </div>

      <!-- Theme switch: sun/moon only -->
      <div class="join join-horizontal">
        {#each themes as t}
          <button
            class="btn btn-sm btn-ghost join-item"
            class:btn-active={theme.preference === t.value}
            onclick={() => setTheme(t.value)}
            title={t.title}
            aria-label={t.title}
          >
            <span class="text-base leading-none">{t.icon}</span>
          </button>
        {/each}
      </div>
    </div>
  </div>
</header>