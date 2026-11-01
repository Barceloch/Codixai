<script lang="ts">
  import { i18n } from '../i18n.svelte'
  import { theme, type ThemePreference } from '../theme.svelte'
  import SelectMenu from './SelectMenu.svelte'

  const monitorIcon = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="3" width="20" height="14" rx="2"></rect><path d="M8 21h8M12 17v4"></path></svg>`

  const sunIcon = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4.5"></circle><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"></path></svg>`

  const moonIcon = `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path></svg>`

  const items = $derived([
    { id: 'system', label: i18n.t('header.themeSystem'), icon: monitorIcon },
    { id: 'light', label: i18n.t('header.themeLight'), icon: sunIcon },
    { id: 'dark', label: i18n.t('header.themeDark'), icon: moonIcon },
  ])
</script>

<SelectMenu
  label={i18n.t('header.theme')}
  {items}
  value={theme.preference}
  onselect={(id) => theme.setPreference(id as ThemePreference)}
  variant="icon"
/>
