<script lang="ts">
  import type { Column, Table, ForeignKey, EnumType } from './types';

  export let column: Column;
  export let table: Table;
  export let foreignKeys: ForeignKey[] = [];
  export let enumTypes: EnumType[] = [];
  export let x: number = 0;
  export let y: number = 0;
  export let visible: boolean = false;

  $: fk = foreignKeys.find(
    f => f.fromSchema === table.schema && f.fromTable === table.name && f.fromColumn === column.name
  );

  $: enumType = enumTypes.find(e => column.type.toLowerCase().includes(e.name.toLowerCase()));

  $: checkConstraints = (table.constraints || []).filter(c => c.type === 'check' && c.expression && c.expression.includes(column.name));

  $: hasProperties = column.primaryKey || column.unique || column.nullable || column.default;
</script>

{#if visible}
  <div class="popover" style="left: {x}px; top: {y}px;">
    <div class="section header-section">
      <span class="col-name">{column.name}</span>
      <span class="col-type">{column.type}</span>
    </div>

    {#if hasProperties}
      <div class="section">
        <div class="label">Properties</div>
        {#if column.primaryKey}<div class="value prop">Primary Key</div>{/if}
        {#if column.unique}<div class="value prop">Unique</div>{/if}
        {#if column.nullable}<div class="value prop">Nullable</div>{/if}
        {#if column.default}<div class="value prop">Default: {column.default}</div>{/if}
      </div>
    {/if}

    {#if column.comment}
      <div class="section">
        <div class="label">Comment</div>
        <div class="value">{column.comment}</div>
      </div>
    {/if}

    {#if column.generated}
      <div class="section">
        <div class="label">Generated</div>
        <div class="value" style="font-family: monospace; font-size: 12px;">{column.generated}</div>
      </div>
    {/if}

    {#if column.sourceSchema && column.sourceTable && column.sourceColumn}
      <div class="section">
        <div class="label">Source</div>
        <div class="value fk-target">{column.sourceSchema}.{column.sourceTable}.{column.sourceColumn}</div>
      </div>
    {/if}

    {#if fk}
      <div class="section">
        <div class="label">Foreign Key</div>
        <div class="value fk-target">References: {fk.toSchema}.{fk.toTable}.{fk.toColumn}</div>
        {#if fk.onDelete}<div class="value">On Delete: {fk.onDelete}</div>{/if}
        {#if fk.onUpdate}<div class="value">On Update: {fk.onUpdate}</div>{/if}
      </div>
    {/if}

    {#if enumType}
      <div class="section">
        <div class="label">Enum Values</div>
        <div class="enum-values">
          {#each enumType.values as val}
            <span class="enum-tag">{val}</span>
          {/each}
        </div>
      </div>
    {/if}

    {#if checkConstraints.length > 0}
      <div class="section">
        <div class="label">Check Constraints</div>
        {#each checkConstraints as cc}
          <div class="value">
            {#if cc.name}<span class="check-name">{cc.name}:</span>{/if}
            {#if cc.expression}<span>{cc.expression}</span>{/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
{/if}

<style>
  .popover {
    position: fixed;
    z-index: 100;
    background: rgba(15, 23, 42, 0.95);
    border: 1px solid rgba(71, 85, 105, 0.6);
    border-radius: 8px;
    padding: 14px 18px;
    min-width: 240px;
    max-width: 380px;
    backdrop-filter: blur(12px);
    pointer-events: none;
    font-size: 14px;
    color: #e2e8f0;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
  }

  .section {
    margin-bottom: 8px;
  }

  .section:last-child {
    margin-bottom: 0;
  }

  .header-section {
    display: flex;
    align-items: baseline;
    gap: 8px;
  }

  .col-name {
    font-weight: 600;
    font-size: 15px;
    color: #f1f5f9;
  }

  .col-type {
    color: #94a3b8;
    font-size: 13px;
  }

  .label {
    color: #94a3b8;
    font-size: 12px;
    margin-bottom: 3px;
  }

  .value {
    color: #e2e8f0;
  }

  .prop {
    font-size: 13px;
  }

  .fk-target {
    color: #22d3ee;
  }

  .enum-values {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 2px;
  }

  .enum-tag {
    background: rgba(71, 85, 105, 0.4);
    border-radius: 3px;
    padding: 1px 6px;
    font-size: 12px;
    color: #cbd5e1;
  }

  .check-name {
    color: #94a3b8;
    margin-right: 4px;
  }
</style>
