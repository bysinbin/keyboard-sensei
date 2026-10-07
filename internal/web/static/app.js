// Keyboard Sensei — Web Interface Logic v1.1.0

let state = {
  status: null,
  rules: [],
  profiles: [],
  activeProfileID: 'profile-ansi-tr',
  keycodes: [],
  recordingActive: false,
  recordedShortcut: null,
  devices: [],
  hyperKey: null,
  sequences: [],
  enableSequences: true
};

// Browser event.code to macOS virtual keycode mapping
const CODE_TO_MACOS_KEYCODE = {
  // Special Turkish Q on ANSI & ISO mappings
  'Comma': 43,        // Turkish 'ö' (ANSI Comma)
  'Period': 47,       // Turkish 'ç' (ANSI Period)
  'Slash': 44,        // Turkish '.' (ANSI Slash)
  'Backslash': 42,    // Turkish ',' (ANSI Backslash)
  'Semicolon': 41,    // Turkish 'ş' (ANSI Semicolon)
  'Quote': 39,        // Turkish 'i' (ANSI Quote)
  'BracketLeft': 33,  // Turkish 'ğ' (ANSI Left Bracket)
  'BracketRight': 30, // Turkish 'ü' (ANSI Right Bracket)
  'Minus': 27,        // '-'
  'Equal': 24,        // '='
  'Backquote': 50,    // Turkish '"' (çift tırnak) / ANSI Grave / Tilde
  'IntlBackslash': 50,// ISO '< >' key (or 10)
  'IntlRo': 94,
  'IntlYen': 93,

  // Letters
  'KeyA': 0, 'KeyS': 1, 'KeyD': 2, 'KeyF': 3, 'KeyH': 4, 'KeyG': 5,
  'KeyZ': 6, 'KeyX': 7, 'KeyC': 8, 'KeyV': 9, 'KeyB': 11, 'KeyQ': 12,
  'KeyW': 13, 'KeyE': 14, 'KeyR': 15, 'KeyY': 16, 'KeyT': 17, 'KeyU': 32,
  'KeyI': 34, 'KeyO': 31, 'KeyP': 35, 'KeyJ': 38, 'KeyK': 40, 'KeyL': 37,
  'KeyN': 45, 'KeyM': 46,

  // Numbers
  'Digit1': 18, 'Digit2': 19, 'Digit3': 20, 'Digit4': 21, 'Digit5': 23,
  'Digit6': 22, 'Digit7': 26, 'Digit8': 28, 'Digit9': 25, 'Digit0': 29,

  // Function Keys
  'F1': 122, 'F2': 120, 'F3': 99, 'F4': 118, 'F5': 96, 'F6': 97,
  'F7': 98, 'F8': 100, 'F9': 101, 'F10': 109, 'F11': 103, 'F12': 111,

  // Controls & Navigation
  'Space': 49, 'Enter': 36, 'NumpadEnter': 76, 'Tab': 48, 'Backspace': 51, 'Escape': 53,
  'ArrowLeft': 123, 'ArrowRight': 124, 'ArrowDown': 125, 'ArrowUp': 126,
  'Home': 115, 'End': 119, 'PageUp': 116, 'PageDown': 121, 'Delete': 117
};

// Fallback: Browser event.keyCode (VK codes) to macOS virtual keycodes
const VK_TO_MACOS_KEYCODE = {
  65: 0,  83: 1,  68: 2,  70: 3,  72: 4,  71: 5,  90: 6,  88: 7,
  67: 8,  86: 9,  66: 11, 81: 12, 87: 13, 69: 14, 82: 15, 89: 16,
  84: 17, 85: 32, 73: 34, 79: 31, 80: 35, 74: 38, 75: 40, 76: 37,
  78: 45, 77: 46,
  49: 18, 50: 19, 51: 20, 52: 21, 53: 23,
  54: 22, 55: 26, 56: 28, 57: 25, 48: 29,
  188: 43, 190: 47, 191: 44, 220: 42, 186: 41, 59: 41, 222: 39,
  219: 33, 221: 30, 189: 27, 173: 27, 187: 24, 61: 24,
  192: 50, 226: 50, 32: 49, 13: 36, 9: 48, 8: 51, 27: 53
};

// Physical ANSI Keyboard Rows for Visualization
const KEYBOARD_LAYOUT = [
  // Row 1: Numbers & symbols
  [
    { main: '"', sub: '~', code: 50, special: true },
    { main: '1', sub: '!', code: 18 },
    { main: '2', sub: "'", code: 19 },
    { main: '3', sub: '^', code: 20 },
    { main: '4', sub: '+', code: 21 },
    { main: '5', sub: '%', code: 23 },
    { main: '6', sub: '&', code: 22 },
    { main: '7', sub: '/', code: 26 },
    { main: '8', sub: '(', code: 28 },
    { main: '9', sub: ')', code: 25 },
    { main: '0', sub: '=', code: 29 },
    { main: '*', sub: '-', code: 27, special: true },
    { main: '-', sub: '_', code: 24 },
    { main: '⌫', sub: '', code: 51, mod: true, cls: 'w-1-5' }
  ],
  // Row 2: QWERTY top row
  [
    { main: 'Tab', sub: '⇥', code: 48, mod: true, cls: 'w-1-5' },
    { main: 'Q', code: 12 },
    { main: 'W', code: 13 },
    { main: 'E', code: 14 },
    { main: 'R', code: 15 },
    { main: 'T', code: 17 },
    { main: 'Y', code: 16 },
    { main: 'U', code: 32 },
    { main: 'I', sub: 'İ', code: 34 },
    { main: 'O', code: 31 },
    { main: 'P', code: 35 },
    { main: 'Ğ', sub: '[', code: 33, special: true },
    { main: 'Ü', sub: ']', code: 30, special: true },
    { main: ',', sub: '\\', code: 42, special: true, cls: 'w-1-5' }
  ],
  // Row 3: Home row
  [
    { main: 'Caps', sub: '⚡', code: 57, mod: true, cls: 'w-1-75' },
    { main: 'A', code: 0 },
    { main: 'S', code: 1 },
    { main: 'D', code: 2 },
    { main: 'F', code: 3 },
    { main: 'G', code: 5 },
    { main: 'H', code: 4 },
    { main: 'J', code: 38 },
    { main: 'K', code: 40 },
    { main: 'L', code: 37 },
    { main: 'Ş', sub: ';', code: 41, special: true },
    { main: 'İ', sub: "'", code: 39, special: true },
    { main: 'Enter', sub: '↩', code: 36, mod: true, cls: 'w-2-25' }
  ],
  // Row 4: Bottom letter row
  [
    { main: 'Shift', sub: '⇧', code: null, mod: true, cls: 'w-2-25' },
    { main: 'Z', code: 6 },
    { main: 'X', code: 7 },
    { main: 'C', code: 8 },
    { main: 'V', code: 9 },
    { main: 'B', code: 11 },
    { main: 'N', code: 45 },
    { main: 'M', code: 46 },
    { main: 'Ö', sub: '< (Virgül)', code: 43, special: true },
    { main: 'Ç', sub: '> (Nokta)', code: 47, special: true },
    { main: '.', sub: '| (Bölü)', code: 44, special: true },
    { main: 'Shift', sub: '⇧', code: null, mod: true, cls: 'w-2-25' }
  ],
  // Row 5: Modifiers & Space
  [
    { main: '⌃ Ctrl', code: null, mod: true, cls: 'w-1-25' },
    { main: '⌥ Opt', code: null, mod: true, cls: 'w-1-25' },
    { main: '⌘ Cmd', code: null, mod: true, cls: 'w-1-5' },
    { main: 'Space', sub: '', code: 49, cls: 'w-space' },
    { main: '⌘ Cmd', code: null, mod: true, cls: 'w-1-5' },
    { main: '⌥ Opt', code: null, mod: true, cls: 'w-1-25' },
    { main: '⌃ Ctrl', code: null, mod: true, cls: 'w-1-25' }
  ]
];

