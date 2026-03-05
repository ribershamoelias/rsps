package ui

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"rsps/internal/ast"
	"rsps/internal/db"
	"rsps/internal/runtime"
)

type Handler struct {
	repository   *db.Repository
	registry     *runtime.Registry
	listTemplate *template.Template
	formTemplate *template.Template
}

type ListPageData struct {
	AppName  string
	Entity   *runtime.EntityMeta
	Entities []*runtime.EntityMeta
	Rows     []ListRow
	Error    string
}

type ListRow struct {
	ID     int64
	Values map[string]string
}

type FormPageData struct {
	AppName  string
	Entity   *runtime.EntityMeta
	Entities []*runtime.EntityMeta
	Mode     string
	Action   string
	Fields   []FormField
	Error    string
}

type FormField struct {
	Name      string
	Label     string
	Kind      string
	Value     string
	Checked   bool
	Required  bool
	Nullable  bool
	Options   []FormOption
	TypeLabel string
}

type FormOption struct {
	Value    string
	Label    string
	Selected bool
}

func NewHandler(repository *db.Repository, registry *runtime.Registry, templateDir string) (*Handler, error) {
	listTemplate, err := template.ParseFiles(filepath.Join(templateDir, "ui_list.html"))
	if err != nil {
		return nil, fmt.Errorf("parse ui list template: %w", err)
	}
	formTemplate, err := template.ParseFiles(filepath.Join(templateDir, "ui_form.html"))
	if err != nil {
		return nil, fmt.Errorf("parse ui form template: %w", err)
	}

	return &Handler{
		repository:   repository,
		registry:     registry,
		listTemplate: listTemplate,
		formTemplate: formTemplate,
	}, nil
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/ui/", h.handle)
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/ui/"))
	if len(parts) == 0 {
		h.redirectToDefaultEntity(w, r)
		return
	}

	entity, ok := h.registry.Entity(parts[0])
	if !ok {
		http.NotFound(w, r)
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		h.renderList(w, entity, "")
		return
	}

	if len(parts) == 2 && parts[1] == "new" {
		switch r.Method {
		case http.MethodGet:
			h.renderCreateForm(w, entity, "")
		case http.MethodPost:
			h.submitCreate(w, r, entity)
		default:
			methodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	}

	if len(parts) == 3 && parts[2] == "edit" {
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.renderEditForm(w, entity, id, "")
		case http.MethodPost:
			h.submitEdit(w, r, entity, id)
		default:
			methodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	}

	if len(parts) == 3 && parts[2] == "delete" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w, http.MethodPost)
			return
		}
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		if err := h.repository.Delete(entity.Name, id); err != nil {
			h.renderList(w, entity, err.Error())
			return
		}
		http.Redirect(w, r, "/ui/"+entity.Name, http.StatusSeeOther)
		return
	}

	http.NotFound(w, r)
}

