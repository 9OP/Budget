package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/9op/budget/internal/service"
)

// chartColors is a fixed palette assigned to categories in order of descending spend.
var chartColors = []string{ //nolint:gochecknoglobals // fixed palette, not mutable state
	"#4e79a7",
	"#f28e2b",
	"#e15759",
	"#76b7b2",
	"#59a14f",
	"#edc948",
	"#b07aa1",
	"#ff9da7",
	"#9c755f",
	"#bab0ac",
}

// chartData is the JSON payload consumed by charts.js.
type chartData struct {
	Labels  []string  `json:"labels"`
	Amounts []float64 `json:"amounts"`
	Colors  []string  `json:"colors"`
}

// categoryView is a presentation-layer view of a CategoryTotal with an assigned chart color.
type categoryView struct {
	Name    string
	Amount  float64
	Percent float64
	Color   string
}

// DashboardData is passed to the dashboard template.
type DashboardData struct {
	Month             string
	MonthLabel        string
	PrevMonth         string
	NextMonth         string
	TotalIncome       float64
	TotalExpense      float64
	Net               float64
	BudgetRows        []service.BudgetConsumption
	ExpenseByCategory []categoryView
	ExpenseChartJSON  template.JS
}

// Dashboard renders the dashboard page for the requested month.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format(monthLayout)
	}

	monthTime, err := time.Parse(monthLayout, month)
	if err != nil {
		http.Error(w, "invalid month parameter", http.StatusBadRequest)

		return
	}

	summary, err := h.svc.GetDashboardSummary(r.Context(), monthTime)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	expCats := withColors(summary.ExpenseByCategory)

	h.renderPage(w, r, "dashboard", &DashboardData{
		Month:             month,
		MonthLabel:        monthTime.Format(monthLabelLayout),
		PrevMonth:         monthTime.AddDate(0, prevMonthOffset, 0).Format(monthLayout),
		NextMonth:         monthTime.AddDate(0, nextMonthOffset, 0).Format(monthLayout),
		TotalIncome:       summary.TotalIncome,
		TotalExpense:      summary.TotalExpense,
		Net:               summary.Net,
		BudgetRows:        summary.BudgetRows,
		ExpenseByCategory: expCats,
		ExpenseChartJSON:  toChartJSON(expCats),
	})
}

func withColors(cats []service.CategoryTotal) []categoryView {
	views := make([]categoryView, len(cats))

	for i, c := range cats {
		views[i] = categoryView{
			Name:    c.Name,
			Amount:  c.Amount,
			Percent: c.Percent,
			Color:   chartColors[i%len(chartColors)],
		}
	}

	return views
}

func toChartJSON(cats []categoryView) template.JS {
	cd := chartData{
		Labels:  make([]string, len(cats)),
		Amounts: make([]float64, len(cats)),
		Colors:  make([]string, len(cats)),
	}

	for i, c := range cats {
		cd.Labels[i] = c.Name
		cd.Amounts[i] = c.Amount
		cd.Colors[i] = c.Color
	}

	b, _ := json.Marshal(cd) //nolint:errchkjson // chartData contains only string/float64 slices; Marshal cannot fail

	return template.JS(b) //nolint:gosec // JSON is generated internally, not user-controlled
}
