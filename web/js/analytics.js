// analytics.js — first-party pageview beacon.
// Kept external so the edge CSP does not need script-src 'unsafe-inline'.
(function () {
  'use strict';

  try {
    fetch('/api/analytics/pageview', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path: location.pathname }),
    }).catch(function () {});
  } catch {
    // Analytics is non-essential and must never break page rendering.
  }
})();
