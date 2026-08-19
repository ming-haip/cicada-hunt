import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  View, Text, StyleSheet, TouchableOpacity, Modal,
  Dimensions, Animated, ActivityIndicator,
} from 'react-native';
import * as Location from 'expo-location';
import { useGame } from '../hooks/useGame';
import { useWeather } from '../hooks/useWeather';
import { apiService } from '../services/api';
import { COLORS, RARITY_COLORS, TUTORIAL_TIPS } from '../utils/constants';
import TutorialTip from '../components/TutorialTip';
import WeatherBackground from '../components/WeatherBackground';

const { width, height } = Dimensions.get('window');

// Simple map simulation — in production, use react-native-maps
function MiniMap({ player, nymphs, trackedNymph, onNymphPress, getTreeVisual }) {
  const mapW = width - 32;
  const mapH = height * 0.55;

  const toMapXY = (lat, lng) => {
    const scale = 8000; // pixels per degree
    const cx = mapW / 2;
    const cy = mapH / 2;
    return {
      x: cx + (lng - player.lng) * scale * Math.cos(player.lat * Math.PI / 180),
      y: cy - (lat - player.lat) * scale,
    };
  };

  return (
    <View style={[s.mapBox, { width: mapW, height: mapH }]}>
      {/* Trees scattered on map */}
      {Array.from({ length: 15 }).map((_, i) => {
        const x = (i * 137 + 50) % mapW;
        const y = (i * 89 + 30) % mapH;
        return (
          <Text key={`tree_${i}`} style={[s.mapTree, { left: x, top: y }]}>
            {getTreeVisual()}
          </Text>
        );
      })}

      {/* Nymph markers */}
      {nymphs.map((n, i) => {
        const { x, y } = toMapXY(n.lat, n.lng);
        if (x < -20 || x > mapW + 20 || y < -20 || y > mapH + 20) return null;
        const isTracked = trackedNymph?.id === n.id;
        const color = RARITY_COLORS[n.quality] || '#888';
        return (
          <TouchableOpacity
            key={n.id || i}
            style={[s.nymphDot, {
              left: x - (isTracked ? 16 : 8),
              top: y - (isTracked ? 16 : 8),
              backgroundColor: color,
              width: isTracked ? 32 : 16,
              height: isTracked ? 32 : 16,
              borderRadius: isTracked ? 16 : 8,
              borderWidth: isTracked ? 3 : 1,
              borderColor: isTracked ? COLORS.gold : COLORS.white,
            }]}
            onPress={() => onNymphPress(n)}
          >
            <Text style={{ fontSize: isTracked ? 16 : 8 }}>🐛</Text>
          </TouchableOpacity>
        );
      })}

      {/* Player marker */}
      <View style={[s.playerDot, {
        left: mapW / 2 - 12, top: mapH / 2 - 12,
      }]}>
        <Text style={{ fontSize: 14 }}>📍</Text>
      </View>

      {/* Compass */}
      <View style={s.compass}>
        <Text style={s.compassText}>N ↑</Text>
      </View>
    </View>
  );
}

