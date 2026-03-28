<script lang="ts">
  import { onMount } from 'svelte';
  import { GetMCPPath } from '../wailsjs/go/main/App';

  export let visible = false;
  export let onClose: () => void = () => {};

  type Mode = 'Claude Code' | 'Config';

  let selectedMode: Mode = 'Claude Code';
  let mcpPath = 'schemata-mcp.exe';
  let copied = false;
  let copyTimeout: ReturnType<typeof setTimeout> | null = null;

  const modes: Mode[] = ['Claude Code', 'Config'];

  onMount(async () => {
    try {
      mcpPath = await GetMCPPath();
    } catch (e) {
      console.error('Failed to get MCP path:', e);
    }
  });

  $: configText = (() => {
    if (selectedMode === 'Claude Code') {
      return `claude mcp add schemata -- "${mcpPath}"`;
    }
    const config = {
      schemata: {
        command: mcpPath,
        args: [],
        type: 'stdio',
      },
    };
    return JSON.stringify(config, null, 2);
  })();

  $: hint = selectedMode === 'Claude Code'
    ? 'Run this command in your terminal to register the MCP server.'
    : 'Add this to your MCP config file (Cursor, Windsurf, Cline, etc.).';

  function handleCopy() {
    navigator.clipboard.writeText(configText).then(() => {
      copied = true;
      if (copyTimeout) clearTimeout(copyTimeout);
      copyTimeout = setTimeout(() => {
        copied = false;
      }, 2000);
    });
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      onClose();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onClose();
    }
  }
</script>

<svelte:window on:keydown={visible ? handleKeydown : undefined} />

{#if visible}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="backdrop" on:click={handleBackdropClick}>
    <div class="modal">
      <h2>MCP Setup</h2>

      <div class="agents">
        {#each modes as mode}
          <button
            class="agent-btn"
            class:active={selectedMode === mode}
            on:click={() => selectedMode = mode}
          >
            {mode}
          </button>
        {/each}
      </div>

      <div class="config-block">{configText}</div>

      <div class="actions">
        <button class="copy-btn" on:click={handleCopy}>
          Copy
        </button>
        {#if copied}
          <span class="copied">Copied!</span>
        {/if}
      </div>

      <p class="hint">{hint}</p>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    z-index: 50;
    display: flex;
    align-items: center;
    justify-content: center;
    backdrop-filter: blur(4px);
  }

  .modal {
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 12px;
    padding: 24px;
    width: 520px;
    max-height: 80vh;
    overflow-y: auto;
    color: #e2e8f0;
  }

  .modal h2 {
    margin: 0 0 16px;
    font-size: 18px;
    font-weight: 600;
  }

  .agents {
    display: flex;
    gap: 8px;
    margin-bottom: 16px;
    flex-wrap: wrap;
  }

  .agent-btn {
    padding: 6px 14px;
    background: #0f172a;
    border: 1px solid #334155;
    border-radius: 6px;
    color: #e2e8f0;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .agent-btn:hover {
    background: #334155;
  }

  .agent-btn.active {
    background: #6366f1;
    border-color: #6366f1;
  }

  .config-block {
    background: #0f172a;
    border: 1px solid #334155;
    border-radius: 8px;
    padding: 14px;
    font-family: 'Consolas', 'Courier New', monospace;
    font-size: 13px;
    white-space: pre-wrap;
    word-break: break-all;
    color: #94a3b8;
    position: relative;
    margin-bottom: 12px;
  }

  .actions {
    display: flex;
    align-items: center;
  }

  .copy-btn {
    padding: 6px 14px;
    background: #6366f1;
    border: none;
    border-radius: 6px;
    color: white;
    font-size: 13px;
    cursor: pointer;
    transition: background 0.15s;
  }

  .copy-btn:hover {
    background: #4f46e5;
  }

  .copied {
    color: #22c55e;
    font-size: 12px;
    margin-left: 8px;
  }

  .hint {
    font-size: 12px;
    color: #64748b;
    margin-top: 8px;
  }
</style>
