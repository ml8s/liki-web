/* global apiPost, escapeHTML */
// payment.js — Generic checkout flow with QR code support.
// Provides product-agnostic payment tools for all pages.
// Uses apiPost from api.js and i18next for translations.
// Products use: POST /api/{product}/checkout → { qrcode_url?, checkout_url }

// ── Checkout: POST /api/{product}/checkout ──
async function makeCheckout(product, payload) {
  var data = await apiPost('/' + product + '/checkout', payload);
  if (!data) throw new Error(i18next.t('error.noCheckoutUrl'));
  return data;
}

// ── QR Modal (native <dialog>) ──
function showQRModal(product, orderID, qrcodeUrl, fallbackUrl) {
  var existing = document.querySelector('.qr-modal');
  if (existing) existing.remove();

  var dialog = document.createElement('dialog');
  dialog.className = 'qr-modal';
  dialog.innerHTML =
    '<form method="dialog">' +
      '<button class="qr-modal-close" aria-label="' + escapeHTML(i18next.t('payment.qrClose')) + '">&times;</button>' +
    '</form>' +
    '<p class="qr-modal-title">' + escapeHTML(i18next.t('payment.scanQR')) + '</p>' +
    '<img class="qr-modal-img" alt="QR Code">' +
    '<p class="qr-modal-hint" data-qr-hint>' + escapeHTML(i18next.t('payment.qrHint')) + '</p>';

  var img = dialog.querySelector('.qr-modal-img');
  img.src = qrcodeUrl;
  if (fallbackUrl) {
    img.onerror = function() { this.onerror = null; location.href = fallbackUrl; };
  }

  var hintEl = dialog.querySelector('[data-qr-hint]');
  var pollTimer = null;
  var stopped = false;
  var pollCount = 0;
  var MAX_POLLS = 100;

  function stop() {
    stopped = true;
    if (pollTimer) { clearTimeout(pollTimer); pollTimer = null; }
  }

  dialog.addEventListener('close', function() { stop(); });

  function poll() {
    if (stopped) return;
    pollCount++;
    if (pollCount > MAX_POLLS) { stop(); return; }
    hintEl.textContent = i18next.t('payment.qrChecking');
    fetch('/api/' + product + '/orders/' + encodeURIComponent(orderID) + '/status')
      .then(function(r) {
        if (!r.ok && r.status === 401) { stop(); dialog.close(); return; }
        return r.json();
      })
      .then(function(d) {
        if (!d || stopped) return;
        if (d.data && d.data.status === 'paid') {
          hintEl.textContent = i18next.t('payment.qrPaid');
          setTimeout(function() {
            dialog.close();
            document.dispatchEvent(new CustomEvent('payment:paid', { detail: { orderID: orderID } }));
          }, 800);
        } else {
          pollTimer = setTimeout(poll, 3000);
        }
      })
      .catch(function() {
        if (!stopped) pollTimer = setTimeout(poll, 3000);
      });
  }

  document.body.appendChild(dialog);
  if (typeof dialog.showModal !== 'function') {
    if (fallbackUrl) location.href = fallbackUrl;
    return;
  }
  dialog.showModal();
  pollTimer = setTimeout(poll, 3000);
}

// ── Go through checkout
// ── Go through checkout: create QR or redirect ──
async function goCheckout(product, orderID, extra) {
  var payload = { order_id: orderID };
  if (extra) Object.assign(payload, extra);
  if (!payload.provider) {
    payload.provider = window.Liki.currency === 'CNY' ? 'xunhu' : 'dodo';
  }
  var data = await makeCheckout(product, payload);
  if (!data) throw new Error(i18next.t('error.noCheckoutUrl'));

  if (data.qrcode_url) {
    showQRModal(product, orderID, data.qrcode_url, data.checkout_url);
    return { mode: 'qr' };
  }
  var url = data.checkout_url;
  if (!url) throw new Error(i18next.t('error.noCheckoutUrl'));
  window.location.href = url;
  return { mode: 'redirect' };
}

// ── namespace extension ──
window.Liki = window.Liki || {};
window.Liki.goCheckout = goCheckout;
window.Liki.showQRModal = showQRModal;
