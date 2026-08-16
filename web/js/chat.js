(function(){
var API_BASE = '/api';
var t = window.Liki.t;
var locale = window.Liki.locale || 'zh-Hant';
var esc = function(s) { var d = document.createElement('div'); d.textContent = s; return d.innerHTML; };
var renderMD = function(s) { try { return DOMPurify.sanitize(marked.parse(s), { ALLOWED_TAGS: ['h2','h3','h4','p','br','strong','em','ul','ol','li','code','pre','blockquote','table','thead','tbody','tr','th','td','hr'], ALLOWED_ATTR: [] }); } catch { return esc(s); } };

var SSE_EVT = { THINKING: 'thinking', PHASE: 'phase', THINKING_DELTA: 'thinking-delta', TEXT_DELTA: 'text-delta', ERROR: 'error' };
var DRAFT_KEY = 'likiChatDraft';
var MAX_RETRIES = 3;

// ── DOM refs ──
var $ = function(id) { return document.getElementById(id); };
var viewLogin = $('view-login');
var viewSelect = $('view-select');
var viewChat = $('view-chat');
var loginEmail = $('login-email');
var btnLogin = $('btn-login');
var loginBtnText = $('login-btn-text');
var loginLoading = $('login-loading');
var loginError = $('login-error');
var orderList = $('order-list');
var chatCountdown = $('chat-countdown');
var btnReset = $('btn-reset');
var chatMessages = $('chat-messages');
var errorToast = $('error-toast');
var errorToastMsg = $('error-toast-msg');
var chatInputBar = $('chat-input-bar');
var chatInput = $('chat-input');
var btnSend = $('btn-send');
var btnStop = $('btn-stop');
var expiredBanner = $('expired-banner');
var errorToastRetry = $('error-toast-retry');

// ── State ──
var orderID = '';
var email = '';
var loginLoadingState = false;
var orderListData = [];
var messages = [];
var ui = { phase: 'welcome', substate: 'idle', error: '' };
var phaseStatus = '';
var chatExpiresAt = null;
var expired = false;
var pending = false;
var isComposing = false;
var sseCtrl = null;
var renderer = null;

// ── View switching ──
function showView(v) {
  viewLogin.classList.toggle('hidden', v !== 'login');
  viewSelect.classList.toggle('hidden', v !== 'select');
  viewChat.classList.toggle('hidden', v !== 'chat');
  if (v === 'login') updateSendButton();
}

// ── Utilities ──
function formatTime(ts) {
  if (!ts) return '';
  var d = new Date(ts);
  var now = new Date();
  var pad = function(n) { return String(n).padStart(2, '0'); };
  var hm = pad(d.getHours()) + ':' + pad(d.getMinutes());
  if (d.toDateString() === now.toDateString()) return hm;
  return pad(d.getMonth() + 1) + '-' + pad(d.getDate()) + ' ' + hm;
}

function formatExpiry(expiresAt) {
  if (!expiresAt) return '';
  try {
    var d = new Date(expiresAt.replace(' ', 'T') + 'Z');
    var now = new Date();
    var diff = d - now;
    if (diff <= 0) return t('chat.expired');
    var days = Math.floor(diff / 86400000);
    var hours = Math.floor((diff % 86400000) / 3600000);
    if (days > 0) return t('chat.remainingDays').replace('{n}', days);
    return t('chat.remainingHours').replace('{n}', hours);
  } catch { return ''; }
}

function refreshMeta() {
  var len = messages.length;
  if (!len) return;
  messages[len - 1].showTime = true;
  if (len >= 2) messages[len - 2].showTime = messages[len - 2].role !== messages[len - 1].role;
}

function scrollDown(force) {
  var el = chatMessages;
  if (!el) return;
  if (force || el.scrollHeight - el.scrollTop - el.clientHeight < 40) {
    el.scrollTop = el.scrollHeight;
  }
}

function focusInput() {
  chatInput.focus();
}

function saveDraft() {
  try { sessionStorage.setItem(DRAFT_KEY, chatInput.value); } catch {}
}

function clearDraft() {
  try { sessionStorage.removeItem(DRAFT_KEY); } catch {}
}

function showError(msg) {
  ui.error = msg;
  errorToastMsg.textContent = msg;
  errorToast.classList.remove('hidden');
}

function hideError() {
  ui.error = '';
  errorToast.classList.add('hidden');
}

function retryLastMessage() {
  hideError();
  var lastUserText = '';
  for (var i = messages.length - 1; i >= 0; i--) {
    if (messages[i].role === 'user') {
      lastUserText = messages[i].content;
      break;
    }
  }
  if (!lastUserText) return;
  while (messages.length > 0 && messages[messages.length - 1].role !== 'user') {
    messages.pop();
  }
  if (messages.length > 0 && messages[messages.length - 1].role === 'user') {
    messages.pop();
  }
  renderMessages();
  sendMessage(lastUserText);
}

// ── Countdown ──
function tickCountdown() {
  if (!chatExpiresAt) return;
  var txt = formatExpiry(chatExpiresAt);
  chatCountdown.textContent = txt;
  if (txt === t('chat.expired')) {
    expired = true;
    chatCountdown.classList.add('hidden');
    chatInputBar.classList.add('hidden');
    expiredBanner.classList.remove('hidden');
    btnReset.classList.add('hidden');
    return;
  }
  chatCountdown.classList.remove('hidden');
  setTimeout(tickCountdown, 60000);
}

// ── Message rendering ──
var COPY_SVG = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>';

function createRow(m, idx) {
  var row = document.createElement('div');
  if (m.role === 'user') {
    row.className = 'msg-row msg-row-user';
    row.setAttribute('data-role', 'user');
    row.innerHTML = '<div class="msg msg-user">' + esc(m.content) + '</div>' +
      (m.time && m.showTime ? '<div class="msg-time">' + formatTime(m.time) + '</div>' : '');
  } else if (m.html) {
    row.className = 'msg-row msg-row-advisor';
    row.setAttribute('data-role', 'advisor');
    row.innerHTML = '<div class="msg-asst-wrap"><div class="msg msg-asst">' + m.html +
      '<button class="msg-copy" data-msg-idx="' + idx + '" aria-label="' + t('chat.copy') + '">' + COPY_SVG + '</button></div></div>' +
      (m.time && m.showTime ? '<div class="msg-time">' + formatTime(m.time) + '</div>' : '');
    row.querySelector('.msg-copy').addEventListener('click', function() {
      copyMsg(messages[parseInt(this.getAttribute('data-msg-idx'), 10)], this);
    });
  } else {
    row.className = 'msg-row msg-row-advisor';
    row.setAttribute('data-role', 'advisor');
    row.innerHTML = '<div class="msg msg-asst">' +
      (phaseStatus ? '<div class="phase-status">' + esc(phaseStatus) + '</div>' : '') +
      '<div class="typing-dots"><span></span><span></span><span></span></div></div>';
  }
  row._html = m.html;
  row._showTime = m.showTime;
  return row;
}

function syncPhaseStatus(row) {
  var ps = row.querySelector('.phase-status');
  if (phaseStatus) {
    if (ps) ps.textContent = phaseStatus;
    else {
      var td = row.querySelector('.typing-dots');
      if (td) { var el = document.createElement('div'); el.className = 'phase-status'; el.textContent = phaseStatus; td.parentNode.insertBefore(el, td); }
    }
  } else if (ps) {
    ps.remove();
  }
}

function renderMessages() {
  var children = chatMessages.children;
  var childCount = children.length;
  var msgCount = messages.length;
  var minLen = childCount < msgCount ? childCount : msgCount;

  for (var i = 0; i < minLen; i++) {
    var m = messages[i];
    var row = children[i];

    if (row._html === m.html && row._showTime === m.showTime) {
      if (!m.html) syncPhaseStatus(row);
      continue;
    }

    var replaced = false;
    if (!row._html && !m.html && row.getAttribute('data-role') === m.role) {
      syncPhaseStatus(row);
    } else if (!m.html || !row._html) {
      var newRow = createRow(m, i);
      chatMessages.replaceChild(newRow, row);
      replaced = true;
    } else {
      var asst = row.querySelector('.msg-asst');
      if (asst) asst.innerHTML = m.html;
      if (m.showTime !== row._showTime) {
        var timeEl = row.querySelector('.msg-time');
        if (m.showTime && m.time) {
          if (timeEl) timeEl.textContent = formatTime(m.time);
          else row.insertAdjacentHTML('beforeend', '<div class="msg-time">' + formatTime(m.time) + '</div>');
        } else if (timeEl) {
          timeEl.remove();
        }
      }
    }
    if (!replaced) {
      row._html = m.html;
      row._showTime = m.showTime;
    }
  }

  for (var j = childCount; j < msgCount; j++) {
    chatMessages.appendChild(createRow(messages[j], j));
  }

  while (chatMessages.children.length > msgCount) {
    chatMessages.removeChild(chatMessages.lastChild);
  }

  scrollDown(true);
}

// ── Stream renderer ──
function createStreamRenderer(asst) {
  var timer = null;
  var lastRender = 0;
  var lastRenderedLen = 0;

  function thinkingHTML() {
    if (!asst.thinking) return '';
    var open = (!asst.content && !asst._thinkingDismissed) ? ' open' : '';
    return '<details class="thinking-block"' + open + '><summary>' + t('chat.thinking') + '</summary><div>' + esc(asst.thinking) + '</div></details>';
  }

  function doRender(final_) {
    timer = null;
    lastRender = Date.now();

    if (final_) {
      asst.html = thinkingHTML() + renderMD(asst.content);
      lastRenderedLen = asst.content.length;
    } else {
      var newContent = asst.content.slice(lastRenderedLen);
      if (!newContent) {
        if (lastRenderedLen === 0) asst.html = thinkingHTML();
        renderMessages();
        return;
      }
      if (lastRenderedLen === 0) {
        asst.html = thinkingHTML() + renderMD(newContent);
      } else {
        asst.html += renderMD(newContent);
      }
      lastRenderedLen = asst.content.length;
    }
    renderMessages();
  }

  return {
    accept: function() {
      if (timer) return;
      var elapsed = Date.now() - lastRender;
      if (elapsed >= 120) doRender();
      else timer = setTimeout(function() { doRender(); }, 120 - elapsed);
    },
    flush: function() {
      if (timer) { clearTimeout(timer); timer = null; }
      doRender(true);
    },
    cancel: function() {
      if (timer) { clearTimeout(timer); timer = null; }
    }
  };
}

// ── SSE ──
function parseEvents(buf, onEvent) {
  var events = buf.split('\n\n');
  for (var i = 0; i < events.length; i++) {
    var lines = events[i].split('\n');
    for (var j = 0; j < lines.length; j++) {
      var line = lines[j];
      if (line.indexOf('data: ') !== 0) continue;
      var raw = line.slice(6).trim();
      if (!raw || raw === '[DONE]') continue;
      try { onEvent(JSON.parse(raw)); } catch {}
    }
  }
}

function readStream(reader, onEvent) {
  return new Promise(function(resolve, reject) {
    var decoder = new TextDecoder();
    var buf = '';
    function pump() {
      reader.read().then(function(result) {
        if (result.done) {
          buf += decoder.decode();
          parseEvents(buf, onEvent);
          resolve();
          return;
        }
        buf += decoder.decode(result.value, { stream: true });
        var events = buf.split('\n\n');
        buf = events.pop();
        parseEvents(events.join('\n\n'), onEvent);
        pump();
      }).catch(reject);
    }
    pump();
  });
}

function sseSend(url, body, onEvent) {
  return new Promise(function(resolve, reject) {
    var attempt = 0;
    function trySend() {
      sseCtrl = new AbortController();
      fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
        signal: sseCtrl.signal
      }).then(function(resp) {
        if (!resp.ok) {
          return resp.json().catch(function() { return {}; }).then(function(err) {
            var e = new Error((err.error && err.error.message) || t('error.requestFailed'));
            e.code = (err.error && err.error.code) || '';
            e.status = resp.status;
            throw e;
          });
        }
        if (!resp.body) throw new Error('Empty response body');
        return readStream(resp.body.getReader(), onEvent).then(function() { resolve(resp); });
      }).catch(function(err) {
        if (err.name === 'AbortError') { resolve(); return; }
        if (err.status) { reject(err); return; } // HTTP error — don't retry
        if (attempt < MAX_RETRIES) {
          attempt++;
          setTimeout(trySend, Math.pow(2, attempt - 1) * 1000);
        } else {
          reject(err);
        }
      });
    }
    trySend();
  });
}

