// Keyboard Sensei — Web Interface Logic

let state = {
  status: null,
  rules: [],
  keycodes: [],
  recordingActive: false,
  recordedShortcut: null
};

// Browser event.code to macOS virtual keycode mapping
const CODE_TO_MACOS_KEYCODE = {
  // Special Turkish Q on ANSI mappings
  'Comma': 43,        // Turkish 'ö'
  'Period': 47,       // Turkish 'ç'
  'Slash': 44,        // Turkish '.'
  'Backslash': 42,    // Turkish ','
  'Semicolon': 41,    // Turkish 'ş'
  'Quote': 39,        // Turkish 'i'
  'BracketLeft': 33,  // Turkish 'ğ'
  'BracketRight': 30, // Turkish 'ü'
  'Minus': 27,        // '-'
  'Equal': 24,        // '='

  // Letters
  'KeyA': 0, 'KeyS': 1, 'KeyD': 2, 'KeyF': 3, 'KeyH': 4, 'KeyG': 5,
  'KeyZ': 6, 'KeyX': 7, 'KeyC': 8, 'KeyV': 9, 'KeyB': 11, 'KeyQ': 12,
  'KeyW': 13, 'KeyE': 14, 'KeyR': 15, 'KeyY': 16, 'KeyT': 17, 'KeyU': 32,
  'KeyI': 34, 'KeyO': 31, 'KeyP': 35, 'KeyJ': 38, 'KeyK': 40, 'KeyL': 37,
  'KeyN': 45, 'KeyM': 46,

  // Numbers
  'Digit1': 18, 'Digit2': 19, 'Digit3': 20, 'Digit4': 21, 'Digit5': 23,
  'Digit6': 22, 'Digit7': 26, 'Digit8': 28, 'Digit9': 25, 'Digit0': 29,

  // Controls
  'Space': 49, 'Enter': 36, 'Tab': 48, 'Backspace': 51, 'Escape': 53,
  'ArrowLeft': 123, 'ArrowRight': 124, 'ArrowDown': 125, 'ArrowUp': 126
};

document.addEventListener('DOMContentLoaded', () => {
  initApp();
});

async function initApp() {
  await loadKeycodes();
  await refreshStatus();
  await loadRules();
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

async function loadRules() {
  try {
    const res = await fetch('/api/config');
    if (!res.ok) return;
    const cfg = await res.json();
    state.rules = cfg.rules || [];
    renderRules(state.rules);
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
}

function updateStats() {
  const activeCount = state.rules.filter(r => r.enabled).length;
  document.getElementById('stat-active-rules').textContent = activeCount;
}

function renderRules(rules) {
  const container = document.getElementById('rules-container');
  if (!rules || rules.length === 0) {
    container.innerHTML = `
      <div class="stream-empty">
        Henüz tanımlı kural yok. Yukarıdaki "Şablonu Uygula" butonuna basabilir veya "+ Yeni Kural Ekle" ile ekleyebilirsiniz.
      </div>
    `;
    return;
  }

  container.innerHTML = rules.map(rule => {
    const shortcutHtml = formatShortcutHtml(rule.modifiers, rule.keycode);
    return `
      <div class="rule-card ${rule.enabled ? '' : 'disabled'}" data-id="${rule.id}">
        <div class="rule-left">
          <div class="keycap-combo">
            ${shortcutHtml}
          </div>
          <span class="arrow-divider">➔</span>
          <div class="output-pill">${escapeHtml(rule.output)}</div>
          <div class="rule-details">
            <span class="rule-name">${escapeHtml(rule.name || 'Özel Kural')}</span>
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

  // Group by priority
  const turkishAnsiGroup = document.createElement('optgroup');
  turkishAnsiGroup.label = 'Türkçe Q / ANSI Özel Tuşları (Önerilen)';

  const otherGroup = document.createElement('optgroup');
  otherGroup.label = 'Diğer Standart Tuşlar';

  state.keycodes.forEach(k => {
    const opt = document.createElement('option');
    opt.value = k.code;
    opt.textContent = `${k.turkish_desc} (Kod: ${k.code})`;

    if ([43, 47, 44, 42, 41, 39, 33, 30, 27, 24, 50].includes(k.code)) {
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

    // Prevent default browser shortcuts while recording (like Cmd+W, Cmd+S, etc.)
    e.preventDefault();
    e.stopPropagation();

    // Ignore pure modifier presses
    if (['Meta', 'Control', 'Alt', 'Shift'].includes(e.key)) {
      return;
    }

    const mods = [];
    if (e.ctrlKey) mods.push('ctrl');
    if (e.altKey) mods.push('alt');
    if (e.shiftKey) mods.push('shift');
    if (e.metaKey) mods.push('cmd');

    let keycode = CODE_TO_MACOS_KEYCODE[e.code];
    if (keycode === undefined) {
      // Fallback: check if keyCode is valid
      keycode = e.keyCode;
    }

    // Update state
    state.recordedShortcut = {
      modifiers: mods,
      keycode: keycode,
      code: e.code,
      key: e.key
    };

    // Update checkboxes
    document.getElementById('mod-cmd').checked = mods.includes('cmd');
    document.getElementById('mod-alt').checked = mods.includes('alt');
    document.getElementById('mod-ctrl').checked = mods.includes('ctrl');
    document.getElementById('mod-shift').checked = mods.includes('shift');

    // Update dropdown
    document.getElementById('form-keycode').value = keycode;

    // Show badges
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

  // Apply Preset
  document.getElementById('btn-apply-preset').addEventListener('click', async () => {
    if (!confirm('ANSI Türkçe Karakter Paketi şablonu yüklenecek. Onaylıyor musunuz?')) return;
    try {
      const res = await fetch('/api/presets/apply', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ preset: 'ansi_turkish' })
      });
      if (res.ok) {
        await loadRules();
        showNotification('Şablon başarıyla uygulandı!');
      }
    } catch (err) {
      alert('Şablon uygulanırken hata: ' + err);
    }
  });

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

    updateRecordedDisplayFromManual();
  } else {
    title.textContent = 'Yeni Kural Ekle';
    document.getElementById('rule-form').reset();
    document.getElementById('form-rule-id').value = '';
    prompt.classList.remove('hidden');
    display.classList.add('hidden');
    label.textContent = 'Tuş kombinasyonunu kaydetmek için buraya tıklayın ve tuşlara basın';
  }

  modal.classList.remove('hidden');
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

  const ruleData = {
    name: name || `Kısayol (${output})`,
    modifiers: mods,
    keycode: keycode,
    output: output,
    enabled: true,
    description: description
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

      // Increment stats counter in real-time
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

  // Keep max 20 items in stream
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
  // Simple toast effect
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
