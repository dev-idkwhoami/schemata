<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { LayoutNode, ForeignKey } from './types';

  export let node: LayoutNode;
  export let foreignKeys: ForeignKey[] = [];
  export let highlighted: boolean = false;
  export let selected: boolean = false;

  const dispatch = createEventDispatcher();

  const HEADER_HEIGHT = 32;
  const ROW_HEIGHT = 26;
  const BORDER_RADIUS = 6;

  type IconDef = { type: 'key'; color: string } | { type: 'lock'; color: string } | { type: 'dashed'; color: string };

  function isForeignKey(colName: string): boolean {
    return foreignKeys.some(
      fk =>
        (fk.fromSchema === node.table.schema &&
          fk.fromTable === node.table.name &&
          fk.fromColumn === colName) ||
        (fk.toSchema === node.table.schema &&
          fk.toTable === node.table.name &&
          fk.toColumn === colName)
    );
  }

  $: tableConstraints = (node.table.constraints || []) as import('./types').TableConstraint[];

  function shareCompositeConstraint(i: number): boolean {
    if (tableConstraints.length === 0) return false;
    if (i < 0 || i + 1 >= node.table.columns.length) return false;
    const col1 = node.table.columns[i].name;
    const col2 = node.table.columns[i + 1].name;
    return tableConstraints.some(
      c => c.type !== 'check' && c.columns && c.columns.length >= 2 && c.columns.includes(col1) && c.columns.includes(col2)
    );
  }

  function getConstraintColor(i: number): string {
    const col1 = node.table.columns[i].name;
    const col2 = node.table.columns[i + 1].name;
    const constraint = tableConstraints.find(
      c => c.type !== 'check' && c.columns && c.columns.length >= 2 && c.columns.includes(col1) && c.columns.includes(col2)
    );
    return constraint?.type === 'unique' ? '#06b6d4' : '#f59e0b';
  }

  function getIcons(col: typeof node.table.columns[0], isFK: boolean): IconDef[] {
    const icons: IconDef[] = [];
    if (col.primaryKey) icons.push({ type: 'key', color: '#f59e0b' });
    if (isFK) icons.push({ type: 'key', color: '#94a3b8' });
    if (col.unique && !col.primaryKey) icons.push({ type: 'lock', color: '#06b6d4' });
    if (col.nullable && !col.primaryKey) icons.push({ type: 'dashed', color: '#64748b' });
    return icons;
  }
</script>

<g transform="translate({node.x}, {node.y})"
  on:mouseenter={() => dispatch('tablehover', { schema: node.table.schema, name: node.table.name })}
  on:mouseleave={() => dispatch('tablehoverend')}
  on:click|stopPropagation={(e) => dispatch('tableclick', { event: e, schema: node.table.schema, name: node.table.name })}
  style="user-select: none; -webkit-user-select: none;"
