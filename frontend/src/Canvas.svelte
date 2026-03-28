<script lang="ts">
  import { onMount, afterUpdate } from 'svelte';
  import { select } from 'd3-selection';
  import { zoom, zoomIdentity, zoomTransform, type ZoomBehavior } from 'd3-zoom';
  import { tree, hierarchy } from 'd3-hierarchy';
  import type { AppState, LayoutNode, FKLineData, Table, ForeignKey } from './types';
  import TableCard from './TableCard.svelte';
  import FKLine from './FKLine.svelte';
  import ColumnPopover from './ColumnPopover.svelte';
  import { SetTablePosition, ClearTablePosition, SetViewPosition, ClearViewPosition } from '../wailsjs/go/main/App';

  export let state: AppState;
  export let activeTab: 'tables' | 'views' = 'tables';
  export let resetZoom: (() => void) | undefined = undefined;
  export let panToTable: ((schema: string, name: string) => void) | undefined = undefined;

  const TABLE_WIDTH = 280;
  const HEADER_HEIGHT = 32;
  const ROW_HEIGHT = 26;
  const TABLE_PADDING = 8;
  const NODE_GAP_X = 120;
  const NODE_GAP_Y = 60;

  let svgEl: SVGSVGElement;
  let gEl: SVGGElement;
  let zoomBehavior: ZoomBehavior<SVGSVGElement, unknown>;
  let layoutNodes: LayoutNode[] = [];
  let fkLines: FKLineData[] = [];
  let lineOffsets: LineOffsets = { vertical: new Map(), sourceH: new Map(), targetH: new Map() };
  let highlightedTable = '';

  let popoverVisible = false;
  let hoveredTableKey = '';
  let currentZoomScale = 1;
  let popoverColumn: any = null;
  let popoverTable: any = null;
  let popoverX = 0;
  let popoverY = 0;

  let dragging: { schema: string; name: string; startX: number; startY: number; nodeStartX: number; nodeStartY: number; width: number; height: number; color: string } | null = null;
  let ghostX = 0;
  let ghostY = 0;

  function handleDragStart(e: MouseEvent, node: LayoutNode) {
    if (e.button !== 0) return;
    e.stopPropagation();
    e.preventDefault();
    dragging = {
      schema: node.table.schema,
      name: node.table.name,
      startX: e.clientX,
      startY: e.clientY,
      nodeStartX: node.x,
      nodeStartY: node.y,
      width: node.width,
      height: node.height,
      color: node.color,
    };
    ghostX = node.x;
    ghostY = node.y;
    document.body.style.cursor = 'grabbing';
    window.addEventListener('mousemove', handleDragMove);
    window.addEventListener('mouseup', handleDragEnd);
  }

  function handleDragMove(e: MouseEvent) {
    if (!dragging || !svgEl) return;

    const transform = zoomTransform(svgEl);
    const dx = (e.clientX - dragging.startX) / transform.k;
    const dy = (e.clientY - dragging.startY) / transform.k;

    ghostX = dragging.nodeStartX + dx;
    ghostY = dragging.nodeStartY + dy;
  }

  function handleDragEnd(e: MouseEvent) {
    if (!dragging || !svgEl) return;

    const transform = zoomTransform(svgEl);
    const dx = (e.clientX - dragging.startX) / transform.k;
    const dy = (e.clientY - dragging.startY) / transform.k;

    const newX = dragging.nodeStartX + dx;
    const newY = dragging.nodeStartY + dy;

    if (activeTab === 'views') {
      SetViewPosition(dragging.schema, dragging.name, newX, newY);
    } else {
      SetTablePosition(dragging.schema, dragging.name, newX, newY);
    }

    window.removeEventListener('mousemove', handleDragMove);
    window.removeEventListener('mouseup', handleDragEnd);
    document.body.style.cursor = '';
    dragging = null;
  }

  function handleUnpin(e: CustomEvent) {
    const { schema, table } = e.detail;
    if (activeTab === 'views') {
      ClearViewPosition(schema, table);
    } else {
      ClearTablePosition(schema, table);
    }
  }

  function recomputeFKLines(nodes: LayoutNode[]): FKLineData[] {
    const nodeMap = new Map<string, LayoutNode>();
    for (const n of nodes) {
      nodeMap.set(tableKey(n.table.schema, n.table.name), n);
    }

    const lines: FKLineData[] = [];
    for (const fk of state.foreignKeys) {
      const sourceNode = nodeMap.get(tableKey(fk.fromSchema, fk.fromTable));
      const targetNode = nodeMap.get(tableKey(fk.toSchema, fk.toTable));
      if (!sourceNode || !targetNode) continue;

      const sourceColIdx = sourceNode.table.columns.findIndex(c => c.name === fk.fromColumn);
      const targetColIdx = targetNode.table.columns.findIndex(c => c.name === fk.toColumn);

      lines.push({
        fk,
        sourceNode,
        targetNode,
        sourceColumnIndex: sourceColIdx >= 0 ? sourceColIdx : 0,
        targetColumnIndex: targetColIdx >= 0 ? targetColIdx : 0,
        color: getSchemaColor(fk.fromSchema),
      });
    }
    return lines;
  }

  function getTableHeight(t: Table): number {
    const indexCount = t.indexes?.length || 0;
    const indexSection = indexCount > 0 ? indexCount * ROW_HEIGHT + 4 : 0;
    return HEADER_HEIGHT + t.columns.length * ROW_HEIGHT + indexSection + TABLE_PADDING;
  }

  function getSchemaColor(schemaName: string): string {
    const s = state.schemas.find(s => s.name === schemaName);
    return s ? s.color : '#64748b';
  }

  function tableKey(schema: string, name: string): string {
    return `${schema}.${name}`;
  }

  function computeLayout(state: AppState): { nodes: LayoutNode[]; lines: FKLineData[] } {
    if (state.tables.length === 0) return { nodes: [], lines: [] };

    // Build adjacency for FK connections (undirected for finding most-connected root)
    const connectionCount = new Map<string, number>();
    const children = new Map<string, Set<string>>();

    for (const t of state.tables) {
      const key = tableKey(t.schema, t.name);
      connectionCount.set(key, 0);
      children.set(key, new Set());
    }

    for (const fk of state.foreignKeys) {
      const fromKey = tableKey(fk.fromSchema, fk.fromTable);
      const toKey = tableKey(fk.toSchema, fk.toTable);
      connectionCount.set(fromKey, (connectionCount.get(fromKey) || 0) + 1);
      connectionCount.set(toKey, (connectionCount.get(toKey) || 0) + 1);
      // FK direction: from (child) -> to (parent), so in tree: to is parent of from
      // But for tree layout, we want parent -> children, so the "to" table owns the "from" table
      children.get(toKey)?.add(fromKey);
    }

    // Separate tables with FK connections from orphans
    const connectedKeys = new Set<string>();
    for (const fk of state.foreignKeys) {
      connectedKeys.add(tableKey(fk.fromSchema, fk.fromTable));
      connectedKeys.add(tableKey(fk.toSchema, fk.toTable));
    }

    const orphanTables = state.tables.filter(t => !connectedKeys.has(tableKey(t.schema, t.name)));
    const connectedTables = state.tables.filter(t => connectedKeys.has(tableKey(t.schema, t.name)));

    const tableMap = new Map<string, Table>();
    for (const t of state.tables) {
      tableMap.set(tableKey(t.schema, t.name), t);
    }

    const nodes: LayoutNode[] = [];

    if (connectedTables.length > 0) {
      // Find root: table with most FK connections
      let rootKey = '';
      let maxConn = -1;
      for (const [key, count] of connectionCount) {
        if (connectedKeys.has(key) && count > maxConn) {
          maxConn = count;
          rootKey = key;
        }
      }

      // Build tree data structure via BFS
      const visited = new Set<string>();
      interface TreeNode { key: string; children: TreeNode[] }

      function buildTree(key: string): TreeNode {
        visited.add(key);
        const childKeys = children.get(key) || new Set();
        const treeChildren: TreeNode[] = [];
        for (const ck of childKeys) {
          if (!visited.has(ck)) {
            treeChildren.push(buildTree(ck));
          }
        }
        return { key, children: treeChildren };
      }

      const treeData = buildTree(rootKey);

      // Add any connected but unreachable tables (disconnected subgraphs)
      for (const t of connectedTables) {
        const key = tableKey(t.schema, t.name);
        if (!visited.has(key)) {
          treeData.children.push(buildTree(key));
        }
      }

      // Use d3-hierarchy tree layout
      const root = hierarchy(treeData, d => d.children);

      // Compute node sizes based on actual table heights
      const maxHeight = Math.max(...connectedTables.map(t => getTableHeight(t)));
      const treeLayout = tree<TreeNode>()
        .nodeSize([maxHeight + NODE_GAP_Y, TABLE_WIDTH + NODE_GAP_X * 2])
        .separation(() => 1);

      treeLayout(root);

      for (const d3node of root.descendants()) {
        const t = tableMap.get(d3node.data.key);
        if (t) {
          const finalX = d3node.y!;
          const finalY = d3node.x!;
          nodes.push({
            table: t,
            x: finalX - TABLE_WIDTH / 2,
            y: finalY,
            width: TABLE_WIDTH,
            height: getTableHeight(t),
            color: getSchemaColor(t.schema),
            kind: 'table',
          });
        }
      }
    }

    // Place orphan tables in a column to the right of the tree
    if (orphanTables.length > 0) {
      const maxX = nodes.length > 0 ? Math.max(...nodes.map(n => n.x + n.width)) + NODE_GAP_X * 2 : 0;
      let currentY = nodes.length > 0 ? Math.min(...nodes.map(n => n.y)) : 0;

      for (let i = 0; i < orphanTables.length; i++) {
        const t = orphanTables[i];
        nodes.push({
          table: t,
          x: maxX,
          y: currentY,
          width: TABLE_WIDTH,
          height: getTableHeight(t),
          color: getSchemaColor(t.schema),
          kind: 'table',
        });
        currentY += getTableHeight(t) + NODE_GAP_Y;
      }
    }

    // Override positions for pinned tables
    for (const n of nodes) {
      if (n.table.position) {
        n.x = n.table.position.x;
        n.y = n.table.position.y;
      }
    }

    // Compute FK lines
    const nodeMap = new Map<string, LayoutNode>();
    for (const n of nodes) {
      nodeMap.set(tableKey(n.table.schema, n.table.name), n);
    }

    const lines: FKLineData[] = [];
    for (const fk of state.foreignKeys) {
      const sourceNode = nodeMap.get(tableKey(fk.fromSchema, fk.fromTable));
      const targetNode = nodeMap.get(tableKey(fk.toSchema, fk.toTable));
      if (!sourceNode || !targetNode) continue;

      const sourceColIdx = sourceNode.table.columns.findIndex(c => c.name === fk.fromColumn);
      const targetColIdx = targetNode.table.columns.findIndex(c => c.name === fk.toColumn);

      lines.push({
        fk,
        sourceNode,
        targetNode,
        sourceColumnIndex: sourceColIdx >= 0 ? sourceColIdx : 0,
        targetColumnIndex: targetColIdx >= 0 ? targetColIdx : 0,
        color: getSchemaColor(fk.fromSchema),
      });
    }

    return { nodes, lines };
  }

  function computeViewLayout(state: AppState): { nodes: LayoutNode[]; lines: FKLineData[] } {
    if (!state.views || state.views.length === 0) return { nodes: [], lines: [] };

    const nodes: LayoutNode[] = [];
    const cols = Math.ceil(Math.sqrt(state.views.length));

    for (let i = 0; i < state.views.length; i++) {
      const v = state.views[i];
      const col = i % cols;
      const row = Math.floor(i / cols);
      const height = HEADER_HEIGHT + (v.columns?.length || 0) * ROW_HEIGHT + TABLE_PADDING;
      const color = getSchemaColor(v.schema);

      const x = v.position ? v.position.x : col * (TABLE_WIDTH + NODE_GAP_X);
      const y = v.position ? v.position.y : row * (height + NODE_GAP_Y);

      nodes.push({
        table: {
          schema: v.schema,
          name: v.name,
          columns: (v.columns || []).map(vc => ({
            name: vc.name,
            type: vc.type,
            nullable: false,
            primaryKey: false,
            unique: false,
            default: '',
            sourceSchema: vc.sourceSchema,
            sourceTable: vc.sourceTable,
            sourceColumn: vc.sourceColumn,
          })),
          constraints: [],
          indexes: [],
          position: v.position,
        },
        x,
        y,
        width: TABLE_WIDTH,
        height,
        color,
        kind: 'view',
      });
    }

    return { nodes, lines: [] };
  }

  function fitToViewport() {
    if (!svgEl || !gEl || layoutNodes.length === 0) return;

    const svgRect = svgEl.getBoundingClientRect();
    const padding = 60;

    // Compute bounding box of all nodes
    let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
    for (const n of layoutNodes) {
      minX = Math.min(minX, n.x);
      minY = Math.min(minY, n.y);
      maxX = Math.max(maxX, n.x + n.width);
      maxY = Math.max(maxY, n.y + n.height);
    }

    const contentWidth = maxX - minX;
    const contentHeight = maxY - minY;
    const availWidth = svgRect.width - padding * 2;
    const availHeight = svgRect.height - padding * 2;

    const scale = Math.min(1, availWidth / contentWidth, availHeight / contentHeight);
    const tx = (svgRect.width - contentWidth * scale) / 2 - minX * scale;
    const ty = (svgRect.height - contentHeight * scale) / 2 - minY * scale;

    const transform = zoomIdentity.translate(tx, ty).scale(scale);
    select(svgEl).transition().duration(300).call(zoomBehavior.transform, transform);
  }

  resetZoom = fitToViewport;

  function handleColumnHover(e: CustomEvent) {
    if (currentZoomScale < 0.5) return;
    const { column, table, event } = e.detail;
    const mouseEvent = event as MouseEvent;
    popoverX = mouseEvent.clientX + 16;
    popoverY = mouseEvent.clientY - 10;
    popoverColumn = column;
    popoverTable = table;
    popoverVisible = true;
  }

  function handleColumnHoverEnd() {
    popoverVisible = false;
  }

  function handleTableHover(e: CustomEvent) {
    hoveredTableKey = e.detail.schema + '.' + e.detail.name;
  }

  function handleTableHoverEnd() {
    hoveredTableKey = '';
  }

  function doPanToTable(schema: string, name: string) {
    const node = layoutNodes.find(n => n.table.schema === schema && n.table.name === name);
    if (!node || !svgEl || !zoomBehavior) return;

    const svgRect = svgEl.getBoundingClientRect();
    const centerX = node.x + node.width / 2;
    const centerY = node.y + node.height / 2;
    const scale = 1;
    const tx = svgRect.width / 2 - centerX * scale;
    const ty = svgRect.height / 2 - centerY * scale;

    const transform = zoomIdentity.translate(tx, ty).scale(scale);
    select(svgEl).transition().duration(500).call(zoomBehavior.transform, transform);

    highlightedTable = schema + '.' + name;
    setTimeout(() => { highlightedTable = ''; }, 2000);
  }

  panToTable = doPanToTable;

  onMount(() => {
    zoomBehavior = zoom<SVGSVGElement, unknown>()
      .scaleExtent([0.1, 4])
      .on('zoom', (event) => {
        if (gEl) {
          gEl.setAttribute('transform', event.transform.toString());
        }
        currentZoomScale = event.transform.k;
      });

    select(svgEl).call(zoomBehavior);
  });

  interface LineOffsets {
    vertical: Map<string, number>;
    sourceH: Map<string, number>;
    targetH: Map<string, number>;
  }

  function computeLineOffsets(lines: FKLineData[]): LineOffsets {
    // Vertical corridor offsets (existing logic)
    const corridors = new Map<string, string[]>();
    for (const line of lines) {
      const { sourceNode, targetNode } = line;
      const sourceCenterX = sourceNode.x + sourceNode.width / 2;
      const targetCenterX = targetNode.x + targetNode.width / 2;
      let exitX: number;
      if (sourceCenterX <= targetCenterX) {
        exitX = sourceNode.x + sourceNode.width + 40;
      } else {
        exitX = sourceNode.x - 40;
      }
      const corridorKey = String(Math.round(exitX / 20) * 20);
      const lineKey = `${line.fk.fromSchema}.${line.fk.fromTable}.${line.fk.fromColumn}`;
      if (!corridors.has(corridorKey)) corridors.set(corridorKey, []);
      corridors.get(corridorKey)!.push(lineKey);
    }
    const vertical = new Map<string, number>();
    for (const [_, lineKeys] of corridors) {
      const count = lineKeys.length;
      for (let i = 0; i < count; i++) {
        vertical.set(lineKeys[i], (i - (count - 1) / 2) * 6);
      }
    }

    // Horizontal offsets: group lines sharing the same source or target column row
    const sourceGroups = new Map<string, string[]>(); // key: "schema.table.colIdx.side"
    const targetGroups = new Map<string, string[]>();
    for (const line of lines) {
      const lineKey = `${line.fk.fromSchema}.${line.fk.fromTable}.${line.fk.fromColumn}`;
      const sourceCenterX = line.sourceNode.x + line.sourceNode.width / 2;
      const targetCenterX = line.targetNode.x + line.targetNode.width / 2;
      const side = sourceCenterX <= targetCenterX ? 'R' : 'L';

      const srcKey = `${line.fk.fromSchema}.${line.fk.fromTable}.${line.sourceColumnIndex}.${side}`;
      if (!sourceGroups.has(srcKey)) sourceGroups.set(srcKey, []);
      sourceGroups.get(srcKey)!.push(lineKey);

      const tgtKey = `${line.fk.toSchema}.${line.fk.toTable}.${line.targetColumnIndex}.${side}`;
      if (!targetGroups.has(tgtKey)) targetGroups.set(tgtKey, []);
      targetGroups.get(tgtKey)!.push(lineKey);
    }

    const sourceH = new Map<string, number>();
    for (const [_, lineKeys] of sourceGroups) {
      const count = lineKeys.length;
      for (let i = 0; i < count; i++) {
        sourceH.set(lineKeys[i], (i - (count - 1) / 2) * 5);
      }
    }

    const targetH = new Map<string, number>();
    for (const [_, lineKeys] of targetGroups) {
      const count = lineKeys.length;
      for (let i = 0; i < count; i++) {
        targetH.set(lineKeys[i], (i - (count - 1) / 2) * 5);
      }
    }

    return { vertical, sourceH, targetH };
  }

  $: {
    if (activeTab === 'tables') {
      const result = computeLayout(state);
      layoutNodes = result.nodes;
      fkLines = result.lines;
      lineOffsets = computeLineOffsets(result.lines);
    } else {
      const result = computeViewLayout(state);
      layoutNodes = result.nodes;
      fkLines = [];
      lineOffsets = { vertical: new Map(), sourceH: new Map(), targetH: new Map() };
    }
  }

  let initialFitDone = false;

  $: if (activeTab) { initialFitDone = false; }

  afterUpdate(() => {
    if (layoutNodes.length > 0 && svgEl && !initialFitDone) {
      initialFitDone = true;
      fitToViewport();
    }
  });
