/**
 * 抓知了猴 (Cicada Hunt) — Mobile PWA Application
 *
 * Complete mobile web app for the cicada hunting AR game.
 * Connects to the Go backend API for real gameplay data.
 */

// ================================================================
// Configuration
// ================================================================
const CONFIG = {
  // API server — change this to your server's address
  // For local dev: http://localhost:8080
  // For LAN: http://192.168.x.x:8080
  API_BASE: (() => {
    const host = window.location.hostname;
    if (host === 'localhost' || host === '127.0.0.1') return 'http://localhost:8080';
    // When served from same host
    return window.location.origin + ':8080';
  })(),
  PLAYER_ID: 'mobile_player_' + Date.now(),
  REFRESH_INTERVAL: 5000, // ms between API polls
  DEFAULT_LAT: 39.9042,
  DEFAULT_LNG: 116.4074,
};

// ================================================================
// Application State
// ================================================================
const STATE = {
  currentScreen: 'screen-map',
  player: { lat: CONFIG.DEFAULT_LAT, lng: CONFIG.DEFAULT_LNG, heading: 0 },
  playerProfile: { level: 1, gold: 100, diamonds: 5 },
  dailyStats: { digs: 0, limit: 50 },
  nymphs: [],
  trackedNymph: null,
  isTracking: false,
  isDigging: false,
  digProgress: 0,
  digMax: 8,
  map: null,
  markers: [],
  heatmapLayer: null,
};

// ================================================================
// Initialization
// ================================================================
document.addEventListener('DOMContentLoaded', () => {
  initMap();
  initNavigation();
  initGPS();
  startPolling();
  loadProfile();
  initDigScreen();
  initCatchScreen();
  console.log('🦗 抓知了猴 App ready!');
  console.log('API:', CONFIG.API_BASE);
});

// ================================================================
// MAP
// ================================================================
function initMap() {
  STATE.map = L.map('map', {
    center: [CONFIG.DEFAULT_LAT, CONFIG.DEFAULT_LNG],
    zoom: 17,
    zoomControl: false,
    attributionControl: false,
  });

  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
  }).addTo(STATE.map);

  // Player marker
  const playerIcon = L.divIcon({
    html: '<div style="width:24px;height:24px;border-radius:50%;background:#4caf50;border:3px solid white;box-shadow:0 0 10px rgba(76,175,80,0.6);"></div>',
    className: '',
    iconSize: [24, 24],
    iconAnchor: [12, 12],
  });

  STATE.playerMarker = L.marker([CONFIG.DEFAULT_LAT, CONFIG.DEFAULT_LNG], { icon: playerIcon })
    .addTo(STATE.map)
    .bindPopup('📍 你的位置');

  // Map click to simulate placing nymphs for demo
  STATE.map.on('click', (e) => {
    if (STATE.isTracking) return;
    showToast(`点击位置: ${e.latlng.lat.toFixed(5)}, ${e.latlng.lng.toFixed(5)}`);
  });

  setTimeout(() => STATE.map.invalidateSize(), 500);
}

// ================================================================
// GPS
// ================================================================
function initGPS() {
  if (!navigator.geolocation) {
    console.warn('GPS not available, using default location');
    updatePlayerPosition(CONFIG.DEFAULT_LAT, CONFIG.DEFAULT_LNG);
    return;
  }

  navigator.geolocation.watchPosition(
    (pos) => {
      const lat = pos.coords.latitude;
      const lng = pos.coords.longitude;
      updatePlayerPosition(lat, lng);
    },
    (err) => {
      console.warn('GPS error:', err.message);
      showToast('⚠️ 无法获取GPS，使用默认位置');
      updatePlayerPosition(CONFIG.DEFAULT_LAT, CONFIG.DEFAULT_LNG);
    },
    { enableHighAccuracy: true, maximumAge: 5000, timeout: 10000 }
  );
}

function updatePlayerPosition(lat, lng) {
  STATE.player.lat = lat;
  STATE.player.lng = lng;

  if (STATE.playerMarker) {
    STATE.playerMarker.setLatLng([lat, lng]);
  }
  if (STATE.map && !STATE.isTracking) {
    STATE.map.setView([lat, lng], STATE.map.getZoom(), { animate: true });
  }
}

