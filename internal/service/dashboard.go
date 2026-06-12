package service

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"time"

	"github.com/9op/budget/internal/domain"
)

const (
	warningThreshold = 80.0
	percentFactor    = 100.0
)

// CategoryTotal holds aggregated spend and percentage share for a single category.
type CategoryTotal struct {
	Name    string
	Amount  float64
	Percent float64
}

// BudgetConsumption holds budget vs actual spend for one category.
type BudgetConsumption struct {
	Category string
	Budget   float64
	Spent    float64
	Percent  float64
	BarWidth float64 // 0–100, clamped for CSS progress bar
	Status   string  // "ok" | "warning" | "over"
}

// DashboardSummary holds all computed data for the dashboard page.
type DashboardSummary struct {
	TotalIncome       float64
	TotalExpense      float64
	Net               float64
	BudgetRows        []BudgetConsumption
	ExpenseByCategory []CategoryTotal
	IncomeByCategory  []CategoryTotal
}

// GetDashboardSummary computes aggregated totals and budget consumption for the given month.
func (s *Service) GetDashboardSummary(ctx context.Context, month time.Time) (DashboardSummary, error) {
	items, err := s.repo.ListItems(ctx, domain.ItemFilter{Month: &month})
	if err != nil {
		return DashboardSummary{}, err
	}

	budgets, err := s.ListBudgets(ctx, domain.BudgetFilter{Month: &month})
	if err != nil {
		return DashboardSummary{}, err
	}

	return computeDashboardSummary(items, budgets), nil
}

func computeDashboardSummary(items []domain.Item, budgets []domain.Budget) DashboardSummary {
	var totalIncome, totalExpense float64

	expByCat := map[string]float64{}
	incByCat := map[string]float64{}

	for _, item := range items {
		switch item.Type {
		case domain.Income:
			totalIncome += item.Amount
			incByCat[item.Category] += item.Amount
		case domain.Expense:
			totalExpense += item.Amount
			expByCat[item.Category] += item.Amount
		default:
			// unknown type; skip
		}
	}

	expenseCats := buildCategoryTotals(expByCat, totalExpense)
	incomeCats := buildCategoryTotals(incByCat, totalIncome)

	spentByCategory := make(map[string]float64, len(expenseCats))
	for _, c := range expenseCats {
		spentByCategory[strings.ToLower(c.Name)] = c.Amount
	}

	budgetRows := make([]BudgetConsumption, 0, len(budgets))

	for _, b := range budgets {
		spent := spentByCategory[strings.ToLower(b.Category)]

		var pct float64
		if b.Amount > 0 {
			pct = spent / b.Amount * percentFactor
		}

		barWidth := pct
		if barWidth > percentFactor {
			barWidth = percentFactor
		}

		status := "ok"
		if pct >= percentFactor {
			status = "over"
		} else if pct >= warningThreshold {
			status = "warning"
		}

		budgetRows = append(budgetRows, BudgetConsumption{
			Category: b.Category,
			Budget:   b.Amount,
			Spent:    spent,
			Percent:  pct,
			BarWidth: barWidth,
			Status:   status,
		})
	}

	slices.SortFunc(budgetRows, func(a, b BudgetConsumption) int {
		return cmp.Compare(b.Percent, a.Percent) // descending
	})

	return DashboardSummary{
		TotalIncome:       totalIncome,
		TotalExpense:      totalExpense,
		Net:               totalIncome - totalExpense,
		BudgetRows:        budgetRows,
		ExpenseByCategory: expenseCats,
		IncomeByCategory:  incomeCats,
	}
}

func buildCategoryTotals(byCat map[string]float64, total float64) []CategoryTotal {
	cats := make([]CategoryTotal, 0, len(byCat))

	for name, amount := range byCat {
		cats = append(cats, CategoryTotal{Name: name, Amount: amount})
	}

	slices.SortFunc(cats, func(a, b CategoryTotal) int {
		return cmp.Compare(b.Amount, a.Amount) // descending
	})

	for i := range cats {
		if total > 0 {
			cats[i].Percent = cats[i].Amount / total * percentFactor
		}
	}

	return cats
}
