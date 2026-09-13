/* global escapeHTML */
// payment.js — shared provider checkout UI and intent polling.

function showIntentQRModal(paymentIntentID, qrURL, fallbackURL, successURL, statusToken) {
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
  img.src = qrURL;
  if (fallbackURL) {
    img.onerror = function() { this.onerror = null; location.href = fallbackURL; };
  }

  var hintEl = dialog.querySelector('[data-qr-hint]');
  var poller = window.LikiPayment.createIntentStatusPoller(paymentIntentID, {
    token: statusToken || '',
    interval: 3000,
    maxPolls: 200
  });

  function closePaid() {
    hintEl.textContent = i18next.t('payment.qrPaid');
    setTimeout(function() {
      poller.stop();
      dialog.close();
      if (successURL) window.location.href = successURL;
      document.dispatchEvent(new CustomEvent('payment:paid'));
    }, 700);
  }

  dialog.addEventListener('close', function() { poller.stop(); });

  poller.start(function(intent) {
    if (intent.status === 'paid') closePaid();
    else hintEl.textContent = i18next.t('payment.qrChecking');
  }, function() {
    hintEl.textContent = i18next.t('error.requestFailed');
  });

  document.body.appendChild(dialog);
  if (typeof dialog.showModal !== 'function') {
    if (fallbackURL) location.href = fallbackURL;
    return;
  }
  dialog.showModal();
}

async function startIntentCheckout(paymentIntentID, checkout, successURL) {
  if (checkout.qr_url) {
    showIntentQRModal(paymentIntentID, checkout.qr_url, checkout.checkout_url, successURL, checkout.status_token);
    return { mode: 'qr' };
  }
  if (!checkout.checkout_url) throw new Error(i18next.t('error.noCheckoutUrl'));
  location.href = checkout.checkout_url;
  return { mode: 'redirect' };
}

window.Liki = window.Liki || {};
Object.assign(window.Liki, { startIntentCheckout: startIntentCheckout, showIntentQRModal: showIntentQRModal });