export default function MapScreen() {
  const { player, addGold, setTrackedNymph, trackedNymph, dailyStats,
    showTutorial, currentTutorial, dismissTutorial } = useGame();
  const { weather, getNymphBonus, getTreeVisual, getGroundVisual, getSceneData } = useWeather();
  const [nymphs, setNymphs] = useState([]);
  const [playerLoc, setPlayerLoc] = useState({ lat: 39.9042, lng: 116.4074 });
  const [selectedNymph, setSelectedNymph] = useState(null);
  const [loading, setLoading] = useState(true);
  const [signalStrength, setSignalStrength] = useState(0);

  // GPS
  useEffect(() => {
    (async () => {
      const { status } = await Location.requestForegroundPermissionsAsync();
      if (status !== 'granted') return;
      const loc = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.High });
      setPlayerLoc({ lat: loc.coords.latitude, lng: loc.coords.longitude });
      setLoading(false);
    })();

    // Show first tutorial on map open
    setTimeout(() => showTutorial('firstMapOpen', TUTORIAL_TIPS.firstMapOpen), 1500);
  }, []);

  // Poll nymphs
  useEffect(() => {
    const poll = async () => {
      const data = await apiService.queryNymphs(playerLoc.lat, playerLoc.lng, 200, 15);
      if (data?.nymphs) setNymphs(data.nymphs.filter(n => n.status === 'active'));
    };
    poll();
    const interval = setInterval(poll, 5000);
    return () => clearInterval(interval);
  }, [playerLoc]);

  // Track proximity updates
  useEffect(() => {
    if (!trackedNymph) { setSignalStrength(0); return; }
    const interval = setInterval(() => {
      const dist = haversine(playerLoc.lat, playerLoc.lng, trackedNymph.lat, trackedNymph.lng);
      const sig = dist <= 2 ? 1 : dist >= 50 ? 0.05 : 0.05 + (50 - dist) / 48 * 0.95;
      setSignalStrength(sig);
    }, 1000);
    return () => clearInterval(interval);
  }, [trackedNymph, playerLoc]);

  const handleTrackNymph = (nymph) => {
    setTrackedNymph(nymph);
    setSelectedNymph(null);
    showTutorial('firstTrack', TUTORIAL_TIPS.firstTrack);
  };

  const scene = getSceneData();

  return (
    <WeatherBackground>
      <View style={s.container}>
        {/* Top bar */}
        <View style={s.topBar}>
          <View style={s.topLeft}>
            <Text style={s.weatherIcon}>{scene.icon}</Text>
            <Text style={s.topText}>{scene.label}</Text>
          </View>
          <View style={s.topRight}>
            <Text style={s.topText}>Lv.{player.level}</Text>
            <Text style={s.topGold}>💰 {player.gold}</Text>
            <Text style={s.topText}>⛏ {dailyStats.digs}/{dailyStats.limit}</Text>
          </View>
        </View>

        {/* Season & weather tip */}
        <View style={s.weatherTip}>
          <Text style={s.weatherTipText}>
            {weather.season === 'summer' && '☀️ 旺季：知了猴活跃中!'}
            {weather.season === 'winter' && '❄️ 冬季：知了猴冬眠中，数量极少'}
            {weather.season === 'spring' && '🌱 春季：知了猴开始苏醒'}
            {weather.season === 'autumn' && '🍂 秋季：知了猴活动减少'}
            {weather.isNight && ' 🌙 夜间加成!'}
          </Text>
          <Text style={s.bonusText}>出现率 ×{getNymphBonus().toFixed(1)}</Text>
        </View>

        {/* Map */}
        {loading ? (
          <ActivityIndicator size="large" color={COLORS.primary} style={{ marginTop: 40 }} />
        ) : (
          <MiniMap
            player={playerLoc}
            nymphs={nymphs}
            trackedNymph={trackedNymph}
            onNymphPress={setSelectedNymph}
            getTreeVisual={getTreeVisual}
          />
        )}

        {/* Signal bar (when tracking) */}
        {trackedNymph && (
          <View style={s.signalContainer}>
            <View style={[s.signalBar, { width: `${signalStrength * 100}%` }]} />
            <Text style={s.signalText}>
              {signalStrength > 0.8 ? '🔴 就在脚下!' :
               signalStrength > 0.5 ? '🟠 很近了' :
               signalStrength > 0.2 ? '🟡 接近中' : '⚪ 正在追踪'}
            </Text>
            <TouchableOpacity onPress={() => setTrackedNymph(null)} style={s.cancelTrack}>
              <Text style={{ color: COLORS.danger, fontSize: 12 }}>取消追踪</Text>
            </TouchableOpacity>
          </View>
        )}

        {/* Nymph count */}
        <View style={s.countBar}>
          <Text style={s.countText}>
            🐛 附近 {nymphs.length} 只知了猴 | {getGroundVisual()} {getTreeVisual()}
          </Text>
        </View>

        {/* Selected nymph modal */}
        <Modal visible={!!selectedNymph} transparent animationType="slide">
          <View style={s.modalOverlay}>
            <View style={s.modalCard}>
              <Text style={s.modalRarity}>
                {'★'.repeat(selectedNymph?.quality || 1)}{'☆'.repeat(5 - (selectedNymph?.quality || 1))}
              </Text>
              <Text style={s.modalSpecies}>{selectedNymph?.species_name}</Text>
              <Text style={s.modalDetail}>📏 {selectedNymph?.size_cm}cm | ⚖ {selectedNymph?.weight_g}g</Text>
              <Text style={s.modalDetail}>📐 深度 {selectedNymph?.depth_cm}cm</Text>
              <Text style={s.modalValue}>💰 ¥{selectedNymph?.estimated_value}</Text>
              <TouchableOpacity
                style={s.trackBtn}
                onPress={() => handleTrackNymph(selectedNymph)}
              >
                <Text style={s.trackBtnText}>🎯 追踪此目标</Text>
              </TouchableOpacity>
              <TouchableOpacity onPress={() => setSelectedNymph(null)} style={s.closeBtn}>
                <Text style={{ color: COLORS.textDim }}>关闭</Text>
              </TouchableOpacity>
            </View>
          </View>
        </Modal>

        {/* Tutorial tip */}
        {currentTutorial && (
          <TutorialTip
            message={currentTutorial.message}
            onDismiss={() => dismissTutorial(currentTutorial.key)}
          />
        )}

        {/* Dig action FAB */}
        {trackedNymph && signalStrength > 0.8 && (
          <TouchableOpacity style={s.fab}>
            <Text style={s.fabText}>⛏️ 开始挖掘!</Text>
          </TouchableOpacity>
        )}
      </View>
    </WeatherBackground>
  );
}

