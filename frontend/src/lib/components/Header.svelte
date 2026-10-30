<script lang="ts">
  import { i18n } from '../i18n.svelte'
  import LanguageSelector from './LanguageSelector.svelte'
  import ThemeSelector from './ThemeSelector.svelte'

  // The header holds the identity of the window and the two switches.
  // The language and theme selectors are compact dropdowns with full
  // names, so the header is readable and the controls are discoverable.
</script>

<header class="glass-surface border-b border border-base-300 bg-base-100/80 backdrop-blur-lg">
  <div class="mx-auto flex w-full max-w-7xl flex-col gap-3 px-4 py-3 lg:flex-row lg:items-center lg:justify-between">
    <div class="flex items-center gap-4 min-w-0">
      <!-- Logo horizontal -->
      <img
        src="/images/logoH.png"
        alt="Codixai"
        class="h-9 w-auto max-w-[200px] lg:max-w-[280px] shrink-0"
        loading="eager"
      />
      <span class="badge badge-glass text-xs text-primary-color whitespace-nowrap shrink-0">
        {i18n.t('app.localOnly')}
      </span>
    </div>

    <div class="flex items-center gap-3 shrink-0">
      <LanguageSelector />
      <ThemeSelector />
    </div>
  </div>
</header>