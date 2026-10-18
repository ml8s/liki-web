// Locale dropdown for statically generated marketing pages. The switcher is a
// native <details> element and its menu is a list of real links, so switching
// works without JS; this script only mirrors the disclosure state for assistive
// tech, closes on outside click / Escape, and remembers the choice in a cookie.
const details = document.querySelector<HTMLDetailsElement>('#lang-switcher');
const toggle = document.querySelector<HTMLElement>('#lang-toggle');
const dropdown = document.querySelector<HTMLElement>('#lang-dropdown');

if (details && toggle && dropdown) {
  const sync = () => {
    toggle.setAttribute('aria-expanded', details.open ? 'true' : 'false');
  };
  sync();
  details.addEventListener('toggle', sync);

  const close = () => {
    details.open = false;
    toggle.focus();
  };

  const rememberLang = (locale: string) => {
    const secure = location.protocol === 'https:' ? '; Secure' : '';
    document.cookie = `liki_locale=${encodeURIComponent(locale)}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`;
  };

  dropdown.querySelectorAll<HTMLAnchorElement>('a[data-lang]').forEach((link) => {
    link.addEventListener('click', () => {
      const locale = link.dataset.lang;
      if (locale) rememberLang(locale);
    });

    link.addEventListener('keydown', (event) => {
      if (event.key === 'Escape') close();
    });
  });

  document.addEventListener('click', (event) => {
    if (details.open && !details.contains(event.target as Node)) {
      details.open = false;
    }
  });

  toggle.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && details.open) close();
  });
}
