<script lang="ts">
  import { onMount, afterUpdate } from 'svelte';
  import { select } from 'd3-selection';
  import { zoom, zoomIdentity, zoomTransform, type ZoomBehavior } from 'd3-zoom';
  import type { AppState, LayoutNode, FKLineData, Table, ForeignKey } from './types';
  import TableCard from './TableCard.svelte';
  import FKLine from './FKLine.svelte';
  import FKContextMenu from './FKContextMenu.svelte';
  import CanvasContextMenu from './CanvasContextMenu.svelte';
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
  let lineOffsets: LineOffsets = { corridorX: new Map(), sourceH: new Map(), targetH: new Map() };
  let highlightedTable = '';

  let popoverVisible = false;
  let hoveredTableKey = '';
  let currentZoomScale = 1;
  let popoverColumn: any = null;
  let popoverTable: any = null;
  let popoverX = 0;
  let popoverY = 0;

  let fkMenuVisible = false;
  let fkMenuX = 0;
  let fkMenuY = 0;
  let fkMenuTargets: { schema: string; table: string }[] = [];

  let ctxMenuVisible = false;
  let ctxMenuX = 0;
  let ctxMenuY = 0;

  let selectedTables = new Set<string>();
  let rubberBand: { startX: number; startY: number; curX: number; curY: number } | null = null;
  let justDragged = false;

  let dragging: { startX: number; startY: number; nodes: { key: string; schema: string; name: string; startX: number; startY: number; width: number; height: number; color: string }[] } | null = null;
  let dragGhosts: { x: number; y: number; width: number; height: number; color: string }[] = [];

  function handleDragStart(e: MouseEvent, node: LayoutNode) {
    if (e.button !== 0) return;
    e.stopPropagation();
    e.preventDefault();
    fkMenuVisible = false;

    const nodeKey = tableKey(node.table.schema, node.table.name);

    // If dragged table isn't in selection, select it alone
    if (!selectedTables.has(nodeKey)) {
      selectedTables = new Set([nodeKey]);
    }

    // Gather all selected nodes for group drag
    const dragNodes = layoutNodes
      .filter(n => selectedTables.has(tableKey(n.table.schema, n.table.name)))
      .map(n => ({ key: tableKey(n.table.schema, n.table.name), schema: n.table.schema, name: n.table.name, startX: n.x, startY: n.y, width: n.width, height: n.height, color: n.color }));

    dragging = { startX: e.clientX, startY: e.clientY, nodes: dragNodes };
    dragGhosts = dragNodes.map(n => ({ x: n.startX, y: n.startY, width: n.width, height: n.height, color: n.color }));

    document.body.style.cursor = 'grabbing';
    window.addEventListener('mousemove', handleDragMove);
    window.addEventListener('mouseup', handleDragEnd);
  }

  function handleDragMove(e: MouseEvent) {
    if (!dragging || !svgEl) return;
    const transform = zoomTransform(svgEl);
    const dx = (e.clientX - dragging.startX) / transform.k;
    const dy = (e.clientY - dragging.startY) / transform.k;
    dragGhosts = dragging.nodes.map(n => ({ x: n.startX + dx, y: n.startY + dy, width: n.width, height: n.height, color: n.color }));
  }

  function handleDragEnd(e: MouseEvent) {
    if (!dragging || !svgEl) return;
    const transform = zoomTransform(svgEl);
    const dx = (e.clientX - dragging.startX) / transform.k;
    const dy = (e.clientY - dragging.startY) / transform.k;

    for (const n of dragging.nodes) {
      if (activeTab === 'views') {
        SetViewPosition(n.schema, n.name, n.startX + dx, n.startY + dy);
      } else {
        SetTablePosition(n.schema, n.name, n.startX + dx, n.startY + dy);
      }
    }

    window.removeEventListener('mousemove', handleDragMove);
    window.removeEventListener('mouseup', handleDragEnd);
    document.body.style.cursor = '';
    dragging = null;
    dragGhosts = [];
    // Suppress the click event that fires after drag
    justDragged = true;
    setTimeout(() => { justDragged = false; }, 50);
  }

  function handleUnpin(e: CustomEvent) {
    const { schema, table } = e.detail;
    if (activeTab === 'views') {
      ClearViewPosition(schema, table);
    } else {
      ClearTablePosition(schema, table);
    }
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

  function makeNode(t: Table, x: number, y: number): LayoutNode {
    return { table: t, x, y, width: TABLE_WIDTH, height: getTableHeight(t), color: getSchemaColor(t.schema), kind: 'table' };
  }

  const MAX_PER_ROW = 5;

  function computeLayout(state: AppState): { nodes: LayoutNode[]; lines: FKLineData[] } {
    if (state.tables.length === 0) return { nodes: [], lines: [] };

    // Build undirected adjacency for BFS ordering
    const neighbors = new Map<string, Set<string>>();
    const connectionCount = new Map<string, number>();
    for (const t of state.tables) {
      const k = tableKey(t.schema, t.name);
      neighbors.set(k, new Set());
      connectionCount.set(k, 0);
    }
    for (const fk of state.foreignKeys) {
      const fk1 = tableKey(fk.fromSchema, fk.fromTable);
      const fk2 = tableKey(fk.toSchema, fk.toTable);
      connectionCount.set(fk1, (connectionCount.get(fk1) || 0) + 1);
      connectionCount.set(fk2, (connectionCount.get(fk2) || 0) + 1);
      if (fk1 !== fk2) { neighbors.get(fk1)?.add(fk2); neighbors.get(fk2)?.add(fk1); }
    }

    // BFS from most-connected table across ALL schemas — connectivity grouping
    const tableMap = new Map<string, Table>();
    for (const t of state.tables) tableMap.set(tableKey(t.schema, t.name), t);

    let rootKey = '';
    let maxC = -1;
    for (const [k, c] of connectionCount) {
      if (c > maxC) { maxC = c; rootKey = k; }
    }

    const visited = new Set<string>();
    const ordered: Table[] = [];
    const queue = [rootKey];
    if (rootKey) visited.add(rootKey);

    while (queue.length > 0) {
      const k = queue.shift()!;
      const t = tableMap.get(k);
      if (t) ordered.push(t);
      const nbrs = neighbors.get(k) || new Set();
      for (const nk of nbrs) {
        if (!visited.has(nk)) { visited.add(nk); queue.push(nk); }
      }
    }
    // Add disconnected tables (no FK connections)
    for (const t of state.tables) {
      if (!visited.has(tableKey(t.schema, t.name))) ordered.push(t);
    }

    // Place in grid: max MAX_PER_ROW per row, actual heights
    const nodes: LayoutNode[] = [];
    const colWidth = TABLE_WIDTH + NODE_GAP_X;
    let rowY = 0;

    for (let i = 0; i < ordered.length; i += MAX_PER_ROW) {
      const row = ordered.slice(i, i + MAX_PER_ROW);
      let rowMaxH = 0;
      for (let j = 0; j < row.length; j++) {
        const t = row[j];
        const h = getTableHeight(t);
        nodes.push(makeNode(t, j * colWidth, rowY));
        if (h > rowMaxH) rowMaxH = h;
      }
      rowY += rowMaxH + NODE_GAP_Y;
    }

    // Override positions for pinned tables
    for (const n of nodes) {
      if (n.table.position) { n.x = n.table.position.x; n.y = n.table.position.y; }
    }

    // Compute FK lines
    const nodeMap = new Map<string, LayoutNode>();
    for (const n of nodes) nodeMap.set(tableKey(n.table.schema, n.table.name), n);

    const lines: FKLineData[] = [];
    for (const fk of state.foreignKeys) {
      const sourceNode = nodeMap.get(tableKey(fk.fromSchema, fk.fromTable));
      const targetNode = nodeMap.get(tableKey(fk.toSchema, fk.toTable));
      if (!sourceNode || !targetNode) continue;
      const sourceColIdx = sourceNode.table.columns.findIndex(c => c.name === fk.fromColumn);
      const targetColIdx = targetNode.table.columns.findIndex(c => c.name === fk.toColumn);
      lines.push({
        fk, sourceNode, targetNode,
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
    fkMenuVisible = false;
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

  function handleFKClick(e: CustomEvent) {
    const { fk, x, y, nearTarget } = e.detail;

    let targets: { schema: string; table: string }[];

    if (nearTarget) {
      // Clicked the collapsed final segment — show all connections to this target column
      const targetKey = `${fk.toSchema}.${fk.toTable}.${fk.toColumn}`;
      const seen = new Set<string>();
      targets = [];

      for (const line of fkLines) {
        const lineTargetKey = `${line.fk.toSchema}.${line.fk.toTable}.${line.fk.toColumn}`;
        if (lineTargetKey !== targetKey) continue;

        const fromKey = `${line.fk.fromSchema}.${line.fk.fromTable}`;
        if (!seen.has(fromKey)) {
          seen.add(fromKey);
          targets.push({ schema: line.fk.fromSchema, table: line.fk.fromTable });
        }
      }

      const toKey = `${fk.toSchema}.${fk.toTable}`;
      if (!seen.has(toKey)) {
        seen.add(toKey);
        targets.push({ schema: fk.toSchema, table: fk.toTable });
      }
    } else {
      // Clicked the spread-out part — show only this connection's two tables
      targets = [
        { schema: fk.fromSchema, table: fk.fromTable },
        { schema: fk.toSchema, table: fk.toTable },
      ];
    }

    fkMenuTargets = targets;
    fkMenuX = x;
    fkMenuY = y;
    fkMenuVisible = true;
    popoverVisible = false;
  }

  function handleFKNavigate(e: CustomEvent) {
    const { schema, table } = e.detail;
    fkMenuVisible = false;
    doPanToTable(schema, table);
  }

  function handleFKMenuClose() {
    fkMenuVisible = false;
  }

  function handleTableClick(e: CustomEvent) {
    const { event, schema, name } = e.detail;
    const mouseEvent = event as MouseEvent;
    const key = tableKey(schema, name);

    if (mouseEvent.ctrlKey || mouseEvent.metaKey) {
      // Ctrl+click: toggle selection
      const next = new Set(selectedTables);
      if (next.has(key)) next.delete(key); else next.add(key);
      selectedTables = next;
    } else {
      selectedTables = new Set([key]);
    }
  }

  function handleCanvasClick(e: MouseEvent) {
    if (justDragged) return;
    // Click on empty space clears selection
    const target = e.target as Element;
    if (target === svgEl || target === gEl) {
      selectedTables = new Set();
    }
  }

  function handleRubberBandStart(e: MouseEvent) {
    if (!e.ctrlKey || e.button !== 0 || !svgEl) return;
    // Start rubber band selection
    const transform = zoomTransform(svgEl);
    const rect = svgEl.getBoundingClientRect();
    const x = (e.clientX - rect.left - transform.x) / transform.k;
    const y = (e.clientY - rect.top - transform.y) / transform.k;
    rubberBand = { startX: x, startY: y, curX: x, curY: y };

    // Disable d3 zoom temporarily during rubber band
    e.stopPropagation();
    e.preventDefault();

    function onMove(ev: MouseEvent) {
      if (!rubberBand || !svgEl) return;
      const t = zoomTransform(svgEl);
      const r = svgEl.getBoundingClientRect();
      rubberBand = { ...rubberBand, curX: (ev.clientX - r.left - t.x) / t.k, curY: (ev.clientY - r.top - t.y) / t.k };

      // Preview: highlight tables within bounds
      const minX = Math.min(rubberBand.startX, rubberBand.curX);
      const maxX = Math.max(rubberBand.startX, rubberBand.curX);
      const minY = Math.min(rubberBand.startY, rubberBand.curY);
      const maxY = Math.max(rubberBand.startY, rubberBand.curY);

      const next = new Set<string>();
      for (const n of layoutNodes) {
        if (n.x + n.width > minX && n.x < maxX && n.y + n.height > minY && n.y < maxY) {
          next.add(tableKey(n.table.schema, n.table.name));
        }
      }
      selectedTables = next;
    }

    function onUp() {
      rubberBand = null;
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup', onUp);
    }

    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup', onUp);
  }

  function handleContextMenu(e: MouseEvent) {
    e.preventDefault();
    ctxMenuVisible = true;
    ctxMenuX = e.clientX;
    ctxMenuY = e.clientY;
    fkMenuVisible = false;
    popoverVisible = false;
  }

  function handleCtxMenuClose() {
    ctxMenuVisible = false;
  }

  function handleExportVisual(e: CustomEvent) {
    ctxMenuVisible = false;
    const { format } = e.detail;

    // Determine which tables to export
    const targetKeys = selectedTables.size > 0 ? selectedTables : new Set(layoutNodes.map(n => tableKey(n.table.schema, n.table.name)));
    let exportNodes = layoutNodes.filter(n => targetKeys.has(tableKey(n.table.schema, n.table.name)));
    if (exportNodes.length === 0) return;

    // When exporting a selection, re-layout into a compact square-ish grid
    if (selectedTables.size > 0) {
      const colW = TABLE_WIDTH + NODE_GAP_X * 2; // extra gap for line channels
      const perRow = Math.max(2, Math.ceil(Math.sqrt(exportNodes.length)));
      const compact: LayoutNode[] = [];
      const margin = 80;
      let rowY = margin;
      for (let i = 0; i < exportNodes.length; i += perRow) {
        const row = exportNodes.slice(i, i + perRow);
        let rowMaxH = 0;
        for (let j = 0; j < row.length; j++) {
          const n = row[j];
          compact.push({ ...n, x: margin + j * colW, y: rowY });
          if (n.height > rowMaxH) rowMaxH = n.height;
        }
        rowY += rowMaxH + NODE_GAP_Y;
      }
      exportNodes = compact;
    }

    // Filter FK lines: only include where BOTH source and target are in export set
    // Re-map lines to use the (possibly re-laid-out) export node positions
    const exportNodeMap = new Map<string, LayoutNode>();
    for (const n of exportNodes) exportNodeMap.set(tableKey(n.table.schema, n.table.name), n);

    const exportLines = fkLines
      .filter(l =>
        targetKeys.has(tableKey(l.fk.fromSchema, l.fk.fromTable)) &&
        targetKeys.has(tableKey(l.fk.toSchema, l.fk.toTable))
      )
      .map(l => ({
        ...l,
        sourceNode: exportNodeMap.get(tableKey(l.fk.fromSchema, l.fk.fromTable))!,
        targetNode: exportNodeMap.get(tableKey(l.fk.toSchema, l.fk.toTable))!,
      }));

    // Recompute channels for the export layout
    const exportOffsets = computeLineOffsets(exportLines, exportNodes);

    // Bounding box
    const pad = 60;
    let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
    for (const n of exportNodes) {
      minX = Math.min(minX, n.x); minY = Math.min(minY, n.y);
      maxX = Math.max(maxX, n.x + n.width); maxY = Math.max(maxY, n.y + n.height);
    }
    minX -= pad * 2; minY -= pad * 2; maxX += pad * 2; maxY += pad * 2;
    const w = maxX - minX;
    const h = maxY - minY;

    // Build SVG from data
    const HH = 32, RH = 26, BR = 6, TP = 8;
    let svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="${minX} ${minY} ${w} ${h}">`;
    svg += `<style>text,tspan{font-family:'Inter',-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;}</style>`;
    svg += `<rect width="100%" height="100%" fill="#111827"/>`;

    // Render FK lines
    const useOffsets = selectedTables.size > 0 ? exportOffsets : lineOffsets;
    for (const line of exportLines) {
      const lk = `${line.fk.fromSchema}.${line.fk.fromTable}.${line.fk.fromColumn}`;
      const cx = useOffsets.corridorX.get(lk) || (line.sourceNode.x + line.sourceNode.width + 40);
      const srcOff = useOffsets.sourceH.get(lk) || 0;
      const isSelf = line.sourceNode.table.schema === line.targetNode.table.schema && line.sourceNode.table.name === line.targetNode.table.name;
      const sx = line.sourceNode.x + line.sourceNode.width;
      const sy = line.sourceNode.y + HH + line.sourceColumnIndex * RH + RH / 2 + srcOff;
      const ty = line.targetNode.y + HH + line.targetColumnIndex * RH + RH / 2;
      const tx = isSelf ? sx : line.targetNode.x;
      const vx = cx;
      const d = `M ${sx} ${sy} H ${vx} V ${ty} H ${tx}`;
      const col = line.color.replace('#', '');
      svg += `<defs><marker id="ah-${col}" markerWidth="8" markerHeight="6" refX="7" refY="3" orient="auto"><polygon points="0 0,8 3,0 6" fill="${line.color}" fill-opacity="0.7"/></marker></defs>`;
      svg += `<path d="${d}" fill="none" stroke="${line.color}" stroke-width="2" stroke-opacity="0.5" marker-end="url(#ah-${col})"/>`;
    }

    // Render table cards
    for (const n of exportNodes) {
      const t = n.table;
      const idxCount = t.indexes?.length || 0;
      const idxSection = idxCount > 0 ? idxCount * RH + 4 : 0;
      const th = HH + t.columns.length * RH + idxSection + TP;

      svg += `<g transform="translate(${n.x},${n.y})">`;
      // Card bg
      svg += `<rect width="${n.width}" height="${th}" rx="${BR}" fill="#1e293b" stroke="#334155" stroke-width="1"/>`;
      // Header
      svg += `<rect width="${n.width}" height="${HH}" rx="${BR}" fill="${n.color}"/>`;
      svg += `<rect y="${HH - BR}" width="${n.width}" height="${BR}" fill="${n.color}"/>`;
      // Table name
      svg += `<text x="28" y="${HH / 2}" dominant-baseline="central" fill="white" font-weight="600" font-size="13px">${escapeXml(t.name)}</text>`;
      // Columns
      for (let i = 0; i < t.columns.length; i++) {
        const col = t.columns[i];
        const cy = HH + i * RH;
        if (i > 0) svg += `<line x1="0" y1="${cy}" x2="${n.width}" y2="${cy}" stroke="#334155" stroke-width="0.5"/>`;
        svg += `<text x="28" y="${cy + RH / 2}" dominant-baseline="central" fill="#e2e8f0" font-size="13px">${escapeXml(col.name)}</text>`;
        svg += `<text x="${n.width - 12}" y="${cy + RH / 2}" dominant-baseline="central" fill="#64748b" font-size="12px" text-anchor="end">${escapeXml(col.type)}</text>`;
      }
      svg += `</g>`;
    }

    svg += `</svg>`;

    function escapeXml(s: string): string {
      return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
    }

    if (format === 'svg') {
      const blob = new Blob([svg], { type: 'image/svg+xml' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url; a.download = 'schema.svg'; a.click();
      URL.revokeObjectURL(url);
    } else {
      const img = new Image();
      const scale = 2;
      img.onload = () => {
        const canvas = document.createElement('canvas');
        canvas.width = w * scale; canvas.height = h * scale;
        const ctx = canvas.getContext('2d')!;
        ctx.scale(scale, scale);
        ctx.drawImage(img, 0, 0, w, h);
        canvas.toBlob(blob => {
          if (!blob) return;
          const url = URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url; a.download = 'schema.png'; a.click();
          URL.revokeObjectURL(url);
        }, 'image/png');
      };
      img.src = 'data:image/svg+xml;base64,' + btoa(unescape(encodeURIComponent(svg)));
    }
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
    corridorX: Map<string, number>;
    sourceH: Map<string, number>;
    targetH: Map<string, number>;
  }

  function computeLineOffsets(lines: FKLineData[], nodes: LayoutNode[]): LineOffsets {
    const BUFFER = 30;
    const CHANNEL_GAP = 8;

    const corridorXMap = new Map<string, number>();

    // Group lines by source table
    const bySource = new Map<string, FKLineData[]>();
    for (const line of lines) {
      const sk = `${line.fk.fromSchema}.${line.fk.fromTable}`;
      if (!bySource.has(sk)) bySource.set(sk, []);
      bySource.get(sk)!.push(line);
    }

    for (const [_, srcLines] of bySource) {
      const sourceRightEdge = srcLines[0].sourceNode.x + srcLines[0].sourceNode.width;

      // Sub-group by target column — same target = same channel (converge)
      const byTargetCol = new Map<string, FKLineData[]>();
      for (const line of srcLines) {
        const tck = `${line.fk.toSchema}.${line.fk.toTable}.${line.fk.toColumn}`;
        if (!byTargetCol.has(tck)) byTargetCol.set(tck, []);
        byTargetCol.get(tck)!.push(line);
      }

      // Assign a channel per target group
      let slot = 0;
      for (const [_, tgtLines] of byTargetCol) {
        let channelX = sourceRightEdge + BUFFER + slot * CHANNEL_GAP;

        // Compute the Y span this channel covers
        let minY = Infinity, maxY = -Infinity;
        for (const line of tgtLines) {
          const sy = line.sourceNode.y + HEADER_HEIGHT + line.sourceColumnIndex * ROW_HEIGHT + ROW_HEIGHT / 2;
          const ty = line.targetNode.y + HEADER_HEIGHT + line.targetColumnIndex * ROW_HEIGHT + ROW_HEIGHT / 2;
          minY = Math.min(minY, sy, ty);
          maxY = Math.max(maxY, sy, ty);
        }

        // Validate: push channel right past any table whose bounding box it crosses
        let pushed = true;
        while (pushed) {
          pushed = false;
          for (const node of nodes) {
            // Skip tables that don't overlap the channel's Y span
            if (node.y + node.height <= minY || node.y >= maxY) continue;
            // If channel falls inside this table (with buffer), push past it
            if (channelX > node.x - BUFFER && channelX < node.x + node.width + BUFFER) {
              channelX = node.x + node.width + BUFFER;
              pushed = true;
            }
          }
        }

        for (const line of tgtLines) {
          const lk = `${line.fk.fromSchema}.${line.fk.fromTable}.${line.fk.fromColumn}`;
          corridorXMap.set(lk, channelX);
        }
        slot++;
      }
    }

    // Source H offsets — spread lines from same source column vertically
    const sourceGroups = new Map<string, string[]>();
    for (const line of lines) {
      const lk = `${line.fk.fromSchema}.${line.fk.fromTable}.${line.fk.fromColumn}`;
      const srcKey = `${line.fk.fromSchema}.${line.fk.fromTable}.${line.sourceColumnIndex}`;
      if (!sourceGroups.has(srcKey)) sourceGroups.set(srcKey, []);
      sourceGroups.get(srcKey)!.push(lk);
    }

    const sourceH = new Map<string, number>();
    for (const [_, lineKeys] of sourceGroups) {
      const count = lineKeys.length;
      for (let i = 0; i < count; i++) {
        sourceH.set(lineKeys[i], (i - (count - 1) / 2) * 5);
      }
    }

    const targetH = new Map<string, number>();

    return { corridorX: corridorXMap, sourceH, targetH };
  }

  $: {
    if (activeTab === 'tables') {
      const result = computeLayout(state);
      layoutNodes = result.nodes;
      fkLines = result.lines;
      lineOffsets = computeLineOffsets(result.lines, result.nodes);
    } else {
      const result = computeViewLayout(state);
      layoutNodes = result.nodes;
      fkLines = [];
      lineOffsets = { corridorX: new Map(), sourceH: new Map(), targetH: new Map() };
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

<svg bind:this={svgEl} class="canvas" on:click={handleCanvasClick} on:mousedown={handleRubberBandStart} on:contextmenu={handleContextMenu}>
  <g bind:this={gEl}>
    {#each fkLines as line (line.fk.fromSchema + '.' + line.fk.fromTable + '.' + line.fk.fromColumn)}
      {@const lk = line.fk.fromSchema + '.' + line.fk.fromTable + '.' + line.fk.fromColumn}
      <FKLine {line} corridorX={lineOffsets.corridorX.get(lk) || (line.sourceNode.x + line.sourceNode.width + 40)} sourceHOffset={lineOffsets.sourceH.get(lk) || 0} targetHOffset={lineOffsets.targetH.get(lk) || 0} highlighted={hoveredTableKey === line.fk.fromSchema + '.' + line.fk.fromTable || hoveredTableKey === line.fk.toSchema + '.' + line.fk.toTable} on:fkclick={handleFKClick} />
    {/each}
    {#each layoutNodes as node (node.table.schema + '.' + node.table.name)}
      <TableCard {node} foreignKeys={state.foreignKeys} highlighted={highlightedTable === node.table.schema + '.' + node.table.name} selected={selectedTables.has(node.table.schema + '.' + node.table.name)} on:columnhover={handleColumnHover} on:columnhoverend={handleColumnHoverEnd} on:tabledragstart={(e) => handleDragStart(e.detail.event, node)} on:unpin={handleUnpin} on:tablehover={handleTableHover} on:tablehoverend={handleTableHoverEnd} on:tableclick={handleTableClick} />
    {/each}
    {#each dragGhosts as ghost}
      <rect
        x={ghost.x}
        y={ghost.y}
        width={ghost.width}
        height={ghost.height}
        rx={6}
        ry={6}
        fill="none"
        stroke={ghost.color}
        stroke-width="2"
        stroke-dasharray="6 3"
        opacity="0.6"
      />
    {/each}
    {#if rubberBand}
      <rect
        x={Math.min(rubberBand.startX, rubberBand.curX)}
        y={Math.min(rubberBand.startY, rubberBand.curY)}
        width={Math.abs(rubberBand.curX - rubberBand.startX)}
        height={Math.abs(rubberBand.curY - rubberBand.startY)}
        fill="rgba(99, 102, 241, 0.1)"
        stroke="#6366f1"
        stroke-width="1"
        stroke-dasharray="4 2"
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

<FKContextMenu
  visible={fkMenuVisible}
  x={fkMenuX}
  y={fkMenuY}
  targets={fkMenuTargets}
  on:navigate={handleFKNavigate}
  on:close={handleFKMenuClose}
/>

<CanvasContextMenu
  visible={ctxMenuVisible}
  x={ctxMenuX}
  y={ctxMenuY}
  hasSelection={selectedTables.size > 0}
  on:export={handleExportVisual}
  on:close={handleCtxMenuClose}
/>

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