function haversine(lat1, lng1, lat2, lng2) {
  const R = 6371000;
  const dLat = (lat2 - lat1) * Math.PI / 180;
  const dLng = (lng2 - lng1) * Math.PI / 180;
  const a = Math.sin(dLat / 2) ** 2 + Math.cos(lat1 * Math.PI / 180) * Math.cos(lat2 * Math.PI / 180) * Math.sin(dLng / 2) ** 2;
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

const s = StyleSheet.create({
  container: { flex: 1, paddingHorizontal: 16 },
  topBar: { flexDirection: 'row', justifyContent: 'space-between', paddingTop: 50, paddingBottom: 8 },
  topLeft: { flexDirection: 'row', alignItems: 'center', gap: 6 },
  topRight: { flexDirection: 'row', alignItems: 'center', gap: 10 },
  weatherIcon: { fontSize: 20 },
  topText: { color: COLORS.text, fontSize: 13, fontWeight: '600' },
  topGold: { color: COLORS.gold, fontSize: 13, fontWeight: 'bold' },
  weatherTip: {
    flexDirection: 'row', justifyContent: 'space-between',
    backgroundColor: 'rgba(76,175,80,0.1)', padding: 8, borderRadius: 8,
    marginBottom: 8, borderWidth: 1, borderColor: COLORS.border,
  },
  weatherTipText: { color: COLORS.primary, fontSize: 12 },
  bonusText: { color: COLORS.gold, fontSize: 12, fontWeight: 'bold' },
  mapBox: {
    backgroundColor: COLORS.bgCard, borderRadius: 16,
    borderWidth: 1, borderColor: COLORS.border, overflow: 'hidden',
    alignSelf: 'center', position: 'relative',
  },
  mapTree: { position: 'absolute', fontSize: 20, opacity: 0.6 },
  nymphDot: {
    position: 'absolute', justifyContent: 'center', alignItems: 'center',
    shadowColor: '#fff', shadowOpacity: 0.3, shadowRadius: 4,
  },
  playerDot: {
    position: 'absolute', backgroundColor: COLORS.primary,
    borderRadius: 12, width: 24, height: 24,
    justifyContent: 'center', alignItems: 'center',
    borderWidth: 2, borderColor: COLORS.white,
    shadowColor: COLORS.primary, shadowOpacity: 0.5, shadowRadius: 8,
    zIndex: 10,
  },
  compass: { position: 'absolute', top: 8, right: 8 },
  compassText: { color: COLORS.textDim, fontSize: 11 },
  signalContainer: {
    marginTop: 10, height: 8, backgroundColor: 'rgba(255,255,255,0.1)',
    borderRadius: 4, overflow: 'hidden',
  },
  signalBar: {
    height: '100%', backgroundColor: COLORS.primary, borderRadius: 4,
    position: 'absolute',
  },
  signalText: { color: COLORS.textDim, fontSize: 11, marginTop: 4, textAlign: 'center' },
  cancelTrack: { alignItems: 'center', marginTop: 4 },
  countBar: { marginTop: 8, alignItems: 'center' },
  countText: { color: COLORS.textDim, fontSize: 12 },
  modalOverlay: {
    flex: 1, justifyContent: 'center', alignItems: 'center',
    backgroundColor: 'rgba(0,0,0,0.6)',
  },
  modalCard: {
    backgroundColor: COLORS.bgCard, borderRadius: 16, padding: 24,
    width: width * 0.8, alignItems: 'center', borderWidth: 1, borderColor: COLORS.border,
  },
  modalRarity: { fontSize: 20, color: COLORS.gold, letterSpacing: 2 },
  modalSpecies: { fontSize: 22, fontWeight: 'bold', color: COLORS.text, marginVertical: 8 },
  modalDetail: { fontSize: 14, color: COLORS.textDim, marginVertical: 2 },
  modalValue: { fontSize: 18, color: COLORS.gold, fontWeight: 'bold', marginVertical: 8 },
  trackBtn: {
    backgroundColor: COLORS.primary, borderRadius: 12, paddingVertical: 12,
    paddingHorizontal: 40, marginTop: 8, width: '100%', alignItems: 'center',
  },
  trackBtnText: { color: COLORS.white, fontSize: 16, fontWeight: 'bold' },
  closeBtn: { marginTop: 12 },
  fab: {
    position: 'absolute', bottom: 20, alignSelf: 'center',
    backgroundColor: COLORS.primaryDark, borderRadius: 30,
    paddingVertical: 16, paddingHorizontal: 40,
    shadowColor: COLORS.primary, shadowOpacity: 0.4, shadowRadius: 12,
    elevation: 8,
  },
  fabText: { color: COLORS.white, fontSize: 18, fontWeight: 'bold' },
});
