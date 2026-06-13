package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/9op/budget/internal/domain"
	"github.com/9op/budget/internal/service"
	"github.com/go-chi/chi/v5"
)

const itemDateLayout = "2006-01-02"

// ErrInvalidAmount is returned when the amount field cannot be parsed as a number.
var ErrInvalidAmount = errors.New("invalid amount, expected a number")

// ErrInvalidItemDate is returned when the date field has an invalid format.
var ErrInvalidItemDate = errors.New("invalid date format, expected YYYY-MM-DD")

// itemView is a template-friendly representation of a domain.Item with pre-formatted fields.
type itemView struct {
	ID       string
	Type     string
	Name     string
	Amount   float64
	Date     string
	Category string
}

// itemFormData is passed to the item_form partial.
// Prefill is non-nil when the form should be pre-populated (e.g. duplicate).
type itemFormData struct {
	Categories []domain.Category
	Prefill    *itemView
}

// itemEditFormData is passed to the item_edit_form partial.
type itemEditFormData struct {
	Item       itemView
	Categories []domain.Category
}

// ItemsData is passed to the items list template.
type ItemsData struct {
	Items           []itemView
	Categories      []domain.Category
	ItemFormData    itemFormData
	Month           string
	MonthLabel      string
	PrevMonth       string
	NextMonth       string
	Search          string
	ItemType        string
	Category        string
	TotalExpense    float64
	TotalIncome     float64
	TotalInvestment float64
}

// ItemsPage renders the items list page.
func (h *Handler) ItemsPage(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	monthStr := q.Get("month")
	if monthStr == "" {
		monthStr = time.Now().Format(monthLayout)
	}

	monthTime, err := parseMonthParam(monthStr)
	if err != nil {
		http.Error(w, "invalid month parameter", http.StatusBadRequest)

		return
	}

	search := q.Get("search")
	filter := domain.ItemFilter{Month: &monthTime, Search: search}

	if itemType := q.Get("type"); itemType != "" {
		t := domain.ItemType(itemType)
		filter.Type = &t
	}

	if cat := q.Get("category"); cat != "" {
		filter.Category = &cat
	}

	items, err := h.svc.ListItems(r.Context(), filter)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	cats, err := h.svc.ListCategories(r.Context())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	var totalExpense, totalIncome, totalInvestment float64

	views := make([]itemView, 0, len(items))
	for _, item := range items {
		switch item.Type {
		case domain.Expense:
			totalExpense += item.Amount
		case domain.Income:
			totalIncome += item.Amount
		case domain.Investment:
			totalInvestment += item.Amount
		}
		views = append(views, toItemView(item))
	}

	h.renderPage(w, r, "items", &ItemsData{
		Items:           views,
		Categories:      cats,
		ItemFormData:    itemFormData{Categories: cats},
		Month:           monthStr,
		MonthLabel:      monthTime.Format(monthLabelLayout),
		PrevMonth:       monthTime.AddDate(0, prevMonthOffset, 0).Format(monthLayout),
		NextMonth:       monthTime.AddDate(0, nextMonthOffset, 0).Format(monthLayout),
		Search:          search,
		ItemType:        q.Get("type"),
		Category:        q.Get("category"),
		TotalExpense:    totalExpense,
		TotalIncome:     totalIncome,
		TotalInvestment: totalInvestment,
	})
}

// DeleteItem removes the item with the given URL id parameter and returns 200.
func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.DeleteItem(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, domain.ErrItemNotFound):
			setToast(w, err.Error(), "error")
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			setToast(w, "internal server error", "error")
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}

		return
	}

	setToast(w, "Item deleted", "success")
	w.WriteHeader(http.StatusOK)
}

// ItemFormPartial renders the blank item creation form partial.
func (h *Handler) ItemFormPartial(w http.ResponseWriter, r *http.Request) {
	cats, err := h.svc.ListCategories(r.Context())
	if err != nil {
		http.Error(w, "failed to load categories", http.StatusInternalServerError)

		return
	}

	h.renderPartial(w, "item_form", itemFormData{Categories: cats})
}