// ================================================================
// API Calls
// ================================================================
async function apiCall(path, options = {}) {
  const url = CONFIG.API_BASE + '/api/v1' + path;
  const headers = {
    'Content-Type': 'application/json',
    'X-Player-ID': CONFIG.PLAYER_ID,
    ...options.headers,
  };

  try {
    const resp = await fetch(url, { ...options, headers });
    if (!resp.ok) {
      // For non-200 responses, still try to parse body
      const body = await resp.text();
      try { return JSON.parse(body); } catch { return { error: body }; }
    }
    return await resp.json();
  } catch (err) {
    console.warn('API call failed:', path, err.message);
    return null;
  }
}

async function queryNymphs() {
  const { lat, lng } = STATE.player;
  const data = await apiCall(`/nymphs?lat=${lat.toFixed(5)}&lng=${lng.toFixed(5)}&radius=200&limit=20`);
  if (data && data.nymphs) {
    STATE.nymphs = data.nymphs.filter(n => n.status === 'active');
    updateNymphMarkers();
  }
  return data;
}

async function digNymph(nymphId, deviation) {
  const { lat, lng } = STATE.player;
  const body = JSON.stringify({
    lat, lng,
    distance_m: 0.5 + Math.random() * 1.5,
    deviation_cm: deviation,
    angle_deg: Math.random() * 20,
    tool_used: 'small_shovel',
  });

  return await apiCall(`/nymphs/${nymphId}/dig`, {
    method: 'POST',
    body,
  });
}

async function getDailyStats() {
  const data = await apiCall('/player/daily-stats');
  if (data) {
    STATE.dailyStats.digs = data.today_digs || 0;
    STATE.dailyStats.limit = data.daily_limit || 50;
    updateTopBar();
  }
}

async function loadProfile() {
  const data = await apiCall('/player');
  if (data && data.player) {
    STATE.playerProfile = data.player;
    updateProfileScreen();
  }
}

// ================================================================
// Nymph Markers
// ================================================================
function updateNymphMarkers() {
  // Clear existing markers
  STATE.markers.forEach(m => STATE.map.removeLayer(m));
  STATE.markers = [];

  STATE.nymphs.forEach((nymph, i) => {
    const dist = haversineDistance(
      STATE.player.lat, STATE.player.lng,
      nymph.lat, nymph.lng
    );

    if (dist > 250) return; // Don't show far markers

    const rarityColors = ['', '#888', '#4caf50', '#2196f3', '#9c27b0', '#ffc107'];
    const color = rarityColors[nymph.quality] || '#888';
    const size = dist < 30 ? 20 : (dist < 100 ? 14 : 10);

    const icon = L.divIcon({
      html: `<div style="
        width:${size}px;height:${size}px;border-radius:50%;
        background:${color};border:2px solid white;
        box-shadow:0 0 ${size/2}px ${color};
        display:flex;align-items:center;justify-content:center;
        font-size:${size*0.5}px;">🐛</div>`,
      className: '',
      iconSize: [size, size],
      iconAnchor: [size/2, size/2],
    });

    const marker = L.marker([nymph.lat, nymph.lng], { icon })
      .addTo(STATE.map)
      .bindPopup(`
        <b>${nymph.species_name}</b><br>
        ⭐ ${'★'.repeat(nymph.quality)}${'☆'.repeat(5-nymph.quality)}<br>
        📏 ${nymph.size_cm}cm | 📐 ${nymph.depth_cm}cm<br>
        💰 ¥${nymph.estimated_value}<br>
        <button onclick="trackNymph('${nymph.id}')"
          style="background:#4caf50;color:white;border:none;padding:6px 12px;border-radius:6px;margin-top:6px;">
          🎯 追踪
        </button>
      `);

    marker.on('click', () => showNymphPopup(nymph));
    STATE.markers.push(marker);
  });
}

// ================================================================
// Nymph Tracking & Popup
// ================================================================
function showNymphPopup(nymph) {
  document.getElementById('popup-species').textContent = nymph.species_name;
  document.getElementById('popup-rarity').textContent = '★'.repeat(nymph.quality) + '☆'.repeat(5 - nymph.quality);
  document.getElementById('popup-size').textContent = nymph.size_cm + 'cm';
  document.getElementById('popup-weight').textContent = nymph.weight_g + 'g';
  document.getElementById('popup-depth').textContent = nymph.depth_cm + 'cm';
  document.getElementById('popup-value').textContent = nymph.estimated_value;
  document.getElementById('popup').classList.remove('hidden');

  document.getElementById('btn-track').onclick = () => trackNymph(nymph.id);
  document.querySelector('.popup-close').onclick = () => {
    document.getElementById('popup').classList.add('hidden');
  };
}