function sseAbort() {
  if (sseCtrl) { sseCtrl.abort(); sseCtrl = null; }
}

// ── Event handlers ──
function handleSSEEvent(evt, asst, rend) {
  switch (evt.type) {
    case SSE_EVT.THINKING:
      phaseStatus = t('chat.thinkingStatus');
      renderMessages();
      break;
    case SSE_EVT.PHASE:
      phaseStatus = evt.content || '';
      renderMessages();
      break;
    case SSE_EVT.THINKING_DELTA:
      asst.thinking += evt.content;
      if (!asst.content) rend.accept();
      break;
    case SSE_EVT.TEXT_DELTA:
      if (!asst.content) {
        asst._thinkingDismissed = true;
        phaseStatus = '';
      }
      ui.phase = 'streaming';
      asst.content += evt.content;
      rend.accept();
      break;
    case SSE_EVT.ERROR:
      rend.flush();
      phaseStatus = '';
      handleError(evt.content);
      break;
  }
}

function handleError(msg) {
  pending = false;
  showError(msg);
  ui.phase = 'chatting';
  ui.substate = 'idle';
  updateInputUI();
}

function updateInputUI() {
  var loading = ui.substate === 'loading';
  var empty = !chatInput.value.trim();
  btnSend.classList.toggle('hidden', loading);
  btnStop.classList.toggle('hidden', !loading);
  btnSend.disabled = empty || loading;
  chatInput.disabled = loading;
}

