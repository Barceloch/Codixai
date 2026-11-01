<script lang="ts">
  import { i18n, type Locale } from '../i18n.svelte'
  import SelectMenu from './SelectMenu.svelte'

  const globeIcon = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9"></circle><path d="M3.6 9h16.8M3.6 15h16.8"></path><path d="M12 3c2.4 2.4 3.6 5.4 3.6 9s-1.2 6.6-3.6 9c-2.4-2.4-3.6-5.4-3.6-9S9.6 5.4 12 3z"></path></svg>`

  const items = [
    { id: 'en', label: 'English', meta: 'EN', icon: globeIcon },
    { id: 'es', label: 'Español', meta: 'ES', icon: globeIcon },
  ]
</script>

<SelectMenu
  label={i18n.t('header.language')}
  {items}
  value={i18n.locale}
  shortLabel={i18n.locale.toUpperCase()}
  onselect={(id) => i18n.setLocale(id as Locale)}
  variant="compact"
/>