function trackNymph(nymphId) {
  const nymph = STATE.nymphs.find(n => n.id === nymphId);
  if (!nymph) { showToast('目标已失效'); return; }

  STATE.trackedNymph = nymph;
  STATE.isTracking = true;
  document.getElementById('popup').classList.add('hidden');
  showToast(`🎯 追踪 ${nymph.species_name}`);

  // Center map on target
  STATE.map.setView([nymph.lat, nymph.lng], 18, { animate: true });

  // Show proximity bar
  document.getElementById('proximity-bar').style.display = 'block';

  // Start proximity updates
  updateProximity();
}

function updateProximity() {
  if (!STATE.isTracking || !STATE.trackedNymph) {
    document.getElementById('proximity-bar').style.display = 'none';
    return;
  }

  const dist = haversineDistance(
    STATE.player.lat, STATE.player.lng,
    STATE.trackedNymph.lat, STATE.trackedNymph.lng
  );

  // Signal strength: 50m → 0.1, 1m → 1.0
  const signal = dist <= 1 ? 1.0 : dist >= 50 ? 0.05 : 0.05 + (50 - dist) / 49 * 0.95;
  const pct = Math.round(signal * 100);

  document.getElementById('prox-fill').style.width = pct + '%';
  document.getElementById('prox-label').textContent =
    dist < 3 ? '🔴 就在脚下!' :
    dist < 10 ? `🟠 很近! ${dist.toFixed(0)}m` :
    dist < 30 ? `🟡 ${dist.toFixed(0)}m` :
    `⚪ ${dist.toFixed(0)}m`;

  if (dist < 3) {
    // Trigger haptic if available
    if (navigator.vibrate) navigator.vibrate([200, 100, 200]);
    document.getElementById('prox-label').textContent = '🔴 到达! 开始挖掘!';
  }

  requestAnimationFrame(() => {
    if (STATE.isTracking) setTimeout(updateProximity, 1000);
  });
}

// ================================================================
// DIG MODE
// ================================================================
function initDigScreen() {
  const btnDig = document.getElementById('btn-dig');
  const xMark = document.getElementById('x-mark');
  const digArea = document.getElementById('dig-area');
  const progressContainer = document.getElementById('dig-progress-container');

  // Enable dig button when tracking
  setInterval(() => {
    if (STATE.isTracking && STATE.trackedNymph && !STATE.isDigging) {
      btnDig.disabled = false;
      btnDig.querySelector('span:last-child').textContent = '开始挖掘';
    } else if (!STATE.isTracking && !STATE.isDigging) {
      btnDig.disabled = true;
    }

    // X-mark oscillation → convergence
    if (STATE.isDigging && xMark.classList.contains('x-mark-scanning')) {
      // Simulate convergence
      const timeInDig = (Date.now() - STATE._digStartTime) / 1000;
      if (timeInDig > 2) {
        xMark.classList.remove('x-mark-scanning');
        xMark.classList.add('x-mark-locked');
        document.getElementById('dig-hint').textContent = 'X标记已锁定! 点击铲子挖掘';
        if (navigator.vibrate) navigator.vibrate(300);
      }
    }
  }, 500);

  // Dig button click → simulate swipe-based digging
  btnDig.addEventListener('click', () => {
    if (!STATE.isTracking || !STATE.trackedNymph) return;

    if (!STATE.isDigging) {
      // Start digging
      startDigging();
    } else {
      // Perform a dig
      performDig();
    }
  });

  // Also allow tapping the X-mark area to dig
  digArea.addEventListener('click', () => {
    if (STATE.isDigging) {
      performDig();
    } else if (STATE.isTracking) {
      startDigging();
    }
  });

  document.getElementById('btn-dig-again').addEventListener('click', () => {
    resetDigScreen();
    switchScreen('screen-map');
  });
}