// ── Send message ──
function sendMessage(msg) {
  if (pending) return;
  if (isComposing) return;
  if (expired) return;
  var text = (msg || chatInput.value).trim();
  if (!text) return;

  pending = true;
  chatInput.value = '';
  clearDraft();
  hideError();
  ui.phase = 'chatting';
  ui.substate = 'loading';
  updateInputUI();

  messages.push({ role: 'user', content: text, time: new Date().toISOString() });

  var asst = { role: 'advisor', content: '', thinking: '', html: '', time: new Date().toISOString() };
  messages.push(asst);
  refreshMeta();
  renderer = createStreamRenderer(asst);
  renderMessages();

  // First turn only: carry the initial advisor greeting as context so the LLM
  // knows what the on-screen options (①②③) refer to. Frontend sees all
  // messages — if any user message exists, the conversation has started and the
  // greeting is already persisted in history (backend only persists on empty
  // history). Reliable: refresh/reset both reflect real "has user" state.
  var ctx = '';
  var hasUser = false;
  for (var i = 0; i < messages.length; i++) {
    if (messages[i].role === 'user') { hasUser = true; break; }
  }
  if (!hasUser && messages.length > 0 && messages[0].role === 'advisor' && messages[0].content) {
    ctx = messages[0].content;
  }
  sseSend(API_BASE + '/naming/agent/chat', { message: text, lang: locale, context: ctx }, function(evt) {
    handleSSEEvent(evt, asst, renderer);
  }).then(function() {
    pending = false;
    if (renderer) renderer.flush();
    if (ui.phase === 'streaming') {
      ui.phase = 'chatting';
      ui.substate = 'idle';
      phaseStatus = '';
      updateInputUI();
      focusInput();
      renderMessages();
    }
    renderer = null;
  }).catch(function(err) {
    if (err.name === 'AbortError') return;
    pending = false;
    if (err.status === 401) {
      showView('login');
      setLoginError(t('chat.sessionExpired'));
      return;
    }
    if (renderer) renderer.flush();
    handleError(err.message);
    if (!asst.content && !asst.thinking && !asst.html) messages.pop();
    renderMessages();
    renderer = null;
  });
}