document.addEventListener('DOMContentLoaded', () => {
  initApp();
});

async function initApp() {
  await loadKeycodes();
  await refreshStatus();
  await loadProfiles();
  await loadDevices();
  await loadHyperKey();
  await loadSequences();
  setupTabs();
  setupEventListeners();
  initSSE();

  // Poll status periodically (every 4s) to check accessibility and running state
  setInterval(refreshStatus, 4000);
}

// -------------------------------------------------------------
// Data Fetching
// -------------------------------------------------------------
async function refreshStatus() {
  try {
    const res = await fetch('/api/status');
    if (!res.ok) return;
    const data = await res.json();
    state.status = data;
    renderStatus(data);
  } catch (err) {
    console.error('Status fetch failed:', err);
  }
}

async function loadKeycodes() {
  try {
    const res = await fetch('/api/keycodes');
    if (!res.ok) return;
    state.keycodes = await res.json();
    populateKeycodeDropdown();
  } catch (err) {
    console.error('Keycodes fetch failed:', err);
  }
}

async function loadProfiles() {
  try {
    const res = await fetch('/api/profiles');
    if (!res.ok) return;
    const data = await res.json();
    state.profiles = data.profiles || [];
    state.activeProfileID = data.active_profile_id;

    renderProfileSelector();
    renderModalProfilesList();

    // Fetch active rules
    await loadRules();
  } catch (err) {
    console.error('Profiles fetch failed:', err);
  }
}

async function loadRules() {
  try {
    const res = await fetch('/api/rules');
    if (!res.ok) return;
    state.rules = await res.json();
    renderRules(state.rules);
    renderVirtualKeyboard();
    updateStats();
  } catch (err) {
    console.error('Rules fetch failed:', err);
  }
}

// -------------------------------------------------------------
// Rendering
// -------------------------------------------------------------
function renderStatus(status) {
  const badge = document.getElementById('status-badge');
  const badgeText = document.getElementById('status-text');
  const pauseIcon = document.getElementById('pause-icon');
  const pauseText = document.getElementById('pause-text');
  const permBanner = document.getElementById('permission-banner');

  if (status.paused) {
    badge.className = 'status-badge status-paused';
    badgeText.textContent = 'Duraklatıldı';
    pauseIcon.textContent = '▶️';
    pauseText.textContent = 'Başlat';
  } else {
    badge.className = 'status-badge status-active';
    badgeText.textContent = 'Motor Aktif';
    pauseIcon.textContent = '⏸';
    pauseText.textContent = 'Duraklat';
  }

  // Accessibility check
  if (!status.accessibility_trusted) {
    permBanner.classList.remove('hidden');
  } else {
    permBanner.classList.add('hidden');
  }

  // Service auto-start toggle
  const serviceToggle = document.getElementById('service-auto-start-toggle');
  if (serviceToggle && document.activeElement !== serviceToggle) {
    serviceToggle.checked = Boolean(status.service_installed);
  }

  // Stats
  if (status.stats) {
    document.getElementById('stat-total-triggers').textContent = status.stats.total_triggered || 0;
  }

  // Hyper Key Stat
  const statHyper = document.getElementById('stat-hyper-status');
  const subHyper = document.getElementById('stat-hyper-sub');
  const badgeHyper = document.getElementById('badge-hyper-status');
  if (status.hyper_key_enabled) {
    if (statHyper) statHyper.textContent = 'Aktif ⚡';
    if (subHyper) subHyper.textContent = 'Caps Lock ➔ ⌘⌥⌃⇧';
    if (badgeHyper) badgeHyper.className = 'tab-status-dot active';
  } else {
    if (statHyper) statHyper.textContent = 'Kapalı';
    if (subHyper) subHyper.textContent = 'Caps Lock Devre Dışı';
    if (badgeHyper) badgeHyper.className = 'tab-status-dot';
  }

  // Sequences count
  const seqBadge = document.getElementById('badge-sequences-count');
  if (seqBadge && status.sequences_count !== undefined) {
    seqBadge.textContent = status.sequences_count;
  }

  if (status.active_profile_name) {
    const headerProfile = document.getElementById('profile-name-header');
    if (headerProfile) headerProfile.textContent = status.active_profile_name;
    const subProfile = document.getElementById('stat-profile-sub');
    if (subProfile) subProfile.textContent = status.active_profile_name;
  }
}

function renderProfileSelector() {
  const select = document.getElementById('profile-select');
  if (!select) return;

  select.innerHTML = state.profiles.map(p => {
    const icon = p.icon || '📁';
    const isSelected = p.id === state.activeProfileID ? 'selected' : '';
    return `<option value="${p.id}" ${isSelected}>${icon} ${escapeHtml(p.name)}</option>`;
  }).join('');

  const active = state.profiles.find(p => p.id === state.activeProfileID);
  if (active) {
    const headerProfile = document.getElementById('profile-name-header');
    if (headerProfile) headerProfile.textContent = active.name;
    const subProfile = document.getElementById('stat-profile-sub');
    if (subProfile) subProfile.textContent = active.name;
  }
}

function renderModalProfilesList() {
  const list = document.getElementById('modal-profiles-list');
  if (!list) return;

  list.innerHTML = state.profiles.map(p => {
    const isActive = p.id === state.activeProfileID;
    const icon = p.icon || '📁';
    return `
      <div class="modal-profile-item ${isActive ? 'active' : ''}">
        <div class="profile-item-info">
          <span class="profile-item-icon">${icon}</span>
          <div>
            <div class="profile-item-name">${escapeHtml(p.name)} ${isActive ? '⭐ (Aktif)' : ''}</div>
            <div class="profile-item-desc">${escapeHtml(p.description || '')} (${p.rules ? p.rules.length : 0} kural)</div>
          </div>
        </div>
        <div class="profile-item-actions">
          ${!isActive ? `<button class="btn btn-secondary btn-sm btn-activate-profile" data-id="${p.id}">Aktif Yap</button>` : ''}
          ${state.profiles.length > 1 ? `<button class="btn btn-danger-ghost btn-sm btn-delete-profile" data-id="${p.id}">Sil</button>` : ''}
        </div>
      </div>
    `;
  }).join('');
}

function updateStats() {
  const activeCount = state.rules.filter(r => r.enabled).length;
  document.getElementById('stat-active-rules').textContent = activeCount;
}