// ItemFormPrefillPartial renders the item form pre-populated with an existing item's data (for duplicate).
func (h *Handler) ItemFormPrefillPartial(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	item, err := h.svc.GetItem(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrItemNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}

		return
	}

	cats, err := h.svc.ListCategories(r.Context())
	if err != nil {
		http.Error(w, "failed to load categories", http.StatusInternalServerError)

		return
	}

	view := toItemView(item)
	view.Date = time.Now().Format(itemDateLayout)

	h.renderPartial(w, "item_form", itemFormData{Categories: cats, Prefill: &view})
}

// ItemEditFormPartial renders the inline edit row for an existing item.
func (h *Handler) ItemEditFormPartial(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	item, err := h.svc.GetItem(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrItemNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}

		return
	}

	cats, err := h.svc.ListCategories(r.Context())
	if err != nil {
		http.Error(w, "failed to load categories", http.StatusInternalServerError)

		return
	}

	h.renderPartial(w, "item_edit_form", itemEditFormData{
		Item:       toItemView(item),
		Categories: cats,
	})
}

// ItemRowPartial renders a single read-only item row (used to cancel an inline edit).
func (h *Handler) ItemRowPartial(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	item, err := h.svc.GetItem(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrItemNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}

		return
	}

	h.renderPartial(w, "item_row", toItemView(item))
}

// UpdateItemPartial handles an HTMX PUT to update an item and returns the updated row partial.
func (h *Handler) UpdateItemPartial(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBodySize)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)

		return
	}

	amount, err := strconv.ParseFloat(r.FormValue("amount"), 64)
	if err != nil {
		http.Error(w, ErrInvalidAmount.Error(), http.StatusBadRequest)

		return
	}

	date, err := time.Parse(itemDateLayout, r.FormValue("date"))
	if err != nil {
		http.Error(w, ErrInvalidItemDate.Error(), http.StatusBadRequest)

		return
	}

	item, err := h.svc.UpdateItem(r.Context(), id, service.UpdateItemInput{
		Type:     domain.ItemType(r.FormValue("type")),
		Name:     r.FormValue("name"),
		Amount:   amount,
		Date:     date,
		Category: r.FormValue("category"),
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrItemNotFound):
			setToast(w, err.Error(), "error")
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			setToast(w, err.Error(), "error")
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		return
	}

	setToast(w, "Item updated", "success")
	h.renderPartial(w, "item_row", toItemView(item))
}

// CreateItemPartial handles an HTMX form submission to create an item and returns the item row partial.
func (h *Handler) CreateItemPartial(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBodySize)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)

		return
	}

	amount, err := strconv.ParseFloat(r.FormValue("amount"), 64)
	if err != nil {
		http.Error(w, ErrInvalidAmount.Error(), http.StatusBadRequest)

		return
	}

	date, err := time.Parse(itemDateLayout, r.FormValue("date"))
	if err != nil {
		http.Error(w, ErrInvalidItemDate.Error(), http.StatusBadRequest)

		return
	}

	item, err := h.svc.CreateItem(r.Context(), service.CreateItemInput{
		Type:     domain.ItemType(r.FormValue("type")),
		Name:     r.FormValue("name"),
		Amount:   amount,
		Date:     date,
		Category: r.FormValue("category"),
	})
	if err != nil {
		setToast(w, err.Error(), "error")
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	setToast(w, "Item created", "success")
	h.renderPartial(w, "item_row", toItemView(item))
}

func toItemView(item domain.Item) itemView {
	return itemView{
		ID:       item.ID,
		Type:     string(item.Type),
		Name:     item.Name,
		Amount:   item.Amount,
		Date:     item.Date.Format(itemDateLayout),
		Category: item.Category,
	}
}