function stopStream() {
  sseAbort();
  if (renderer) renderer.flush();
  phaseStatus = '';
  ui.phase = 'chatting';
  ui.substate = 'idle';
  pending = false;
  renderMessages();
  updateInputUI();
  focusInput();
}

// ── Greeting ──
function greet() {
  var greeting = t('chat.greeting');
  messages.push({ role: 'advisor', content: greeting, html: renderMD(greeting), time: new Date().toISOString() });
  refreshMeta();
  renderMessages();
}

// ── Reset ──
function reset() {
  sseAbort();
  if (renderer) { renderer.cancel(); renderer = null; }
  fetch('/api/naming/orders/' + orderID + '/messages', { method: 'DELETE', credentials: 'include' }).catch(function(){});
  messages = [];
  ui.phase = 'welcome';
  ui.substate = 'idle';
  ui.error = '';
  phaseStatus = '';
  pending = false;
  chatInput.value = ''; clearDraft(); focusInput();
  updateInputUI();
  greet();
}

// ── Copy ──
function copyMsg(m, btn) {
  var done = function() {
    if (btn) {
      var orig = btn.innerHTML;
      btn.innerHTML = '<span style="font-size:12px">✓</span>';
      setTimeout(function() { btn.innerHTML = orig; }, 1200);
    }
  };
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(m.content).then(done).catch(function() {
      fallbackCopy(m.content, done);
    });
  } else {
    fallbackCopy(m.content, done);
  }
}

