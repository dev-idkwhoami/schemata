<script lang="ts">
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';

  export let visible: boolean = false;
  export let x: number = 0;
  export let y: number = 0;
  export let targets: { schema: string; table: string }[] = [];

  const dispatch = createEventDispatcher();
  let menuEl: HTMLDivElement;

  function handleNavigate(schema: string, table: string) {
    dispatch('navigate', { schema, table });
  }

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
  <div class="fk-menu" style="left: {x}px; top: {y}px;" bind:this={menuEl}>
    {#each targets as t (t.schema + '.' + t.table)}
      <button class="fk-menu-item" on:mousedown|stopPropagation on:click={() => handleNavigate(t.schema, t.table)}>
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
          <path d="M1 7h10M8 4l3 3-3 3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <span class="table-name">{t.table}</span>
        <span class="schema-name">{t.schema}</span>
      </button>
    {/each}
  </div>
{/if}

<style>
  .fk-menu {
    position: fixed;
    z-index: 110;
    background: rgba(15, 23, 42, 0.95);
    border: 1px solid rgba(71, 85, 105, 0.6);
    border-radius: 8px;
    padding: 4px;
    min-width: 180px;
    max-height: 320px;
    overflow-y: auto;
    backdrop-filter: blur(12px);
    font-size: 14px;
    color: #e2e8f0;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
    scrollbar-width: thin;
    scrollbar-color: rgba(71, 85, 105, 0.6) transparent;
  }

  .fk-menu::-webkit-scrollbar {
    width: 6px;
  }

  .fk-menu::-webkit-scrollbar-track {
    background: transparent;
  }

  .fk-menu::-webkit-scrollbar-thumb {
    background: rgba(71, 85, 105, 0.6);
    border-radius: 3px;
  }

  .fk-menu::-webkit-scrollbar-thumb:hover {
    background: rgba(100, 116, 139, 0.8);
  }

  .fk-menu-item {
    display: flex;
    align-items: center;
    gap: 8px;
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

  .fk-menu-item:hover {
    background: rgba(71, 85, 105, 0.4);
  }

  .fk-menu-item svg {
    flex-shrink: 0;
    color: #94a3b8;
  }

  .table-name {
    font-weight: 500;
  }

  .schema-name {
    color: #64748b;
    font-size: 12px;
    margin-left: auto;
  }
</style>
