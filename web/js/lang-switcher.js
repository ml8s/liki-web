// lang-switcher.js — locale dropdown for statically generated pages.
(function () {
  'use strict';

  var toggle = document.getElementById('lang-toggle');
  var dropdown = document.getElementById('lang-dropdown');
  if (!toggle || !dropdown) return;

  function open() {
    dropdown.classList.remove('hidden');
    toggle.setAttribute('aria-expanded', 'true');
  }

  function close() {
    dropdown.classList.add('hidden');
    toggle.setAttribute('aria-expanded', 'false');
  }

  function getLang() {
    return (location.pathname.match(/^\/(zh-Hans|zh-Hant|en)\//) || [])[1] || 'zh-Hant';
  }

  function rememberLang(locale) {
    document.cookie = 'liki_locale=' + encodeURIComponent(locale) +
      '; Path=/; Max-Age=31536000; SameSite=Lax';
  }

  window.Liki = window.Liki || {};
  window.Liki.localeFromPath = getLang;

  toggle.addEventListener('click', function (event) {
    event.preventDefault();
    event.stopPropagation();
    dropdown.classList.contains('hidden') ? open() : close();
  });

  dropdown.querySelectorAll('[data-lang]').forEach(function (button) {
    button.addEventListener('click', function (event) {
      event.preventDefault();
      var locale = this.getAttribute('data-lang');
      rememberLang(locale);
      var path = location.pathname.replace(/^\/(zh-Hans|zh-Hant|en)\/?/, '/');
      location.href = '/' + locale + (path === '/' ? '/' : path);
    });
  });

  document.addEventListener('click', function (event) {
    var switcher = document.getElementById('lang-switcher');
    if (switcher && !switcher.contains(event.target)) close();
  });

  toggle.addEventListener('keydown', function (event) {
    if (event.key === 'Escape') {
      close();
      toggle.focus();
    }
  });

  dropdown.querySelectorAll('[data-lang]').forEach(function (button) {
    button.addEventListener('keydown', function (event) {
      if (event.key === 'Escape') {
        close();
        button.focus();
      }
    });
  });
})();
