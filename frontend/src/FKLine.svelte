<script lang="ts">
  import type { FKLineData } from './types';

  export let line: FKLineData;
  export let verticalOffset: number = 0;
  export let sourceHOffset: number = 0;
  export let targetHOffset: number = 0;
  export let highlighted: boolean = false;

  let hovered = false;

  const HEADER_HEIGHT = 32;
  const ROW_HEIGHT = 26;
  const CLEARANCE = 40;

  function computePath(line: FKLineData, verticalOffset: number, srcHOff: number, tgtHOff: number): string {
    const { sourceNode, targetNode, sourceColumnIndex, targetColumnIndex } = line;

    const sourceY = sourceNode.y + HEADER_HEIGHT + sourceColumnIndex * ROW_HEIGHT + ROW_HEIGHT / 2 + srcHOff;
    const targetY = targetNode.y + HEADER_HEIGHT + targetColumnIndex * ROW_HEIGHT + ROW_HEIGHT / 2 + tgtHOff;

    let sourceX: number;
    let targetX: number;
    let exitDir: number;

    const sourceCenterX = sourceNode.x + sourceNode.width / 2;
    const targetCenterX = targetNode.x + targetNode.width / 2;

    if (sourceCenterX <= targetCenterX) {
      sourceX = sourceNode.x + sourceNode.width;
      targetX = targetNode.x;
      exitDir = 1;
    } else {
      sourceX = sourceNode.x;
      targetX = targetNode.x + targetNode.width;
      exitDir = -1;
    }

    const vertX = sourceX + exitDir * CLEARANCE + exitDir * verticalOffset;

    return `M ${sourceX} ${sourceY} H ${vertX} V ${targetY} H ${targetX}`;
  }

  $: path = computePath(line, verticalOffset, sourceHOffset, targetHOffset);
</script>

<g
  on:mouseenter={() => hovered = true}
  on:mouseleave={() => hovered = false}
>
  <!-- Invisible wider hit area for hover -->
  <path
    d={path}
    fill="none"
    stroke="transparent"
    stroke-width="10"
  />
  <path
    d={path}
    fill="none"
    stroke={line.color}
    stroke-width="2"
    stroke-opacity={highlighted || hovered ? 1 : 0.25}
    marker-end="url(#arrowhead-{line.color.replace('#', '')})"
    style="transition: stroke-opacity 0.15s"
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
