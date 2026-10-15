// Locale dropdown for statically generated marketing pages.
const toggle = document.querySelector<HTMLButtonElement>('#lang-toggle');
const dropdown = document.querySelector<HTMLElement>('#lang-dropdown');
const switcher = document.querySelector<HTMLElement>('#lang-switcher');

if (toggle && dropdown && switcher) {
  const open = () => {
    dropdown.classList.remove('hidden');
    toggle.setAttribute('aria-expanded', 'true');
  };

  const close = () => {
    dropdown.classList.add('hidden');
    toggle.setAttribute('aria-expanded', 'false');
  };

  const rememberLang = (locale: string) => {
    const secure = location.protocol === 'https:' ? '; Secure' : '';
    document.cookie = `liki_locale=${encodeURIComponent(locale)}; Path=/; Max-Age=31536000; SameSite=Lax${secure}`;
  };

  toggle.addEventListener('click', (event) => {
    event.preventDefault();
    event.stopPropagation();
    if (dropdown.classList.contains('hidden')) open();
    else close();
  });

  dropdown.querySelectorAll<HTMLButtonElement>('[data-lang]').forEach((button) => {
    button.addEventListener('click', (event) => {
      event.preventDefault();
      const locale = button.dataset.lang;
      if (!locale) return;

      rememberLang(locale);
      const path = location.pathname.replace(/^\/(zh-Hans|zh-Hant|en)\/?/, '/');
      location.href = `/${locale}${path === '/' ? '/' : path}`;
    });

    button.addEventListener('keydown', (event) => {
      if (event.key === 'Escape') {
        close();
        button.focus();
      }
    });
  });

  document.addEventListener('click', (event) => {
    if (!switcher.contains(event.target as Node)) close();
  });

  toggle.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') {
      close();
      toggle.focus();
    }
  });
}