// 降级：navigator.clipboard 不可用（非安全上下文/旧 Safari）时走 execCommand
function fallbackCopy(text, done) {
  var ta = document.createElement('textarea');
  ta.value = text;
  ta.setAttribute('readonly', '');
  ta.style.position = 'fixed';
  ta.style.top = '-9999px';
  document.body.appendChild(ta);
  ta.select();
  var ok = false;
  try { ok = document.execCommand('copy'); } catch {}
  document.body.removeChild(ta);
  if (ok && done) done();
}

// ── Login ──
function setLoginError(msg) {
  loginError.textContent = msg;
  loginError.classList.toggle('hidden', !msg);
  if (msg) {
    loginError.closest('.login-card').classList.add('has-error');
  } else {
    loginError.closest('.login-card').classList.remove('has-error');
  }
}

function setLoginLoading(loading) {
  loginLoadingState = loading;
  loginBtnText.classList.toggle('hidden', loading);
  loginLoading.classList.toggle('hidden', !loading);
  btnLogin.disabled = loading || !loginEmail.value;
}

function doLogin() {
  if (!loginEmail.value || loginLoadingState) return;
  setLoginLoading(true);
  setLoginError('');

  fetch('/api/naming/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: loginEmail.value })
  }).then(function(resp) {
    return resp.json().then(function(data) {
      if (!resp.ok) throw new Error((data.error && data.error.message) || t('error.requestFailed'));
      return data.data;
    });
  }).then(function(result) {
    setLoginLoading(false);
    email = loginEmail.value;
    if (result.orders) {
      orderListData = result.orders;
      renderOrderList();
      showView('select');
    } else if (result.order_id) {
      orderID = result.order_id;
      try { sessionStorage.setItem('likiOrderID', result.order_id); } catch {}
      enterChat();
    }
  }).catch(function(err) {
    setLoginLoading(false);
    setLoginError(err.message);
  });
}

// ── Order list ──
function renderOrderList() {
  var html = '';
  for (var i = 0; i < orderListData.length; i++) {
    var o = orderListData[i];
    var delay = i * 80;
    html += '<div class="order-card" role="button" tabindex="0" data-order-id="' + o.order_id + '" style="animation-delay: ' + delay + 'ms">' +
      '<div class="flex justify-between items-center">' +
      '<div class="font-medium text-stone-800">' + esc(o.summary) + '</div>' +
      '<span class="text-xs text-stone-400">' + formatExpiry(o.expires_at) + '</span>' +
      '</div></div>';
  }
  orderList.innerHTML = html;

  var cards = orderList.querySelectorAll('.order-card');
  for (var j = 0; j < cards.length; j++) {
    cards[j].addEventListener('click', function() {
      selectOrder(this.getAttribute('data-order-id'));
    });
    cards[j].addEventListener('keydown', function(e) {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        selectOrder(this.getAttribute('data-order-id'));
      }
    });
  }
}

function selectOrder(oid) {
  fetch('/api/naming/orders/select', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ order_id: oid, email: email })
  }).then(function(resp) {
    return resp.json().then(function(data) {
      if (!resp.ok) throw new Error((data.error && data.error.message) || t('error.requestFailed'));
      return data;
    });
  }).then(function() {
    orderID = oid;
    try { sessionStorage.setItem('likiOrderID', oid); } catch {}
    enterChat();
  }).catch(function(err) {
    setLoginError(err.message);
    showView('login');
  });
}

