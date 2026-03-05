package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"rsps/internal/db"
	"rsps/internal/runtime"
)

type Handler struct {
	repository *db.Repository
	registry   *runtime.Registry
}

func NewHandler(repository *db.Repository, registry *runtime.Registry) *Handler {
	return &Handler{repository: repository, registry: registry}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/", h.handle)
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	// Dynamic route parser for /api/{entity} and /api/{entity}/{id}.
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/"))
	if len(parts) == 0 {
		writeJSONError(w, http.StatusNotFound, "resource not found")
		return
	}

	entity, ok := h.registry.Entity(parts[0])
	if !ok {
		writeJSONError(w, http.StatusNotFound, "entity not found")
		return
	}

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			h.list(w, entity)
		case http.MethodPost:
			h.create(w, r, entity)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	}

	if len(parts) == 2 {
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid id")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.get(w, entity, id)
		case http.MethodPut:
			h.update(w, r, entity, id)
		case http.MethodDelete:
			h.delete(w, entity, id)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPut, http.MethodDelete)
		}
		return
	}

	writeJSONError(w, http.StatusNotFound, "resource not found")
}

func (h *Handler) list(w http.ResponseWriter, entity *runtime.EntityMeta) {
	records, err := h.repository.List(entity.Name)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (h *Handler) get(w http.ResponseWriter, entity *runtime.EntityMeta, id int64) {
	record, err := h.repository.Get(entity.Name, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request, entity *runtime.EntityMeta) {
	payload, err := decodePayload(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validation is metadata-driven and enforces unknown field/type/required constraints.
	if err := runtime.ValidatePayload(entity, payload, true); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	record, err := h.repository.Create(entity.Name, payload)
	if err != nil {
		if isConstraintError(err) {
			writeJSONError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, record)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request, entity *runtime.EntityMeta, id int64) {
	payload, err := decodePayload(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := runtime.ValidatePayload(entity, payload, false); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	record, err := h.repository.Update(entity.Name, id, payload)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		if isConstraintError(err) {
			writeJSONError(w, http.StatusConflict, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func (h *Handler) delete(w http.ResponseWriter, entity *runtime.EntityMeta, id int64) {
	err := h.repository.Delete(entity.Name, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func decodePayload(r *http.Request) (map[string]any, error) {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	payload := make(map[string]any)
	if err := decoder.Decode(&payload); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("request body is required")
		}
		return nil, fmt.Errorf("invalid json payload: %w", err)
	}

	if decoder.More() {
		return nil, fmt.Errorf("invalid json payload: multiple objects")
	}

	return payload, nil
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

func isConstraintError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "constraint") || strings.Contains(message, "unique") || strings.Contains(message, "foreign key")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(bytes)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeMethodNotAllowed(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}