function startDigging() {
  STATE.isDigging = true;
  STATE._digStartTime = Date.now();
  STATE.digProgress = 0;

  const nymph = STATE.trackedNymph;
  document.getElementById('dig-species').textContent = nymph.species_name;
  document.getElementById('depth-value').textContent = nymph.depth_cm + 'cm';
  document.getElementById('depth-fill').style.width = (nymph.depth_cm / 50 * 100) + '%';
  document.getElementById('dig-target-info').classList.remove('hidden');
  document.getElementById('dig-progress-container').classList.remove('hidden');
  document.getElementById('x-mark').classList.add('x-mark-scanning');
  document.getElementById('x-mark').classList.remove('x-mark-locked');
  document.getElementById('dig-hint').textContent = '正在扫描地面...移动手机对准目标';

  const btnDig = document.getElementById('btn-dig');
  btnDig.querySelector('span:last-child').textContent = '🪒 挖!';
  btnDig.style.animation = 'pulse 0.5s infinite';
}

function performDig() {
  if (!STATE.isDigging) return;

  STATE.digProgress++;
  const progress = STATE.digProgress / STATE.digMax;
  document.getElementById('dig-progress-fill').style.width = (progress * 100) + '%';
  document.getElementById('dig-count').textContent = `挖掘中... ${STATE.digProgress}/${STATE.digMax}`;

  // Haptic feedback
  if (navigator.vibrate) navigator.vibrate(50);

  if (STATE.digProgress >= STATE.digMax) {
    completeDigging();
  }
}

async function completeDigging() {
  STATE.isDigging = false;
  const btnDig = document.getElementById('btn-dig');
  btnDig.style.animation = '';
  btnDig.querySelector('span:last-child').textContent = '挖掘中...';

  const nymph = STATE.trackedNymph;
  const deviation = Math.random() * 20; // Simulated deviation

  const result = await digNymph(nymph.id, deviation);

  const resultDiv = document.getElementById('dig-result');
  const titleEl = document.getElementById('result-title');
  const detailsEl = document.getElementById('result-details');

  resultDiv.classList.remove('hidden');

  if (result && result.success) {
    titleEl.textContent = '🎉 挖到了!';
    titleEl.style.color = '#ffc107';
    detailsEl.innerHTML = `
      <p style="font-size:18px">${nymph.species_name}</p>
      <p>⭐ ${'★'.repeat(nymph.quality)}</p>
      <p>💰 +¥${result.coin_reward || nymph.estimated_value}</p>
      <p>✨ +${result.exp_reward || nymph.quality * 10} EXP</p>
    `;
    STATE.playerProfile.gold += (result.coin_reward || Math.floor(nymph.estimated_value));
    STATE.trackedNymph = null;
    STATE.isTracking = false;
    updateTopBar();
  } else {
    titleEl.textContent = '😢 没挖到';
    titleEl.style.color = '#e53935';
    const reason = result?.fail_reason || '偏差太大，知了猴溜走了';
    detailsEl.innerHTML = `<p>${reason}</p><p>成功率: ${Math.round((result?.success_rate || 0) * 100)}%</p>`;
  }
}

function resetDigScreen() {
  STATE.isDigging = false;
  STATE.digProgress = 0;
  document.getElementById('dig-progress-fill').style.width = '0%';
  document.getElementById('dig-count').textContent = '';
  document.getElementById('dig-target-info').classList.add('hidden');
  document.getElementById('dig-progress-container').classList.add('hidden');
  document.getElementById('dig-result').classList.add('hidden');
  document.getElementById('x-mark').classList.remove('x-mark-locked', 'x-mark-scanning');
  document.getElementById('dig-hint').textContent = '将手机对准地面，寻找X标记...';
  document.getElementById('btn-dig').style.animation = '';
}

// ================================================================
// CATCH MODE
// ================================================================
function initCatchScreen() {
  document.getElementById('btn-swing')?.addEventListener('click', () => {
    showToast('🥅 挥网! (捕蝉功能开发中)');
    if (navigator.vibrate) navigator.vibrate([100, 50, 200]);
  });
}

// ================================================================
// NAVIGATION
// ================================================================
function initNavigation() {
  document.querySelectorAll('.btn-mode').forEach(btn => {
    btn.addEventListener('click', () => {
      const screenId = btn.dataset.screen;
      switchScreen(screenId);
    });
  });

  document.querySelectorAll('.btn-back').forEach(btn => {
    btn.addEventListener('click', () => {
      switchScreen(btn.dataset.screen || 'screen-map');
    });
  });
}

