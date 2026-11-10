// Package storage provides persistence for the structural index.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/barceloch/codixai/internal/domain"
	"github.com/barceloch/codixai/internal/index"
	_ "modernc.org/sqlite"
)

// SchemaVersion is the version of the storage schema.
const SchemaVersion = 2

// IndexStore provides persistence for the structural index.
type IndexStore struct {
	db *sql.DB
}

// NewIndexStore creates a new IndexStore connected to the given SQLite database.
func NewIndexStore(dbPath string) (*IndexStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &IndexStore{db: db}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	if err := store.migrateIfNeeded(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	return store, nil
}

// initSchema creates the database tables if they don't exist.
func (s *IndexStore) initSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS index_metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS index_files (id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL UNIQUE, language TEXT NOT NULL, hash TEXT NOT NULL, size INTEGER NOT NULL, mod_time INTEGER NOT NULL, indexed_at INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS index_symbols (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER NOT NULL REFERENCES index_files(id) ON DELETE CASCADE, name TEXT NOT NULL, kind TEXT NOT NULL, line INTEGER NOT NULL, column INTEGER, language TEXT NOT NULL, container TEXT, signature TEXT, exported INTEGER, UNIQUE(file_id, name, kind, line))`,
		`CREATE TABLE IF NOT EXISTS index_dependencies (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER NOT NULL REFERENCES index_files(id) ON DELETE CASCADE, path TEXT NOT NULL, line INTEGER NOT NULL, language TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS index_references (id INTEGER PRIMARY KEY AUTOINCREMENT, source_file_id INTEGER NOT NULL REFERENCES index_files(id) ON DELETE CASCADE, source_symbol_id INTEGER REFERENCES index_symbols(id) ON DELETE SET NULL, target_file_id INTEGER REFERENCES index_files(id) ON DELETE SET NULL, target_symbol_id INTEGER REFERENCES index_symbols(id) ON DELETE SET NULL, referenced_name TEXT NOT NULL, kind TEXT NOT NULL, language TEXT NOT NULL, resolution TEXT NOT NULL, source_line INTEGER NOT NULL, source_column INTEGER, evidence TEXT, candidates TEXT, created_at INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS index_builds (id INTEGER PRIMARY KEY AUTOINCREMENT, root_path TEXT NOT NULL, tool_version TEXT NOT NULL, schema_version INTEGER NOT NULL, analyzer_version TEXT NOT NULL, started_at INTEGER NOT NULL, completed_at INTEGER, files_total INTEGER NOT NULL DEFAULT 0, files_indexed INTEGER NOT NULL DEFAULT 0, symbols_count INTEGER NOT NULL DEFAULT 0, deps_count INTEGER NOT NULL DEFAULT 0, refs_count INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL DEFAULT 'running')`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to create tables: %w", err)
		}
	}

	// If this is a legacy v1 database, add missing columns before creating v2 indexes.
	if s.looksLikeV1() {
		if err := s.ensureV2Columns(); err != nil {
			return err
		}
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_symbols_name ON index_symbols(name)`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_file ON index_symbols(file_id)`,
		`CREATE INDEX IF NOT EXISTS idx_symbols_container ON index_symbols(container)`,
		`CREATE INDEX IF NOT EXISTS idx_deps_path ON index_dependencies(path)`,
		`CREATE INDEX IF NOT EXISTS idx_deps_file ON index_dependencies(file_id)`,
		`CREATE INDEX IF NOT EXISTS idx_files_path ON index_files(path)`,
		`CREATE INDEX IF NOT EXISTS idx_references_source_file ON index_references(source_file_id)`,
		`CREATE INDEX IF NOT EXISTS idx_references_source_symbol ON index_references(source_symbol_id)`,
		`CREATE INDEX IF NOT EXISTS idx_references_target_symbol ON index_references(target_symbol_id)`,
		`CREATE INDEX IF NOT EXISTS idx_references_resolution ON index_references(resolution)`,
		`CREATE INDEX IF NOT EXISTS idx_references_language ON index_references(language)`,
		`CREATE INDEX IF NOT EXISTS idx_references_name ON index_references(referenced_name)`,
	}
	for _, stmt := range indexes {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	var existing string
	err := s.db.QueryRow(`SELECT value FROM index_metadata WHERE key = 'schema_version'`).Scan(&existing)
	if err == sql.ErrNoRows {
		version := SchemaVersion
		if s.looksLikeV1() {
			version = 1
		}
		if _, err := s.db.Exec(`INSERT INTO index_metadata (key, value) VALUES ('schema_version', ?)`, version); err != nil {
			return fmt.Errorf("failed to set schema version: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to read schema version: %w", err)
	}

	return nil
}

func (s *IndexStore) ensureV2Columns() error {
	columns := []struct{ name, def string }{
		{"column", "INTEGER"},
		{"container", "TEXT"},
		{"signature", "TEXT"},
		{"exported", "INTEGER"},
	}
	for _, col := range columns {
		_, err := s.db.Exec(fmt.Sprintf("ALTER TABLE index_symbols ADD COLUMN %s %s", col.name, col.def))
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return fmt.Errorf("failed to add column %s: %w", col.name, err)
		}
	}
	_, err := s.db.Exec(`ALTER TABLE index_builds ADD COLUMN refs_count INTEGER NOT NULL DEFAULT 0`)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return fmt.Errorf("failed to add refs_count column: %w", err)
	}
	return nil
}

// migrateIfNeeded checks the current schema version and migrates if needed.
func (s *IndexStore) migrateIfNeeded() error {
	var currentVersion int
	err := s.db.QueryRow(`SELECT value FROM index_metadata WHERE key = 'schema_version'`).Scan(&currentVersion)
	if err == sql.ErrNoRows {
		currentVersion = 0
	} else if err != nil {
		return fmt.Errorf("failed to read schema version: %w", err)
	}

	if currentVersion < SchemaVersion && s.looksLikeV1() {
		if err := s.ensureV2Columns(); err != nil {
			return err
		}
		currentVersion = 1
	}
	if currentVersion == 0 {
		_, _ = s.db.Exec(`INSERT OR REPLACE INTO index_metadata (key, value) VALUES ('schema_version', ?)`, SchemaVersion)
		return nil
	}
	if currentVersion >= SchemaVersion {
		return nil
	}
	if err := s.ensureV2Columns(); err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT OR REPLACE INTO index_metadata (key, value) VALUES ('schema_version', ?)`, SchemaVersion)
	return err
}

func (s *IndexStore) looksLikeV1() bool {
	rows, err := s.db.Query(`PRAGMA table_info(index_symbols)`)
	if err != nil {
		return false
	}
	defer rows.Close()
	hasColumn := false
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			continue
		}
		if name == "column" || name == "container" {
			hasColumn = true
		}
	}
	return !hasColumn
}