// ── Enter chat ──
function enterChat(prefetched) {
  var p;
  if (prefetched) {
    p = Promise.resolve(prefetched);
  } else {
    p = fetch('/api/naming/orders/' + orderID + '/status').then(function(resp) {
      return resp.json().then(function(data) {
        if (!resp.ok) throw new Error('order not found');
        return data.data;
      });
    });
  }

  p.then(function(o) {
    if (o.status !== 'paid') {
      setLoginError(t('chat.orderNotPaid'));
      showView('login');
      return;
    }

    chatExpiresAt = o.chat_expires_at || null;
    var cd = formatExpiry(o.chat_expires_at);
    if (cd === t('chat.expired')) expired = true;

    showView('chat');
    if (expired) {
      expiredBanner.classList.remove('hidden');
      chatInputBar.classList.add('hidden');
    } else {
      expiredBanner.classList.add('hidden');
      chatInputBar.classList.remove('hidden');
      btnReset.classList.remove('hidden');
      tickCountdown();
    }

    try { var d = sessionStorage.getItem(DRAFT_KEY); if (d) chatInput.value = d; } catch {}
    focusInput();
    updateInputUI();

    // Show loading indicator while history loads
    messages = [{ role: 'advisor', content: '', html: '', time: new Date().toISOString() }];
    renderMessages();

    // Load history
    fetch('/api/naming/orders/' + orderID + '/messages').then(function(histResp) {
      if (!histResp.ok) {
        messages = [];
        greet();
        showError(t('error.requestFailed'));
        return;
      }
      return histResp.json().then(function(histData) {
        var msgs = histData.data || [];
        messages = [];
        if (msgs.length > 0) {
          for (var i = 0; i < msgs.length; i++) {
            var m = msgs[i];
            if (m.role === 'tool') continue;
            if (m.role === 'assistant' && !m.content) continue;
            messages.push({
              role: m.role === 'assistant' ? 'advisor' : 'user',
              content: m.content,
              html: m.role === 'assistant' ? renderMD(m.content) : undefined,
              time: m.created_at
            });
          }
          refreshMeta();
          renderMessages();
          return;
        }
        greet();
      });
    }).catch(function() {
      messages = [];
      greet();
      showError(t('error.requestFailed'));
    });
  }).catch(function(err) {
    setLoginError(err.message);
    showView('login');
  });
}

// ── Keyboard shortcuts ──
function onGlobalKeydown(e) {
  if (e.key === 'Escape' && ui.substate === 'loading') {
    stopStream();
  }
}

// ── Event bindings ──
btnLogin.addEventListener('click', doLogin);
loginEmail.addEventListener('keydown', function(e) {
  if (e.key === 'Enter') doLogin();
  updateSendButton();
});
loginEmail.addEventListener('input', function() {
  setLoginError('');
  updateSendButton();
});

function updateSendButton() {
  btnLogin.disabled = !loginEmail.value || loginLoadingState;
}

btnReset.addEventListener('click', reset);
btnSend.addEventListener('click', function() { sendMessage(); });
btnStop.addEventListener('click', stopStream);

chatInput.addEventListener('keydown', function(e) {
  if (e.key === 'Enter') sendMessage();
});
chatInput.addEventListener('input', function() { saveDraft(); updateInputUI(); });
chatInput.addEventListener('compositionstart', function() { isComposing = true; });
chatInput.addEventListener('compositionend', function() { isComposing = false; });

$('error-toast-close').addEventListener('click', hideError);
errorToastRetry.addEventListener('click', retryLastMessage);

document.addEventListener('keydown', onGlobalKeydown);

// ── Init ──
(function init() {
  var appEl = $('app');
  if (appEl) appEl.removeAttribute('v-cloak');

  var qp = new URLSearchParams(location.search);
  var oid = qp.get('order_id');
  if (oid) {
    try { sessionStorage.setItem('likiOrderID', oid); } catch {}
    var loc = (location.pathname.match(/^\/(zh-Hans|zh-Hant|en)\//) || [])[1];
    history.replaceState(null, '', '/' + (loc || 'zh-Hant') + '/chat');
  } else {
    try { oid = sessionStorage.getItem('likiOrderID'); } catch {}
  }

  if (oid) {
    orderID = oid;
    fetch('/api/naming/orders/' + oid + '/status').then(function(resp) {
      if (resp.status === 401) {
        showView('login');
        return;
      }
      if (resp.ok) {
        return resp.json().then(function(data) {
          if (data.data && data.data.status === 'paid') {
            enterChat(data.data);
            return;
          }
          throw new Error('not paid');
        });
      }
      throw new Error('not found');
    }).catch(function() {
      try { sessionStorage.removeItem('likiOrderID'); } catch {}
      showView('login');
    });
  } else {
    showView('login');
  }

  // ── mobile: keyboard 适配 ──
  var _shell = $('view-chat');
  if (window.visualViewport && _shell) {
    function fitShell() {
      _shell.style.height = window.visualViewport.height + 'px';
      scrollDown(true);
    }
    window.visualViewport.addEventListener('resize', fitShell);
  }
  if (chatInput) {
    chatInput.addEventListener('focus', function () {
      setTimeout(function () { scrollDown(true); }, 300);
    });
  }
})();

})();
