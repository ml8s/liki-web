/* global createOrderAndCheckout, Liki */
// mingshu.js — Lifebook product page form handler.
// Uses shared createOrderAndCheckout from api.js.

(function(){
  var loading = false;
  var form = document.getElementById('mingshu-form');
  var btn = document.getElementById('form-submit');
  var errEl = document.getElementById('form-error');

  if (!form || !btn || !errEl) return;

  form.addEventListener('submit', async function(e){
    e.preventDefault();
    if (loading) return;

    var name = document.getElementById('lb-name').value.trim();
    var dt = document.getElementById('lb-birth').value;
    if (!dt) { Liki.showErr(errEl, Liki.t('mingshu.form.error.birth')); return; }
    var parts = dt.split('T')[0].split('-');
    var timeParts = (dt.split('T')[1] || '12:00').split(':');
    var city = document.getElementById('lb-city').value.trim();
    var genderEl = form.querySelector('input[name="lb-gender"]:checked');
    var email = document.getElementById('lb-email').value.trim();

    if (!name || !dt || !city || !email) {
      Liki.showErr(errEl, Liki.t('mingshu.form.error.request'));
      return;
    }

    loading = true;
    btn.disabled = true;
    btn.classList.add('btn-loading');
    errEl.classList.add('hidden');

    try {
      await createOrderAndCheckout('/api/mingshu/orders',
        {
          birth: {
            name: name,
            year: parseInt(parts[0], 10),
            month: parseInt(parts[1], 10),
            day: parseInt(parts[2], 10),
            hour: parseInt(timeParts[0], 10),
            minute: timeParts.length > 1 ? parseInt(timeParts[1], 10) : 0,
            city: city,
            gender: genderEl ? genderEl.value : 'male'
          },
          email: email,
          currency: window.Liki.currency,
          lang: i18next.isInitialized ? i18next.language : 'zh-Hans'
        },
        'mingshu',
        { email: email });
      return;
    } catch(e) {
      Liki.showErr(errEl, e.message);
      loading = false;
      btn.disabled = false;
      btn.classList.remove('btn-loading');
    }
  });
})();
