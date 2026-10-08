// skills-page.js — decorative keyword field and install-command copy actions.
(function () {
  'use strict';

  document.querySelectorAll('#kw-bg span').forEach(function (item) {
    item.style.cssText = [
      'position:absolute',
      'left:' + (Math.random() * 85 + 7) + '%',
      'top:' + (Math.random() * 90 + 5) + '%',
      'font-size:' + (12 + Math.random() * 14) + 'px',
      'color:#16775d',
      'opacity:0.06',
      'white-space:nowrap',
      'pointer-events:none',
      'user-select:none',
      'transform:translateX(-50%) rotate(' + (Math.random() * 60 - 30) + 'deg)',
    ].join(';');
  });

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
