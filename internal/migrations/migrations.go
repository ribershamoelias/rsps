package migrations

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"rsps/internal/schema"
)

type Migrator struct {
	db *sql.DB
}

func New(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

func (m *Migrator) Apply(sch *schema.Schema) error {
	if err := m.ensureMigrationsTable(); err != nil {
		return err
	}

	newHash, err := sch.Hash()
	if err != nil {
		return fmt.Errorf("hash schema: %w", err)
	}

	oldHash, hasHash, err := m.currentHash(sch.AppName)
	if err != nil {
		return err
	}
	if hasHash && oldHash == newHash {
		return nil
	}

	existingTables, err := m.listTables()
	if err != nil {
		return err
	}
	desiredTables := make(map[string]schema.Table, len(sch.Tables))
	for _, table := range sch.Tables {
		desiredTables[table.Name] = table
	}

	for _, tableName := range existingTables {
		if tableName == "rsps_migrations" {
			continue
		}
		// Any table disappearance is treated as destructive and rejected in V1.
		if _, ok := desiredTables[tableName]; !ok {
			return fmt.Errorf("unsafe migration: table '%s' exists but is not present in schema", tableName)
		}
	}

	for _, table := range sch.Tables {
		exists, err := m.tableExists(table.Name)
		if err != nil {
			return err
		}
		if !exists {
			// New table creation is always safe.
			if err := m.createTable(table); err != nil {
				return err
			}
			if err := m.ensureIndexes(table); err != nil {
				return err
			}
			continue
		}

		if err := m.reconcileTable(table); err != nil {
			return err
		}
		if err := m.ensureIndexes(table); err != nil {
			return err
		}
	}

	if err := m.storeHash(sch.AppName, newHash); err != nil {
		return err
	}
	return nil
}

func (m *Migrator) ensureMigrationsTable() error {
	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS rsps_migrations (
			app_name TEXT PRIMARY KEY,
			schema_hash TEXT NOT NULL,
			applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	return nil
}

func (m *Migrator) currentHash(appName string) (string, bool, error) {
	row := m.db.QueryRow(`SELECT schema_hash FROM rsps_migrations WHERE app_name = ?`, appName)
	var hash string
	if err := row.Scan(&hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read current migration hash: %w", err)
	}
	return hash, true, nil
}

func (m *Migrator) storeHash(appName, hash string) error {
	_, err := m.db.Exec(`
		INSERT INTO rsps_migrations(app_name, schema_hash, applied_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(app_name) DO UPDATE SET schema_hash = excluded.schema_hash, applied_at = CURRENT_TIMESTAMP
	`, appName, hash)
	if err != nil {
		return fmt.Errorf("store migration hash: %w", err)
	}
	return nil
}

func (m *Migrator) listTables() ([]string, error) {
	rows, err := m.db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table: %w", err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tables: %w", err)
	}
	sort.Strings(tables)
	return tables, nil
}

func (m *Migrator) tableExists(name string) (bool, error) {
	row := m.db.QueryRow(`SELECT 1 FROM sqlite_master WHERE type='table' AND name = ? LIMIT 1`, name)
	var value int
	err := row.Scan(&value)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, fmt.Errorf("check table existence '%s': %w", name, err)
}

func (m *Migrator) createTable(table schema.Table) error {
	definitions := make([]string, 0, len(table.Columns)+len(table.ForeignKeys))
	for _, column := range table.Columns {
		definitions = append(definitions, "  "+columnSQL(column))
	}
	for _, foreignKey := range table.ForeignKeys {
		definitions = append(definitions, fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s(%s)", quoteIdent(foreignKey.Column), quoteIdent(foreignKey.RefTable), quoteIdent(foreignKey.RefColumn)))
	}
	statement := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);", quoteIdent(table.Name), strings.Join(definitions, ",\n"))
	if _, err := m.db.Exec(statement); err != nil {
		return fmt.Errorf("create table '%s': %w", table.Name, err)
	}
	return nil
}

