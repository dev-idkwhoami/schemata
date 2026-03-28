export interface Extension {
  name: string;
}

export interface ViewColumn {
  name: string;
  type: string;
  sourceSchema?: string;
  sourceTable?: string;
  sourceColumn?: string;
}

export interface View {
  schema: string;
  name: string;
  columns: ViewColumn[];
  comment?: string;
  position?: Position;
}

export interface Index {
  name: string;
  columns: string[];
  unique: boolean;
  type?: string;
  where?: string;
}

export interface Column {
  name: string;
  type: string;
  nullable: boolean;
  primaryKey: boolean;
  unique: boolean;
  default: string;
  comment?: string;
  generated?: string;
  sourceSchema?: string;
  sourceTable?: string;
  sourceColumn?: string;
}

export interface ForeignKey {
  fromSchema: string;
  fromTable: string;
  fromColumn: string;
  toSchema: string;
  toTable: string;
  toColumn: string;
  onDelete?: string;
  onUpdate?: string;
}

export interface TableConstraint {
  type: string;
  columns: string[];
  name?: string;
  expression?: string;
}

export interface Table {
  schema: string;
  name: string;
  columns: Column[];
  constraints: TableConstraint[];
  indexes: Index[];
  comment?: string;
  position?: Position;
}

export interface Schema {
  name: string;
  color: string;
}

export interface EnumType {
  schema: string;
  name: string;
  values: string[];
}

export interface Position {
  x: number;
  y: number;
}

export interface AppState {
  schemas: Schema[];
  tables: Table[];
  foreignKeys: ForeignKey[];
  enumTypes: EnumType[];
  extensions?: Extension[];
  views?: View[];
}

export interface LayoutNode {
  table: Table;
  x: number;
  y: number;
  width: number;
  height: number;
  color: string;
  kind: 'table' | 'view';
}

export interface FKLineData {
  fk: ForeignKey;
  sourceNode: LayoutNode;
  targetNode: LayoutNode;
  sourceColumnIndex: number;
  targetColumnIndex: number;
  color: string;
}