>
  {#if selected}
    <rect
      x={-4}
      y={-4}
      width={node.width + 8}
      height={node.height + 8}
      rx={BORDER_RADIUS + 2}
      ry={BORDER_RADIUS + 2}
      fill="none"
      stroke="#6366f1"
      stroke-width="2"
      opacity="0.8"
    />
  {/if}
  {#if highlighted}
    <rect
      x={-4}
      y={-4}
      width={node.width + 8}
      height={node.height + 8}
      rx={BORDER_RADIUS + 2}
      ry={BORDER_RADIUS + 2}
      fill="none"
      stroke={node.color}
      stroke-width="1"
      opacity="0.4"
    />
  {/if}
  <!-- Card background -->
  <rect
    width={node.width}
    height={node.height}
    rx={BORDER_RADIUS}
    ry={BORDER_RADIUS}
    fill="#1e293b"
    stroke={highlighted ? node.color : '#334155'}
    stroke-width={highlighted ? '2' : '1'}
  />

  <!-- Header bar -->
  <rect
    width={node.width}
    height={HEADER_HEIGHT}
    rx={BORDER_RADIUS}
    ry={BORDER_RADIUS}
    fill={node.color}
  />
  <!-- Square off bottom corners of header -->
  <rect
    y={HEADER_HEIGHT - BORDER_RADIUS}
    width={node.width}
    height={BORDER_RADIUS}
    fill={node.color}
  />

  <!-- Entity type icon -->
  {#if node.kind === 'view'}
    <svg x={8} y={HEADER_HEIGHT / 2 - 7} width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M21 17v2a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-2"/><path d="M21 7V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v2"/><circle cx="12" cy="12" r="1"/><path d="M18.944 12.33a1 1 0 0 0 0-.66 7.5 7.5 0 0 0-13.888 0 1 1 0 0 0 0 .66 7.5 7.5 0 0 0 13.888 0"/>
    </svg>
  {:else}
    <svg x={8} y={HEADER_HEIGHT / 2 - 7} width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M12 3v18"/><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="M3 15h18"/>
    </svg>
  {/if}

  <!-- Table name -->
  <text
    x={28}
    y={HEADER_HEIGHT / 2}
    dominant-baseline="central"
    fill="white"
    font-weight="600"
    font-size="13px"
  >
    {node.table.name}
  </text>

  <!-- Drag grip icon (always visible, far right of header) -->
  <g
    transform="translate({node.width - 18}, {HEADER_HEIGHT / 2 - 8})"
    style="cursor: grab"
    on:mousedown|stopPropagation|preventDefault={(e) => dispatch('tabledragstart', { event: e })}
  >
    <rect x={-3} y={-3} width={18} height={22} fill="transparent" />
    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
      <circle cx="12" cy="5" r="1"/><circle cx="19" cy="5" r="1"/><circle cx="5" cy="5" r="1"/>
      <circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/>
      <circle cx="12" cy="19" r="1"/><circle cx="19" cy="19" r="1"/><circle cx="5" cy="19" r="1"/>
    </svg>
  </g>

  <!-- Unpin button (visible when pinned, left of drag grip) -->
  {#if node.table.position}
    <g
      transform="translate({node.width - 36}, {HEADER_HEIGHT / 2 - 6})"
      style="cursor: pointer"
      on:click|stopPropagation={() => dispatch('unpin', { schema: node.table.schema, table: node.table.name })}
    >
      <rect x={-3} y={-3} width={18} height={18} fill="transparent" />
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 17v5"/>
        <path d="M15 9.34V7a1 1 0 0 1 1-1 2 2 0 0 0 0-4H7.89"/>
        <path d="m2 2 20 20"/>
        <path d="M9 9v1.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24V16a1 1 0 0 0 1 1h11"/>
      </svg>
    </g>
  {/if}


  <!-- Columns -->
  {#each node.table.columns as col, i}
    {@const y = HEADER_HEIGHT + i * ROW_HEIGHT}
    {@const isFK = isForeignKey(col.name)}
    <!-- Row separator -->
    {#if i > 0}
      <line
        x1={0}
        y1={y}
        x2={node.width}
        y2={y}
        stroke="#334155"
        stroke-width="0.5"
      />
      {#if shareCompositeConstraint(i - 1)}
        <svg x={node.width / 2 - 6} y={y - 6} width="12" height="12" viewBox="0 0 24 24" fill="none" stroke={getConstraintColor(i - 1)} stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
          <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
        </svg>
      {/if}
    {/if}

    <!-- Left gutter icons (overlapping at 8px offsets) -->
    {#if node.kind !== 'view'}
      {@const icons = getIcons(col, isFK)}
      {#each icons as icon, idx}
        <svg x={4 + idx * 8} y={y + ROW_HEIGHT / 2 - 7} width="14" height="14" viewBox="0 0 24 24" fill="none" stroke={icon.color} stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          {#if icon.type === 'key'}
            <path d="M2.586 17.414A2 2 0 0 0 2 18.828V21a1 1 0 0 0 1 1h3a1 1 0 0 0 1-1v-1a1 1 0 0 1 1-1h1a1 1 0 0 0 1-1v-1a1 1 0 0 1 1-1h.172a2 2 0 0 0 1.414-.586l.814-.814a6.5 6.5 0 1 0-4-4z"/>
            <circle cx="16.5" cy="7.5" r=".5" fill={icon.color}/>
          {:else if icon.type === 'lock'}
            <rect width="18" height="11" x="3" y="11" rx="2" ry="2"/>
            <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
          {:else if icon.type === 'dashed'}
            <path d="M5 3a2 2 0 0 0-2 2"/>
            <path d="M19 3a2 2 0 0 1 2 2"/>
            <path d="M21 19a2 2 0 0 1-2 2"/>
            <path d="M5 21a2 2 0 0 1-2-2"/>
            <path d="M9 3h1"/>
            <path d="M9 21h1"/>
            <path d="M14 3h1"/>
            <path d="M14 21h1"/>
            <path d="M3 9v1"/>
            <path d="M21 9v1"/>
            <path d="M3 14v1"/>
            <path d="M21 14v1"/>
          {/if}
        </svg>
      {/each}
    {/if}

    <!-- Column name -->
    <text
      x={node.kind === 'view' ? 12 : 28}
      y={y + ROW_HEIGHT / 2}
      dominant-baseline="central"
      fill="#e2e8f0"
      font-size="12px"
    >
      {col.name}
    </text>

    <!-- Column type -->
    <text
      x={node.width - 12}
      y={y + ROW_HEIGHT / 2}
      dominant-baseline="central"
      text-anchor="end"
      fill="#64748b"
      font-size="11px"
    >
      {col.type}
    </text>

    <!-- Hit area for hover popover -->
    <rect
      x={0}
      y={y}
      width={node.width}
      height={ROW_HEIGHT}
      fill="transparent"
      on:mouseenter={(e) => dispatch('columnhover', { column: col, table: node.table, x: node.x + node.width + 8, y: node.y + y + ROW_HEIGHT / 2, event: e })}
      on:mouseleave={() => dispatch('columnhoverend')}
      style="cursor: default"
    />
  {/each}

  <!-- Indexes section -->
  {#if node.table.indexes && node.table.indexes.length > 0}
    {@const indexStartY = HEADER_HEIGHT + node.table.columns.length * ROW_HEIGHT}
    <!-- Thick separator -->
    <line
      x1={0}
      y1={indexStartY}
      x2={node.width}
      y2={indexStartY}
      stroke="#475569"
      stroke-width="2"
    />
    {#each node.table.indexes as idx, i}
      {@const iy = indexStartY + 4 + i * ROW_HEIGHT}
      {#if i > 0}
        <line
          x1={0}
          y1={iy}
          x2={node.width}
          y2={iy}
          stroke="#334155"
          stroke-width="0.5"
        />
      {/if}
      <!-- Index name -->
      <text
        x={12}
        y={iy + ROW_HEIGHT / 2}
        dominant-baseline="central"
        fill="#94a3b8"
        font-size="11px"
        font-style="italic"
      >
        {idx.name}{idx.type && idx.type !== 'btree' ? ` (${idx.type})` : ''}
      </text>
      <!-- Index columns -->
      <text
        x={node.width - 12}
        y={iy + ROW_HEIGHT / 2}
        dominant-baseline="central"
        text-anchor="end"
        fill="#64748b"
        font-size="10px"
      >
        {idx.columns.join(', ')}
      </text>
      <!-- Unique badge -->
      {#if idx.unique}
        <svg x={node.width - 12 - idx.columns.join(', ').length * 6 - 22} y={iy + ROW_HEIGHT / 2 - 7} width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#06b6d4" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect width="18" height="11" x="3" y="11" rx="2" ry="2"/>
          <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
        </svg>
      {/if}
    {/each}
  {/if}
</g>
