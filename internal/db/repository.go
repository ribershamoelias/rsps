package db

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"rsps/internal/ast"
	"rsps/internal/runtime"
)

type Repository struct {
	db       *sql.DB
	registry *runtime.Registry
}

func NewRepository(db *sql.DB, registry *runtime.Registry) *Repository {
	return &Repository{db: db, registry: registry}
}

func (r *Repository) Create(entityName string, data map[string]any) (map[string]any, error) {
	entity, err := r.requireEntity(entityName)
	if err != nil {
		return nil, err
	}

	columns := make([]string, 0)
	placeholders := make([]string, 0)
	args := make([]any, 0)

	for _, field := range entity.Fields {
		value, ok := data[field.Name]
		if !ok {
			continue
		}
		normalized, err := normalizeValue(field, value)
		if err != nil {
			return nil, err
		}
		columns = append(columns, quoteIdent(field.Column))
		placeholders = append(placeholders, "?")
		args = append(args, normalized)
	}

	var query string
	if len(columns) == 0 {
		query = fmt.Sprintf("INSERT INTO %s DEFAULT VALUES", quoteIdent(entity.Table))
	} else {
		query = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quoteIdent(entity.Table), strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	}

	statement, err := r.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("prepare insert for '%s': %w", entityName, err)
	}
	defer statement.Close()

	result, err := statement.Exec(args...)
	if err != nil {
		return nil, fmt.Errorf("insert '%s': %w", entityName, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("fetch last insert id for '%s': %w", entityName, err)
	}

	return r.Get(entityName, id)
}

func (r *Repository) Get(entityName string, id int64) (map[string]any, error) {
	entity, err := r.requireEntity(entityName)
	if err != nil {
		return nil, err
	}

	selectColumns := r.selectColumns(entity)
	query := fmt.Sprintf("SELECT %s FROM %s WHERE id = ?", strings.Join(selectColumns, ", "), quoteIdent(entity.Table))

	statement, err := r.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("prepare get for '%s': %w", entityName, err)
	}
	defer statement.Close()

	row := statement.QueryRow(id)
	record, err := scanSingleRow(entity, row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("entity '%s' with id %d not found", entityName, id)
		}
		return nil, fmt.Errorf("query '%s' by id: %w", entityName, err)
	}

	return record, nil
}

func (r *Repository) List(entityName string) ([]map[string]any, error) {
	entity, err := r.requireEntity(entityName)
	if err != nil {
		return nil, err
	}

	selectColumns := r.selectColumns(entity)
	query := fmt.Sprintf("SELECT %s FROM %s ORDER BY id DESC", strings.Join(selectColumns, ", "), quoteIdent(entity.Table))

	statement, err := r.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("prepare list for '%s': %w", entityName, err)
	}
	defer statement.Close()

	rows, err := statement.Query()
	if err != nil {
		return nil, fmt.Errorf("query list for '%s': %w", entityName, err)
	}
	defer rows.Close()

	records := make([]map[string]any, 0)
	for rows.Next() {
		record, err := scanRows(entity, rows)
		if err != nil {
			return nil, fmt.Errorf("scan list row for '%s': %w", entityName, err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate list rows for '%s': %w", entityName, err)
	}

	return records, nil
}

func (r *Repository) Update(entityName string, id int64, data map[string]any) (map[string]any, error) {
	entity, err := r.requireEntity(entityName)
	if err != nil {
		return nil, err
	}

	setClauses := make([]string, 0)
	args := make([]any, 0)

	for _, field := range entity.Fields {
		value, ok := data[field.Name]
		if !ok {
			continue
		}
		normalized, err := normalizeValue(field, value)
		if err != nil {
			return nil, err
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", quoteIdent(field.Column)))
		args = append(args, normalized)
	}

	if len(setClauses) == 0 {
		return r.Get(entityName, id)
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?", quoteIdent(entity.Table), strings.Join(setClauses, ", "))

	statement, err := r.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("prepare update for '%s': %w", entityName, err)
	}
	defer statement.Close()

	result, err := statement.Exec(args...)
	if err != nil {
		return nil, fmt.Errorf("update '%s' id %d: %w", entityName, id, err)
	}

	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return nil, fmt.Errorf("entity '%s' with id %d not found", entityName, id)
	}

	return r.Get(entityName, id)
}

func (r *Repository) Delete(entityName string, id int64) error {
	entity, err := r.requireEntity(entityName)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", quoteIdent(entity.Table))
	statement, err := r.db.Prepare(query)
	if err != nil {
		return fmt.Errorf("prepare delete for '%s': %w", entityName, err)
	}
	defer statement.Close()

	result, err := statement.Exec(id)
	if err != nil {
		return fmt.Errorf("delete '%s' id %d: %w", entityName, id, err)
	}

	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return fmt.Errorf("entity '%s' with id %d not found", entityName, id)
	}

	return nil
}

func (r *Repository) requireEntity(entityName string) (*runtime.EntityMeta, error) {
	entity, ok := r.registry.Entity(entityName)
	if !ok {
		return nil, fmt.Errorf("unknown entity '%s'", entityName)
	}
	return entity, nil
}

