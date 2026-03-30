<script lang="ts">
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';

  export let visible: boolean = false;
  export let x: number = 0;
  export let y: number = 0;
  export let hasSelection: boolean = false;

  const dispatch = createEventDispatcher();
  let menuEl: HTMLDivElement;

  function handleClickOutside(e: MouseEvent) {
    if (visible && menuEl && !menuEl.contains(e.target as Node)) {
      dispatch('close');
    }
  }

  onMount(() => {
    window.addEventListener('mousedown', handleClickOutside, true);
  });

  onDestroy(() => {
    window.removeEventListener('mousedown', handleClickOutside, true);
  });
</script>

{#if visible}
  <div class="ctx-menu" style="left: {x}px; top: {y}px;" bind:this={menuEl}>
    <button class="ctx-item" on:mousedown|stopPropagation on:click={() => dispatch('export', { format: 'svg' })}>
      <span class="ctx-label">Export as SVG</span>
      <span class="ctx-hint">{hasSelection ? 'Selected' : 'All'}</span>
    </button>
    <button class="ctx-item" on:mousedown|stopPropagation on:click={() => dispatch('export', { format: 'png' })}>
      <span class="ctx-label">Export as PNG</span>
      <span class="ctx-hint">{hasSelection ? 'Selected' : 'All'}</span>
    </button>
  </div>
{/if}

<style>
  .ctx-menu {
    position: fixed;
    z-index: 110;
    background: rgba(15, 23, 42, 0.95);
    border: 1px solid rgba(71, 85, 105, 0.6);
    border-radius: 8px;
    padding: 4px;
    min-width: 180px;
    backdrop-filter: blur(12px);
    font-size: 14px;
    color: #e2e8f0;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
  }

  .ctx-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 8px 12px;
    border: none;
    background: none;
    color: #e2e8f0;
    font-size: 14px;
    border-radius: 6px;
    cursor: pointer;
    text-align: left;
  }

  .ctx-item:hover {
    background: rgba(71, 85, 105, 0.4);
  }

  .ctx-label {
    font-weight: 500;
  }

  .ctx-hint {
    font-size: 11px;
    color: #64748b;
  }
</style>
