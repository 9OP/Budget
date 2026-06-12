// Package web wires the HTTP server, middleware, and routes for the budget application.
package web

import (
	"embed"
	"fmt"
	"net/http"
	"time"

	"github.com/9op/budget/internal/service"
	"github.com/9op/budget/internal/web/handlers"
	"github.com/9op/budget/internal/web/middleware"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

//go:embed static templates
var webFS embed.FS

const (
	requestTimeout   = 30 * time.Second
	compressionLevel = 5
	staticMaxAge     = time.Hour
)

// NewServer builds and returns the chi router wired to the given service.
func NewServer(svc *service.Service) (*chi.Mux, error) {
	h, err := handlers.NewHandler(svc, webFS)
	if err != nil {
		return nil, fmt.Errorf("create handlers: %w", err)
	}

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Compress(compressionLevel))
	r.Use(chimw.Timeout(requestTimeout))
	r.Use(chimw.CleanPath)
	r.Use(chimw.StripSlashes)
	r.Use(chimw.GetHead)

	// Static assets — 1h client cache via Cache-Control header.
	cacheControl := fmt.Sprintf("public, max-age=%d", int(staticMaxAge.Seconds()))
	r.With(chimw.SetHeader("Cache-Control", cacheControl)).Handle("/static/*", http.FileServerFS(webFS))

	// Pages.
	r.Get("/", h.Dashboard)
	r.Get("/items", h.ItemsPage)
	r.Get("/budgets", h.BudgetsPage)
	r.Get("/categories", h.CategoriesPage)
	r.Get("/trends", h.TrendsPage)

	// Partials.
	r.Get("/partials/item-form", h.ItemFormPartial)
	r.Get("/partials/item-form/{id}", h.ItemFormPrefillPartial)
	r.Get("/partials/item-edit/{id}", h.ItemEditFormPartial)
	r.Get("/partials/item-row/{id}", h.ItemRowPartial)
	r.Get("/partials/category-form", h.CatFormPartial)
	r.Get("/partials/category-edit/{name}", h.CategoryEditFormPartial)
	r.Get("/partials/category-row/{name}", h.CategoryRowPartial)

	// HTMX actions — all return HTML fragments consumed by the frontend.
	r.Post("/api/items", h.CreateItemPartial)
	r.Put("/api/items/{id}", h.UpdateItemPartial)
	r.Delete("/api/items/{id}", h.DeleteItem)
	r.Post("/api/categories", h.CreateCategoryPartial)
	r.Put("/api/categories/{name}", h.RenameCategoryPartial)
	r.Delete("/api/categories/{name}", h.DeleteCategoryPartial)
	r.Put("/api/budgets/{month}/{category}", h.SetBudgetPartial)
	r.Delete("/api/budgets/{month}/{category}", h.DeleteBudgetPartial)

	return r, nil
}
