<script lang="ts">
  import { i18n } from '../i18n.svelte'
  import { theme, type ThemePreference } from '../theme.svelte'

  const themes: { value: ThemePreference; icon: string; title: string }[] = [
    {
      value: 'system',
      title: i18n.t('header.themeSystem'),
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
  <line x1="8" y1="21" x2="16" y2="21"></line>
  <line x1="12" y1="17" x2="12" y2="21"></line>
</svg>`
    },
    {
      value: 'light',
      title: i18n.t('header.themeLight'),
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <circle cx="12" cy="12" r="5"></circle>
  <path d="M12 1v2M12 21v2M4.2 4.2l1.4 1.4M18.4 18.4l1.4 1.4M1 12h2M21 12h2M4.2 19.8l1.4-1.4M18.4 5.6l1.4 1.4"></path>
</svg>`
    },
    {
      value: 'dark',
      title: i18n.t('header.themeDark'),
      icon: `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
  <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
</svg>`
    }
  ]

  function setTheme(preference: ThemePreference): void {
    theme.setPreference(preference)
  }

  let open = false
</script>

<div class="flex flex-wrap items-center gap-2">
  <div class="text-xs font-medium text-text-secondary uppercase tracking-wide">
    {i18n.t('header.theme')}:
  </div>
  <div class="dropdown">
    <button
      type="button"
      class="btn-glass btn-sm flex items-center gap-2 {theme.preference === 'system' ? 'btn-active' : ''} {theme.preference === 'light' ? 'btn-active' : ''} {theme.preference === 'dark' ? 'btn-active' : ''}"
      onclick={() => open = true}
      aria-haspopup="listbox"
      aria-expanded={open}
    >
      <span class="text-base leading-none">
        {#if theme.preference === 'system'}
          <div class="flex items-center gap-1">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
              <line x1="8" y1="21" x2="16" y2="21"></line>
              <line x1="12" y1="17" x2="12" y2="21"></line>
            </svg>
          </div>
        {:else if theme.preference === 'light'}
          <div class="flex items-center gap-1">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <circle cx="12" cy="12" r="5"></circle>
              <path d="M12 1v2M12 21v2M4.2 4.2l1.4 1.4M18.4 18.4l1.4 1.4M1 12h2M21 12h2M4.2 19.8l1.4-1.4M18.4 5.6l1.4 1.4"></path>
            </svg>
          </div>
        {:else}
          <div class="flex items-center gap-1">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
            </svg>
          </div>
        {/if}
      </span>
      <span class="text-sm font-medium">
        {theme.preference === 'system' ? 'System' : theme.preference === 'light' ? 'Light' : 'Dark'}
      </span>
      <svg class="w-3 h-3 text-text-secondary transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
      </svg>
    </button>

    {#if open}
      <div
        class="dropdown-menu min-w-[180px]"
        role="listbox"
        aria-label="Theme options"
      >
        {#each themes as t (t.value)}
          <button
            type="button"
            class="dropdown-item {theme.preference === t.value ? 'selected' : ''}"
            onclick={() => setTheme(t.value)}
            role="option"
            aria-selected={theme.preference === t.value}
          >
            <div class="flex items-center gap-3">
              <span class="text-lg leading-none">
                <!-- eslint-disable-next-line svelte/no-at-html-tags -->
                {@html t.icon}
              </span>
              <span class="font-medium">
                {t.title}
              </span>
              <span class="text-xs text-text-secondary ml-auto">
                {t.value === 'system' ? 'Auto' : t.value === 'light' ? '☀️' : '🌙'}
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