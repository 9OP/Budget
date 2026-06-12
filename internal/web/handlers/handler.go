package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/9op/budget/internal/service"
)

const (
	monthLayout      = "2006-01"
	monthLabelLayout = "January 2006"
	prevMonthOffset  = -1
	nextMonthOffset  = 1
)

// Handler holds dependencies for page and partial handlers.
type Handler struct {
	svc  *service.Service
	tmpl templates
}

// NewHandler creates a Handler, parsing all templates from the given filesystem.
func NewHandler(svc *service.Service, fsys fs.FS) (*Handler, error) {
	tmpl, err := parseTemplates(fsys)
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return &Handler{svc: svc, tmpl: tmpl}, nil
}

// renderPage renders a full page. ActivePage is injected automatically
// by merging the data struct's fields into a map alongside the base fields.
func (h *Handler) renderPage(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.tmpl[name].ExecuteTemplate(w, "layout", h.enrich(name, data)); err != nil {
		slog.Error("render template", slog.String("name", name), slog.String("error", err.Error()))
	}
}

func (h *Handler) enrich(name string, data any) any {
	m := map[string]any{
		"ActivePage": name,
	}

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		m[t.Field(i).Name] = v.Field(i).Interface()
	}
	return m
}

// renderPartial renders an HTMX fragment by template name.
func (h *Handler) renderPartial(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.tmpl["partials"].ExecuteTemplate(w, name, data); err != nil {
		slog.Error("render partial", slog.String("name", name), slog.String("error", err.Error()))
	}
}

// setToast sets the HX-Trigger header to fire a "toast" event on the client.
// kind should be "success" or "error".
func setToast(w http.ResponseWriter, message, kind string) {
	type payload struct {
		Toast struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"toast"`
	}

	var p payload
	p.Toast.Message = message
	p.Toast.Type = kind

	b, err := json.Marshal(p)
	if err != nil {
		return
	}

	w.Header().Set("HX-Trigger", string(b))
}

func parseMonthParam(s string) (time.Time, error) {
	return time.Parse(monthLayout, s)
}

// templates maps template names to their parsed template sets.
type templates map[string]*template.Template

const layout = "templates/layout.html"

func parseTemplates(fsys fs.FS) (templates, error) {
	t := make(templates)

	partialPaths, err := fs.Glob(fsys, "templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("glob partials: %w", err)
	}

	pagePaths, err := fs.Glob(fsys, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("glob pages: %w", err)
	}

	for _, path := range pagePaths {
		if path == layout {
			continue
		}

		name := strings.TrimSuffix(filepath.Base(path), ".html")
		tmpl, err := template.ParseFS(fsys, append([]string{layout, path}, partialPaths...)...)
		if err != nil {
			return nil, fmt.Errorf("parse page %s: %w", name, err)
		}

		t[name] = tmpl
	}

	partials, err := template.ParseFS(fsys, partialPaths...)
	if err != nil {
		return nil, fmt.Errorf("parse partials: %w", err)
	}

	t["partials"] = partials

	return t, nil
}
