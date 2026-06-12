document.addEventListener('toast', function (e) {
  showToast(e.detail.message, e.detail.type || 'success');
});

function showToast(message, type) {
  var container = document.getElementById('toast-container');
  if (!container) return;

  var toast = document.createElement('div');
  toast.className = 'toast toast-' + type;
  toast.setAttribute('role', 'status');
  toast.setAttribute('aria-live', 'polite');
  toast.textContent = message;

  container.appendChild(toast);

  requestAnimationFrame(function () {
    requestAnimationFrame(function () {
      toast.classList.add('toast-visible');
    });
  });

  setTimeout(function () {
    toast.classList.remove('toast-visible');
    toast.addEventListener('transitionend', function () { toast.remove(); }, { once: true });
  }, 3000);
}
