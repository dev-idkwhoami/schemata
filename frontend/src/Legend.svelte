<script lang="ts">
  import type { Schema } from './types';

  export let schemas: Schema[];

  $: visibleSchemas = schemas.length === 1 && schemas[0].name === 'public' ? [] : schemas;
</script>

{#if visibleSchemas.length > 0}
  <div class="legend">
    {#each visibleSchemas as schema (schema.name)}
      <div class="legend-item">
        <span class="swatch" style="background-color: {schema.color};" />
        <span class="label">{schema.name}</span>
      </div>
    {/each}
  </div>
{/if}

<style>
  .legend {
    position: fixed;
    bottom: 12px;
    left: 12px;
    background: rgba(30, 41, 59, 0.9);
    border: 1px solid rgba(71, 85, 105, 0.5);
    border-radius: 6px;
    padding: 8px 12px;
    backdrop-filter: blur(8px);
    z-index: 10;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .legend-item {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .swatch {
    width: 12px;
    height: 12px;
    border-radius: 3px;
    flex-shrink: 0;
  }

  .label {
    font-size: 12px;
    color: #e2e8f0;
  }
</style>