func (r *Repository) selectColumns(entity *runtime.EntityMeta) []string {
	columns := []string{"id"}
	for _, field := range entity.Fields {
		columns = append(columns, quoteIdent(field.Column))
	}
	columns[0] = quoteIdent(columns[0])
	return columns
}

func scanSingleRow(entity *runtime.EntityMeta, row *sql.Row) (map[string]any, error) {
	values := make([]any, len(entity.Fields)+1)
	pointers := make([]any, len(values))
	for index := range values {
		pointers[index] = &values[index]
	}
	if err := row.Scan(pointers...); err != nil {
		return nil, err
	}
	return rowToMap(entity, values), nil
}

func scanRows(entity *runtime.EntityMeta, rows *sql.Rows) (map[string]any, error) {
	values := make([]any, len(entity.Fields)+1)
	pointers := make([]any, len(values))
	for index := range values {
		pointers[index] = &values[index]
	}
	if err := rows.Scan(pointers...); err != nil {
		return nil, err
	}
	return rowToMap(entity, values), nil
}

func rowToMap(entity *runtime.EntityMeta, values []any) map[string]any {
	record := make(map[string]any, len(values))
	record["id"] = toInt64(values[0])

	for index, field := range entity.Fields {
		record[field.Name] = decodeValue(field, values[index+1])
	}

	return record
}

func decodeValue(field *runtime.FieldMeta, value any) any {
	if value == nil {
		return nil
	}

	switch typed := value.(type) {
	case int64:
		if field.Type == ast.TypeBool {
			return typed != 0
		}
		return typed
	case float64:
		if field.Type == ast.TypeBool {
			return typed != 0
		}
		if field.Type == ast.TypeInt || field.IsReference {
			return int64(typed)
		}
		return typed
	case []byte:
		text := string(typed)
		if field.Type == ast.TypeBool {
			return text == "1" || strings.EqualFold(text, "true")
		}
		if field.Type == ast.TypeInt || field.IsReference {
			parsed, err := strconv.ParseInt(text, 10, 64)
			if err == nil {
				return parsed
			}
		}
		return text
	default:
		return typed
	}
}

func normalizeValue(field *runtime.FieldMeta, value any) (any, error) {
	if value == nil {
		return nil, nil
	}

	switch field.Type {
	case ast.TypeString, ast.TypeText, ast.TypeDate, ast.TypeDatetime, ast.TypeJSON:
		switch typed := value.(type) {
		case string:
			return typed, nil
		default:
			return fmt.Sprintf("%v", typed), nil
		}
	case ast.TypeInt, ast.TypeRef:
		return convertInt(value, field.Name)
	case ast.TypeFloat:
		return convertFloat(value, field.Name)
	case ast.TypeBool:
		return convertBool(value, field.Name)
	default:
		return value, nil
	}
}

func convertInt(value any, fieldName string) (int64, error) {
	switch typed := value.(type) {
	case int:
		return int64(typed), nil
	case int8:
		return int64(typed), nil
	case int16:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case int64:
		return typed, nil
	case uint:
		return int64(typed), nil
	case uint8:
		return int64(typed), nil
	case uint16:
		return int64(typed), nil
	case uint32:
		return int64(typed), nil
	case uint64:
		return int64(typed), nil
	case float64:
		return int64(typed), nil
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("field '%s' expects integer", fieldName)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("field '%s' expects integer", fieldName)
	}
}

func convertFloat(value any, fieldName string) (float64, error) {
	switch typed := value.(type) {
	case float32:
		return float64(typed), nil
	case float64:
		return typed, nil
	case int:
		return float64(typed), nil
	case int8:
		return float64(typed), nil
	case int16:
		return float64(typed), nil
	case int32:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case uint:
		return float64(typed), nil
	case uint8:
		return float64(typed), nil
	case uint16:
		return float64(typed), nil
	case uint32:
		return float64(typed), nil
	case uint64:
		return float64(typed), nil
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		if err != nil {
			return 0, fmt.Errorf("field '%s' expects float", fieldName)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("field '%s' expects float", fieldName)
	}
}

func convertBool(value any, fieldName string) (int64, error) {
	switch typed := value.(type) {
	case bool:
		if typed {
			return 1, nil
		}
		return 0, nil
	case int:
		if typed != 0 {
			return 1, nil
		}
		return 0, nil
	case int64:
		if typed != 0 {
			return 1, nil
		}
		return 0, nil
	case float64:
		if typed != 0 {
			return 1, nil
		}
		return 0, nil
	case string:
		lower := strings.ToLower(strings.TrimSpace(typed))
		switch lower {
		case "1", "true", "yes", "on":
			return 1, nil
		case "0", "false", "no", "off":
			return 0, nil
		default:
			return 0, fmt.Errorf("field '%s' expects bool", fieldName)
		}
	default:
		return 0, fmt.Errorf("field '%s' expects bool", fieldName)
	}
}

func toInt64(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case float64:
		return int64(typed)
	case []byte:
		parsed, err := strconv.ParseInt(string(typed), 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func quoteIdent(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}
