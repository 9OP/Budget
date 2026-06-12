package handlers

import (
	"errors"
	"net/http"

	"github.com/9op/budget/internal/domain"
	"github.com/go-chi/chi/v5"
)

// CategoriesData is passed to the categories page template.
type CategoriesData struct {
	Categories []domain.Category
}

// CategoriesPage renders the categories list page.
func (h *Handler) CategoriesPage(w http.ResponseWriter, r *http.Request) {
	cats, err := h.svc.ListCategories(r.Context())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	h.renderPage(w, "categories", &CategoriesData{
		Categories: cats,
	})
}

// CatFormPartial renders the category creation form partial.
func (h *Handler) CatFormPartial(w http.ResponseWriter, r *http.Request) {
	h.renderPartial(w, "category_form", nil)
}

// DeleteCategoryPartial handles an HTMX DELETE to remove a category.
func (h *Handler) DeleteCategoryPartial(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := h.svc.DeleteCategory(r.Context(), name); err != nil {
		switch {
		case errors.Is(err, domain.ErrCategoryNotFound):
			setToast(w, err.Error(), "error")
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			setToast(w, "internal server error", "error")
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}

		return
	}

	setToast(w, "Category deleted", "success")
	w.WriteHeader(http.StatusOK)
}

// CategoryEditFormPartial renders the inline rename form for a category.
func (h *Handler) CategoryEditFormPartial(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	h.renderPartial(w, "category_edit_form", domain.Category{Name: name})
}

// CategoryRowPartial renders a single read-only category item (used to cancel an inline edit).
func (h *Handler) CategoryRowPartial(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	h.renderPartial(w, "category_item", domain.Category{Name: name})
}

// RenameCategoryPartial handles an HTMX PUT to rename a category and returns the updated category item partial.
func (h *Handler) RenameCategoryPartial(w http.ResponseWriter, r *http.Request) {
	oldName := chi.URLParam(r, "name")

	if err := r.ParseForm(); err != nil {
		setToast(w, "invalid form data", "error")
		http.Error(w, "invalid form data", http.StatusBadRequest)

		return
	}

	cat, err := h.svc.RenameCategory(r.Context(), oldName, r.FormValue("name"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCategoryNotFound):
			setToast(w, err.Error(), "error")
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, domain.ErrCategoryAlreadyExists),
			errors.Is(err, domain.ErrMissingCategoryName):
			setToast(w, err.Error(), "error")
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			setToast(w, "internal server error", "error")
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}

		return
	}

	setToast(w, "Category renamed", "success")
	h.renderPartial(w, "category_item", cat)
}

// CreateCategoryPartial handles an HTMX form submission to create a category and returns the category item partial.
func (h *Handler) CreateCategoryPartial(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		setToast(w, "invalid form data", "error")
		http.Error(w, "invalid form data", http.StatusBadRequest)

		return
	}

	cat, err := h.svc.CreateCategory(r.Context(), r.FormValue("name"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrCategoryAlreadyExists),
			errors.Is(err, domain.ErrMissingCategoryName):
			setToast(w, err.Error(), "error")
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			setToast(w, "internal server error", "error")
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}

		return
	}

	setToast(w, "Category created", "success")
	h.renderPartial(w, "category_item", cat)
}
