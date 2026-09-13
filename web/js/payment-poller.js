// payment-poller.js — reusable, cancellable payment intent polling.
(function (window) {
  'use strict';

  function createIntentStatusPoller(paymentIntentID, options) {
    var opts = Object.assign({ interval: 3000, maxPolls: 200, token: '' }, options || {});
    var stopped = false;
    var polls = 0;
    var timer = null;

    function stop() {
      stopped = true;
      if (timer) clearTimeout(timer);
    }

    function fetchOnce() {
      var url = '/api/payments/intents/' + encodeURIComponent(paymentIntentID) + '/status';
      if (opts.token) url += '?status_token=' + encodeURIComponent(opts.token);
      return fetch(url).then(function (resp) {
        if (!resp.ok) throw new Error('status ' + resp.status);
        return resp.json();
      }).then(function (envelope) { return envelope && envelope.data; });
    }

    function poll(onUpdate, onError) {
      if (stopped) return Promise.resolve({ stopped: true });
      polls += 1;
      if (polls > opts.maxPolls) {
        stop();
        return Promise.reject(new Error('timeout'));
      }
      return fetchOnce().then(function (intent) {
        if (stopped || !intent) return intent;
        if (onUpdate) onUpdate(intent);
        if (intent.status === 'paid' || intent.status === 'failed' || intent.status === 'expired') return intent;
        return new Promise(function (resolve) {
          timer = setTimeout(function () { resolve(poll(onUpdate, onError)); }, opts.interval);
        });
      });
    }

    return {
      stop: stop,
      start: function (onUpdate, onError) {
        return poll(onUpdate, onError).catch(function (err) {
          stop();
          if (onError) onError(err);
          return { error: err };
        });
      }
    };
  }

  function createFulfillmentStatusPoller(paymentIntentID, options) {
    var opts = Object.assign({ interval: 3000, maxPolls: 300, token: '' }, options || {});
    var stopped = false;
    var polls = 0;
    var timer = null;

    function stop() {
      stopped = true;
      if (timer) clearTimeout(timer);
    }

    function fetchOnce() {
      var url = '/api/payments/intents/' + encodeURIComponent(paymentIntentID) + '/fulfillment-status';
      if (opts.token) url += '?status_token=' + encodeURIComponent(opts.token);
      return fetch(url).then(function (resp) {
        if (!resp.ok) throw new Error('status ' + resp.status);
        return resp.json();
      }).then(function (envelope) { return envelope && envelope.data; });
    }

    return {
      stop: stop,
      start: function (onStatus) {
        return new Promise(function (resolve, reject) {
          function tick() {
            if (stopped) return;
            polls += 1;
            if (polls > opts.maxPolls) {
              stop();
              reject(new Error('timeout'));
              return;
            }
            fetchOnce().then(function (status) {
              if (stopped) return;
              if (onStatus) onStatus(status);
              if (status.fulfillment_status === 'succeeded' || status.fulfillment_status === 'failed') {
                stop();
                resolve(status);
                return;
              }
              timer = setTimeout(tick, opts.interval);
            }).catch(function (err) {
              stop();
              reject(err);
            });
          }
          tick();
        });
      }
    };
  }

  window.LikiPayment = window.LikiPayment || {};
  window.LikiPayment.createIntentStatusPoller = createIntentStatusPoller;
  window.LikiPayment.createFulfillmentStatusPoller = createFulfillmentStatusPoller;
})(window);
