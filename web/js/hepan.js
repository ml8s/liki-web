/* global createOrderAndCheckout, Liki */
// Hepan (八紫合盘) — Dual person compatibility report
(function () {
  'use strict';

  var buyBtn = document.getElementById('hp-buy');
  var errEl = document.getElementById('form-error');
  if (!buyBtn || !errEl) return;

  var loading = false;

  function getVal(id) { var el = document.getElementById(id); return el ? el.value.trim() : ''; }

  function getGender(prefix) {
    var male = document.getElementById('hp-' + prefix + '-male');
    if (male && male.checked) return 'male';
    return 'female';
  }

  function formatPerson(prefix) {
    var dt = getVal('hp-' + prefix + '-birth');
    var parts = dt ? dt.split('T') : ['', ''];
    var dateParts = (parts[0] || '').split('-');
    var timeParts = (parts[1] || '12:00').split(':');
    return {
      name: getVal('hp-' + prefix + '-name'),
      birth: dt,
      year: parseInt(dateParts[0]) || 0,
      month: parseInt(dateParts[1]) || 0,
      day: parseInt(dateParts[2]) || 0,
      hour: parseInt(timeParts[0]) || 12,
      minute: parseInt(timeParts[1]) || 0,
      city: getVal('hp-' + prefix + '-city'),
      gender: getGender(prefix)
    };
  }

  buyBtn.addEventListener('click', async function (e) {
    e.preventDefault();
    if (loading) return;
    errEl.classList.add('hidden');

    var personA = formatPerson('a');
    var personB = formatPerson('b');
    var email = getVal('hp-email');

    if (!personA.name || !personB.name) { Liki.showErr(errEl, Liki.t('hepan.form.error.name') || '请填写姓名'); return; }
    if (!personA.birth || !personB.birth) { Liki.showErr(errEl, Liki.t('hepan.form.error.birth') || '请填写出生日期'); return; }
    if (!email) { Liki.showErr(errEl, Liki.t('hepan.form.error.email') || '请填写邮箱'); return; }

    loading = true;
    buyBtn.disabled = true;
    buyBtn.textContent = Liki.t('loading') || '处理中...';

    try {
      await createOrderAndCheckout('/api/hepan/orders',
        {
          person_a: personA,
          person_b: personB,
          email: email,
          currency: window.Liki.currency,
          lang: i18next ? i18next.language : 'zh-Hans'
        },
        'hepan',
        { email: email });
    } catch (e) {
      Liki.showErr(errEl, e.message || Liki.t('err.network') || '网络错误');
    } finally {
      loading = false;
      buyBtn.disabled = false;
      buyBtn.textContent = Liki.t('hepan.cta.button') || '购买报告';
    }
  });

  // ── payment:paid listener → checkout（支付后跳转到 /hepan/report）──
  document.addEventListener('payment:paid', function (e) {
    var oid = e.detail && (e.detail.orderID || e.detail);
    if (!oid) return;
    goCheckout('hepan', oid); // eslint-disable-line no-undef
  });
})();

