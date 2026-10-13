// First-party pageview beacon. Analytics must never block or break rendering.
try {
  void fetch('/api/analytics/pageview', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path: location.pathname }),
  }).catch(() => {});
} catch {
  // Private browsing or a disabled API must not affect the page.
}
