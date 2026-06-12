package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/9op/budget/internal/domain"
	"github.com/go-chi/chi/v5"
)

// budgetRowData is template data for a single budget row.
type budgetRowData struct {
	Month       string
	Category    string
	Amount      float64
	Inherited   bool
	SourceMonth string
	HasBudget   bool
}

// BudgetsData is passed to the budgets page template.
type BudgetsData struct {
	Month      string
	MonthLabel string
	PrevMonth  string
	NextMonth  string
	Rows       []budgetRowData
}

// BudgetsPage renders the budgets page for the requested month.
func (h *Handler) BudgetsPage(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format(monthLayout)
	}

	monthTime, err := time.Parse(monthLayout, month)
	if err != nil {
		http.Error(w, "invalid month parameter", http.StatusBadRequest)

		return
	}

	cats, err := h.svc.ListCategories(r.Context())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	budgets, err := h.svc.ListBudgets(r.Context(), domain.BudgetFilter{Month: &monthTime})
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	budgetMap := make(map[string]domain.Budget, len(budgets))
	for _, b := range budgets {
		budgetMap[strings.ToLower(b.Category)] = b
	}

	rows := make([]budgetRowData, 0, len(cats))
	for _, cat := range cats {
		b, ok := budgetMap[strings.ToLower(cat.Name)]
		sourceMonth := ""
		if !b.SourceMonth.IsZero() {
			sourceMonth = b.SourceMonth.Format(monthLayout)
		}
		rows = append(rows, budgetRowData{
			Month:       month,
			Category:    cat.Name,
			Amount:      b.Amount,
			Inherited:   b.Inherited,
			SourceMonth: sourceMonth,
			HasBudget:   ok,
		})
	}

	h.renderPage(w, r, "budgets", &BudgetsData{
		Month:      month,
		MonthLabel: monthTime.Format(monthLabelLayout),
		PrevMonth:  monthTime.AddDate(0, prevMonthOffset, 0).Format(monthLayout),
		NextMonth:  monthTime.AddDate(0, nextMonthOffset, 0).Format(monthLayout),
		Rows:       rows,
	})
}

// SetBudgetPartial handles an HTMX form submission to set a budget and returns the budget row partial.
func (h *Handler) SetBudgetPartial(w http.ResponseWriter, r *http.Request) {
	monthStr := chi.URLParam(r, "month")
	category := chi.URLParam(r, "category")

	monthTime, err := parseMonthParam(monthStr)
	if err != nil {
		http.Error(w, domain.ErrInvalidMonth.Error(), http.StatusBadRequest)

		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBodySize)

	if parseErr := r.ParseForm(); parseErr != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)

		return
	}

	amount, err := strconv.ParseFloat(r.FormValue("amount"), 64)
	if err != nil {
		http.Error(w, "invalid amount, expected a number", http.StatusBadRequest)

		return
	}

	budget, err := h.svc.SetBudget(r.Context(), domain.Budget{
		Category: category,
		Month:    monthTime,
		Amount:   amount,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	h.renderPartial(w, "budget_row", budgetRowData{
		Month:     monthStr,
		Category:  budget.Category,
		Amount:    budget.Amount,
		HasBudget: true,
	})
}

// DeleteBudgetPartial handles an HTMX delete to remove a budget and returns an empty budget row partial.
func (h *Handler) DeleteBudgetPartial(w http.ResponseWriter, r *http.Request) {
	monthStr := chi.URLParam(r, "month")
	category := chi.URLParam(r, "category")

	monthTime, err := parseMonthParam(monthStr)
	if err != nil {
		http.Error(w, domain.ErrInvalidMonth.Error(), http.StatusBadRequest)

		return
	}

	if err := h.svc.DeleteBudget(r.Context(), category, monthTime); err != nil {
		switch {
		case errors.Is(err, domain.ErrBudgetNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}

		return
	}

	h.renderPartial(w, "budget_row", budgetRowData{
		Month:    monthStr,
		Category: category,
	})
}
