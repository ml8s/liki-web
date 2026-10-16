// boot.ts — update the static app shell before the Vue bundle initializes.
// This preserves the perceived startup state without an inline CSP exception.
const locales = ['zh-Hans', 'zh-Hant', 'en'] as const;

function normalizeLocale(value: unknown): string {
  const normalized = String(value || '').toLowerCase();
  if ((locales as readonly string[]).includes(normalized)) return normalized;
  if (normalized.startsWith('zh')) {
    return normalized.includes('cn') || normalized.includes('hans') || normalized.includes('sg')
      ? 'zh-Hans'
      : 'zh-Hant';
  }
  return normalized.startsWith('en') ? 'en' : 'zh-Hant';
}

const params = new URLSearchParams(location.search);
const cookie = document.cookie.match(/(?:^|;\s*)liki_locale=([^;]+)/);
const locale = normalizeLocale(
  params.get('locale') || (cookie && cookie[1]) || navigator.language,
);
const tail = locale === 'en' ? 'Metaphysics' : '命理';
document.documentElement.lang = locale;
// 启动屏显示加载后 shell 的同一个品牌串；<title> 由 index.html（站点名）与 router 负责。
document.getElementById('app-boot')?.replaceChildren(`Liki·${tail}`);
