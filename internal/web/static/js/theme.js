function applyChartDefaults(isDark) {
  if (typeof Chart === "undefined") return;
  Chart.defaults.color = isDark ? "#94a3b8" : "#64748b";
  Chart.defaults.borderColor = isDark ? "#334155" : "#e2e8f0";
}

function toggleTheme() {
  var isDark = document.documentElement.getAttribute("data-theme") === "dark";
  var next = isDark ? "light" : "dark";
  document.documentElement.setAttribute("data-theme", next);
  localStorage.setItem("theme", next);
  applyChartDefaults(!isDark);
}