func (m *Migrator) reconcileTable(table schema.Table) error {
	existingColumns, err := m.tableColumns(table.Name)
	if err != nil {
		return err
	}
	desiredColumns := make(map[string]schema.Column, len(table.Columns))
	for _, column := range table.Columns {
		desiredColumns[column.Name] = column
	}

	for name, existingColumn := range existingColumns {
		desiredColumn, ok := desiredColumns[name]
		if !ok {
			return fmt.Errorf("unsafe migration: column '%s.%s' exists but is not present in schema", table.Name, name)
		}
		if !typesCompatible(existingColumn.SQLType, desiredColumn.SQLType) {
			return fmt.Errorf("unsafe migration: column '%s.%s' type change '%s' -> '%s'", table.Name, name, existingColumn.SQLType, desiredColumn.SQLType)
		}
	}

	for _, desiredColumn := range table.Columns {
		if _, ok := existingColumns[desiredColumn.Name]; ok {
			continue
		}
		if desiredColumn.PrimaryKey {
			return fmt.Errorf("unsafe migration: cannot add primary key column '%s.%s' to existing table", table.Name, desiredColumn.Name)
		}
		if !desiredColumn.Nullable && desiredColumn.Default == nil {
			return fmt.Errorf("unsafe migration: cannot add required column '%s.%s' without default", table.Name, desiredColumn.Name)
		}
		statement := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s;", quoteIdent(table.Name), columnSQL(desiredColumn))
		if _, err := m.db.Exec(statement); err != nil {
			return fmt.Errorf("add column '%s.%s': %w", table.Name, desiredColumn.Name, err)
		}
	}

	return nil
}

func (m *Migrator) ensureIndexes(table schema.Table) error {
	existingIndexes, err := m.listIndexes(table.Name)
	if err != nil {
		return err
	}

	for _, index := range table.Indexes {
		if _, ok := existingIndexes[index.Name]; ok {
			continue
		}
		keyword := "INDEX"
		if index.Unique {
			keyword = "UNIQUE INDEX"
		}
		columnNames := make([]string, 0, len(index.Columns))
		for _, column := range index.Columns {
			columnNames = append(columnNames, quoteIdent(column))
		}
		statement := fmt.Sprintf("CREATE %s IF NOT EXISTS %s ON %s (%s);", keyword, quoteIdent(index.Name), quoteIdent(index.Table), strings.Join(columnNames, ", "))
		if _, err := m.db.Exec(statement); err != nil {
			return fmt.Errorf("create index '%s': %w", index.Name, err)
		}
	}
	return nil
}

func (m *Migrator) tableColumns(tableName string) (map[string]schema.Column, error) {
	rows, err := m.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", quoteIdent(tableName)))
	if err != nil {
		return nil, fmt.Errorf("read table_info for '%s': %w", tableName, err)
	}
	defer rows.Close()

	columns := make(map[string]schema.Column)
	for rows.Next() {
		var cid int
		var name string
		var sqlType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &sqlType, &notNull, &defaultValue, &pk); err != nil {
			return nil, fmt.Errorf("scan pragma column for '%s': %w", tableName, err)
		}
		column := schema.Column{
			Name:       name,
			SQLType:    strings.ToUpper(sqlType),
			Nullable:   notNull == 0,
			PrimaryKey: pk == 1,
		}
		if defaultValue.Valid {
			value := defaultValue.String
			column.Default = &value
		}
		columns[name] = column
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pragma columns for '%s': %w", tableName, err)
	}

	return columns, nil
}

func (m *Migrator) listIndexes(tableName string) (map[string]struct{}, error) {
	rows, err := m.db.Query(fmt.Sprintf("PRAGMA index_list(%s)", quoteIdent(tableName)))
	if err != nil {
		return nil, fmt.Errorf("list indexes for '%s': %w", tableName, err)
	}
	defer rows.Close()

	indexes := make(map[string]struct{})
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, fmt.Errorf("scan index for '%s': %w", tableName, err)
		}
		indexes[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate indexes for '%s': %w", tableName, err)
	}
	return indexes, nil
}

func typesCompatible(existing, desired string) bool {
	normalizedExisting := strings.ToUpper(strings.TrimSpace(existing))
	normalizedDesired := strings.ToUpper(strings.TrimSpace(desired))
	return normalizedExisting == normalizedDesired
}

func columnSQL(column schema.Column) string {
	if column.PrimaryKey {
		if column.AutoIncrement {
			return fmt.Sprintf("%s INTEGER PRIMARY KEY AUTOINCREMENT", quoteIdent(column.Name))
		}
		return fmt.Sprintf("%s INTEGER PRIMARY KEY", quoteIdent(column.Name))
	}

	parts := []string{quoteIdent(column.Name), column.SQLType}
	if !column.Nullable {
		parts = append(parts, "NOT NULL")
	}
	if column.Default != nil {
		parts = append(parts, "DEFAULT "+*column.Default)
	}
	return strings.Join(parts, " ")
}

func quoteIdent(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}
