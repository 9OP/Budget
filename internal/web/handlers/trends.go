package handlers

import (
	"cmp"
	"encoding/json"
	"html/template"
	"net/http"
	"slices"
	"time"

	"github.com/9op/budget/internal/domain"
)

// TrendsData is passed to the trends template.
type TrendsData struct {
	ChartJSON template.JS
	HasData   bool
}

type trendsChartData struct {
	Labels     []string  `json:"labels"`
	Income     []float64 `json:"income"`
	Expense    []float64 `json:"expense"`
	Net        []float64 `json:"net"`
	CumIncome  []float64 `json:"cum_income"`
	CumExpense []float64 `json:"cum_expense"`
	CumNet     []float64 `json:"cum_net"`
}

// TrendsPage renders the month-over-month trends chart.
func (h *Handler) TrendsPage(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ListItems(r.Context(), domain.ItemFilter{})
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	b, _ := json.Marshal(buildTrendsChartData(items)) //nolint:errchkjson // only safe types; Marshal cannot fail

	h.renderPage(w, r, "trends", &TrendsData{
		ChartJSON: template.JS(b), //nolint:gosec // JSON is generated internally, not user-controlled
		HasData:   len(items) > 0,
	})
}

type monthKey struct {
	year  int
	month time.Month
}

func buildTrendsChartData(items []domain.Item) trendsChartData {
	totals := map[monthKey][2]float64{} // [0]=income [1]=expense

	for _, item := range items {
		key := monthKey{item.Date.Year(), item.Date.Month()}
		t := totals[key]

		switch item.Type {
		case domain.Income:
			t[0] += item.Amount
		case domain.Expense:
			t[1] += item.Amount
		default:
			// unknown type; skip
		}

		totals[key] = t
	}

	keys := make([]monthKey, 0, len(totals))
	for k := range totals {
		keys = append(keys, k)
	}

	slices.SortFunc(keys, func(a, b monthKey) int {
		if a.year != b.year {
			return cmp.Compare(a.year, b.year)
		}

		return cmp.Compare(int(a.month), int(b.month))
	})

	if len(keys) == 0 {
		return trendsChartData{}
	}

	// Fill every month between first and last, including empty ones.
	first := keys[0]
	last := keys[len(keys)-1]
	cur := time.Date(first.year, first.month, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(last.year, last.month, 1, 0, 0, 0, 0, time.UTC)

	var allMonths []monthKey

	for !cur.After(end) {
		allMonths = append(allMonths, monthKey{cur.Year(), cur.Month()})
		cur = cur.AddDate(0, 1, 0)
	}

	n := len(allMonths)
	cd := trendsChartData{
		Labels:     make([]string, n),
		Income:     make([]float64, n),
		Expense:    make([]float64, n),
		Net:        make([]float64, n),
		CumIncome:  make([]float64, n),
		CumExpense: make([]float64, n),
		CumNet:     make([]float64, n),
	}

	var cumInc, cumExp float64

	for i, mk := range allMonths {
		t := time.Date(mk.year, mk.month, 1, 0, 0, 0, 0, time.UTC)
		cd.Labels[i] = t.Format("Jan 2006")

		if data, ok := totals[mk]; ok {
			cd.Income[i] = data[0]
			cd.Expense[i] = data[1]
		}

		cd.Net[i] = cd.Income[i] - cd.Expense[i]

		cumInc += cd.Income[i]
		cumExp += cd.Expense[i]

		cd.CumIncome[i] = cumInc
		cd.CumExpense[i] = cumExp
		cd.CumNet[i] = cumInc - cumExp
	}

	return cd
}
