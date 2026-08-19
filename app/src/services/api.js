import AsyncStorage from '@react-native-async-storage/async-storage';

const DEFAULT_API = 'http://10.12.208.217:8080';

class ApiService {
  constructor() {
    this.baseUrl = DEFAULT_API;
    this.playerId = null;
  }

  async init() {
    this.playerId = await AsyncStorage.getItem('playerId');
    if (!this.playerId) {
      this.playerId = 'player_' + Date.now() + '_' + Math.random().toString(36).slice(2, 8);
      await AsyncStorage.setItem('playerId', this.playerId);
    }
  }

  async _fetch(path, options = {}) {
    if (!this.playerId) await this.init();
    const url = `${this.baseUrl}/api/v1${path}`;
    const headers = {
      'Content-Type': 'application/json',
      'X-Player-ID': this.playerId,
      ...options.headers,
    };
    try {
      const resp = await fetch(url, { ...options, headers });
      const text = await resp.text();
      try { return JSON.parse(text); } catch { return { error: text }; }
    } catch (e) {
      console.warn('API error:', path, e.message);
      return null;
    }
  }

  async queryNymphs(lat, lng, radius = 200, limit = 20) {
    return this._fetch(`/nymphs?lat=${lat.toFixed(5)}&lng=${lng.toFixed(5)}&radius=${radius}&limit=${limit}`);
  }

  async digNymph(nymphId, lat, lng, deviation, tool = 'small_shovel') {
    return this._fetch(`/nymphs/${nymphId}/dig`, {
      method: 'POST',
      body: JSON.stringify({
        lat, lng,
        distance_m: 0.5 + Math.random() * 1.5,
        deviation_cm: deviation,
        angle_deg: Math.random() * 20,
        tool_used: tool,
      }),
    });
  }

  async getProfile() {
    return this._fetch('/player');
  }

  async getDailyStats() {
    return this._fetch('/player/daily-stats');
  }

  async getInventory() {
    return this._fetch('/player/inventory');
  }
}

export const apiService = new ApiService();
