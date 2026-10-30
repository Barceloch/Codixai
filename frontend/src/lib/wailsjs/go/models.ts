export namespace application {
	
	export class VerificationSummary {
	    total: number;
	    by_status?: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new VerificationSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.by_status = source["by_status"];
	    }
	}
	export class VerificationDocument {
	    schema_version: string;
	    tool_version: string;
	    generated_at: string;
	    base: string;
	    head: string;
	    results: domain.ClaimResult[];
	    summary: VerificationSummary;
	    limitations?: string[];
	
	    static createFrom(source: any = {}) {
	        return new VerificationDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema_version = source["schema_version"];
	        this.tool_version = source["tool_version"];
	        this.generated_at = source["generated_at"];
	        this.base = source["base"];
	        this.head = source["head"];
	        this.results = this.convertValues(source["results"], domain.ClaimResult);
	        this.summary = this.convertValues(source["summary"], VerificationSummary);
	        this.limitations = source["limitations"];
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
	
	export class VerifyResult {
	    Document?: VerificationDocument;
	    Failed: boolean;
	    Contradicted: number;
	    Unresolved: number;
	    FailOnUnresolved: boolean;
	
	    static createFrom(source: any = {}) {
	        return new VerifyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Document = this.convertValues(source["Document"], VerificationDocument);
	        this.Failed = source["Failed"];
	        this.Contradicted = source["Contradicted"];
	        this.Unresolved = source["Unresolved"];
	        this.FailOnUnresolved = source["FailOnUnresolved"];
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

export namespace domain {
	
	export class Rule {
	    id: string;
	    version: string;
	    area: string;
	    description: string;
	    severity: string;
	    languages: Record<string, string>;
	    method?: string;
	    limitations?: string[];
	    evidence_kind?: string;
	
	    static createFrom(source: any = {}) {
	        return new Rule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.version = source["version"];
	        this.area = source["area"];
	        this.description = source["description"];
	        this.severity = source["severity"];
	        this.languages = source["languages"];
	        this.method = source["method"];
	        this.limitations = source["limitations"];
	        this.evidence_kind = source["evidence_kind"];
	    }
	}
	export class Summary {
	    total: number;
	    by_severity?: Record<string, number>;
	    by_rule?: Record<string, number>;
	    by_language?: Record<string, number>;
	    by_epistemic_status?: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.by_severity = source["by_severity"];
	        this.by_rule = source["by_rule"];
	        this.by_language = source["by_language"];
	        this.by_epistemic_status = source["by_epistemic_status"];
	    }
	}
	export class Support {
	    level: string;
	    method?: string;
	    limitations?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Support(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.method = source["method"];
	        this.limitations = source["limitations"];
	    }
	}
	export class Evidence {
	    kind: string;
	    ref: string;
	    detail?: string;
	
	    static createFrom(source: any = {}) {
	        return new Evidence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.ref = source["ref"];
	        this.detail = source["detail"];
	    }
	}
	export class Location {
	    path: string;
	    start_line: number;
	    end_line?: number;
	    start_column?: number;
	    end_column?: number;
	
	    static createFrom(source: any = {}) {
	        return new Location(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.start_line = source["start_line"];
	        this.end_line = source["end_line"];
	        this.start_column = source["start_column"];
	        this.end_column = source["end_column"];
	    }
	}
	export class Finding {
	    schema_version: string;
	    rule_id: string;
	    rule_version: string;
	    severity: string;
	    epistemic_status: string;
	    language?: string;
	    message: string;
	    location: Location;
	    revision?: Revision;
	    evidence?: Evidence[];
	    support?: Support;
	    recommendation?: string;
	
	    static createFrom(source: any = {}) {
	        return new Finding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema_version = source["schema_version"];
	        this.rule_id = source["rule_id"];
	        this.rule_version = source["rule_version"];
	        this.severity = source["severity"];
	        this.epistemic_status = source["epistemic_status"];
	        this.language = source["language"];
	        this.message = source["message"];
	        this.location = this.convertValues(source["location"], Location);
	        this.revision = this.convertValues(source["revision"], Revision);
	        this.evidence = this.convertValues(source["evidence"], Evidence);
	        this.support = this.convertValues(source["support"], Support);
	        this.recommendation = source["recommendation"];
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
	export class Revision {
	    base?: string;
	    head?: string;
	
	    static createFrom(source: any = {}) {
	        return new Revision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.base = source["base"];
	        this.head = source["head"];
	    }
	}
	export class AnalysisRun {
	    schema_version: string;
	    tool_version: string;
	    generated_at: string;
	    origin: string;
	    revision: Revision;
	    findings: Finding[];
	    summary: Summary;
	    rules?: Rule[];
	    limitations?: string[];
	
	    static createFrom(source: any = {}) {
	        return new AnalysisRun(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.schema_version = source["schema_version"];
	        this.tool_version = source["tool_version"];
	        this.generated_at = source["generated_at"];
	        this.origin = source["origin"];
	        this.revision = this.convertValues(source["revision"], Revision);
	        this.findings = this.convertValues(source["findings"], Finding);
	        this.summary = this.convertValues(source["summary"], Summary);
	        this.rules = this.convertValues(source["rules"], Rule);
	        this.limitations = source["limitations"];
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
	export class Diagnostic {
	    path: string;
	    message: string;
	    severity: string;
	    line?: number;
	
	    static createFrom(source: any = {}) {
	        return new Diagnostic(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.message = source["message"];
	        this.severity = source["severity"];
	        this.line = source["line"];
	    }
	}
	export class DiffLine {
	    kind: string;
	    old_line?: number;
	    new_line?: number;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new DiffLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.old_line = source["old_line"];
	        this.new_line = source["new_line"];
	        this.content = source["content"];
	    }
	}
	export class Hunk {
	    index: number;
	    old_start: number;
	    old_lines: number;
	    new_start: number;
	    new_lines: number;
	    header?: string;
	    lines: DiffLine[];
	
	    static createFrom(source: any = {}) {
	        return new Hunk(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.old_start = source["old_start"];
	        this.old_lines = source["old_lines"];
	        this.new_start = source["new_start"];
	        this.new_lines = source["new_lines"];
	        this.header = source["header"];
	        this.lines = this.convertValues(source["lines"], DiffLine);
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
	export class FileChange {
	    path: string;
	    old_path?: string;
	    status: string;
	    language?: string;
	    is_binary?: boolean;
	    hunks?: Hunk[];
	
	    static createFrom(source: any = {}) {
	        return new FileChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.old_path = source["old_path"];
	        this.status = source["status"];
	        this.language = source["language"];
	        this.is_binary = source["is_binary"];
	        this.hunks = this.convertValues(source["hunks"], Hunk);
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
	export class ChangeSet {
	    root?: string;
	    origin: string;
	    base?: string;
	    head?: string;
	    files: FileChange[];
	    diagnostics?: Diagnostic[];
	
	    static createFrom(source: any = {}) {
	        return new ChangeSet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.origin = source["origin"];
	        this.base = source["base"];
	        this.head = source["head"];
	        this.files = this.convertValues(source["files"], FileChange);
	        this.diagnostics = this.convertValues(source["diagnostics"], Diagnostic);
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
	export class Claim {
	    id: string;
	    type: string;
	    paths?: string[];
	    symbols?: string[];
	    report?: string;
	    report_revision?: string;
	    dependencies?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Claim(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.paths = source["paths"];
	        this.symbols = source["symbols"];
	        this.report = source["report"];
	        this.report_revision = source["report_revision"];
	        this.dependencies = source["dependencies"];
	    }
	}
	export class ClaimResult {
	    claim: Claim;
	    status: string;
	    message: string;
	    evidence?: Evidence[];
	    limitations?: string[];
	
	    static createFrom(source: any = {}) {
	        return new ClaimResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.claim = this.convertValues(source["claim"], Claim);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.evidence = this.convertValues(source["evidence"], Evidence);
	        this.limitations = source["limitations"];
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

export namespace ui {
	
	export class AuditView {
	    run_id: string;
	    schema_version: string;
	    tool_version: string;
	    root: string;
	    origin: string;
	    base?: string;
	    changed_files: number;
	    fail_on: string;
	    failed: boolean;
	    failing_findings: number;
	    known_findings: number;
	    suppressed_findings: number;
	    excluded_files?: string[];
	    document?: domain.AnalysisRun;
	    changes?: domain.ChangeSet;
	
	    static createFrom(source: any = {}) {
	        return new AuditView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.run_id = source["run_id"];
	        this.schema_version = source["schema_version"];
	        this.tool_version = source["tool_version"];
	        this.root = source["root"];
	        this.origin = source["origin"];
	        this.base = source["base"];
	        this.changed_files = source["changed_files"];
	        this.fail_on = source["fail_on"];
	        this.failed = source["failed"];
	        this.failing_findings = source["failing_findings"];
	        this.known_findings = source["known_findings"];
	        this.suppressed_findings = source["suppressed_findings"];
	        this.excluded_files = source["excluded_files"];
	        this.document = this.convertValues(source["document"], domain.AnalysisRun);
	        this.changes = this.convertValues(source["changes"], domain.ChangeSet);
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
	export class AuditProgress {
	    run_id: string;
	    phase: string;
	    changed_files?: number;
	    findings?: number;
	
	    static createFrom(source: any = {}) {
	        return new AuditProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.run_id = source["run_id"];
	        this.phase = source["phase"];
	        this.changed_files = source["changed_files"];
	        this.findings = source["findings"];
	    }
	}
	export class AuditEvent {
	    run_id: string;
	    kind: string;
	    progress?: AuditProgress;
	    view?: AuditView;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.run_id = source["run_id"];
	        this.kind = source["kind"];
	        this.progress = this.convertValues(source["progress"], AuditProgress);
	        this.view = this.convertValues(source["view"], AuditView);
	        this.error = source["error"];
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
	
	export class AuditRequest {
	    root: string;
	    origin?: string;
	    worktree?: boolean;
	    staged?: boolean;
	    base?: string;
	    run_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.origin = source["origin"];
	        this.worktree = source["worktree"];
	        this.staged = source["staged"];
	        this.base = source["base"];
	        this.run_id = source["run_id"];
	    }
	}
	
	export class ExportRequest {
	    run_id: string;
	    format: string;
	    suggested_name?: string;
	    title?: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.run_id = source["run_id"];
	        this.format = source["format"];
	        this.suggested_name = source["suggested_name"];
	        this.title = source["title"];
	    }
	}
	export class VerifyView {
	    run_id: string;
	    root: string;
	    base: string;
	    failed: boolean;
	    contradicted: number;
	    unresolved: number;
	    document?: application.VerificationDocument;
	
	    static createFrom(source: any = {}) {
	        return new VerifyView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.run_id = source["run_id"];
	        this.root = source["root"];
	        this.base = source["base"];
	        this.failed = source["failed"];
	        this.contradicted = source["contradicted"];
	        this.unresolved = source["unresolved"];
	        this.document = this.convertValues(source["document"], application.VerificationDocument);
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
	export class VerifyProgress {
	    run_id: string;
	    phase: string;
	    total?: number;
	    contradicted?: number;
	    unresolved?: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new VerifyProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.run_id = source["run_id"];
	        this.phase = source["phase"];
	        this.total = source["total"];
	        this.contradicted = source["contradicted"];
	        this.unresolved = source["unresolved"];
	        this.error = source["error"];
	    }
	}
	export class VerifyEvent {
	    run_id: string;
	    kind: string;
	    progress?: VerifyProgress;
	    view?: VerifyView;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new VerifyEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.run_id = source["run_id"];
	        this.kind = source["kind"];
	        this.progress = this.convertValues(source["progress"], VerifyProgress);
	        this.view = this.convertValues(source["view"], VerifyView);
	        this.error = source["error"];
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
	
	export class VerifyRequest {
	    root: string;
	    base: string;
	    claims_path: string;
	    run_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new VerifyRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.root = source["root"];
	        this.base = source["base"];
	        this.claims_path = source["claims_path"];
	        this.run_id = source["run_id"];
	    }
	}

}

export namespace visualization {
	
	export class DependencyInfo {
	    path: string;
	    file: string;
	    language: string;
	
	    static createFrom(source: any = {}) {
	        return new DependencyInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.file = source["file"];
	        this.language = source["language"];
	    }
	}
	export class SymbolInfo {
	    name: string;
	    kind: string;
	    file: string;
	    language: string;
	
	    static createFrom(source: any = {}) {
	        return new SymbolInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.file = source["file"];
	        this.language = source["language"];
	    }
	}
	export class VisualizationResult {
	    symbols: SymbolInfo[];
	    dependencies: DependencyInfo[];
	    summary: Record<string, number>;
	
	    static createFrom(source: any = {}) {
	        return new VisualizationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.symbols = this.convertValues(source["symbols"], SymbolInfo);
	        this.dependencies = this.convertValues(source["dependencies"], DependencyInfo);
	        this.summary = source["summary"];
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