// migrateToV2 migrates the database from schema v1 to v2.
func (s *IndexStore) migrateToV2() error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin migration transaction: %w", err)
	}
	defer tx.Rollback()

	// Add new columns to index_symbols if they don't exist
	columns := []struct {
		name string
		def  string
	}{
		{"column", "INTEGER"},
		{"container", "TEXT"},
		{"signature", "TEXT"},
		{"exported", "INTEGER"},
	}

	for _, col := range columns {
		_, err := tx.Exec(fmt.Sprintf("ALTER TABLE index_symbols ADD COLUMN %s %s", col.name, col.def))
		if err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("failed to add column %s: %w", col.name, err)
		}
	}

	// Create index_references table
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS index_references (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_file_id INTEGER NOT NULL REFERENCES index_files(id) ON DELETE CASCADE,
			source_symbol_id INTEGER REFERENCES index_symbols(id) ON DELETE SET NULL,
			target_file_id INTEGER REFERENCES index_files(id) ON DELETE SET NULL,
			target_symbol_id INTEGER REFERENCES index_symbols(id) ON DELETE SET NULL,
			referenced_name TEXT NOT NULL,
			kind TEXT NOT NULL,
			language TEXT NOT NULL,
			resolution TEXT NOT NULL,
			source_line INTEGER NOT NULL,
			source_column INTEGER,
			evidence TEXT,
			candidates TEXT,
			created_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create index_references table: %w", err)
	}

	// Add indexes for references
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_references_source_file ON index_references(source_file_id)`,
		`CREATE INDEX IF NOT EXISTS idx_references_source_symbol ON index_references(source_symbol_id)`,
		`CREATE INDEX IF NOT EXISTS idx_references_target_symbol ON index_references(target_symbol_id)`,
		`CREATE INDEX IF NOT EXISTS idx_references_resolution ON index_references(resolution)`,
		`CREATE INDEX IF NOT EXISTS idx_references_language ON index_references(language)`,
		`CREATE INDEX IF NOT EXISTS idx_references_name ON index_references(referenced_name)`,
	}

	for _, idx := range indexes {
		if _, err := tx.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Add refs_count column to index_builds
	_, err = tx.Exec(`ALTER TABLE index_builds ADD COLUMN refs_count INTEGER NOT NULL DEFAULT 0`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column") {
		return fmt.Errorf("failed to add refs_count column: %w", err)
	}

	// Add indexes for symbols container
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_symbols_container ON index_symbols(container)`)
	if err != nil {
		return fmt.Errorf("failed to create idx_symbols_container: %w", err)
	}

	// Update schema version
	_, err = tx.Exec(`INSERT OR REPLACE INTO index_metadata (key, value) VALUES ('schema_version', ?)`, SchemaVersion)
	if err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	return tx.Commit()
}

