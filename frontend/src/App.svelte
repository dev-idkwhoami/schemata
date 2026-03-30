<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime';
  import { GetState, SaveSQL, OpenProjectDialog } from '../wailsjs/go/main/App';
  import Canvas from './Canvas.svelte';
  import Legend from './Legend.svelte';
  import McpSetup from './McpSetup.svelte';
  import type { AppState } from './types';

  let state: AppState = {
    schemas: [],
    tables: [],
    foreignKeys: [],
    enumTypes: [],
    extensions: [],
    views: [],
  };

  let resetZoom: (() => void) | undefined;
  let panToTable: ((schema: string, name: string) => void) | undefined;
  let mcpSetupOpen = false;
  let activeTab: 'tables' | 'views' = 'tables';
  let searchOpen = false;
  let searchQuery = '';
  let searchSelectedIndex = 0;

  interface SearchResult {
    schema: string;
    table: string;
    label: string;
    sublabel: string;
    type: 'table' | 'column';
  }

  $: searchResults = (() => {
    if (!searchQuery || searchQuery.length === 0) return [] as SearchResult[];
    const q = searchQuery.toLowerCase();
    const results: SearchResult[] = [];
    for (const t of state.tables) {
      if (t.name.toLowerCase().includes(q)) {
        results.push({ schema: t.schema, table: t.name, label: t.name, sublabel: t.schema, type: 'table' });
      }
      for (const c of t.columns) {
        if (c.name.toLowerCase().includes(q)) {
          results.push({ schema: t.schema, table: t.name, label: c.name, sublabel: `${t.name}.${c.name}`, type: 'column' });
        }
      }
    }
    return results.slice(0, 12);
  })();

  $: if (searchResults) searchSelectedIndex = Math.min(searchSelectedIndex, Math.max(0, searchResults.length - 1));

  function selectSearchResult(r: SearchResult) {
    if (panToTable) panToTable(r.schema, r.table);
    searchOpen = false;
    searchQuery = '';
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.ctrlKey && e.key === 'f') {
      e.preventDefault();
      searchOpen = !searchOpen;
      if (!searchOpen) searchQuery = '';
    }
    if (e.key === 'Escape' && searchOpen) {
      searchOpen = false;
      searchQuery = '';
    }
    if (e.ctrlKey && e.key === 'z') {
      e.preventDefault();
      // Undo placeholder - will be connected in Batch 4
    }
    if (e.ctrlKey && e.key === 'y') {
      e.preventDefault();
      // Redo placeholder - will be connected in Batch 4
    }
  }

  onMount(() => {
    window.addEventListener('keydown', handleKeydown);

    // Listen for state updates from Go backend
    EventsOn('state_update', (newState: AppState) => {
      state = newState;
    });

    // Load initial state
    GetState().then((s: AppState) => {
      if (s) state = s;
    });
  });

  onDestroy(() => {
    window.removeEventListener('keydown', handleKeydown);
    EventsOff('state_update');
  });

  async function handleExport() {
    try {
      const path = await SaveSQL();
      if (path) {
        console.log('Exported to:', path);
      }
    } catch (e) {
      console.error('Export failed:', e);
    }
  }

  async function handleOpenProject() {
    try {
      await OpenProjectDialog();
    } catch (e) {
      console.error('Open failed:', e);
    }
  }

  function handleResetZoom() {
    if (resetZoom) resetZoom();
  }
</script>