// -------------------------------------------------------------
// Virtual Keyboard Rendering
// -------------------------------------------------------------
function renderVirtualKeyboard() {
  const container = document.getElementById('virtual-keyboard');
  if (!container) return;

  // Map keycode to rules
  const ruleMap = {};
  state.rules.forEach(rule => {
    if (rule.enabled) {
      if (!ruleMap[rule.keycode]) ruleMap[rule.keycode] = [];
      ruleMap[rule.keycode].push(rule);
    }
  });

  container.innerHTML = KEYBOARD_LAYOUT.map(row => {
    const keysHtml = row.map(key => {
      const cls = ['kb-key'];
      if (key.cls) cls.push(key.cls);
      if (key.mod) cls.push('key-mod');
      if (key.special) cls.push('key-special');

      let badgeHtml = '';
      let tooltipText = '';

      if (key.code !== null && ruleMap[key.code]) {
        cls.push('key-active');
        const rList = ruleMap[key.code];
        const outputs = rList.map(r => r.output).join(', ');
        badgeHtml = `<span class="key-badge" title="${escapeHtml(outputs)}">${escapeHtml(rList[0].output)}</span>`;
        tooltipText = rList.map(r => `${formatShortcutText(r.modifiers, r.keycode)} ➔ ${r.output}`).join(' | ');
      } else if (key.code === 57 && state.hyperKey && state.hyperKey.enabled) {
        cls.push('key-hyper-active');
        badgeHtml = `<span class="key-badge key-badge-hyper" title="Hyper Key Aktif">⚡</span>`;
        tooltipText = 'Caps Lock ➔ Hyper Key (⌘⌥⌃⇧)';
      }

      const dataAttr = key.code !== null ? `data-keycode="${key.code}"` : '';
      const titleAttr = tooltipText ? `title="${escapeHtml(tooltipText)}"` : '';

      return `
        <div class="${cls.join(' ')}" ${dataAttr} ${titleAttr}>
          <span class="key-main">${escapeHtml(key.main)}</span>
          ${key.sub ? `<span class="key-sub">${escapeHtml(key.sub)}</span>` : ''}
          ${badgeHtml}
        </div>
      `;
    }).join('');

    return `<div class="kb-row">${keysHtml}</div>`;
  }).join('');
}

function formatShortcutText(modifiers, keycode) {
  const syms = [];
  const mods = modifiers || [];
  if (mods.includes('ctrl')) syms.push('⌃');
  if (mods.includes('alt')) syms.push('⌥');
  if (mods.includes('shift')) syms.push('⇧');
  if (mods.includes('cmd')) syms.push('⌘');

  let keyStr = `#${keycode}`;
  const info = state.keycodes.find(k => k.code === keycode);
  if (info) keyStr = info.name.toUpperCase();
  return syms.concat(keyStr).join(' + ');
}

function renderRules(rules) {
  const container = document.getElementById('rules-container');
  if (!rules || rules.length === 0) {
    container.innerHTML = `
      <div class="stream-empty">
        Bu profilde henüz tanımlı kural yok. Yukarıdaki sanal klavyeden bir tuşa tıklayabilir veya "+ Yeni Kural Ekle" ile başlayabilirsiniz.
      </div>
    `;
    return;
  }

  container.innerHTML = rules.map(rule => {
    const shortcutHtml = formatShortcutHtml(rule.modifiers, rule.keycode);
    const hasApps = (rule.target_apps && rule.target_apps.length > 0) || (rule.excluded_apps && rule.excluded_apps.length > 0);
    return `
      <div class="rule-card ${rule.enabled ? '' : 'disabled'}" data-id="${rule.id}">
        <div class="rule-left">
          <div class="keycap-combo">
            ${shortcutHtml}
          </div>
          <span class="arrow-divider">➔</span>
          <div class="output-pill">${escapeHtml(rule.output)}</div>
          <div class="rule-details">
            <div style="display:flex;align-items:center;gap:6px;">
              <span class="rule-name">${escapeHtml(rule.name || 'Özel Kural')}</span>
              ${rule.is_snippet ? '<span class="version-tag" style="background:rgba(56,189,248,0.15);color:#38bdf8;border-color:rgba(56,189,248,0.3);">Makro</span>' : ''}
              ${hasApps ? '<span class="version-tag" style="background:rgba(168,85,247,0.15);color:#c084fc;border-color:rgba(168,85,247,0.3);">Uygulama Filtresi</span>' : ''}
            </div>
            <span class="rule-desc">${escapeHtml(rule.description || '')}</span>
          </div>
        </div>

        <div class="rule-right">
          <label class="switch" title="${rule.enabled ? 'Devre Dışı Bırak' : 'Etkinleştir'}">
            <input type="checkbox" class="rule-toggle" data-id="${rule.id}" ${rule.enabled ? 'checked' : ''}>
            <span class="slider"></span>
          </label>

          <button class="btn-icon-action btn-edit-rule" data-id="${rule.id}" title="Düzenle">
            ✏️
          </button>

          <button class="btn-icon-action delete btn-delete-rule" data-id="${rule.id}" title="Sil">
            🗑️
          </button>
        </div>
      </div>
    `;
  }).join('');
}

function formatShortcutHtml(modifiers, keycode) {
  const mods = modifiers || [];
  let html = '';
  if (mods.includes('ctrl')) html += '<kbd class="keycap">⌃</kbd>';
  if (mods.includes('alt')) html += '<kbd class="keycap">⌥</kbd>';
  if (mods.includes('shift')) html += '<kbd class="keycap">⇧</kbd>';
  if (mods.includes('cmd')) html += '<kbd class="keycap">⌘</kbd>';

  let keyName = `#${keycode}`;
  const keyInfo = state.keycodes.find(k => k.code === keycode);
  if (keyInfo) {
    keyName = keyInfo.name.toUpperCase();
  }

  html += `<kbd class="keycap highlight-key">${escapeHtml(keyName)}</kbd>`;
  return html;
}

function populateKeycodeDropdown() {
  const select = document.getElementById('form-keycode');
  select.innerHTML = '<option value="">Tuş Seçin...</option>';

  const turkishAnsiGroup = document.createElement('optgroup');
  turkishAnsiGroup.label = 'Türkçe Q / ANSI Özel Tuşları (Önerilen)';

  const otherGroup = document.createElement('optgroup');
  otherGroup.label = 'Diğer Standart Tuşlar';

  state.keycodes.forEach(k => {
    const opt = document.createElement('option');
    opt.value = k.code;
    opt.textContent = `${k.turkish_desc} (Kod: ${k.code})`;

    if ([43, 47, 44, 42, 41, 39, 33, 30, 27, 24, 50, 10].includes(k.code)) {
      turkishAnsiGroup.appendChild(opt);
    } else {
      otherGroup.appendChild(opt);
    }
  });

  select.appendChild(turkishAnsiGroup);
  select.appendChild(otherGroup);
}

// -------------------------------------------------------------
// Interactive Key Recorder
// -------------------------------------------------------------
function setupKeyRecorder() {
  const box = document.getElementById('key-recorder-box');
  const prompt = document.getElementById('recorder-prompt');
  const label = document.getElementById('recorder-label');
  const display = document.getElementById('recorded-display');
  const badges = document.getElementById('recorded-badges');

  box.addEventListener('click', () => {
    state.recordingActive = true;
    box.classList.add('recording');
    label.textContent = 'Klavyenizdeki tuş kombinasyonuna basın (örn. ⌘ + ö)...';
  });

  box.addEventListener('blur', () => {
    if (state.recordingActive && !state.recordedShortcut) {
      state.recordingActive = false;
      box.classList.remove('recording');
      label.textContent = 'Tuş kombinasyonunu kaydetmek için buraya tıklayın';
    }
  });

  box.addEventListener('keydown', (e) => {
    if (!state.recordingActive) return;

    e.preventDefault();
    e.stopPropagation();

    if (['Meta', 'Control', 'Alt', 'Shift'].includes(e.key)) {
      return;
    }

    const mods = [];
    if (e.ctrlKey) mods.push('ctrl');
    if (e.altKey) mods.push('alt');
    if (e.shiftKey) mods.push('shift');
    if (e.metaKey) mods.push('cmd');

    let keycode = CODE_TO_MACOS_KEYCODE[e.code];
    if (keycode === undefined && e.keyCode) {
      keycode = VK_TO_MACOS_KEYCODE[e.keyCode];
    }
    if (keycode === undefined) {
      keycode = e.keyCode;
    }

    state.recordedShortcut = {
      modifiers: mods,
      keycode: keycode,
      code: e.code,
      key: e.key
    };

    document.getElementById('mod-cmd').checked = mods.includes('cmd');
    document.getElementById('mod-alt').checked = mods.includes('alt');
    document.getElementById('mod-ctrl').checked = mods.includes('ctrl');
    document.getElementById('mod-shift').checked = mods.includes('shift');

    const selectEl = document.getElementById('form-keycode');
    let opt = selectEl.querySelector(`option[value="${keycode}"]`);
    if (!opt && keycode !== undefined && !isNaN(keycode)) {
      opt = document.createElement('option');
      opt.value = keycode;
      opt.textContent = `Özel Tuş #${keycode} (Kod: ${keycode})`;
      selectEl.appendChild(opt);
    }
    selectEl.value = keycode;

    prompt.classList.add('hidden');
    display.classList.remove('hidden');
    badges.innerHTML = formatShortcutHtml(mods, keycode);

    box.classList.remove('recording');
    state.recordingActive = false;
  });
}