// Close closes the database connection.
func (s *IndexStore) Close() error {
	return s.db.Close()
}

// Save persists the index to the database.
func (s *IndexStore) Save(ctx context.Context, idx *index.Index, rootPath, toolVersion, analyzerVersion string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Record build start
	buildStart := time.Now().Unix()
	res, err := tx.ExecContext(ctx, `
		INSERT INTO index_builds (root_path, tool_version, schema_version, analyzer_version, started_at, status)
		VALUES (?, ?, ?, ?, ?, 'running')
	`, rootPath, toolVersion, SchemaVersion, analyzerVersion, buildStart)
	if err != nil {
		return fmt.Errorf("failed to record build start: %w", err)
	}
	buildID, _ := res.LastInsertId()

	// Clear existing data for this root (we rebuild completely for now)
	_, err = tx.ExecContext(ctx, `DELETE FROM index_files WHERE id IN (
		SELECT id FROM index_files WHERE path LIKE ? || '%'
	)`, rootPath)
	if err != nil {
		return fmt.Errorf("failed to clear existing files: %w", err)
	}

	symbols := idx.Symbols()
	dependencies := idx.Dependencies()

	// Group symbols and dependencies by file
	fileSymbols := make(map[string][]index.Symbol)
	fileDeps := make(map[string][]index.Dependency)

	for _, sym := range symbols {
		fileSymbols[sym.File] = append(fileSymbols[sym.File], sym)
	}
	for _, dep := range dependencies {
		fileDeps[dep.File] = append(fileDeps[dep.File], dep)
	}

	// Get all unique file paths
	allFiles := make(map[string]bool)
	for f := range fileSymbols {
		allFiles[f] = true
	}
	for f := range fileDeps {
		allFiles[f] = true
	}

	filesIndexed := 0
	symbolsCount := 0
	depsCount := 0

	for filePath := range allFiles {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read file to get hash, size, mod time
		// For now, we'll use placeholder values since we don't have the source here
		fileHash := ""
		fileSize := int64(0)
		modTime := time.Now().Unix()

		// Insert file
		res, err := tx.ExecContext(ctx, `
			INSERT INTO index_files (path, language, hash, size, mod_time, indexed_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, filePath, "", fileHash, fileSize, modTime, time.Now().Unix())
		if err != nil {
			return fmt.Errorf("failed to insert file %s: %w", filePath, err)
		}
		fileID, _ := res.LastInsertId()

		// Insert symbols for this file
		if syms, ok := fileSymbols[filePath]; ok {
			for _, sym := range syms {
				exportedVal := 0
				if sym.Exported != nil && *sym.Exported {
					exportedVal = 1
				}
				_, err := tx.ExecContext(ctx, `
					INSERT INTO index_symbols (file_id, name, kind, line, column, language, container, signature, exported)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, fileID, sym.Name, sym.Kind, sym.Line, sym.Column, sym.Language, sym.Container, sym.Signature, exportedVal)
				if err != nil {
					return fmt.Errorf("failed to insert symbol %s: %w", sym.Name, err)
				}
				symbolsCount++
			}
		}

		// Insert dependencies for this file
		if deps, ok := fileDeps[filePath]; ok {
			for _, dep := range deps {
				_, err := tx.ExecContext(ctx, `
					INSERT INTO index_dependencies (file_id, path, line, language)
					VALUES (?, ?, ?, ?)
				`, fileID, dep.Path, dep.Line, dep.Language)
				if err != nil {
					return fmt.Errorf("failed to insert dependency %s: %w", dep.Path, err)
				}
				depsCount++
			}
		}

		filesIndexed++
	}

	// Update build record
	buildEnd := time.Now().Unix()
	_, err = tx.ExecContext(ctx, `
		UPDATE index_builds
		SET completed_at = ?, files_total = ?, files_indexed = ?, symbols_count = ?, deps_count = ?, refs_count = ?, status = 'completed'
		WHERE id = ?
	`, buildEnd, len(allFiles), filesIndexed, symbolsCount, depsCount, 0, buildID)
	if err != nil {
		return fmt.Errorf("failed to update build record: %w", err)
	}

	// Update metadata
	_, err = tx.ExecContext(ctx, `
		INSERT OR REPLACE INTO index_metadata (key, value) VALUES ('last_indexed_at', ?)
	`, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	return tx.Commit()
}

