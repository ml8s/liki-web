// paipan-hook.js — save a free-chart draft and enter onboarding.
(function () {
  'use strict';

  var form = document.getElementById('paipanHook');
  if (!form) return;

  form.addEventListener('submit', function (event) {
    event.preventDefault();
    var date = form.date.value;
    var time = form.time.value;
    var gender = form.gender.value;
    if (!date || !time || !gender) return;

    try {
      sessionStorage.setItem('liki-draft', JSON.stringify({
        birth_date: date,
        birth_time: time,
        gender: gender,
        calendar: 'solar',
      }));
    } catch {
      // Private browsing can reject storage; still continue to onboarding.
    }

    location.href = '/app/profiles/new';
  });
})();
