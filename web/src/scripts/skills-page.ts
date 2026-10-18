// Install-command copy actions on the public skills page.
document.querySelectorAll<HTMLButtonElement>('.copy-btn').forEach((button) => {
  button.addEventListener('click', () => {
    const command = button.dataset.cmd;
    if (!command || !navigator.clipboard) return;

    navigator.clipboard.writeText(command).then(() => {
      const status = button.closest('.install-block')?.querySelector('[role="status"]');
      if (status) status.textContent = button.dataset.copiedLabel ?? '';

      const original = button.innerHTML;
      button.innerHTML = '<span class="text-xs">✓</span>';
      setTimeout(() => {
        button.innerHTML = original;
      }, 1500);
    }).catch(() => {});
  });
});
