/* The page only renders plugin state; all network, secrets and files use the backend. */
(() => {
  'use strict';
  const host = window.lithe;
  const locale = host?.context?.locale || navigator.language;
  const lang = /^zh/i.test(locale) ? 'zh' : 'en';
  const defaults = window.leetcodeMessages[lang];
  const uiVersion = 1;
  let text = defaults;
  let compatible = true;
  let state = null;
  let activeCall = '';
  let failedMore = false;
  let lastFailure = null;
  let sequence = 0;
  let frame = 0;
  let pendingSelection = null;
  let selecting = false;
  let infoButton = null;
  let communitySelection = '';
  const instance = `page.${crypto.randomUUID()}`;
  const $ = id => document.getElementById(id);
  const t = key => text[key] || key;
  const format = (key, ...args) => { let i = 0; return t(key).replace(/%[ds]/g, () => String(args[i++])); };
  document.documentElement.lang = lang;
  function translate() {
    document.querySelectorAll('[data-text]').forEach(el => { el.textContent = t(el.dataset.text); });
    document.querySelectorAll('[data-placeholder]').forEach(el => { el.placeholder = t(el.dataset.placeholder); el.setAttribute('aria-label', el.placeholder); });
    document.querySelectorAll('[data-tip]').forEach(el => {
      el.type = 'button';
      el.setAttribute('aria-label', t(el.dataset.tip));
      el.setAttribute('aria-expanded', String(infoButton === el));
      el.setAttribute('aria-controls', 'info-popover');
    });
    $('list').setAttribute('aria-label', t(state?.community?.kind || 'problems'));
  }
  const info = document.createElement('div');
  info.id = 'info-popover'; info.className = 'info-popover';
  info.setAttribute('role', 'note'); info.hidden = true;
  document.body.append(info);
  function closeInfo() {
    infoButton?.setAttribute('aria-expanded', 'false');
    infoButton = null; info.hidden = true;
  }
  function positionInfo() {
    if (!infoButton) return;
    const anchor = infoButton.getBoundingClientRect();
    info.style.left = `${Math.max(8, Math.min(anchor.left, innerWidth - info.offsetWidth - 8))}px`;
    info.style.top = `${Math.max(8, Math.min(anchor.bottom + 6, innerHeight - info.offsetHeight - 8))}px`;
  }
  document.querySelectorAll('[data-tip]').forEach(button => button.addEventListener('click', () => {
    const opening = infoButton !== button;
    closeInfo();
    if (opening) {
      infoButton = button; info.textContent = t(button.dataset.tip); info.hidden = false;
      button.setAttribute('aria-expanded', 'true'); positionInfo();
    }
  }));
  document.addEventListener('pointerdown', event => {
    if (!event.target.closest('[data-tip],#info-popover')) closeInfo();
  });
  document.addEventListener('focusin', event => {
    if (!event.target.closest('[data-tip],#info-popover')) closeInfo();
  });
  document.addEventListener('keydown', event => { if (event.key === 'Escape' && infoButton) { event.preventDefault(); closeInfo(); } });
  window.addEventListener('resize', positionInfo);
  function updateControls() {
    const busy = !!activeCall;
    $('app').setAttribute('aria-busy', String(busy));
    document.querySelectorAll('button,input,select').forEach(el => { if (!el.classList.contains('info')) el.disabled = busy || !host || !compatible; });
    $('cancel').hidden = !busy;
    $('cancel').disabled = false;
    $('retry').hidden = !lastFailure || !compatible;
    $('retry').textContent = lastFailure?.action === 'open' && state ? t('syncState') : t('retry');
    if (!state || !compatible) return;
    $('site').disabled = busy || state.pending;
    ['run', 'submit', 'connect', 'logout'].forEach(action => {
      document.querySelectorAll(`[data-action="${action}"]`).forEach(el => { el.disabled = busy || state.pending; });
    });
    ['preview', 'create', 'solutions', 'discussion'].forEach(action => {
      document.querySelector(`[data-action="${action}"]`).disabled = busy || !state.selected;
    });
    $('check').hidden = !state.pending;
    $('result').hidden = !state.hasResult;
  }
  function render(next) {
    // The packaged dictionary keeps newly added labels readable when the
    // host still has an older backend loaded. Reject that mixed installation
    // before sending actions that its backend cannot understand.
    text = { ...defaults, ...next?.text };
    compatible = next?.uiVersion === uiVersion;
    if (!compatible) {
      pendingSelection = null;
      translate();
      $('error').textContent = t('versionMismatch');
      $('error').hidden = false;
      $('notice').textContent = '';
      updateControls();
      return;
    }
    const resetScroll = state && (next.site !== state.site || next.query !== state.query || next.difficulty !== state.difficulty || next.status !== state.status || next.tag !== state.tag || (next.selected !== state.selected && !!next.community) || next.community?.kind !== state.community?.kind);
    if (resetScroll) communitySelection = '';
    state = next;
    translate();
    for (const key of ['site', 'language', 'difficulty', 'status', 'tag']) $(key).value = next[key];
    $('search').value = next.query;
    if (!$('language').options.length) {
      for (const item of next.languages) { const option = new Option(item.name, item.id); $('language').add(option); }
    }
    $('language').value = next.language;
    $('selected').textContent = next.title || t('none');
    $('selected').title = next.title || t('none');
    const items = next.community?.items || next.questions;
    $('count').textContent = next.community ? format('communityCount', items.length, next.community.total) : next.loaded ? format('count', items.length, next.total) : '';
    $('community-bar').hidden = !next.community;
    $('community-title').textContent = next.community ? t(next.community.kind) : '';
    $('account').textContent = next.account ? format('signedIn', next.account) : next.credentialStored ? t('savedCredential') : t('signedOut');
    $('verify').hidden = !next.credentialStored || !!next.account;
    $('logout').hidden = !next.credentialStored;
    $('login').hidden = !!next.account;
    $('connect').hidden = !!next.account;
    $('notice').textContent = next.notice || (next.pending ? t('pending') : !next.community && next.questions.length >= 400 ? t('loadLimit') : '');
    $('result').hidden = !next.hasResult;
    $('empty').hidden = !!items.length;
    $('empty').textContent = next.community ? t('communityEmpty') : next.loaded ? t('nothingFound') : t('empty');
    $('spacer').style.height = `${items.length * 48}px`;
    if (resetScroll) $('list').scrollTop = 0;
    renderRows();
    updateControls();
  }
  function renderRows() {
    if (!state) return;
    const list = $('list');
    const items = state.community?.items || state.questions;
    if (state.community && !items.some(item => item.slug === communitySelection)) communitySelection = items[0]?.slug || '';
    const selected = items.findIndex(q => q.slug === (state.community ? communitySelection : state.selected));
    const start = Math.max(0, Math.floor(list.scrollTop / 48) - 4);
    const end = Math.min(items.length, Math.ceil((list.scrollTop + list.clientHeight) / 48) + 4);
    const fragment = document.createDocumentFragment();
    for (let i = start; i < end; i++) {
      const q = items[i];
      const row = document.createElement('div');
      row.className = 'problem'; row.id = `problem-${i}`; row.dataset.slug = q.slug;
      row.setAttribute('role', 'option'); row.setAttribute('aria-selected', String(i === selected));
      row.setAttribute('aria-posinset', String(i + 1)); row.setAttribute('aria-setsize', String(state.community?.total ?? state.total));
      const title = document.createElement('span'); title.className = 'problem-title'; title.textContent = state.community ? q.title : `${q.number}. ${q.title}`;
      const meta = document.createElement('span'); meta.className = `problem-meta ${(q.difficulty || '').toLowerCase()}`;
      meta.textContent = state.community ? q.author : [t(q.difficulty), q.paid ? t('paidLabel') : '', q.status ? t(q.status.toUpperCase()) : ''].filter(Boolean).join(' · ');
      row.title = title.textContent; row.append(title, meta); fragment.append(row);
    }
    $('rows').replaceChildren(fragment); $('rows').style.transform = `translateY(${start * 48}px)`;
    if (selected >= start && selected < end) list.setAttribute('aria-activedescendant', `problem-${selected}`);
    else list.removeAttribute('aria-activedescendant');
  }
  async function invoke(action, extra = {}) {
    if (!host || activeCall || !compatible) return;
    activeCall = `${instance}.${++sequence}`;
    const call = activeCall;
    lastFailure = null;
    $('error').hidden = true;
    $('notice').textContent = t('loading');
    updateControls();
    try {
      const result = await host.request('functions.invoke', { function: 'leetcode.dispatch', params: { instance, action, language: locale, ...extra }, call });
      render(result.value);
      if (action !== 'select' && action !== 'open') failedMore = false;
      return compatible;
    } catch (error) {
      const cancelled = /cancel/i.test(error?.code || '');
      // A cancelled wait can still have created a remote job. A manual read
      // recovers that job without repeating the upload or the failed action.
      lastFailure = { action: 'open', extra: {} };
      if (!cancelled) {
        $('error').textContent = error?.pluginErrorCode?.startsWith('leetcode.') ? (error.message || t('requestFailed')) : t('requestFailed');
        $('error').hidden = false;
        if (action === 'more' || action === 'communityMore') failedMore = true;
        // Retrying a mutation can duplicate a remote operation. Only reads
        // get a retry button. The backend keeps pending state itself, so an
        // extra open request is neither needed nor safe while the original
        // call is still unwinding.
        if (error?.pluginErrorCode !== 'leetcode.resultDisplayFailed' && ['open', 'refresh', 'more', 'search', 'difficulty', 'status', 'tag', 'check', 'preview', 'result', 'solutions', 'discussion', 'communityMore', 'communityOpen'].includes(action)) lastFailure = { action, extra };
      }
    } finally {
      if (activeCall === call) activeCall = '';
      if ($('notice').textContent === t('loading')) $('notice').textContent = state?.pending ? t('pending') : '';
      updateControls(); renderRows();
      if (pendingSelection && !selecting && compatible) void openSelection();
    }
  }
  function maybeMore() {
    const page = state?.community || state;
    const items = state?.community?.items || state?.questions || [];
    if (compatible && page?.hasMore && !activeCall && !failedMore && $('list').scrollTop + $('list').clientHeight >= items.length * 48 - 180) {
      void invoke(state.community ? 'communityMore' : 'more', { cursor: page.cursor });
    }
  }
  async function openSelection() {
    if (selecting || activeCall || !compatible) return;
    selecting = true;
    try {
      while (pendingSelection && compatible) {
        const slug = pendingSelection; pendingSelection = null;
        if (await invoke('select', { item: slug }) && !pendingSelection) await invoke('preview');
      }
    } finally { selecting = false; }
  }
  $('list').addEventListener('scroll', () => {
    if (frame) return;
    frame = requestAnimationFrame(() => { frame = 0; renderRows(); maybeMore(); });
  }, { passive: true });
  new ResizeObserver(() => renderRows()).observe($('list'));
  $('rows').addEventListener('click', e => {
    const row = e.target.closest('[data-slug]');
    if (!row || !compatible) return;
    const slug = row.dataset.slug;
    $('list').focus();
    // Keep the latest click if a page is still loading. A single click opens
    // the statement; replacing virtual rows cannot swallow a double click.
    if (state?.community) {
      if (activeCall) return;
      communitySelection = slug; renderRows();
      void invoke('communityOpen', { item: slug }); return;
    }
    pendingSelection = slug; void openSelection();
  });
  $('list').addEventListener('keydown', e => {
    if (!state || activeCall || !compatible) return;
    const items = state.community?.items || state.questions;
    if (!items.length) return;
    if (state.community) {
      const current = Math.max(0, items.findIndex(item => item.slug === communitySelection));
      if (e.key === 'Enter' && items[current]) { e.preventDefault(); void invoke('communityOpen', { item: items[current].slug }); return; }
      if (!['ArrowDown','ArrowUp','Home','End'].includes(e.key)) return;
      e.preventDefault();
      const next = e.key === 'Home' ? 0 : e.key === 'End' ? items.length - 1 : Math.max(0, Math.min(items.length - 1, current + (e.key === 'ArrowDown' ? 1 : -1)));
      communitySelection = items[next].slug;
      revealRow(next); renderRows(); return;
    }
    if (e.key === 'Enter') { e.preventDefault(); void invoke('preview'); return; }
    if (!['ArrowDown','ArrowUp','Home','End'].includes(e.key)) return;
    e.preventDefault();
    const i = state.questions.findIndex(q => q.slug === state.selected);
    const next = e.key === 'Home' ? 0 : e.key === 'End' ? state.questions.length - 1 : Math.max(0, Math.min(state.questions.length - 1, i + (e.key === 'ArrowDown' ? 1 : -1)));
    revealRow(next);
    void invoke('select', { item: state.questions[next].slug });
  });
  function revealRow(index) {
    const list = $('list');
    const y = index * 48;
    if (y < list.scrollTop) list.scrollTop = y;
    else if (y + 48 > list.scrollTop + list.clientHeight) list.scrollTop = y + 48 - list.clientHeight;
  }
  document.querySelectorAll('[data-action]').forEach(el => el.addEventListener('click', () => { void invoke(el.dataset.action); }));
  ['site','language','difficulty','status','tag'].forEach(key => $(key).addEventListener('change', () => { failedMore = false; void invoke(key, { value: $(key).value }); }));
  $('search-form').addEventListener('submit', e => { e.preventDefault(); if (!e.isComposing) { failedMore = false; void invoke('search', { value: $('search').value.trim() }); } });
  $('retry').addEventListener('click', () => { const retry = lastFailure; if (retry) void invoke(retry.action, retry.extra); });
  $('cancel').addEventListener('click', () => { if (activeCall) void host.request('functions.cancel', { call: activeCall }).catch(() => { $('error').textContent = t('requestFailed'); $('error').hidden = false; }); });
  translate();
  if (!host) { $('error').textContent = t('unavailable'); $('error').hidden = false; updateControls(); return; }
  document.documentElement.dataset.theme = host.context.theme;
  host.on('functions.progress', event => { if (event.call === activeCall) $('notice').textContent = event.message || t('loading'); });
  host.on('ui.themeChanged', event => { document.documentElement.dataset.theme = event.theme; });
  void invoke('open');
})();
