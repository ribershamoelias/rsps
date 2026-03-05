package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"rsps/internal/ast"
)

type Schema struct {
	AppName string  `json:"app_name"`
	Tables  []Table `json:"tables"`
}

type Table struct {
	Name        string       `json:"name"`
	Entity      string       `json:"entity"`
	Columns     []Column     `json:"columns"`
	ForeignKeys []ForeignKey `json:"foreign_keys"`
	Indexes     []Index      `json:"indexes"`
}

type Column struct {
	Name          string  `json:"name"`
	SQLType       string  `json:"sql_type"`
	Nullable      bool    `json:"nullable"`
	Default       *string `json:"default,omitempty"`
	PrimaryKey    bool    `json:"primary_key,omitempty"`
	AutoIncrement bool    `json:"auto_increment,omitempty"`
}

type ForeignKey struct {
	Column    string `json:"column"`
	RefTable  string `json:"ref_table"`
	RefColumn string `json:"ref_column"`
}

type Index struct {
	Name    string   `json:"name"`
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

func Generate(app *ast.Application) (*Schema, error) {
	schema := &Schema{AppName: app.Name}

	for _, entity := range app.Entities {
		table := Table{
			Name:   entity.Name,
			Entity: entity.Name,
			Columns: []Column{
				{
					Name:          "id",
					SQLType:       "INTEGER",
					PrimaryKey:    true,
					AutoIncrement: true,
				},
			},
		}

		for _, field := range entity.Fields {
			columnName := field.Name
			if field.Type == ast.TypeRef {
				columnName = field.Name + "_id"
			}

			column := Column{
				Name:     columnName,
				SQLType:  sqlTypeForField(field.Type),
				Nullable: field.Nullable,
				Default:  sqlDefaultForField(field),
			}
			table.Columns = append(table.Columns, column)

			if field.Type == ast.TypeRef {
				table.ForeignKeys = append(table.ForeignKeys, ForeignKey{
					Column:    columnName,
					RefTable:  field.Reference.Entity,
					RefColumn: "id",
				})
			}

			if field.HasAttribute(ast.AttrUnique) {
				table.Indexes = append(table.Indexes, Index{
					Name:    fmt.Sprintf("udx_%s_%s", table.Name, columnName),
					Table:   table.Name,
					Columns: []string{columnName},
					Unique:  true,
				})
			}
			if field.HasAttribute(ast.AttrIndex) {
				table.Indexes = append(table.Indexes, Index{
					Name:    fmt.Sprintf("idx_%s_%s", table.Name, columnName),
					Table:   table.Name,
					Columns: []string{columnName},
					Unique:  false,
				})
			}
		}

		schema.Tables = append(schema.Tables, table)
	}

	return schema, nil
}

func (s *Schema) SQL() string {
	var statements []string
	for _, table := range s.Tables {
		columnDefs := make([]string, 0, len(table.Columns)+len(table.ForeignKeys))
		for _, column := range table.Columns {
			columnDefs = append(columnDefs, "  "+columnDefinition(column))
		}
		for _, foreignKey := range table.ForeignKeys {
			columnDefs = append(columnDefs, fmt.Sprintf("  FOREIGN KEY (%s) REFERENCES %s(%s)", quoteIdent(foreignKey.Column), quoteIdent(foreignKey.RefTable), quoteIdent(foreignKey.RefColumn)))
		}

		statements = append(statements, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);", quoteIdent(table.Name), strings.Join(columnDefs, ",\n")))
		for _, index := range table.Indexes {
			indexKeyword := "INDEX"
			if index.Unique {
				indexKeyword = "UNIQUE INDEX"
			}
			columnList := make([]string, 0, len(index.Columns))
			for _, column := range index.Columns {
				columnList = append(columnList, quoteIdent(column))
			}
			statements = append(statements, fmt.Sprintf("CREATE %s IF NOT EXISTS %s ON %s (%s);", indexKeyword, quoteIdent(index.Name), quoteIdent(index.Table), strings.Join(columnList, ", ")))
		}
	}

	return strings.Join(statements, "\n\n") + "\n"
}

func (s *Schema) Hash() (string, error) {
	payload, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func sqlTypeForField(fieldType ast.FieldType) string {
	switch fieldType {
	case ast.TypeString, ast.TypeText:
		return "TEXT"
	case ast.TypeInt, ast.TypeBool, ast.TypeRef:
		return "INTEGER"
	case ast.TypeFloat:
		return "REAL"
	case ast.TypeDate, ast.TypeDatetime, ast.TypeJSON:
		return "TEXT"
	default:
		return "TEXT"
	}
}

func sqlDefaultForField(field *ast.Field) *string {
	if field.Default == nil {
		return nil
	}

	value := field.Default.Value
	switch field.Default.Kind {
	case "now":
		defaultValue := "CURRENT_TIMESTAMP"
		return &defaultValue
	case "bool":
		if value == "true" {
			defaultValue := "1"
			return &defaultValue
		}
		defaultValue := "0"
		return &defaultValue
	case "number":
		return &value
	case "string", "identifier":
		escaped := strings.ReplaceAll(value, "'", "''")
		defaultValue := fmt.Sprintf("'%s'", escaped)
		return &defaultValue
	default:
		escaped := strings.ReplaceAll(value, "'", "''")
		defaultValue := fmt.Sprintf("'%s'", escaped)
		return &defaultValue
	}
}

func columnDefinition(column Column) string {
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
