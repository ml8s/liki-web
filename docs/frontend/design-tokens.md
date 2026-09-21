# Design Tokens

## Canonical source

`web/css/tokens.css` is the single source of truth for shared brand values.

Every surface must consume the same semantic tokens:

```css
var(--brand-primary)
var(--brand-primary-hover)
var(--brand-primary-soft)
var(--brand-gold)
var(--brand-paper)
var(--brand-surface)
var(--brand-text)
var(--brand-text-muted)
var(--brand-line)
```

Do not introduce a second product palette.

## Surfaces

| Surface | Entry | Styling |
|---|---|---|
| Marketing / legal | Astro `BaseLayout` | `tokens.css` + `app.css` |
| User product | Vue `/app` | imports `tokens.css` + Element Plus aliases |
| KOL Console | `/kol` | `tokens.css` + `app.css` + `console.css` |
| Scout Console | `/scout` | `tokens.css` + `app.css` + `console.css` |
| Admin Console | `/admin.html` | `tokens.css` + `app.css` + `console.css` |

## Brand contract

- Primary: `--brand-primary`
- Primary hover: `--brand-primary-hover`
- Primary tint: `--brand-primary-soft`
- Gold accent: `--brand-gold`
- Gold text: `--brand-gold-text`
- Ink / body text: `--brand-text`
- Secondary text: `--brand-text-muted`
- Page background: `--brand-paper`
- Card background: `--brand-surface`
- Border: `--brand-line`

## Runtime modules

Console pages share `web/js/console-core.js` for stateless utilities, email-code auth, and API plumbing.
Page behavior lives in `console-kol.js`, `console-scout.js`, and `console-admin.js`.
These files are copied byte-for-byte into `web/public/js`; neither partner surface imports the other.

## Performance & mobile contract

Console pages do not load `app.css`. They load only:

1. `tokens.css`
2. `console.css`

`tokens.css` owns the shared primitive tokens; `console.css` owns console
components and the mobile breakpoint. Console scripts use `defer`.

Mobile rules:

- viewport must include `width=device-width` and `initial-scale=1`
- independent consoles may add `viewport-fit=cover`
- primary breakpoint is `640px`
- touch targets stay at least 40px, preferably 44px on coarse pointers
- tables scroll horizontally inside a bordered container
- dialogs and forms span the available width on small screens
- reduced motion is always respected

## Rules

1. New UI must use semantic CSS variables.
2. Do not hardcode brand hex values in components.
3. Do not invent a second success, warning, or primary color.
4. Cards use `12px` radius; controls use `8px`; small tags use `6px`.
5. Tables and numeric data use `font-variant-numeric: tabular-nums`.
6. Legal text uses the same page card, type scale, and spacing as marketing content.
