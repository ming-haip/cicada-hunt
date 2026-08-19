import React, { useState, useEffect, useRef } from 'react';
import {
  View, Text, StyleSheet, TouchableOpacity, Animated, Vibration,
} from 'react-native';
import * as Haptics from 'expo-haptics';
import { useGame } from '../hooks/useGame';
import { useWeather } from '../hooks/useWeather';
import { COLORS, TUTORIAL_TIPS } from '../utils/constants';
import TutorialTip from '../components/TutorialTip';

export default function CatchScreen() {
  const { addGold, addExp, showTutorial, currentTutorial, dismissTutorial } = useGame();
  const { weather, getSceneData, getTreeVisual } = useWeather();
  const [scanning, setScanning] = useState(true);
  const [cicada, setCicada] = useState(null);
  const [catchState, setCatchState] = useState('scanning'); // scanning | approaching | ready | swinging | result
  const [catchResult, setCatchResult] = useState(null);
  const scanRotate = useRef(new Animated.Value(0)).current;

  // Radar scan rotation animation
  useEffect(() => {
    const anim = Animated.loop(
      Animated.timing(scanRotate, {
        toValue: 360, duration: 3000, useNativeDriver: true,
      })
    );
    anim.start();
    return () => anim.stop();
  }, []);

  // Simulate finding a cicada
  useEffect(() => {
    if (!scanning) return;
    const timer = setTimeout(() => {
      const species = ['黑蚱蝉', '鸣鸣蝉', '蟪蛄', '寒蝉', '蒙古寒蝉'][Math.floor(Math.random() * 5)];
      const rarity = [1, 1, 2, 2, 3, 3, 4, 5][Math.floor(Math.random() * 8)];
      setCicada({ id: 'cic_' + Date.now(), species, rarity, distance: 45 + Math.random() * 100, value: (rarity * 5 + Math.random() * 10).toFixed(0) });
      setScanning(false);
      setCatchState('approaching');
      showTutorial('firstCatch', TUTORIAL_TIPS.firstCatch);
    }, 3000 + Math.random() * 4000);
    return () => clearTimeout(timer);
  }, [scanning]);

  const swingNet = async () => {
    setCatchState('swinging');
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Heavy);
    Vibration.vibrate([100, 50, 200]);

    // Simulate catch result
    setTimeout(() => {
      const success = Math.random() < 0.6;
      if (success) {
        setCatchResult({ success: true, cicada, coin: cicada.value * 2, exp: cicada.rarity * 15 });
        addGold(cicada.value * 2);
        addExp(cicada.rarity * 15);
      } else {
        setCatchResult({ success: false, reason: '蝉躲开了！它飞走了...' });
      }
      setCatchState('result');
    }, 500);
  };

  const reset = () => { setScanning(true); setCicada(null); setCatchState('scanning'); setCatchResult(null); };

  const scene = getSceneData();
  const spinInterp = scanRotate.interpolate({ inputRange: [0, 360], outputRange: ['0deg', '360deg'] });

  return (
    <View style={[cs.container, { backgroundColor: scene.bg }]}>
      <View style={cs.header}>
        <Text style={cs.headerTitle}>🥅 捕蝉雷达</Text>
        <Text style={cs.treeIcon}>{getTreeVisual()}</Text>
      </View>

      {/* Radar display */}
      <View style={cs.radarOuter}>
        <View style={cs.radarInner}>
          <Animated.View style={[cs.scanLine, { transform: [{ rotate: spinInterp }] }]} />
          {/* Grid lines */}
          <View style={cs.radarRing} />
          <View style={[cs.radarRing, { width: '66%', height: '66%', borderRadius: 100 }]} />
          <View style={[cs.radarRing, { width: '33%', height: '33%', borderRadius: 60 }]} />

          {/* Cicada blip */}
          {cicada && (
            <View style={cs.blip}>
              <Text style={cs.blipText}>🐞</Text>
            </View>
          )}

          {scanning && (
            <Text style={cs.scanningText}>扫描中...</Text>
          )}
        </View>
      </View>

      {/* Info */}
      {cicada && (
        <View style={cs.infoCard}>
          <Text style={cs.infoSpecies}>{cicada.species}</Text>
          <Text style={cs.infoStars}>{'★'.repeat(cicada.rarity)}</Text>
          <Text style={cs.infoDist}>📡 距离 {cicada.distance.toFixed(0)}m</Text>
          <Text style={cs.infoValue}>💰 ¥{cicada.value}</Text>
        </View>
      )}

      {/* Status */}
      <Text style={cs.status}>
        {catchState === 'scanning' && '📡 雷达正在扫描周边蝉信号...'}
        {catchState === 'approaching' && '🎯 发现蝉! 悄悄靠近... (从背后接近!)'}
        {catchState === 'ready' && '✅ 已进入捕蝉范围! 挥网!'}
        {catchState === 'swinging' && '🥅 挥网中...'}
      </Text>

      {/* Swing button */}
      {cicada && catchState !== 'swinging' && catchState !== 'result' && (
        <TouchableOpacity style={cs.swingBtn} onPress={swingNet}>
          <Text style={cs.swingText}>🥅 挥网捕捉!</Text>
        </TouchableOpacity>
      )}

      {/* Result */}
      {catchState === 'result' && catchResult && (
        <View style={[cs.resultCard, catchResult.success ? cs.resultWin : cs.resultLose]}>
          <Text style={cs.resultTitle}>{catchResult.success ? '🎉 抓到了!' : '😢 让它跑了'}</Text>
          {catchResult.success ? (
            <>
              <Text style={cs.resultSpecies}>{catchResult.cicada?.species}</Text>
              <Text style={cs.resultReward}>💰 +¥{catchResult.coin}  ✨ +{catchResult.exp}EXP</Text>
            </>
          ) : (
            <Text style={cs.resultReason}>{catchResult.reason}</Text>
          )}
          <TouchableOpacity style={cs.againBtn} onPress={reset}>
            <Text style={cs.againText}>继续扫描</Text>
          </TouchableOpacity>
        </View>
      )}

      {currentTutorial && (
        <TutorialTip message={currentTutorial.message} onDismiss={() => dismissTutorial(currentTutorial.key)} />
      )}
    </View>
  );
}

