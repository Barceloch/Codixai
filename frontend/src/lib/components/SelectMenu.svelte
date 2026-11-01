<script lang="ts">
  import { tick } from 'svelte'

  interface Item {
    id: string
    label: string
    meta?: string
    icon?: string
  }

  let {
    label,
    items = [],
    value = '',
    onselect,
    variant = 'compact',
    shortLabel = '',
  }: {
    label: string
    items?: Item[]
    value: string
    onselect: (id: string) => void
    variant?: 'compact' | 'icon'
    shortLabel?: string
  } = $props()

  let open = $state(false)
  let wrap = $state<HTMLDivElement | null>(null)
  let trigger = $state<HTMLButtonElement | null>(null)
  let popup = $state<HTMLDivElement | null>(null)

  const active = $derived(items.find((item) => item.id === value) ?? null)

  function itemButtons(): HTMLButtonElement[] {
    if (!popup) return []
    return Array.from(popup.querySelectorAll<HTMLButtonElement>('.menu-item'))
  }

  async function openMenu(position: 'first' | 'last' | null): Promise<void> {
    open = true
    if (!position) return
    await tick()
    const buttons = itemButtons()
    const target = position === 'first' ? buttons[0] : buttons[buttons.length - 1]
    target?.focus()
  }

  function close(refocus = false): void {
    open = false
    if (refocus) trigger?.focus()
  }

  function toggle(): void {
    if (open) {
      close()
    } else {
      void openMenu(null)
    }
  }

  function select(item: Item): void {
    onselect(item.id)
    close(true)
  }

  function moveFocus(event: KeyboardEvent): void {
    const buttons = itemButtons()
    if (buttons.length === 0) return
    const current = buttons.findIndex((button) => button === document.activeElement)
    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault()
        buttons[Math.min(current + 1, buttons.length - 1)]?.focus()
        break
      case 'ArrowUp':
        event.preventDefault()
        buttons[Math.max(current - 1, 0)]?.focus()
        break
      case 'Home':
        event.preventDefault()
        buttons[0]?.focus()
        break
      case 'End':
        event.preventDefault()
        buttons[buttons.length - 1]?.focus()
        break
      case 'Escape':
        event.preventDefault()
        close(true)
        break
      case 'Tab':
        close()
        break
    }
  }

  function triggerKeydown(event: KeyboardEvent): void {
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      void openMenu('first')
    } else if (event.key === 'ArrowUp') {
      event.preventDefault()
      void openMenu('last')
    } else if (event.key === 'Escape' && open) {
      close(true)
    } else if (event.key === 'Tab' && open) {
      close()
    }
  }

  $effect(() => {
    if (!open) return
    const onPointerDown = (event: PointerEvent) => {
      if (wrap && !wrap.contains(event.target as Node)) close()
    }
    window.addEventListener('pointerdown', onPointerDown)
    return () => window.removeEventListener('pointerdown', onPointerDown)
  })
</script>

<div class="menu-wrap" bind:this={wrap}>
  <button
    type="button"
    class="icon-btn"
    aria-haspopup="true"
    aria-expanded={open}
    aria-label={label}
    title={label}
    onclick={toggle}
    onkeydown={triggerKeydown}
    bind:this={trigger}
  >
    {#if active?.icon}
      <span aria-hidden="true">
        <!-- eslint-disable-next-line svelte/no-at-html-tags -->
        {@html active.icon}
      </span>
    {/if}
    {#if variant === 'compact' && shortLabel}
      <span class="text-[11px] font-semibold tracking-[0.08em]">{shortLabel}</span>
    {/if}
  </button>

  {#if open}
    <div class="menu" tabindex="-1" bind:this={popup}>
      {#each items as item (item.id)}
        <button
          type="button"
          class="menu-item"
          class:is-active={item.id === value}
          onclick={() => select(item)}
          onkeydown={moveFocus}
        >
          {#if item.icon}
            <span aria-hidden="true">
              <!-- eslint-disable-next-line svelte/no-at-html-tags -->
              {@html item.icon}
            </span>
          {/if}
          <span>{item.label}</span>
          {#if item.meta}
            <span class="menu-meta">{item.meta}</span>
          {/if}
          {#if item.id === value}
            <svg class="menu-check" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
            </svg>
          {/if}
        </button>
      {/each}
    </div>
  {/if}
</div>