// -------------------------------------------------------------
// Event Listeners
// -------------------------------------------------------------
function setupEventListeners() {
  setupKeyRecorder();

  // Profile Selection Change
  const profileSelect = document.getElementById('profile-select');
  if (profileSelect) {
    profileSelect.addEventListener('change', async (e) => {
      const selectedId = e.target.value;
      try {
        const res = await fetch('/api/profiles/activate', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ profile_id: selectedId })
        });
        if (res.ok) {
          state.activeProfileID = selectedId;
          await loadProfiles();
          showNotification('Aktif profil değiştirildi!');
        }
      } catch (err) {
        alert('Profil değiştirme hatası: ' + err);
      }
    });
  }

  // Manage Profiles Modal Trigger
  document.getElementById('btn-manage-profiles').addEventListener('click', () => {
    document.getElementById('profile-modal').classList.remove('hidden');
  });

  document.getElementById('btn-close-profile-modal').addEventListener('click', () => {
    document.getElementById('profile-modal').classList.add('hidden');
  });

  // Activate Profile from Modal
  document.getElementById('modal-profiles-list').addEventListener('click', async (e) => {
    const actBtn = e.target.closest('.btn-activate-profile');
    if (actBtn) {
      const pId = actBtn.dataset.id;
      await fetch('/api/profiles/activate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ profile_id: pId })
      });
      state.activeProfileID = pId;
      await loadProfiles();
      showNotification('Profil aktif edildi!');
      return;
    }

    const delBtn = e.target.closest('.btn-delete-profile');
    if (delBtn) {
      const pId = delBtn.dataset.id;
      if (confirm('Bu profili ve içindeki tüm kuralları silmek istediğinizden emin misiniz?')) {
        const res = await fetch(`/api/profiles/${pId}`, { method: 'DELETE' });
        if (res.ok) {
          await loadProfiles();
          showNotification('Profil silindi.');
        } else {
          alert('Profil silinemedi.');
        }
      }
      return;
    }
  });

  // New Profile Form
  document.getElementById('new-profile-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const name = document.getElementById('new-profile-name').value.trim();
    const preset = document.getElementById('new-profile-preset').value;
    if (!name) return;

    let rules = [];
    if (preset === 'profile-ansi-tr') {
      rules = [
        { id: `rule-${Date.now()}-1`, name: "Küçüktür (<)", modifiers: ["cmd"], keycode: 43, key_label: "ö", output: "<", enabled: true },
        { id: `rule-${Date.now()}-2`, name: "Büyüktür (>)", modifiers: ["cmd"], keycode: 47, key_label: "ç", output: ">", enabled: true },
        { id: `rule-${Date.now()}-3`, name: "Dikey Çizgi (|)", modifiers: ["alt"], keycode: 44, key_label: ".", output: "|", enabled: true }
      ];
    }

    const newP = {
      id: `profile-${Date.now()}`,
      name: name,
      icon: '✨',
      description: `${name} özel profili`,
      rules: rules
    };

    const res = await fetch('/api/profiles', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(newP)
    });

    if (res.ok) {
      document.getElementById('new-profile-name').value = '';
      await loadProfiles();
      showNotification('Yeni profil oluşturuldu!');
    }
  });

  // Virtual Keyboard Key Click -> Open Modal Pre-filled
  document.getElementById('virtual-keyboard').addEventListener('click', (e) => {
    const keyEl = e.target.closest('.kb-key');
    if (!keyEl || !keyEl.dataset.keycode) return;

    const keycode = parseInt(keyEl.dataset.keycode, 10);
    if (!isNaN(keycode)) {
      openModalWithKeycode(keycode);
    }
  });

  // Dynamic Token Chips in Modal
  document.querySelectorAll('.btn-token').forEach(btn => {
    btn.addEventListener('click', () => {
      const token = btn.dataset.token;
      const outputInput = document.getElementById('form-output');
      outputInput.value += token;
      outputInput.focus();
    });
  });

  // Detect Active App Button in Modal
  const detectAppBtn = document.getElementById('btn-detect-app');
  if (detectAppBtn) {
    detectAppBtn.addEventListener('click', async () => {
      try {
        const res = await fetch('/api/active_app');
        if (res.ok) {
          const app = await res.json();
          const tip = document.getElementById('detected-app-tip');
          if (app.name) {
            tip.innerHTML = `Algılanan: <strong>${escapeHtml(app.name)}</strong> (${escapeHtml(app.bundle_id)}) — <a href="#" id="link-add-target" style="color:#38bdf8;">Yalnızca bu uygulamaya ekle</a> | <a href="#" id="link-add-exclude" style="color:#ef4444;">İstisnalara ekle</a>`;
            
            document.getElementById('link-add-target')?.addEventListener('click', (ev) => {
              ev.preventDefault();
              const inp = document.getElementById('form-target-apps');
              inp.value = inp.value ? `${inp.value}, ${app.name}` : app.name;
            });

            document.getElementById('link-add-exclude')?.addEventListener('click', (ev) => {
              ev.preventDefault();
              const inp = document.getElementById('form-excluded-apps');
              inp.value = inp.value ? `${inp.value}, ${app.name}` : app.name;
            });
          } else {
            tip.textContent = 'Ön plandaki uygulama okunamadı.';
          }
        }
      } catch (err) {
        console.error('Active app error:', err);
      }
    });
  }

  // Export JSON
  document.getElementById('btn-export-json').addEventListener('click', () => {
    window.location.href = '/api/export';
  });

  // Import JSON Trigger & Handler
  const importTrigger = document.getElementById('btn-import-json-trigger');
  const importInput = document.getElementById('file-import-input');
  if (importTrigger && importInput) {
    importTrigger.addEventListener('click', () => importInput.click());
    importInput.addEventListener('change', async (e) => {
      const file = e.target.files[0];
      if (!file) return;

      const reader = new FileReader();
      reader.onload = async (ev) => {
        try {
          const res = await fetch('/api/import', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: ev.target.result
          });
          if (res.ok) {
            await loadProfiles();
            showNotification('Konfigürasyon başarıyla içe aktarıldı!');
          } else {
            const errTxt = await res.text();
            alert('İçe aktarma hatası: ' + errTxt);
          }
        } catch (err) {
          alert('Hata: ' + err);
        }
        importInput.value = '';
      };
      reader.readAsText(file);
    });
  }

  // Pause / Resume
  document.getElementById('btn-toggle-pause').addEventListener('click', async () => {
    await fetch('/api/status/pause', { method: 'POST' });
    await refreshStatus();
  });

  // Service Auto-Start Toggle
  const serviceToggle = document.getElementById('service-auto-start-toggle');
  if (serviceToggle) {
    serviceToggle.addEventListener('change', async (e) => {
      const isChecked = e.target.checked;
      try {
        const res = await fetch('/api/service', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ action: isChecked ? 'install' : 'uninstall' })
        });
        if (res.ok) {
          showNotification(isChecked ? '🚀 Başlangıç servisi aktif edildi!' : '🛑 Başlangıç servisi kaldırıldı.');
          await refreshStatus();
        } else {
          const txt = await res.text();
          alert('Servis işlemi başarısız: ' + txt);
          e.target.checked = !isChecked;
        }
      } catch (err) {
        alert('Bağlantı hatası: ' + err);
        e.target.checked = !isChecked;
      }
    });
  }

  // Open System Settings
  document.getElementById('btn-open-settings').addEventListener('click', async () => {
    await fetch('/api/status/open_settings', { method: 'POST' });
  });

  document.getElementById('btn-fix-permission').addEventListener('click', async () => {
    await fetch('/api/status/open_settings', { method: 'POST' });
  });

  // Quit App & Service
  const quitBtn = document.getElementById('btn-quit-app');
  if (quitBtn) {
    quitBtn.addEventListener('click', async () => {
      if (!confirm('Keyboard Sensei uygulamasını ve arka plan servisini tamamen kapatmak istiyor musunuz?')) return;
      try {
        showNotification('Uygulama kapatılıyor...');
        await fetch('/api/quit', { method: 'POST' });
        setTimeout(() => {
          document.body.innerHTML = `
            <div style="display:flex;flex-direction:column;align-items:center;justify-content:center;height:100vh;color:#94a3b8;font-family:sans-serif;text-align:center;">
              <h2 style="color:#fff;margin-bottom:12px;font-size:1.8rem;">🛑 Keyboard Sensei Kapatıldı</h2>
              <p>Arka plan servisi durduruldu. Bu sekmeyi kapatabilirsiniz.</p>
            </div>
          `;
        }, 400);
      } catch (err) {
        console.log('Quit dispatched');
      }
    });
  }

  // Open Add Rule Modal
  document.getElementById('btn-add-rule').addEventListener('click', () => {
    openModal();
  });

  // Close Modal
  document.getElementById('btn-close-modal').addEventListener('click', closeModal);
  document.getElementById('btn-cancel-modal').addEventListener('click', closeModal);

  // Form Submit
  document.getElementById('rule-form').addEventListener('submit', handleFormSubmit);

  // Rules Container Event Delegation (Toggle, Edit, Delete)
  document.getElementById('rules-container').addEventListener('click', async (e) => {
    const toggle = e.target.closest('.rule-toggle');
    if (toggle) {
      const id = toggle.dataset.id;
      const rule = state.rules.find(r => r.id === id);
      if (rule) {
        rule.enabled = toggle.checked;
        await updateRule(rule);
      }
      return;
    }

    const editBtn = e.target.closest('.btn-edit-rule');
    if (editBtn) {
      const id = editBtn.dataset.id;
      const rule = state.rules.find(r => r.id === id);
      if (rule) openModal(rule);
      return;
    }

    const delBtn = e.target.closest('.btn-delete-rule');
    if (delBtn) {
      const id = delBtn.dataset.id;
      if (confirm('Bu kuralı silmek istediğinizden emin misiniz?')) {
        await deleteRule(id);
      }
      return;
    }
  });

  // Sandbox Clear
  document.getElementById('btn-clear-sandbox').addEventListener('click', () => {
    document.getElementById('sandbox-textarea').value = '';
    document.getElementById('sandbox-textarea').focus();
  });

  // Clear Logs
  document.getElementById('btn-clear-logs').addEventListener('click', () => {
    const stream = document.getElementById('activity-stream');
    stream.innerHTML = '<div class="stream-empty">Kayıtlar temizlendi.</div>';
  });

  // Sync manual modifier checkboxes to recorder display
  ['mod-cmd', 'mod-alt', 'mod-ctrl', 'mod-shift'].forEach(id => {
    document.getElementById(id).addEventListener('change', updateRecordedDisplayFromManual);
  });
  document.getElementById('form-keycode').addEventListener('change', updateRecordedDisplayFromManual);

  // Hardware Devices Listeners
  document.getElementById('btn-refresh-devices')?.addEventListener('click', async () => {
    const btn = document.getElementById('btn-refresh-devices');
    if (btn) btn.classList.add('loading');
    await loadDevices();
    if (btn) btn.classList.remove('loading');
    showNotification('Klavyeler yeniden tarandı!');
  });

  // Hyper Key Listeners
  document.getElementById('hyperkey-master-toggle')?.addEventListener('change', saveHyperKeySettings);
  document.getElementById('btn-save-hyperkey')?.addEventListener('click', saveHyperKeySettings);
  ['hk-mod-cmd', 'hk-mod-alt', 'hk-mod-ctrl', 'hk-mod-shift'].forEach(id => {
    document.getElementById(id)?.addEventListener('change', updateHyperKeyPreview);
  });

  // Sequences Listeners
  document.getElementById('sequences-master-toggle')?.addEventListener('change', async (e) => {
    state.enableSequences = e.target.checked;
    await saveSequences(state.sequences, state.enableSequences);
  });
  document.getElementById('btn-add-sequence')?.addEventListener('click', openSequenceModal);
  document.getElementById('btn-close-seq-modal')?.addEventListener('click', closeSequenceModal);
  document.getElementById('btn-cancel-seq-modal')?.addEventListener('click', closeSequenceModal);
  document.getElementById('sequence-form')?.addEventListener('submit', handleSequenceFormSubmit);
}

