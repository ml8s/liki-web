/* global createOrderAndCheckout, Liki */
// purchase.js — Naming product page form handler.
// Uses shared createOrderAndCheckout from api.js, currency from window.Liki.currency.
(function(){
  var loading = false;
  var form = document.getElementById('purchase-form');
  var emailEl = document.getElementById('purchase-email');
  var btn = document.getElementById('purchase-btn');
  var errEl = document.getElementById('purchase-error');

  if (!form || !emailEl || !btn) return;

  // QR payment success: redirect to chat
  document.addEventListener('payment:paid', function(e) {
    var lang = (i18next.language || 'zh-Hant');
    location.href = '/' + lang + '/chat?order_id=' + encodeURIComponent(e.detail.orderID);
  });

  form.addEventListener('submit', async function(){
    if (loading) return;
    var email = emailEl.value.trim();
    if (!email) { Liki.showErr(errEl, Liki.t('naming.cta.errorEmail')); return; }

    loading = true;
    btn.disabled = true;
    errEl.classList.add('hidden');

    try {
      await createOrderAndCheckout('/api/naming/orders',
        { email: email, currency: window.Liki.currency, lang: i18next.isInitialized ? i18next.language : 'zh-Hant' },
        'naming',
        { email: email });
      return;
    } catch(e) {
      Liki.showErr(errEl, e.message);
      loading = false;
      btn.disabled = false;
    }
  });

})();
