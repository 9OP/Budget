// Package handlers contains the HTTP page and HTMX partial handlers for the budget web UI.
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
	maxFormBodySize  = 1 << 20 // 1 MB — applied via http.MaxBytesReader before ParseForm
)

// Handler holds dependencies for page and partial handlers.
type Handler struct {
	svc     *service.Service
	tmpl    templates
	authCfg AuthConfig
}

// NewHandler creates a Handler, parsing all templates from the given filesystem.
func NewHandler(svc *service.Service, fsys fs.FS, authCfg AuthConfig) (*Handler, error) {
	tmpl, err := parseTemplates(fsys)
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return &Handler{svc: svc, tmpl: tmpl, authCfg: authCfg}, nil
}

// renderPage renders a full page. ActivePage is injected automatically
// by merging the data struct's fields into a map alongside the base fields.
func (h *Handler) renderPage(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.tmpl[name].ExecuteTemplate(w, "layout", h.enrich(name, data)); err != nil {
		slog.Error("render template", slog.String("name", name), slog.String("error", err.Error()))
	}
}

func (*Handler) enrich(name string, data any) any {
	m := map[string]any{
		"ActivePage": name,
	}

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	t := v.Type()
	for i := range v.NumField() {
		m[t.Field(i).Name] = v.Field(i).Interface()
	}

	return m
}

// renderPartial renders an HTMX fragment by template name.
func (h *Handler) renderPartial(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.tmpl["partials"].ExecuteTemplate(w, name, data); err != nil {
		//nolint:gosec // name is an internal template identifier, not user input
		slog.Error("render partial", slog.String("name", name), slog.String("error", err.Error()))
	}
}

// toastContent holds the message and kind for a client-side toast notification.
type toastContent struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// toastPayload is the HX-Trigger JSON payload that fires a "toast" event on the client.
type toastPayload struct {
	Toast toastContent `json:"toast"`
}

// setToast sets the HX-Trigger header to fire a "toast" event on the client.
// kind should be "success" or "error".
func setToast(w http.ResponseWriter, message, kind string) {
	p := toastPayload{Toast: toastContent{Message: message, Type: kind}}

	b, err := json.Marshal(p)
	if err != nil {
		return
	}

	w.Header().Set("HX-Trigger", string(b))
}

func parseMonthParam(s string) (time.Time, error) {
	t, err := time.Parse(monthLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse month %q: %w", s, err)
	}

	return t, nil
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

		parsed, parseErr := template.ParseFS(fsys, append([]string{layout, path}, partialPaths...)...)
		if parseErr != nil {
			return nil, fmt.Errorf("parse page %s: %w", name, parseErr)
		}

		t[name] = parsed
	}

	partials, err := template.ParseFS(fsys, partialPaths...)
	if err != nil {
		return nil, fmt.Errorf("parse partials: %w", err)
	}

	t["partials"] = partials

	// Standalone pages — parsed without the layout so they render as complete documents.
	for _, name := range []string{"login"} {
		parsed, parseErr := template.ParseFS(fsys, "templates/"+name+".html")
		if parseErr != nil {
			return nil, fmt.Errorf("parse standalone %s: %w", name, parseErr)
		}

		t[name] = parsed
	}

	return t, nil
}