// -------------------------------------------------------------
// Feature Tabs Navigation
// -------------------------------------------------------------
function setupTabs() {
  const tabs = document.querySelectorAll('.feature-tab');
  tabs.forEach(tab => {
    tab.addEventListener('click', () => {
      const targetPaneId = 'tab-pane-' + tab.dataset.tab.replace('tab-', '');
      tabs.forEach(t => t.classList.remove('active'));
      tab.classList.add('active');

      document.querySelectorAll('.tab-pane').forEach(p => p.classList.add('hidden'));
      const activePane = document.getElementById(targetPaneId);
      if (activePane) {
        activePane.classList.remove('hidden');
      }
    });
  });
}

// -------------------------------------------------------------
// Hardware Devices Management
// -------------------------------------------------------------
async function loadDevices() {
  try {
    const res = await fetch('/api/devices');
    if (!res.ok) return;
    const data = await res.json();
    state.devices = data.devices || [];
    renderDevices(state.devices);
  } catch (err) {
    console.error('Devices fetch failed:', err);
  }
}

function renderDevices(devices) {
  const container = document.getElementById('devices-container');
  if (!container) return;

  const countBadge = document.getElementById('badge-devices-count');
  if (countBadge) countBadge.textContent = devices.length;

  const statDevCount = document.getElementById('stat-devices-count');
  const statDevSub = document.getElementById('stat-devices-sub');
  if (statDevCount) {
    statDevCount.textContent = devices.length;
  }
  if (statDevSub) {
    const internalCount = devices.filter(d => d.is_internal).length;
    const externalCount = devices.length - internalCount;
    statDevSub.textContent = `${internalCount} Dahili, ${externalCount} Harici`;
  }

  if (devices.length === 0) {
    container.innerHTML = `
      <div class="empty-state" style="grid-column: 1 / -1; text-align: center; padding: 30px;">
        <div style="font-size: 2rem; margin-bottom: 8px;">⌨️</div>
        <h3 style="color:#fff; margin-bottom: 4px;">Bağlı Klavye Bulunamadı</h3>
        <p style="color:var(--text-muted); font-size:0.85rem;">Mac'inize bağlı donanım veya sanal klavye algılanamadı.</p>
      </div>
    `;
    return;
  }

  container.innerHTML = devices.map(dev => {
    const isInternal = dev.is_internal;
    const iconClass = isInternal ? 'device-icon-internal' : 'device-icon-external';
    const iconSymbol = isInternal ? '💻' : '🔌';
    const typeLabel = isInternal ? 'Dahili MacBook Klavyesi' : 'Harici Klavye';
    const pillClass = isInternal ? 'pill-internal' : 'pill-external';
    const hexVid = '0x' + (dev.vendor_id ? dev.vendor_id.toString(16).padStart(4, '0') : '0000');
    const hexPid = '0x' + (dev.product_id ? dev.product_id.toString(16).padStart(4, '0') : '0000');

    return `
      <div class="device-card ${dev.enabled ? '' : 'disabled'}" data-id="${dev.id}">
        <div class="device-header">
          <div class="device-title-row">
            <div class="device-icon-box ${iconClass}">
              <span>${iconSymbol}</span>
            </div>
            <div class="device-name-group">
              <h4 class="device-name">${escapeHtml(dev.name || 'Klavye')}</h4>
              <div class="device-badge-row">
                <span class="device-pill ${pillClass}">${typeLabel}</span>
                <span class="device-pill pill-transport">${escapeHtml(dev.transport || 'HID')}</span>
              </div>
            </div>
          </div>
          <div class="device-toggle-box">
            <label class="switch" title="Sensei Dönüşümlerini Bu Klavyede Aç/Kapat">
              <input type="checkbox" class="device-toggle-switch" data-vid="${dev.vendor_id}" data-pid="${dev.product_id}" data-id="${dev.id}" ${dev.enabled ? 'checked' : ''}>
              <span class="slider"></span>
            </label>
          </div>
        </div>

        <div class="device-meta-row">
          <span class="device-id-code">VID: ${hexVid} | PID: ${hexPid}</span>
          <span class="device-status-text ${dev.enabled ? 'enabled' : 'disabled'}">
            ${dev.enabled ? '✅ Sensei Aktif' : '⏸️ Devre Dışı'}
          </span>
        </div>
      </div>
    `;
  }).join('');

  // Attach event listeners to device toggle switches
  container.querySelectorAll('.device-toggle-switch').forEach(sw => {
    sw.addEventListener('change', async (e) => {
      const vid = parseInt(e.target.dataset.vid, 10);
      const pid = parseInt(e.target.dataset.pid, 10);
      const id = e.target.dataset.id;
      const enabled = e.target.checked;

      // Update state locally
      const found = state.devices.find(d => (d.vendor_id === vid && d.product_id === pid) || d.id === id);
      if (found) {
        found.enabled = enabled;
      }

      // Visual update
      const card = e.target.closest('.device-card');
      if (card) {
        card.classList.toggle('disabled', !enabled);
        const statusTxt = card.querySelector('.device-status-text');
        if (statusTxt) {
          statusTxt.className = `device-status-text ${enabled ? 'enabled' : 'disabled'}`;
          statusTxt.textContent = enabled ? '✅ Sensei Aktif' : '⏸️ Devre Dışı';
        }
      }

      try {
        await fetch('/api/devices', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            device: {
              id: id,
              vendor_id: vid,
              product_id: pid,
              enabled: enabled
            }
          })
        });
        showNotification(enabled ? 'Klavye için dönüşümler aktif edildi' : 'Klavye devre dışı bırakıldı');
      } catch (err) {
        console.error('Device update failed:', err);
      }
    });
  });
}

