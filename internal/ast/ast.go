package ast

type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Application struct {
	Name     string    `json:"name"`
	Entities []*Entity `json:"entities"`
	Pos      Position  `json:"pos"`
}

type Entity struct {
	Name   string   `json:"name"`
	Fields []*Field `json:"fields"`
	Pos    Position `json:"pos"`
}

type FieldType string

const (
	TypeString   FieldType = "string"
	TypeText     FieldType = "text"
	TypeInt      FieldType = "int"
	TypeFloat    FieldType = "float"
	TypeBool     FieldType = "bool"
	TypeDate     FieldType = "date"
	TypeDatetime FieldType = "datetime"
	TypeJSON     FieldType = "json"
	TypeRef      FieldType = "ref"
)

type FieldAttribute string

const (
	AttrUnique FieldAttribute = "unique"
	AttrIndex  FieldAttribute = "index"
)

type Reference struct {
	Entity string   `json:"entity"`
	Pos    Position `json:"pos"`
}

type Literal struct {
	Kind  string   `json:"kind"`
	Value string   `json:"value"`
	Pos   Position `json:"pos"`
}

type Field struct {
	Name       string           `json:"name"`
	Type       FieldType        `json:"type"`
	Nullable   bool             `json:"nullable"`
	Attributes []FieldAttribute `json:"attributes"`
	Default    *Literal         `json:"default,omitempty"`
	Reference  *Reference       `json:"reference,omitempty"`
	Pos        Position         `json:"pos"`
}

func (f *Field) HasAttribute(attr FieldAttribute) bool {
	for _, candidate := range f.Attributes {
		if candidate == attr {
			return true
		}
	}
	return false
}