func (h *Handler) renderList(w http.ResponseWriter, entity *runtime.EntityMeta, errMessage string) {
	rows, err := h.repository.List(entity.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	viewRows := make([]ListRow, 0, len(rows))
	for _, row := range rows {
		view := ListRow{
			ID:     toInt64(row["id"]),
			Values: make(map[string]string, len(entity.Fields)),
		}
		for _, field := range entity.Fields {
			view.Values[field.Name] = formatDisplayValue(field, row[field.Name])
		}
		viewRows = append(viewRows, view)
	}

	data := ListPageData{
		AppName:  h.registry.AppName,
		Entity:   entity,
		Entities: h.registry.Entities,
		Rows:     viewRows,
		Error:    errMessage,
	}
	if err := h.listTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) renderCreateForm(w http.ResponseWriter, entity *runtime.EntityMeta, errMessage string) {
	fields, err := h.buildFormFields(entity, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := FormPageData{
		AppName:  h.registry.AppName,
		Entity:   entity,
		Entities: h.registry.Entities,
		Mode:     "create",
		Action:   "/ui/" + entity.Name + "/new",
		Fields:   fields,
		Error:    errMessage,
	}
	if err := h.formTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) renderEditForm(w http.ResponseWriter, entity *runtime.EntityMeta, id int64, errMessage string) {
	record, err := h.repository.Get(entity.Name, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	fields, err := h.buildFormFields(entity, record)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := FormPageData{
		AppName:  h.registry.AppName,
		Entity:   entity,
		Entities: h.registry.Entities,
		Mode:     "edit",
		Action:   fmt.Sprintf("/ui/%s/%d/edit", entity.Name, id),
		Fields:   fields,
		Error:    errMessage,
	}
	if err := h.formTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) submitCreate(w http.ResponseWriter, r *http.Request, entity *runtime.EntityMeta) {
	payload, err := h.parseFormPayload(r, entity)
	if err != nil {
		h.renderCreateForm(w, entity, err.Error())
		return
	}

	if err := runtime.ValidatePayload(entity, payload, true); err != nil {
		h.renderCreateForm(w, entity, err.Error())
		return
	}

	if _, err := h.repository.Create(entity.Name, payload); err != nil {
		h.renderCreateForm(w, entity, err.Error())
		return
	}

	http.Redirect(w, r, "/ui/"+entity.Name, http.StatusSeeOther)
}

func (h *Handler) submitEdit(w http.ResponseWriter, r *http.Request, entity *runtime.EntityMeta, id int64) {
	payload, err := h.parseFormPayload(r, entity)
	if err != nil {
		h.renderEditForm(w, entity, id, err.Error())
		return
	}

	if err := runtime.ValidatePayload(entity, payload, false); err != nil {
		h.renderEditForm(w, entity, id, err.Error())
		return
	}

	if _, err := h.repository.Update(entity.Name, id, payload); err != nil {
		h.renderEditForm(w, entity, id, err.Error())
		return
	}

	http.Redirect(w, r, "/ui/"+entity.Name, http.StatusSeeOther)
}

func (h *Handler) parseFormPayload(r *http.Request, entity *runtime.EntityMeta) (map[string]any, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("parse form: %w", err)
	}

	payload := make(map[string]any, len(entity.Fields))
	for _, field := range entity.Fields {
		if field.Type == ast.TypeBool {
			payload[field.Name] = r.FormValue(field.Name) == "on"
			continue
		}

		raw := strings.TrimSpace(r.FormValue(field.Name))
		if raw == "" {
			if field.Nullable {
				payload[field.Name] = nil
				continue
			}
			if field.Type == ast.TypeString || field.Type == ast.TypeText {
				payload[field.Name] = ""
				continue
			}
			continue
		}

		value, err := runtime.ParseStringValue(field, raw)
		if err != nil {
			return nil, err
		}
		payload[field.Name] = value
	}

	return payload, nil
}

func (h *Handler) buildFormFields(entity *runtime.EntityMeta, record map[string]any) ([]FormField, error) {
	fields := make([]FormField, 0, len(entity.Fields))

	for _, field := range entity.Fields {
		value := ""
		checked := false
		if record != nil {
			if raw, ok := record[field.Name]; ok && raw != nil {
				if field.Type == ast.TypeBool {
					checked = toBool(raw)
				} else {
					value = formatInputValue(field, raw)
				}
			}
		}

		viewField := FormField{
			Name:      field.Name,
			Label:     field.Name,
			Kind:      inputKind(field),
			Value:     value,
			Checked:   checked,
			Required:  !field.Nullable && field.Default == nil,
			Nullable:  field.Nullable,
			TypeLabel: string(field.Type),
		}

		if field.IsReference {
			options, err := h.referenceOptions(field, value)
			if err != nil {
				return nil, err
			}
			viewField.Options = options
		}

		fields = append(fields, viewField)
	}

	return fields, nil
}

func (h *Handler) referenceOptions(field *runtime.FieldMeta, selected string) ([]FormOption, error) {
	referenceEntity, ok := h.registry.Entity(field.ReferenceEntity)
	if !ok {
		return nil, fmt.Errorf("reference entity '%s' not found", field.ReferenceEntity)
	}

	rows, err := h.repository.List(referenceEntity.Name)
	if err != nil {
		return nil, err
	}

	labelField := firstTextField(referenceEntity)
	options := make([]FormOption, 0, len(rows)+1)
	options = append(options, FormOption{
		Value:    "",
		Label:    "-- none --",
		Selected: selected == "",
	})

	for _, row := range rows {
		id := toInt64(row["id"])
		label := fmt.Sprintf("#%d", id)
		if labelField != "" {
			if value, ok := row[labelField]; ok && value != nil {
				label = fmt.Sprintf("%v", value)
			}
		}
		idValue := strconv.FormatInt(id, 10)
		options = append(options, FormOption{
			Value:    idValue,
			Label:    label,
			Selected: selected == idValue,
		})
	}

	return options, nil
}

func firstTextField(entity *runtime.EntityMeta) string {
	for _, field := range entity.Fields {
		if field.Type == ast.TypeString || field.Type == ast.TypeText {
			return field.Name
		}
	}
	return ""
}

func inputKind(field *runtime.FieldMeta) string {
	if field.IsReference {
		return "select"
	}

	switch field.Type {
	case ast.TypeText:
		return "textarea"
	case ast.TypeInt, ast.TypeFloat:
		return "number"
	case ast.TypeBool:
		return "checkbox"
	case ast.TypeDate:
		return "date"
	case ast.TypeDatetime:
		return "datetime-local"
	default:
		return "text"
	}
}

func formatDisplayValue(field *runtime.FieldMeta, value any) string {
	if value == nil {
		return ""
	}
	if field.Type == ast.TypeBool {
		if toBool(value) {
			return "true"
		}
		return "false"
	}
	return fmt.Sprintf("%v", value)
}

func formatInputValue(field *runtime.FieldMeta, value any) string {
	if value == nil {
		return ""
	}

	switch field.Type {
	case ast.TypeDate:
		text := fmt.Sprintf("%v", value)
		if len(text) >= 10 {
			return text[:10]
		}
		return text
	case ast.TypeDatetime:
		text := fmt.Sprintf("%v", value)
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
			parsed, err := time.Parse(layout, text)
			if err == nil {
				return parsed.Format("2006-01-02T15:04")
			}
		}
		return text
	default:
		return fmt.Sprintf("%v", value)
	}
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "/")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func methodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (h *Handler) redirectToDefaultEntity(w http.ResponseWriter, r *http.Request) {
	if len(h.registry.Entities) == 0 {
		http.Error(w, "no entities configured", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/ui/"+h.registry.Entities[0].Name, http.StatusFound)
}

func toInt64(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func toBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case int64:
		return typed != 0
	case int:
		return typed != 0
	case float64:
		return typed != 0
	case string:
		lower := strings.ToLower(typed)
		return lower == "1" || lower == "true" || lower == "yes" || lower == "on"
	default:
		return false
	}
}