// -------------------------------------------------------------
// Hyper Key Management
// -------------------------------------------------------------
async function loadHyperKey() {
  try {
    const res = await fetch('/api/hyperkey');
    if (!res.ok) return;
    state.hyperKey = await res.json();
    renderHyperKey(state.hyperKey);
  } catch (err) {
    console.error('HyperKey fetch failed:', err);
  }
}

function renderHyperKey(hk) {
  if (!hk) return;

  const masterToggle = document.getElementById('hyperkey-master-toggle');
  if (masterToggle) masterToggle.checked = Boolean(hk.enabled);

  const modCmd = document.getElementById('hk-mod-cmd');
  const modAlt = document.getElementById('hk-mod-alt');
  const modCtrl = document.getElementById('hk-mod-ctrl');
  const modShift = document.getElementById('hk-mod-shift');

  const mods = hk.modifiers || ['cmd', 'alt', 'ctrl', 'shift'];
  if (modCmd) modCmd.checked = mods.includes('cmd');
  if (modAlt) modAlt.checked = mods.includes('alt');
  if (modCtrl) modCtrl.checked = mods.includes('ctrl');
  if (modShift) modShift.checked = mods.includes('shift');

  const tapRadios = document.querySelectorAll('input[name="hyper-tap-action"]');
  tapRadios.forEach(r => {
    r.checked = (r.value === (hk.tap_action || 'escape'));
  });

  updateHyperKeyPreview();

  // Status stats
  const statHyper = document.getElementById('stat-hyper-status');
  const subHyper = document.getElementById('stat-hyper-sub');
  const badgeHyper = document.getElementById('badge-hyper-status');
  if (hk.enabled) {
    if (statHyper) statHyper.textContent = 'Aktif ⚡';
    if (subHyper) subHyper.textContent = `Tap: ${(hk.tap_action || 'esc').toUpperCase()} / ⌘⌥⌃⇧`;
    if (badgeHyper) badgeHyper.className = 'tab-status-dot active';
  } else {
    if (statHyper) statHyper.textContent = 'Kapalı';
    if (subHyper) subHyper.textContent = 'Caps Lock Devre Dışı';
    if (badgeHyper) badgeHyper.className = 'tab-status-dot';
  }
}

function updateHyperKeyPreview() {
  const preview = document.getElementById('hyperkey-combo-preview');
  if (!preview) return;

  const mods = [];
  if (document.getElementById('hk-mod-cmd')?.checked) mods.push('⌘');
  if (document.getElementById('hk-mod-alt')?.checked) mods.push('⌥');
  if (document.getElementById('hk-mod-ctrl')?.checked) mods.push('⌃');
  if (document.getElementById('hk-mod-shift')?.checked) mods.push('⇧');

  if (mods.length === 0) {
    preview.innerHTML = '<span class="text-muted">(Hiçbir modifier seçilmedi)</span>';
    return;
  }

  preview.innerHTML = mods.map((m, i) => {
    return `<kbd class="kbd-badge">${m}</kbd>${i < mods.length - 1 ? '<span class="plus">+</span>' : ''}`;
  }).join('');
}