// Load loads the index from the database.
func (s *IndexStore) Load(ctx context.Context, rootPath string) (*index.Index, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// We need to rebuild the index from stored data
	// First, get all files for this root
	rows, err := tx.QueryContext(ctx, `
		SELECT id, path, language FROM index_files WHERE path LIKE ? || '%'
	`, rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to query files: %w", err)
	}
	defer rows.Close()

	idx := index.New()

	type fileInfo struct {
		id       int64
		path     string
		language string
	}

	files := []fileInfo{}
	for rows.Next() {
		var f fileInfo
		if err := rows.Scan(&f.id, &f.path, &f.language); err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	// Load symbols
	for _, f := range files {
		symRows, err := tx.QueryContext(ctx, `
			SELECT name, kind, line, column, language, container, signature, exported FROM index_symbols WHERE file_id = ?
		`, f.id)
		if err != nil {
			return nil, fmt.Errorf("failed to query symbols: %w", err)
		}
		defer symRows.Close()

		for symRows.Next() {
			var sym index.Symbol
			var exportedInt int
			sym.File = f.path
			if err := symRows.Scan(&sym.Name, &sym.Kind, &sym.Line, &sym.Column, &sym.Language, &sym.Container, &sym.Signature, &exportedInt); err != nil {
				return nil, fmt.Errorf("failed to scan symbol: %w", err)
			}
			if exportedInt == 1 {
				exported := true
				sym.Exported = &exported
			}
			idx.AddSymbol(sym)
		}
		if err := symRows.Err(); err != nil {
			return nil, fmt.Errorf("symbol rows error: %w", err)
		}
	}

	// Load dependencies
	for _, f := range files {
		depRows, err := tx.QueryContext(ctx, `
			SELECT path, line, language FROM index_dependencies WHERE file_id = ?
		`, f.id)
		if err != nil {
			return nil, fmt.Errorf("failed to query dependencies: %w", err)
		}
		defer depRows.Close()

		for depRows.Next() {
			var dep index.Dependency
			dep.File = f.path
			if err := depRows.Scan(&dep.Path, &dep.Line, &dep.Language); err != nil {
				return nil, fmt.Errorf("failed to scan dependency: %w", err)
			}
			idx.AddDependency(dep)
		}
		if err := depRows.Err(); err != nil {
			return nil, fmt.Errorf("dependency rows error: %w", err)
		}
	}

	return idx, tx.Commit()
}

// Status returns the status of the index for a given root.
func (s *IndexStore) Status(ctx context.Context, rootPath string) (*IndexStatus, error) {
	var status IndexStatus
	var lastIndexed sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as file_count,
			(SELECT COUNT(*) FROM index_symbols WHERE file_id IN (SELECT id FROM index_files WHERE path LIKE ? || '%')) as symbol_count,
			(SELECT COUNT(*) FROM index_dependencies WHERE file_id IN (SELECT id FROM index_files WHERE path LIKE ? || '%')) as dep_count,
			(SELECT COUNT(*) FROM index_references WHERE source_file_id IN (SELECT id FROM index_files WHERE path LIKE ? || '%')) as ref_count,
			MAX(indexed_at) as last_indexed
		FROM index_files WHERE path LIKE ? || '%'
	`, rootPath, rootPath, rootPath, rootPath).Scan(&status.FileCount, &status.SymbolCount, &status.DepCount, &status.RefCount, &lastIndexed)
	if err != nil {
		if err == sql.ErrNoRows {
			return &IndexStatus{FileCount: 0, SymbolCount: 0, DepCount: 0, RefCount: 0}, nil
		}
		return nil, fmt.Errorf("failed to get status: %w", err)
	}
	status.LastIndexed = lastIndexed.Int64
	status.RootPath = rootPath
	return &status, nil
}

// IndexStatus holds status information about an index.
type IndexStatus struct {
	RootPath    string
	FileCount   int
	SymbolCount int
	DepCount    int
	RefCount    int
	LastIndexed int64
}

// ListSymbols returns symbols matching a query.
func (s *IndexStore) ListSymbols(ctx context.Context, rootPath, query string) ([]SymbolResult, error) {
	var args []any
	sqlQuery := `
		SELECT s.name, s.kind, s.line, s.column, s.language, s.container, s.signature, s.exported, f.path
		FROM index_symbols s
		JOIN index_files f ON s.file_id = f.id
		WHERE f.path LIKE ? || '%'
	`
	args = append(args, rootPath)
	if strings.TrimSpace(query) != "" {
		sqlQuery += " AND LOWER(s.name) LIKE LOWER(?)"
		args = append(args, "%"+query+"%")
	}
	sqlQuery += " ORDER BY s.name LIMIT 1000"

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query symbols: %w", err)
	}
	defer rows.Close()

	var results []SymbolResult
	for rows.Next() {
		var r SymbolResult
		var exportedInt int
		if err := rows.Scan(&r.Name, &r.Kind, &r.Line, &r.Column, &r.Language, &r.Container, &r.Signature, &exportedInt, &r.File); err != nil {
			return nil, fmt.Errorf("failed to scan symbol: %w", err)
		}
		if exportedInt == 1 {
			exported := true
			r.Exported = &exported
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// SymbolResult represents a symbol from the stored index.
type SymbolResult struct {
	Name      string
	Kind      string
	Line      int
	Column    int
	Language  string
	Container string
	Signature string
	Exported  *bool
	File      string
}

// ListDependencies returns dependencies from the stored index.
func (s *IndexStore) ListDependencies(ctx context.Context, rootPath string) ([]DependencyResult, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.path, d.line, d.language, f.path
		FROM index_dependencies d
		JOIN index_files f ON d.file_id = f.id
		WHERE f.path LIKE ? || '%'
		ORDER BY d.path
	`, rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to query dependencies: %w", err)
	}
	defer rows.Close()

	var results []DependencyResult
	for rows.Next() {
		var r DependencyResult
		if err := rows.Scan(&r.Path, &r.Line, &r.Language, &r.File); err != nil {
			return nil, fmt.Errorf("failed to scan dependency: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// DependencyResult represents a dependency from the stored index.
type DependencyResult struct {
	Path     string
	Line     int
	Language string
	File     string
}

// SaveReferences saves references to the database.
func (s *IndexStore) SaveReferences(ctx context.Context, refs []domain.Reference, rootPath string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Build a map of symbol name+file to symbol ID for fast lookup
	symbolIDMap := make(map[string]int64)
	rows, err := tx.QueryContext(ctx, `
		SELECT s.id, s.name, f.path FROM index_symbols s
		JOIN index_files f ON s.file_id = f.id
		WHERE f.path LIKE ? || '%'
	`, rootPath)
	if err != nil {
		return fmt.Errorf("failed to query symbols for reference resolution: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var name, path string
		if err := rows.Scan(&id, &name, &path); err != nil {
			return err
		}
		key := path + ":" + name
		symbolIDMap[key] = id
	}
	rows.Close()

	// Delete existing references for this root
	_, err = tx.ExecContext(ctx, `
		DELETE FROM index_references WHERE source_file_id IN (
			SELECT id FROM index_files WHERE path LIKE ? || '%'
		)
	`, rootPath)
	if err != nil {
		return fmt.Errorf("failed to delete existing references: %w", err)
	}

	createdAt := time.Now().Unix()
	// Build file path -> ID map
	fileIDMap := make(map[string]int64)
	fileRows, err := tx.QueryContext(ctx, `
		SELECT id, path FROM index_files WHERE path LIKE ? || '%'
	`, rootPath)
	if err != nil {
		return fmt.Errorf("failed to query files for reference persistence: %w", err)
	}
	for fileRows.Next() {
		var id int64
		var path string
		if err := fileRows.Scan(&id, &path); err != nil {
			fileRows.Close()
			return err
		}
		fileIDMap[path] = id
	}
	fileRows.Close()

	absRoot, _ := filepath.Abs(rootPath)
	for _, ref := range refs {
		candidates := []string{
			ref.Location.Path,
			ref.Source.File,
			filepath.Join(rootPath, ref.Location.Path),
			filepath.Join(rootPath, ref.Source.File),
			filepath.Join(absRoot, ref.Location.Path),
			filepath.Join(absRoot, ref.Source.File),
		}
		var sourceFileID int64
		ok := false
		for _, c := range candidates {
			if c == "" {
				continue
			}
			if id, found := fileIDMap[c]; found {
				sourceFileID = id
				ok = true
				break
			}
			if id, found := fileIDMap[filepath.Clean(c)]; found {
				sourceFileID = id
				ok = true
				break
			}
		}
		if !ok {
			// Skip references whose source file is not in the index.
			continue
		}

		var sourceSymbolID, targetSymbolID, targetFileID sql.NullInt64
		var evidenceJSON, candidatesJSON sql.NullString

		// Look up source symbol ID
		if ref.Source.File != "" && ref.Source.Name != "" {
			key := ref.Source.File + ":" + ref.Source.Name
			if id, found := symbolIDMap[key]; found {
				sourceSymbolID = sql.NullInt64{Int64: id, Valid: true}
			}
		}

		// Look up target symbol ID
		if ref.Target != nil && ref.Target.File != "" && ref.Target.Name != "" {
			key := ref.Target.File + ":" + ref.Target.Name
			if id, found := symbolIDMap[key]; found {
				targetSymbolID = sql.NullInt64{Int64: id, Valid: true}
			}
			if fid, found := fileIDMap[ref.Target.File]; found {
				targetFileID = sql.NullInt64{Int64: fid, Valid: true}
			}
		}

		// Serialize evidence
		if len(ref.Evidence) > 0 {
			evidenceBytes, err := json.Marshal(ref.Evidence)
			if err != nil {
				return fmt.Errorf("failed to marshal evidence: %w", err)
			}
			evidenceJSON = sql.NullString{String: string(evidenceBytes), Valid: true}
		}

		// Serialize candidates
		if len(ref.Candidates) > 0 {
			candidatesBytes, err := json.Marshal(ref.Candidates)
			if err != nil {
				return fmt.Errorf("failed to marshal candidates: %w", err)
			}
			candidatesJSON = sql.NullString{String: string(candidatesBytes), Valid: true}
		}

		referencedName := ref.Source.Name
		if ref.Target != nil && ref.Target.Name != "" {
			referencedName = ref.Target.Name
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO index_references (
				source_file_id, source_symbol_id, target_file_id, target_symbol_id,
				referenced_name, kind, language, resolution,
				source_line, source_column, evidence, candidates, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			sourceFileID,
			sourceSymbolID,
			targetFileID,
			targetSymbolID,
			referencedName,
			string(ref.Kind),
			ref.Language,
			string(ref.Resolution),
			ref.Location.StartLine,
			ref.Location.StartColumn,
			evidenceJSON,
			candidatesJSON,
			createdAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert reference: %w", err)
		}
	}

	return tx.Commit()
}

// LoadReferences loads references from the database.
func (s *IndexStore) LoadReferences(ctx context.Context, rootPath string) ([]domain.Reference, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT 
			r.id, r.referenced_name, r.kind, r.language, r.resolution,
			r.source_line, r.source_column, r.evidence, r.candidates, r.created_at,
			sf.path as source_file, ssf.name as source_symbol,
			tf.path as target_file, tsf.name as target_symbol
		FROM index_references r
		JOIN index_files sf ON r.source_file_id = sf.id
		LEFT JOIN index_symbols ssf ON r.source_symbol_id = ssf.id
		LEFT JOIN index_files tf ON r.target_file_id = tf.id
		LEFT JOIN index_symbols tsf ON r.target_symbol_id = tsf.id
		WHERE sf.path LIKE ? || '%'
		ORDER BY sf.path, r.source_line, COALESCE(r.source_column, 0), COALESCE(ssf.name, r.referenced_name), r.kind, r.language, r.resolution, r.id
	`, rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to query references: %w", err)
	}
	defer rows.Close()

	var refs []domain.Reference
	for rows.Next() {
		var (
			id             int64
			referencedName string
			kind           string
			language       string
			resolution     string
			sourceLine     int
			sourceColumn   sql.NullInt64
			evidenceJSON   sql.NullString
			candidatesJSON sql.NullString
			createdAt      sql.NullInt64
			sourceFile     string
			sourceSymbol   sql.NullString
			targetFile     sql.NullString
			targetSymbol   sql.NullString
		)

		if err := rows.Scan(
			&id, &referencedName, &kind, &language, &resolution,
			&sourceLine, &sourceColumn,
			&evidenceJSON, &candidatesJSON, &createdAt,
			&sourceFile, &sourceSymbol,
			&targetFile, &targetSymbol,
		); err != nil {
			return nil, fmt.Errorf("failed to scan reference: %w", err)
		}

		ref := domain.Reference{
			ID:         fmt.Sprintf("%d", id),
			Kind:       domain.ReferenceKind(kind),
			Language:   language,
			Resolution: domain.ResolutionStatus(resolution),
			Location: domain.Location{
				Path:        sourceFile,
				StartLine:   sourceLine,
				StartColumn: int(sourceColumn.Int64),
			},
			Source: domain.SymbolRef{
				Name:     sourceSymbol.String,
				File:     sourceFile,
				Line:     sourceLine,
				Column:   int(sourceColumn.Int64),
				Language: language,
			},
		}
		if !sourceSymbol.Valid || ref.Source.Name == "" {
			ref.Source.Name = referencedName
		}

		if targetFile.Valid || targetSymbol.Valid {
			ref.Target = &domain.SymbolRef{
				Name:     targetSymbol.String,
				File:     targetFile.String,
				Language: language,
			}
			if ref.Target.Name == "" {
				ref.Target.Name = referencedName
			}
		}

		if evidenceJSON.Valid && evidenceJSON.String != "" {
			if err := json.Unmarshal([]byte(evidenceJSON.String), &ref.Evidence); err != nil {
				return nil, fmt.Errorf("failed to unmarshal evidence: %w", err)
			}
		}
		if candidatesJSON.Valid && candidatesJSON.String != "" {
			if err := json.Unmarshal([]byte(candidatesJSON.String), &ref.Candidates); err != nil {
				return nil, fmt.Errorf("failed to unmarshal candidates: %w", err)
			}
		}

		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return refs, nil
}

// ReferenceFilter filters references.
type ReferenceFilter struct {
	Resolution []domain.ResolutionStatus
	Language   []string
	Kind       []domain.ReferenceKind
	SourceFile string
}

// ListReferences returns references matching a filter.
func (s *IndexStore) ListReferences(ctx context.Context, rootPath string, filter ReferenceFilter) ([]domain.Reference, error) {
	// Simplified implementation - just load all and filter in memory
	refs, err := s.LoadReferences(ctx, rootPath)
	if err != nil {
		return nil, err
	}

	// Apply filters
	filtered := make([]domain.Reference, 0, len(refs))
	for _, ref := range refs {
		if len(filter.Resolution) > 0 {
			match := false
			for _, r := range filter.Resolution {
				if ref.Resolution == r {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if len(filter.Language) > 0 {
			match := false
			for _, l := range filter.Language {
				if ref.Language == l {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if len(filter.Kind) > 0 {
			match := false
			for _, k := range filter.Kind {
				if ref.Kind == k {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if filter.SourceFile != "" && ref.Source.File != filter.SourceFile {
			continue
		}
		filtered = append(filtered, ref)
	}

	return filtered, nil
}
