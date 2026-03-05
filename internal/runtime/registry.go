package runtime

import (
	"encoding/json"
	"fmt"
	"os"

	"rsps/internal/ast"
)

type Registry struct {
	AppName  string        `json:"app_name"`
	Entities []*EntityMeta `json:"entities"`

	entityByName map[string]*EntityMeta
}

type EntityMeta struct {
	Name   string       `json:"name"`
	Table  string       `json:"table"`
	Fields []*FieldMeta `json:"fields"`

	fieldByName map[string]*FieldMeta
}

type FieldMeta struct {
	Name            string               `json:"name"`
	Column          string               `json:"column"`
	Type            ast.FieldType        `json:"type"`
	Nullable        bool                 `json:"nullable"`
	IsReference     bool                 `json:"is_reference"`
	ReferenceEntity string               `json:"reference_entity,omitempty"`
	Attributes      []ast.FieldAttribute `json:"attributes,omitempty"`
	Default         *ast.Literal         `json:"default,omitempty"`
}

func NewRegistry(app *ast.Application) *Registry {
	registry := &Registry{AppName: app.Name}
	for _, entity := range app.Entities {
		entityMeta := &EntityMeta{
			Name:  entity.Name,
			Table: entity.Name,
		}
		for _, field := range entity.Fields {
			column := field.Name
			referenceEntity := ""
			isReference := false
			if field.Type == ast.TypeRef {
				column = field.Name + "_id"
				referenceEntity = field.Reference.Entity
				isReference = true
			}

			entityMeta.Fields = append(entityMeta.Fields, &FieldMeta{
				Name:            field.Name,
				Column:          column,
				Type:            field.Type,
				Nullable:        field.Nullable,
				IsReference:     isReference,
				ReferenceEntity: referenceEntity,
				Attributes:      field.Attributes,
				Default:         field.Default,
			})
		}
		registry.Entities = append(registry.Entities, entityMeta)
	}
	registry.Reindex()
	return registry
}

func (r *Registry) Reindex() {
	r.entityByName = make(map[string]*EntityMeta, len(r.Entities))
	for _, entity := range r.Entities {
		r.entityByName[entity.Name] = entity
		entity.fieldByName = make(map[string]*FieldMeta, len(entity.Fields))
		for _, field := range entity.Fields {
			entity.fieldByName[field.Name] = field
		}
	}
}

func (r *Registry) Entity(name string) (*EntityMeta, bool) {
	entity, ok := r.entityByName[name]
	return entity, ok
}

func (r *Registry) Save(path string) error {
	bytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return fmt.Errorf("write metadata file '%s': %w", path, err)
	}
	return nil
}

func Load(path string) (*Registry, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metadata file '%s': %w", path, err)
	}

	var registry Registry
	if err := json.Unmarshal(bytes, &registry); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	registry.Reindex()
	return &registry, nil
}

func (e *EntityMeta) Field(name string) (*FieldMeta, bool) {
	field, ok := e.fieldByName[name]
	return field, ok
}