<div class="app">
  <Canvas {state} {activeTab} bind:resetZoom bind:panToTable />

  {#if state.schemas.length > 0}
    <Legend schemas={state.schemas} />
  {/if}

  <div class="tab-toggle">
    <button class="tab-btn" class:active={activeTab === 'tables'} on:click={() => activeTab = 'tables'}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 3v18"/><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="M3 15h18"/>
      </svg>
      Tables
    </button>
    <button class="tab-btn" class:active={activeTab === 'views'} on:click={() => activeTab = 'views'}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M21 17v2a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-2"/><path d="M21 7V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v2"/><circle cx="12" cy="12" r="1"/><path d="M18.944 12.33a1 1 0 0 0 0-.66 7.5 7.5 0 0 0-13.888 0 1 1 0 0 0 0 .66 7.5 7.5 0 0 0 13.888 0"/>
      </svg>
      Views
    </button>
  </div>

  <div class="toolbar">
    {#if searchOpen}
      <div class="search-box">
        <input
          type="text"
          placeholder="Search tables & columns..."
          bind:value={searchQuery}
          on:keydown={(e) => {
            if (e.key === 'ArrowDown') {
              e.preventDefault();
              searchSelectedIndex = Math.min(searchSelectedIndex + 1, searchResults.length - 1);
            } else if (e.key === 'ArrowUp') {
              e.preventDefault();
              searchSelectedIndex = Math.max(searchSelectedIndex - 1, 0);
            } else if (e.key === 'Enter' && searchResults.length > 0) {
              e.preventDefault();
              selectSearchResult(searchResults[searchSelectedIndex]);
            }
          }}
          autofocus
        />
        {#if searchResults.length > 0}
          <div class="search-dropdown">
            {#each searchResults as result, i}
              <button
                class="search-item"
                class:selected={i === searchSelectedIndex}
                on:click={() => selectSearchResult(result)}
                on:mouseenter={() => searchSelectedIndex = i}
              >
                <span class="search-icon">{result.type === 'table' ? '◻' : '·'}</span>
                <span class="search-label">{result.label}</span>
                <span class="search-sublabel">{result.sublabel}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
    <button class="btn" on:click={handleResetZoom} title="Fit to viewport">
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <path d="M2 5V2h3M11 2h3v3M14 11v3h-3M5 14H2v-3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
      Reset View
    </button>
    <button class="btn" on:click={handleExport} title="Export PostgreSQL DDL">
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <path d="M8 2v8M5 7l3 3 3-3M3 12v1h10v-1" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
      Export SQL
    </button>
    <button class="btn btn-icon" on:click={handleOpenProject} title="Load Project">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="11.5" cy="12.5" r="2.5"/><path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/><path d="M13.3 14.3 15 16"/>
      </svg>
    </button>
  </div>

  <div class="toolbar-left">
    <button class="btn" on:click={() => mcpSetupOpen = true} title="MCP Setup">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 8V4H8"/>
        <rect width="16" height="12" x="4" y="8" rx="2"/>
        <path d="M2 14h2"/>
        <path d="M20 14h2"/>
        <path d="M15 13v2"/>
        <path d="M9 13v2"/>
      </svg>
      MCP Setup
    </button>
  </div>

  {#if activeTab === 'tables' && state.tables.length === 0}
    <div class="empty-state">
      <p>No tables yet.</p>
      <p class="hint">Use the MCP tools to create schemas and tables.</p>
    </div>
  {:else if activeTab === 'views' && (!state.views || state.views.length === 0)}
    <div class="empty-state">
      <p>No views yet.</p>
      <p class="hint">Use the MCP tools to create views.</p>
    </div>
  {/if}
</div>

<McpSetup visible={mcpSetupOpen} onClose={() => mcpSetupOpen = false} />

<style>
  .app {
    width: 100%;
    height: 100%;
    position: relative;
    overflow: hidden;
  }

  .toolbar-left {
    position: fixed;
    top: 12px;
    left: 12px;
    z-index: 10;
  }

  .toolbar {
    position: fixed;
    top: 12px;
    right: 12px;
    display: flex;
    gap: 8px;
    z-index: 10;
  }

  .btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px;
    background: rgba(30, 41, 59, 0.9);
    border: 1px solid rgba(71, 85, 105, 0.5);
    color: #e2e8f0;
    border-radius: 6px;
    cursor: pointer;
    font-size: 12px;
    backdrop-filter: blur(8px);
    transition: background 0.15s;
  }

  .btn:hover {
    background: rgba(51, 65, 85, 0.9);
  }

  .btn-icon {
    padding: 6px;
  }

  .search-box {
    position: relative;
  }

  .search-box input {
    padding: 6px 12px;
    background: rgba(30, 41, 59, 0.95);
    border: 1px solid rgba(71, 85, 105, 0.5);
    color: #e2e8f0;
    border-radius: 6px;
    font-size: 13px;
    outline: none;
    width: 260px;
    backdrop-filter: blur(8px);
  }

  .search-box input:focus {
    border-color: #6366f1;
  }

  .search-dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    margin-top: 4px;
    background: rgba(15, 23, 42, 0.95);
    border: 1px solid rgba(71, 85, 105, 0.5);
    border-radius: 6px;
    overflow: hidden;
    backdrop-filter: blur(12px);
    max-height: 320px;
    overflow-y: auto;
  }

  .search-item {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 7px 12px;
    border: none;
    background: none;
    color: #e2e8f0;
    font-size: 13px;
    cursor: pointer;
    text-align: left;
  }

  .search-item:hover,
  .search-item.selected {
    background: rgba(51, 65, 85, 0.6);
  }

  .search-icon {
    color: #64748b;
    font-size: 10px;
    width: 12px;
    text-align: center;
    flex-shrink: 0;
  }

  .search-label {
    flex: 1;
    font-weight: 500;
  }

  .search-sublabel {
    color: #64748b;
    font-size: 11px;
    flex-shrink: 0;
  }

  .tab-toggle {
    position: fixed;
    top: 12px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    background: rgba(15, 23, 42, 0.9);
    border: 1px solid rgba(71, 85, 105, 0.5);
    border-radius: 8px;
    padding: 3px;
    z-index: 10;
    backdrop-filter: blur(8px);
  }
  .tab-btn {
    padding: 5px 16px;
    border: none;
    background: none;
    color: rgba(226, 232, 240, 0.4);
    font-size: 13px;
    cursor: pointer;
    border-radius: 6px;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: all 0.15s;
  }
  .tab-btn.active {
    background: rgba(99, 102, 241, 0.8);
    color: #e2e8f0;
  }

  .empty-state {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    text-align: center;
    color: #64748b;
    pointer-events: none;
  }

  .empty-state p {
    font-size: 16px;
    margin-bottom: 4px;
  }

  .empty-state .hint {
    font-size: 13px;
    color: #475569;
  }
</style>
