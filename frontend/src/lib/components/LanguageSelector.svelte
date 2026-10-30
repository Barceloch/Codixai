<script lang="ts">
  import { i18n, type Locale } from '../i18n.svelte'

  const locales: { value: Locale; flag: string; title: string; icon: string }[] = [
    {
      value: 'en',
      flag: '🇬🇧',
      title: 'English',
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <circle cx="12" cy="12" r="10"></circle>
  <path d="M12 2a15.3 15.3 0 0 1 4 10.5V12a10 10 0 1 0 10 10 15.3-4 15.3 0 0 1-4-10.5V12A15.3 15.3 0 0 1 12 2z"></path>
</svg>`
    },
    {
      value: 'es',
      flag: '🇪🇸',
      title: 'Español',
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <circle cx="12" cy="12" r="10"></circle>
  <path d="M12 2a15.3 15.3 0 0 1 4 10.5V12a10 10 0 1 0 10 10 15.3-4 15.3 0 0 1-4-10.5V12A15.3 15.3 0 0 1 12 2z"></path>
  <path d="M12 12l-4-4"></path>
  <path d="M12 12l4 4"></path>
  <path d="M12 12l-4 4"></path>
  <path d="M12 12l4-4"></path>
</svg>`
    }
  ]

  function setLocale(locale: Locale): void {
    i18n.setLocale(locale)
  }

  let open = false
</script>

<div class="flex flex-wrap items-center gap-2">
  <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">
    {i18n.t('header.language')}:
  </div>
  <div class="dropdown">
    <button
      type="button"
      class="btn-glass btn-sm flex items-center gap-2 {i18n.locale === 'en' ? 'btn-active' : ''} {i18n.locale === 'es' ? 'btn-active' : ''}"
      onclick={() => open = true}
      aria-haspopup="listbox"
      aria-expanded={open}
    >
      <span class="text-base leading-none">
        {i18n.locale === 'en' ? '🇬🇧' : '🇪🇸'}
      </span>
      <span class="text-sm font-medium">
        {i18n.locale.toUpperCase()}
      </span>
      <svg class="w-3 h-3 text-text-secondary transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
      </svg>
    </button>

    {#if open}
      <div
        class="dropdown-menu min-w-[180px]"
        role="listbox"
        aria-label="Language options"
        onclick={(e) => e.stopPropagation()}
      >
        {#each locales as loc (loc.value)}
          <button
            type="button"
            class="dropdown-item {i18n.locale === loc.value ? 'selected' : ''}"
            onclick={() => setLocale(loc.value)}
            role="option"
            aria-selected={i18n.locale === loc.value}
          >
            <div class="flex items-center gap-3">
              <span class="text-lg leading-none">
                {loc.flag}
              </span>
              <span class="font-medium">
                {loc.title}
              </span>
              <span class="text-xs text-text-secondary ml-auto">
                {loc.value.toUpperCase()}
              </span>
            </div>
          </button>
        {/each}
      </div>
      <button
        class="dropdown-backdrop"
        type="button"
        aria-hidden="true"
        tabindex="-1"
        onclick={() => open = false}
        onkeydown={(e) => { if (e.key === 'Escape') open = false; }}
      ></button>
    {/if}
  </div>
</div>