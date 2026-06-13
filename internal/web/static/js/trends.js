var trendsChart = null;
var trendsData = null;

function initTrends(data) {
  trendsData = data;
  renderTrends('monthly');
}

function setTrendsView(view) {
  document.getElementById('btn-monthly').className    = view === 'monthly'    ? 'btn btn-primary' : 'btn btn-outline';
  document.getElementById('btn-cumulative').className = view === 'cumulative' ? 'btn btn-primary' : 'btn btn-outline';
  renderTrends(view);
}

function renderTrends(view) {
  var d = trendsData;
  var cum = view === 'cumulative';

  var datasets = [
    {
      label: 'Income',
      data: cum ? d.cum_income : d.income,
      borderColor: '#16a34a',
      backgroundColor: 'rgba(22,163,74,0.08)',
      tension: 0.3,
      fill: false,
    },
    {
      label: 'Expenses',
      data: cum ? d.cum_expense : d.expense,
      borderColor: '#dc2626',
      backgroundColor: 'rgba(220,38,38,0.08)',
      tension: 0.3,
      fill: false,
    },
    {
      label: 'Investments',
      data: cum ? d.cum_investment : d.investment,
      borderColor: '#f59e0b',
      backgroundColor: 'rgba(245,158,11,0.08)',
      tension: 0.3,
      fill: false,
    },
    {
      label: 'Net',
      data: cum ? d.cum_net : d.net,
      borderColor: '#3b82f6',
      backgroundColor: 'rgba(59,130,246,0.08)',
      tension: 0.3,
      fill: false,
    },
  ];

  if (trendsChart) {
    trendsChart.data.datasets = datasets;
    trendsChart.update();

    return;
  }

  var ctx = document.getElementById('trends-chart');
  if (!ctx) return;

  trendsChart = new Chart(ctx, {
    type: 'line',
    data: { labels: d.labels, datasets: datasets },
    options: {
      responsive: true,
      interaction: { mode: 'index', intersect: false },
      plugins: { legend: { position: 'top' } },
      scales: { y: { beginAtZero: false } },
    },
  });
}
