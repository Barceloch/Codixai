<script lang="ts">
  import { i18n } from '../i18n.svelte'
  import LanguageSelector from './LanguageSelector.svelte'
  import ThemeSelector from './ThemeSelector.svelte'

  // The header holds the identity of the window and the two switches. Both
  // switches are self-evident icon controls, so no visible label is needed.
</script>

<header class="topbar">
  <div class="topbar-inner">
    <div class="topbar-brand">
      <img class="topbar-mark" src="/images/logoH.png" alt="Codixai" />
      <span class="topbar-sep"></span>
      <span class="chip chip-accent">{i18n.t('app.localOnly')}</span>
    </div>
    <div class="flex items-center gap-2">
      <LanguageSelector />
      <ThemeSelector />
    </div>
  </div>
</header>