const cs = StyleSheet.create({
  container: { flex: 1, padding: 16, paddingTop: 50, alignItems: 'center' },
  header: { flexDirection: 'row', alignItems: 'center', gap: 10, marginBottom: 20 },
  headerTitle: { fontSize: 20, fontWeight: 'bold', color: COLORS.text },
  treeIcon: { fontSize: 28 },

  radarOuter: {
    width: 250, height: 250, borderRadius: 125,
    backgroundColor: COLORS.bg, borderWidth: 3, borderColor: COLORS.primaryDark,
    justifyContent: 'center', alignItems: 'center',
    shadowColor: COLORS.primary, shadowOpacity: 0.2, shadowRadius: 20,
  },
  radarInner: {
    width: 230, height: 230, borderRadius: 115,
    backgroundColor: COLORS.bgCard, justifyContent: 'center', alignItems: 'center',
    overflow: 'hidden',
  },
  scanLine: {
    position: 'absolute', width: 2, height: '100%',
    backgroundColor: COLORS.primary, opacity: 0.6,
  },
  radarRing: {
    position: 'absolute', borderWidth: 1, borderColor: 'rgba(76,175,80,0.2)',
    width: '100%', height: '100%', borderRadius: 115,
  },
  blip: {
    position: 'absolute', top: '30%', right: '30%',
    backgroundColor: 'rgba(255,193,7,0.3)', borderRadius: 12,
    padding: 4,
  },
  blipText: { fontSize: 18 },
  scanningText: { color: COLORS.primary, fontSize: 14, fontWeight: '600' },

  infoCard: {
    backgroundColor: COLORS.bgCard, borderRadius: 12, padding: 16,
    borderWidth: 1, borderColor: COLORS.border, marginTop: 16,
    alignItems: 'center', width: '100%',
  },
  infoSpecies: { fontSize: 18, fontWeight: 'bold', color: COLORS.text },
  infoStars: { fontSize: 16, color: COLORS.gold, marginVertical: 4, letterSpacing: 2 },
  infoDist: { fontSize: 14, color: COLORS.textDim },
  infoValue: { fontSize: 16, color: COLORS.gold, fontWeight: 'bold', marginTop: 4 },

  status: { color: COLORS.textDim, fontSize: 14, marginTop: 16, textAlign: 'center' },

  swingBtn: {
    backgroundColor: COLORS.primary, borderRadius: 16, paddingVertical: 16,
    paddingHorizontal: 50, marginTop: 20, shadowColor: COLORS.primary,
    shadowOpacity: 0.4, shadowRadius: 10,
  },
  swingText: { color: COLORS.white, fontSize: 20, fontWeight: 'bold' },

  resultCard: { borderRadius: 16, padding: 24, marginTop: 20, alignItems: 'center', width: '100%' },
  resultWin: { backgroundColor: 'rgba(76,175,80,0.15)', borderWidth: 1, borderColor: COLORS.primary },
  resultLose: { backgroundColor: 'rgba(229,57,53,0.15)', borderWidth: 1, borderColor: COLORS.danger },
  resultTitle: { fontSize: 22, fontWeight: 'bold', color: COLORS.text, marginBottom: 8 },
  resultSpecies: { fontSize: 18, color: COLORS.gold },
  resultReward: { fontSize: 16, color: COLORS.gold, fontWeight: 'bold', marginTop: 4 },
  resultReason: { fontSize: 14, color: COLORS.textDim, textAlign: 'center' },
  againBtn: { backgroundColor: COLORS.bgSurface, borderRadius: 12, paddingVertical: 12, paddingHorizontal: 30, marginTop: 12, borderWidth: 1, borderColor: COLORS.border },
  againText: { color: COLORS.text, fontSize: 15, fontWeight: '600' },
});
