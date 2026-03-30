<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { FKLineData } from './types';

  export let line: FKLineData;
  export let corridorX: number = 0;
  export let sourceHOffset: number = 0;
  export let targetHOffset: number = 0;
  export let highlighted: boolean = false;

  const dispatch = createEventDispatcher();
  let hovered = false;
  let lastVertX = 0;
  let lastTargetX = 0;
  let lastTargetY = 0;
  let lastSelfRef = false;

  function handleClick(e: MouseEvent) {
    e.stopPropagation();

    // Determine if the click is on the collapsed final horizontal segment
    // (between vertX and targetX) by converting SVG coords to screen coords.
    // Self-ref FKs share the same X range for both segments, so never treat as nearTarget.
    let nearTarget = false;
    if (!lastSelfRef) {
      const el = e.currentTarget as SVGElement;
      const ctm = el.getScreenCTM();
      if (ctm) {
        const screenVertX = ctm.a * lastVertX + ctm.e;
        const screenTargetX = ctm.a * lastTargetX + ctm.e;
        const screenTargetY = ctm.d * lastTargetY + ctm.f;
        const minX = Math.min(screenVertX, screenTargetX);
        const maxX = Math.max(screenVertX, screenTargetX);
        // Check X is in the target segment range AND Y is close to target row
        nearTarget = e.clientX >= minX && e.clientX <= maxX
          && Math.abs(e.clientY - screenTargetY) < 15;
      }
    }

    dispatch('fkclick', {
      fk: line.fk,
      x: e.clientX,
      y: e.clientY,
      nearTarget,
    });
  }

  const HEADER_HEIGHT = 32;
  const ROW_HEIGHT = 26;

  function computePath(line: FKLineData, corridorX: number, srcHOff: number, tgtHOff: number): string {
    const { sourceNode, targetNode, sourceColumnIndex, targetColumnIndex } = line;

    const sourceY = sourceNode.y + HEADER_HEIGHT + sourceColumnIndex * ROW_HEIGHT + ROW_HEIGHT / 2 + srcHOff;
    const targetY = targetNode.y + HEADER_HEIGHT + targetColumnIndex * ROW_HEIGHT + ROW_HEIGHT / 2 + tgtHOff;

    const isSelfRef = sourceNode.table.schema === targetNode.table.schema
      && sourceNode.table.name === targetNode.table.name;

    // Hard rule: exit always RIGHT, enter always LEFT.
    // Exception: self-ref FKs exit and re-enter on the right side.
    const sourceX = sourceNode.x + sourceNode.width; // right edge
    const targetX = isSelfRef
      ? sourceNode.x + sourceNode.width  // self-ref: re-enter right side
      : targetNode.x;                     // normal: enter left edge

    // Self-ref uses the channel too (exits and re-enters right side)
    const vertX = corridorX;

    lastVertX = vertX;
    lastTargetX = targetX;
    lastTargetY = targetY;
    lastSelfRef = isSelfRef;

    return `M ${sourceX} ${sourceY} H ${vertX} V ${targetY} H ${targetX}`;
  }

  $: path = computePath(line, corridorX, sourceHOffset, targetHOffset);
</script>

<g
  on:mouseenter={() => hovered = true}
  on:mouseleave={() => hovered = false}
>
  <!-- Invisible wider hit area for hover + click -->
  <path
    d={path}
    fill="none"
    stroke="transparent"
    stroke-width="10"
    style="cursor: {lastSelfRef ? 'default' : 'pointer'}"
    on:click={lastSelfRef ? undefined : handleClick}
  />
  <path
    d={path}
    fill="none"
    stroke={line.color}
    stroke-width="2"
    stroke-opacity={highlighted || hovered ? 1 : 0.25}
    marker-end="url(#arrowhead-{line.color.replace('#', '')})"
    style="transition: stroke-opacity 0.15s"
    pointer-events="none"
  />

  <defs>
    <marker
      id="arrowhead-{line.color.replace('#', '')}"
      markerWidth="8"
      markerHeight="6"
      refX="7"
      refY="3"
      orient="auto"
    >
      <polygon
        points="0 0, 8 3, 0 6"
        fill={line.color}
        fill-opacity="0.7"
      />
    </marker>
  </defs>
</g>