async function saveHyperKeySettings() {
  const masterToggle = document.getElementById('hyperkey-master-toggle');
  const mods = [];
  if (document.getElementById('hk-mod-cmd')?.checked) mods.push('cmd');
  if (document.getElementById('hk-mod-alt')?.checked) mods.push('alt');
  if (document.getElementById('hk-mod-ctrl')?.checked) mods.push('ctrl');
  if (document.getElementById('hk-mod-shift')?.checked) mods.push('shift');

  const selectedTap = document.querySelector('input[name="hyper-tap-action"]:checked')?.value || 'escape';

  const payload = {
    enabled: masterToggle ? masterToggle.checked : false,
    source_keycode: 57,
    modifiers: mods,
    tap_action: selectedTap
  };

  try {
    const res = await fetch('/api/hyperkey', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      state.hyperKey = await res.json();
      renderHyperKey(state.hyperKey);
      renderVirtualKeyboard();
      showNotification('⚡ Hyper Key ayarları başarıyla uygulandı!');
    }
  } catch (err) {
    alert('Hyper Key kaydedilemedi: ' + err);
  }
}

// -------------------------------------------------------------
// Double-Tap Sequences Management
// -------------------------------------------------------------
async function loadSequences() {
  try {
    const res = await fetch('/api/sequences');
    if (!res.ok) return;
    const data = await res.json();
    state.sequences = data.sequences || [];
    state.enableSequences = Boolean(data.enabled);
    renderSequences(state.sequences, state.enableSequences);
    populateSeqKeycodeDropdown();
  } catch (err) {
    console.error('Sequences fetch failed:', err);
  }
}

function renderSequences(sequences, enabled) {
  const container = document.getElementById('sequences-container');
  if (!container) return;

  const countBadge = document.getElementById('badge-sequences-count');
  if (countBadge) countBadge.textContent = sequences.length;

  const masterToggle = document.getElementById('sequences-master-toggle');
  if (masterToggle) masterToggle.checked = enabled;

  if (sequences.length === 0) {
    container.innerHTML = `
      <div class="empty-state" style="text-align: center; padding: 30px;">
        <div style="font-size: 2rem; margin-bottom: 8px;">🔁</div>
        <h3 style="color:#fff; margin-bottom: 4px;">Dizilim Bulunmuyor</h3>
        <p style="color:var(--text-muted); font-size:0.85rem;">Yukarıdaki "+ Yeni Dizilim Ekle" butonuna basarak çift dokunma kısayolu oluşturabilirsiniz.</p>
      </div>
    `;
    return;
  }

  container.innerHTML = sequences.map((seq, idx) => {
    const label = seq.key_label || `#${seq.keycode}`;
    return `
      <div class="sequence-card ${seq.enabled && enabled ? '' : 'disabled'}" data-id="${seq.id}">
        <div class="sequence-formula">
          <div class="seq-double-key">
            <span class="seq-key-pill">${escapeHtml(label)}</span>
            <span class="seq-key-pill">${escapeHtml(label)}</span>
          </div>
          <span class="seq-arrow">➔</span>
          <span class="seq-output-badge">${escapeHtml(seq.output)}</span>
        </div>

        <div class="sequence-meta">
          <span class="sequence-name">${escapeHtml(seq.name || `${label}${label} ➔ ${seq.output}`)}</span>
          <span class="sequence-details">${escapeHtml(seq.description || 'Hızlı çift dokunma')} &bull; ${seq.timeout_ms || 280}ms eşik</span>
        </div>

        <div class="sequence-actions">
          <label class="switch" title="Bu dizilimi aktif / pasif yap">
            <input type="checkbox" class="seq-toggle-switch" data-index="${idx}" ${seq.enabled ? 'checked' : ''}>
            <span class="slider"></span>
          </label>
          <button class="btn btn-ghost btn-sm btn-delete-seq" data-index="${idx}" title="Dizilimi Sil">🗑️</button>
        </div>
      </div>
    `;
  }).join('');

  // Bind sequence toggle and delete handlers
  container.querySelectorAll('.seq-toggle-switch').forEach(sw => {
    sw.addEventListener('change', async (e) => {
      const idx = parseInt(e.target.dataset.index, 10);
      state.sequences[idx].enabled = e.target.checked;
      await saveSequences(state.sequences, state.enableSequences);
    });
  });

  container.querySelectorAll('.btn-delete-seq').forEach(btn => {
    btn.addEventListener('click', async (e) => {
      const idx = parseInt(e.target.closest('.btn-delete-seq').dataset.index, 10);
      if (confirm('Bu çift dokunma kuralını silmek istediğinize emin misiniz?')) {
        state.sequences.splice(idx, 1);
        await saveSequences(state.sequences, state.enableSequences);
      }
    });
  });
}

async function saveSequences(sequences, enabled) {
  try {
    const res = await fetch('/api/sequences', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        enabled: enabled,
        sequences: sequences
      })
    });
    if (res.ok) {
      const data = await res.json();
      state.sequences = data.sequences || [];
      state.enableSequences = Boolean(data.enabled);
      renderSequences(state.sequences, state.enableSequences);
      showNotification('Çift dokunma dizilimleri güncellendi!');
    }
  } catch (err) {
    console.error('Save sequences failed:', err);
  }
}

function populateSeqKeycodeDropdown() {
  const select = document.getElementById('form-seq-keycode');
  if (!select || !state.keycodes) return;

  select.innerHTML = '<option value="">Tuş Seçin...</option>' + state.keycodes.map(k => {
    return `<option value="${k.code}">${k.name} (Kod: ${k.code})</option>`;
  }).join('');
}

function openSequenceModal() {
  const modal = document.getElementById('sequence-modal');
  if (!modal) return;
  document.getElementById('sequence-form').reset();
  document.getElementById('form-seq-id').value = '';
  document.getElementById('form-seq-timeout').value = '280';
  modal.classList.remove('hidden');
}

function closeSequenceModal() {
  const modal = document.getElementById('sequence-modal');
  if (modal) modal.classList.add('hidden');
}

async function handleSequenceFormSubmit(e) {
  e.preventDefault();
  const id = document.getElementById('form-seq-id').value || `seq-${Date.now()}`;
  const keycode = parseInt(document.getElementById('form-seq-keycode').value, 10);
  const output = document.getElementById('form-seq-output').value.trim();
  const name = document.getElementById('form-seq-name').value.trim() || `Çift Dokunma ➔ ${output}`;
  const timeoutMs = parseInt(document.getElementById('form-seq-timeout').value, 10) || 280;

  let keyLabel = `#${keycode}`;
  const keyInfo = state.keycodes.find(k => k.code === keycode);
  if (keyInfo) keyLabel = keyInfo.name;

  const newSeq = {
    id: id,
    name: name,
    keycode: keycode,
    key_label: keyLabel,
    tap_count: 2,
    output: output,
    timeout_ms: timeoutMs,
    enabled: true
  };

  const existingIdx = state.sequences.findIndex(s => s.id === id);
  if (existingIdx >= 0) {
    state.sequences[existingIdx] = newSeq;
  } else {
    state.sequences.push(newSeq);
  }

  await saveSequences(state.sequences, state.enableSequences);
  closeSequenceModal();
}

function updateRecordedDisplayFromManual() {
  const mods = [];
  if (document.getElementById('mod-cmd').checked) mods.push('cmd');
  if (document.getElementById('mod-alt').checked) mods.push('alt');
  if (document.getElementById('mod-ctrl').checked) mods.push('ctrl');
  if (document.getElementById('mod-shift').checked) mods.push('shift');

  const keycodeVal = document.getElementById('form-keycode').value;
  if (!keycodeVal) return;

  const keycode = parseInt(keycodeVal, 10);
  const prompt = document.getElementById('recorder-prompt');
  const display = document.getElementById('recorded-display');
  const badges = document.getElementById('recorded-badges');

  prompt.classList.add('hidden');
  display.classList.remove('hidden');
  badges.innerHTML = formatShortcutHtml(mods, keycode);
}