function switchScreen(screenId) {
  document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
  document.querySelectorAll('.btn-mode').forEach(b => b.classList.remove('active'));

  const screen = document.getElementById(screenId);
  if (screen) screen.classList.add('active');

  const navBtn = document.querySelector(`[data-screen="${screenId}"]`);
  if (navBtn) navBtn.classList.add('active');

  STATE.currentScreen = screenId;

  // Refresh map size when switching to map
  if (screenId === 'screen-map' && STATE.map) {
    setTimeout(() => STATE.map.invalidateSize(), 300);
  }

  // Load profile when switching to profile
  if (screenId === 'screen-profile') {
    loadProfile();
  }
}

// ================================================================
// TOP BAR & POLLING
// ================================================================
function updateTopBar() {
  document.getElementById('top-level').textContent = STATE.playerProfile.level;
  document.getElementById('top-gold').textContent = STATE.playerProfile.gold;
  document.getElementById('top-digs').textContent = STATE.dailyStats.digs;
}

function startPolling() {
  // Initial load
  queryNymphs();
  getDailyStats();

  // Periodic refresh
  setInterval(() => {
    queryNymphs();
    getDailyStats();
    updateTopBar();
  }, CONFIG.REFRESH_INTERVAL);
}

function updateProfileScreen() {
  const p = STATE.playerProfile;
  document.getElementById('profile-name').textContent = p.nickname || 'Player';
  document.getElementById('profile-level').textContent = p.level || 1;
  document.getElementById('profile-gold').textContent = p.gold_coins || 100;
  document.getElementById('profile-diamonds').textContent = p.diamonds || 5;

  document.getElementById('player-stats').innerHTML = `
    <div>⛏ 总挖掘: ${p.total_digs || 0} 次</div>
    <div>🐛 知了猴: ${p.total_nymphs || 0} 只</div>
    <div>🥅 捕蝉: ${p.total_catches || 0} 次</div>
    <div>💎 稀有: ${p.rare_nymphs || 0} 只</div>
    <div>🏆 传说: ${p.legendary_captures || 0} 只</div>
  `;

  // Load tools
  loadTools();
}

async function loadTools() {
  try {
    const data = await apiCall('/player/inventory');
    if (!data || !data.tools) return;

    const toolsList = document.getElementById('tools-list');
    toolsList.innerHTML = '';

    Object.entries(data.tools).forEach(([id, tool]) => {
      if (tool.type !== 'shovel' && tool.type !== 'net') return;
      toolsList.innerHTML += `
        <div class="tool-item">
          <div>
            <div class="tool-name">${tool.type === 'shovel' ? '🪒' : '🥅'} ${tool.name}</div>
            <div class="tool-desc">${tool.description}</div>
          </div>
          <div class="tool-level">Lv.${tool.level}</div>
        </div>
      `;
    });
  } catch(e) { /* ignore */ }
}

// ================================================================
// UTILITIES
// ================================================================
function haversineDistance(lat1, lng1, lat2, lng2) {
  const R = 6371000;
  const dLat = (lat2 - lat1) * Math.PI / 180;
  const dLng = (lng2 - lng1) * Math.PI / 180;
  const a = Math.sin(dLat/2) ** 2 +
    Math.cos(lat1 * Math.PI/180) * Math.cos(lat2 * Math.PI/180) *
    Math.sin(dLng/2) ** 2;
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

function showToast(msg) {
  let toast = document.getElementById('toast');
  if (!toast) {
    toast = document.createElement('div');
    toast.id = 'toast';
    toast.style.cssText = `
      position:fixed;bottom:140px;left:50%;transform:translateX(-50%);
      background:rgba(0,0,0,0.85);color:white;padding:10px 20px;
      border-radius:20px;font-size:14px;z-index:100;
      pointer-events:none;transition:opacity 0.3s;
    `;
    document.body.appendChild(toast);
  }

  toast.textContent = msg;
  toast.style.opacity = '1';
  clearTimeout(toast._timeout);
  toast._timeout = setTimeout(() => { toast.style.opacity = '0'; }, 2000);
}

// ================================================================
// Service Worker Registration (PWA)
// ================================================================
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js').catch(() => {});
}

// Expose for popup onclick
window.trackNymph = trackNymph;