</script>

<svg bind:this={svgEl} class="canvas">
  <g bind:this={gEl}>
    {#each fkLines as line (line.fk.fromSchema + '.' + line.fk.fromTable + '.' + line.fk.fromColumn)}
      {@const lk = line.fk.fromSchema + '.' + line.fk.fromTable + '.' + line.fk.fromColumn}
      <FKLine {line} verticalOffset={lineOffsets.vertical.get(lk) || 0} sourceHOffset={lineOffsets.sourceH.get(lk) || 0} targetHOffset={lineOffsets.targetH.get(lk) || 0} highlighted={hoveredTableKey === line.fk.fromSchema + '.' + line.fk.fromTable || hoveredTableKey === line.fk.toSchema + '.' + line.fk.toTable} />
    {/each}
    {#each layoutNodes as node (node.table.schema + '.' + node.table.name)}
      <TableCard {node} foreignKeys={state.foreignKeys} highlighted={highlightedTable === node.table.schema + '.' + node.table.name} on:columnhover={handleColumnHover} on:columnhoverend={handleColumnHoverEnd} on:tabledragstart={(e) => handleDragStart(e.detail.event, node)} on:unpin={handleUnpin} on:tablehover={handleTableHover} on:tablehoverend={handleTableHoverEnd} />
    {/each}
    {#if dragging}
      <rect
        x={ghostX}
        y={ghostY}
        width={dragging.width}
        height={dragging.height}
        rx={6}
        ry={6}
        fill="none"
        stroke={dragging.color}
        stroke-width="2"
        stroke-dasharray="6 3"
        opacity="0.6"
      />
    {/if}
  </g>
</svg>

{#if popoverVisible && popoverColumn}
  <ColumnPopover
    column={popoverColumn}
    table={popoverTable}
    foreignKeys={state.foreignKeys}
    enumTypes={state.enumTypes || []}
    x={popoverX}
    y={popoverY}
    visible={popoverVisible}
  />
{/if}

<style>
  .canvas {
    width: 100%;
    height: 100%;
    display: block;
    cursor: default;
  }

  .canvas:active {
    cursor: default;
  }
</style>