// -------------------------------------------------------------
// Modal & Rule Actions
// -------------------------------------------------------------
function openModal(rule = null) {
  const modal = document.getElementById('rule-modal');
  const title = document.getElementById('modal-title');
  const prompt = document.getElementById('recorder-prompt');
  const display = document.getElementById('recorded-display');
  const label = document.getElementById('recorder-label');
  const detectedTip = document.getElementById('detected-app-tip');
  if (detectedTip) detectedTip.textContent = '';

  state.recordingActive = false;
  state.recordedShortcut = null;

  if (rule) {
    title.textContent = 'Kuralı Düzenle';
    document.getElementById('form-rule-id').value = rule.id;
    document.getElementById('form-name').value = rule.name || '';
    document.getElementById('form-description').value = rule.description || '';
    document.getElementById('form-output').value = rule.output || '';
    document.getElementById('form-keycode').value = rule.keycode;

    document.getElementById('mod-cmd').checked = rule.modifiers.includes('cmd');
    document.getElementById('mod-alt').checked = rule.modifiers.includes('alt');
    document.getElementById('mod-ctrl').checked = rule.modifiers.includes('ctrl');
    document.getElementById('mod-shift').checked = rule.modifiers.includes('shift');

    document.getElementById('form-target-apps').value = (rule.target_apps || []).join(', ');
    document.getElementById('form-excluded-apps').value = (rule.excluded_apps || []).join(', ');

    updateRecordedDisplayFromManual();
  } else {
    title.textContent = 'Yeni Kural Ekle';
    document.getElementById('rule-form').reset();
    document.getElementById('form-rule-id').value = '';
    document.getElementById('form-target-apps').value = '';
    document.getElementById('form-excluded-apps').value = '';
    prompt.classList.remove('hidden');
    display.classList.add('hidden');
    label.textContent = 'Tuş kombinasyonunu kaydetmek için buraya tıklayın ve tuşlara basın';
  }

  modal.classList.remove('hidden');
}

function openModalWithKeycode(keycode) {
  openModal();
  const selectEl = document.getElementById('form-keycode');
  selectEl.value = keycode;

  // Default to ⌘ Command if empty
  document.getElementById('mod-cmd').checked = true;
  updateRecordedDisplayFromManual();
}

function closeModal() {
  document.getElementById('rule-modal').classList.add('hidden');
}

async function handleFormSubmit(e) {
  e.preventDefault();

  const id = document.getElementById('form-rule-id').value;
  const name = document.getElementById('form-name').value;
  const description = document.getElementById('form-description').value;
  const output = document.getElementById('form-output').value;
  const keycode = parseInt(document.getElementById('form-keycode').value, 10);

  const targetAppsStr = document.getElementById('form-target-apps').value.trim();
  const excludedAppsStr = document.getElementById('form-excluded-apps').value.trim();

  const targetApps = targetAppsStr ? targetAppsStr.split(',').map(s => s.trim()).filter(Boolean) : [];
  const excludedApps = excludedAppsStr ? excludedAppsStr.split(',').map(s => s.trim()).filter(Boolean) : [];

  if (isNaN(keycode)) {
    alert('Lütfen bir tuş seçin veya tuş kaydediciyi kullanın.');
    return;
  }
  if (!output) {
    alert('Lütfen yazılacak karakteri belirtin.');
    return;
  }

  const mods = [];
  if (document.getElementById('mod-cmd').checked) mods.push('cmd');
  if (document.getElementById('mod-alt').checked) mods.push('alt');
  if (document.getElementById('mod-ctrl').checked) mods.push('ctrl');
  if (document.getElementById('mod-shift').checked) mods.push('shift');

  if (mods.length === 0) {
    alert('En az bir modifier tuşu (⌘, ⌥, ⌃ veya ⇧) seçmelisiniz.');
    return;
  }

  const isSnippet = output.includes('{') || output.length > 4;

  const ruleData = {
    name: name || `Kısayol (${output})`,
    modifiers: mods,
    keycode: keycode,
    output: output,
    enabled: true,
    description: description,
    target_apps: targetApps,
    excluded_apps: excludedApps,
    is_snippet: isSnippet
  };

  try {
    if (id) {
      ruleData.id = id;
      await fetch(`/api/rules/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(ruleData)
      });
    } else {
      await fetch('/api/rules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(ruleData)
      });
    }

    closeModal();
    await loadRules();
    showNotification(id ? 'Kural güncellendi!' : 'Yeni kural eklendi!');
  } catch (err) {
    alert('Kaydetme hatası: ' + err);
  }
}

async function updateRule(rule) {
  try {
    await fetch(`/api/rules/${rule.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(rule)
    });
    updateStats();
    renderRules(state.rules);
    renderVirtualKeyboard();
  } catch (err) {
    console.error('Update rule error:', err);
  }
}

async function deleteRule(id) {
  try {
    await fetch(`/api/rules/${id}`, { method: 'DELETE' });
    await loadRules();
    showNotification('Kural silindi.');
  } catch (err) {
    alert('Silme hatası: ' + err);
  }
}

// -------------------------------------------------------------
// Live SSE Stream
// -------------------------------------------------------------
function initSSE() {
  const eventSource = new EventSource('/api/events');

  eventSource.addEventListener('trigger', (e) => {
    try {
      const data = JSON.parse(e.data);
      appendActivity(data);

      const statElem = document.getElementById('stat-total-triggers');
      const current = parseInt(statElem.textContent, 10) || 0;
      statElem.textContent = current + 1;
    } catch (err) {
      console.error('SSE parse error:', err);
    }
  });

  eventSource.onerror = () => {
    // Reconnects automatically by browser
  };
}

function appendActivity(data) {
  const stream = document.getElementById('activity-stream');
  const empty = stream.querySelector('.stream-empty');
  if (empty) empty.remove();

  const timeStr = new Date(data.timestamp).toLocaleTimeString();
  const item = document.createElement('div');
  item.className = 'activity-item';
  item.innerHTML = `
    <div class="activity-left">
      <span class="activity-badge">${escapeHtml(data.shortcut || 'Kısayol')}</span>
      <span>➔</span>
      <span class="output-pill" style="min-width:24px;height:24px;font-size:0.8rem;padding:0 6px;">${escapeHtml(data.output)}</span>
    </div>
    <span class="activity-time">${timeStr}</span>
  `;

  stream.insertBefore(item, stream.firstChild);

  while (stream.children.length > 20) {
    stream.removeChild(stream.lastChild);
  }
}

// -------------------------------------------------------------
// Helpers
// -------------------------------------------------------------
function escapeHtml(text) {
  if (!text) return '';
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

function showNotification(msg) {
  const toast = document.createElement('div');
  toast.style.position = 'fixed';
  toast.style.bottom = '24px';
  toast.style.right = '24px';
  toast.style.background = 'linear-gradient(135deg, #10b981, #059669)';
  toast.style.color = '#fff';
  toast.style.padding = '12px 20px';
  toast.style.borderRadius = '10px';
  toast.style.boxShadow = '0 8px 24px rgba(0,0,0,0.4)';
  toast.style.zIndex = '9999';
  toast.style.fontWeight = '600';
  toast.style.fontSize = '0.9rem';
  toast.style.animation = 'slide-in 0.2s ease-out';
  toast.textContent = msg;

  document.body.appendChild(toast);
  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transition = 'opacity 0.3s ease';
    setTimeout(() => toast.remove(), 300);
  }, 2500);
}
