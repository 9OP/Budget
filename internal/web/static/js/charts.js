function initChart(data, doughnutId) {
  if (!data || !data.labels || data.labels.length === 0) return;

  const ctx = document.getElementById(doughnutId);
  if (ctx) {
    new Chart(ctx, {
      type: 'doughnut',
      data: {
        labels: data.labels,
        datasets: [{ data: data.amounts, backgroundColor: data.colors }],
      },
      options: {
        responsive: false,
        plugins: { legend: { display: false } },
      },
    });
  }
}
