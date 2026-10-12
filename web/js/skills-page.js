// skills-page.js — install-command copy actions.
(function () {
  'use strict';

  document.querySelectorAll('.copy-btn').forEach(function (button) {
    button.addEventListener('click', function () {
      var command = button.getAttribute('data-cmd');
      if (!command || !navigator.clipboard || !navigator.clipboard.writeText) return;

      navigator.clipboard.writeText(command).then(function () {
        var original = button.innerHTML;
        button.innerHTML = '<span class="text-xs">✓</span>';
        setTimeout(function () {
          button.innerHTML = original;
        }, 1500);
      }).catch(function () {});
    });
  });
})();
