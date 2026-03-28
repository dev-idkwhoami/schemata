export namespace main {
	
	export class ViewColumn {
	    name: string;
	    type: string;
	    sourceSchema?: string;
	    sourceTable?: string;
	    sourceColumn?: string;
	
	    static createFrom(source: any = {}) {
	        return new ViewColumn(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.sourceSchema = source["sourceSchema"];
	        this.sourceTable = source["sourceTable"];
	        this.sourceColumn = source["sourceColumn"];
	    }
	}
	export class View {
	    schema: string;
	    name: string;
	    columns: ViewColumn[];
	    comment?: string;
	    position?: Position;
	
	    static createFrom(source: any = {}) {
	        return new View(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema = source["schema"];
	        this.name = source["name"];
	        this.columns = this.convertValues(source["columns"], ViewColumn);
	        this.comment = source["comment"];
	        this.position = this.convertValues(source["position"], Position);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Extension {
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Extension(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	    }
	}
	export class EnumType {
	    schema: string;
	    name: string;
	    values: string[];
	
	    static createFrom(source: any = {}) {
	        return new EnumType(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema = source["schema"];
	        this.name = source["name"];
	        this.values = source["values"];
	    }
	}
	export class ForeignKey {
	    fromSchema: string;
	    fromTable: string;
	    fromColumn: string;
	    toSchema: string;
	    toTable: string;
	    toColumn: string;
	    onDelete?: string;
	    onUpdate?: string;
	
	    static createFrom(source: any = {}) {
	        return new ForeignKey(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fromSchema = source["fromSchema"];
	        this.fromTable = source["fromTable"];
	        this.fromColumn = source["fromColumn"];
	        this.toSchema = source["toSchema"];
	        this.toTable = source["toTable"];
	        this.toColumn = source["toColumn"];
	        this.onDelete = source["onDelete"];
	        this.onUpdate = source["onUpdate"];
	    }
	}
	export class Position {
	    x: number;
	    y: number;
	
	    static createFrom(source: any = {}) {
	        return new Position(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	export class Index {
	    name: string;
	    columns: string[];
	    unique: boolean;
	    type?: string;
	    where?: string;
	
	    static createFrom(source: any = {}) {
	        return new Index(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.columns = source["columns"];
	        this.unique = source["unique"];
	        this.type = source["type"];
	        this.where = source["where"];
	    }
	}
	export class TableConstraint {
	    type: string;
	    columns: string[];
	    name?: string;
	    expression?: string;
	
	    static createFrom(source: any = {}) {
	        return new TableConstraint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.columns = source["columns"];
	        this.name = source["name"];
	        this.expression = source["expression"];
	    }
	}
	export class Column {
	    name: string;
	    type: string;
	    nullable: boolean;
	    primaryKey: boolean;
	    unique: boolean;
	    default: string;
	    comment?: string;
	    generated?: string;
	
	    static createFrom(source: any = {}) {
	        return new Column(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.nullable = source["nullable"];
	        this.primaryKey = source["primaryKey"];
	        this.unique = source["unique"];
	        this.default = source["default"];
	        this.comment = source["comment"];
	        this.generated = source["generated"];
	    }
	}
	export class Table {
	    schema: string;
	    name: string;
	    comment?: string;
	    columns: Column[];
	    constraints: TableConstraint[];
	    indexes: Index[];
	    position?: Position;
	
	    static createFrom(source: any = {}) {
	        return new Table(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema = source["schema"];
	        this.name = source["name"];
	        this.comment = source["comment"];
	        this.columns = this.convertValues(source["columns"], Column);
	        this.constraints = this.convertValues(source["constraints"], TableConstraint);
	        this.indexes = this.convertValues(source["indexes"], Index);
	        this.position = this.convertValues(source["position"], Position);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Schema {
	    name: string;
	    color: string;
	
	    static createFrom(source: any = {}) {
	        return new Schema(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.color = source["color"];
	    }
	}
	export class AppState {
	    schemas: Schema[];
	    tables: Table[];
	    foreignKeys: ForeignKey[];
	    enumTypes: EnumType[];
	    extensions?: Extension[];
	    views?: View[];
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schemas = this.convertValues(source["schemas"], Schema);
	        this.tables = this.convertValues(source["tables"], Table);
	        this.foreignKeys = this.convertValues(source["foreignKeys"], ForeignKey);
	        this.enumTypes = this.convertValues(source["enumTypes"], EnumType);
	        this.extensions = this.convertValues(source["extensions"], Extension);
	        this.views = this.convertValues(source["views"], View);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	
	
	

}

