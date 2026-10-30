<script lang="ts">
  interface DropdownOption {
    value: string
    label: string
    description?: string
    selected?: boolean
    icon?: string
  }

  let {
    options = [],
    value = $bindable(''),
    placeholder = '',
    disabled = false,
    onselect,
    class: className = '',
  }: {
    options: DropdownOption[]
    value: string
    placeholder?: string
    disabled?: boolean
    onselect?: (value: string) => void
    class?: string
  } = $props()

  let open = $state(false)
  let container = $state<HTMLElement | null>(null)
  let button = $state<HTMLElement | null>(null)

  const selectedOption = $derived.by(() => options.find(opt => opt.value === value))

  function toggle() {
    if (!disabled) {
      open = !open
    }
  }

  function selectOption(option: DropdownOption) {
    if (option.value !== value) {
      value = option.value
      onselect?.(option.value)
    }
    close()
  }

  function close() {
    open = false
  }

  function handleKeydown(event: KeyboardEvent) {
    if (disabled) return

    switch (event.key) {
      case 'Enter':
      case ' ':
        event.preventDefault()
        open = true
        break
      case 'Escape':
        close()
        button?.focus()
        break
    }
  }

  function handleMenuKeydown(event: KeyboardEvent) {
    if (!open) return

    const items = Array.from(container?.querySelectorAll<HTMLButtonElement>('.dropdown-item') || [])
    const currentIndex = items.findIndex(item => document.activeElement === item)

    switch (event.key) {
      case 'ArrowDown': {
        event.preventDefault()
        const nextIndex = currentIndex === -1 ? 0 : Math.min(currentIndex + 1, items.length - 1)
        items[nextIndex]?.focus()
        break
      }
      case 'ArrowUp': {
        event.preventDefault()
        const prevIndex = currentIndex === -1 ? items.length - 1 : Math.max(currentIndex - 1, 0)
        items[prevIndex]?.focus()
        break
      }
      case 'Enter':
      case ' ': {
        event.preventDefault()
        items[currentIndex]?.click()
        break
      }
    }
  }

  // Close on outside click
  function handleClickOutside(event: MouseEvent) {
    if (container && !container.contains(event.target as Node)) {
      open = false
    }
  }

  // Attach and clean up event listeners
  $effect(() => {
    if (open) {
      document.addEventListener('click', handleClickOutside)
      return () => {
        document.removeEventListener('click', handleClickOutside)
      }
    }
  })
</script>

<div class="dropdown {className}" bind:this={container}>
  <button
    type="button"
    class="dropdown-toggle btn-glass btn-sm {selectedOption?.value ? 'btn-active' : ''} flex items-center gap-2"
    class:disabled
    onclick={toggle}
    onkeydown={handleKeydown}
    bind:this={button}
    aria-haspopup="listbox"
    aria-expanded={open}
    aria-disabled={disabled}
  >
    <span class="flex items-center gap-2">
      {#if selectedOption?.icon}
        <!-- eslint-disable-next-line svelte/no-at-html-tags -->
        {@html selectedOption.icon}
      {/if}
      <span class="text-sm font-medium">
        {selectedOption?.value ? selectedOption.value : placeholder || 'Select...'}
      </span>
      {#if selectedOption?.value}
        <span class="text-xs text-text-secondary">({selectedOption.value})</span>
      {/if}
    </span>
    <svg class="w-4 h-4 text-text-secondary transition-transform {open ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
    </svg>
  </button>

  {#if open}
    <div
      class="dropdown-menu"
      onkeydown={handleMenuKeydown}
      role="listbox"
      aria-labelledby="dropdown-selected"
    >
      {#each options as option (option.value)}
        <button
          type="button"
          class="dropdown-item {option.selected ? 'selected' : ''} {option.value === value ? 'selected' : ''}"
          onclick={() => selectOption(option)}
          role="option"
          aria-selected={option.value === value}
          tabindex={option.value === value ? 0 : -1}
        >
          <div class="flex flex-col items-start">
            <div class="flex items-center gap-2">
              {#if option.icon}
                <!-- eslint-disable-next-line svelte/no-at-html-tags -->
                {@html option.icon}
              {/if}
              <span class="font-medium {option.value === value ? 'text-primary' : ''}">
                {option.label}
              </span>
              {#if option.selected}
                <svg class="w-3 h-3 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                </svg>
              {/if}
            </div>
            {#if option.description}
              <div class="text-xs text-text-secondary mt-1 ml-6">
                {option.description}
              </div>
            {/if}
          </div>
        </button>
      {/each}
    </div>
  {/if}
</div>