import React, { useState, useRef, useEffect } from 'react';
import {
  View, Text, StyleSheet, TouchableOpacity, Animated,
  PanResponder, Vibration, Dimensions,
} from 'react-native';
import * as Haptics from 'expo-haptics';
import { useGame } from '../hooks/useGame';
import { useWeather } from '../hooks/useWeather';
import { apiService } from '../services/api';
import { COLORS, TUTORIAL_TIPS } from '../utils/constants';
import TutorialTip from '../components/TutorialTip';

const { width } = Dimensions.get('window');

export default function DigScreen() {
  const { trackedNymph, setTrackedNymph, player, addGold, addExp, showTutorial, currentTutorial, dismissTutorial } = useGame();
  const { weather, getSceneData, getGroundVisual } = useWeather();
  const [digState, setDigState] = useState('idle'); // idle | scanning | locked | digging | complete
  const [digProgress, setDigProgress] = useState(0);
  const [digCount, setDigCount] = useState(0);
  const [result, setResult] = useState(null);
  const [xMarkPhase, setXMarkPhase] = useState(0);
  const [xMarkLocked, setXMarkLocked] = useState(false);
  const maxDigs = 8;
  const pulseAnim = useRef(new Animated.Value(1)).current;

  // X-mark pulsing animation
  useEffect(() => {
    const anim = Animated.loop(
      Animated.sequence([
        Animated.timing(pulseAnim, { toValue: 0.3, duration: 800, useNativeDriver: true }),
        Animated.timing(pulseAnim, { toValue: 1, duration: 800, useNativeDriver: true }),
      ])
    );
    if (digState === 'scanning' && !xMarkLocked) anim.start();
    else { anim.stop(); pulseAnim.setValue(1); }
    return () => anim.stop();
  }, [digState, xMarkLocked]);

  // Auto-lock X mark after scanning
  useEffect(() => {
    if (digState !== 'scanning') return;
    const timer = setTimeout(() => {
      setXMarkLocked(true);
      setDigState('locked');
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      Vibration.vibrate(300);
    }, 2500);
    return () => clearTimeout(timer);
  }, [digState]);

  const startScanning = () => {
    if (!trackedNymph) return;
    setDigState('scanning');
    setXMarkLocked(false);
    setDigProgress(0);
    setDigCount(0);
    setResult(null);
    showTutorial('firstDig', TUTORIAL_TIPS.firstDig);
  };

  const performDig = async () => {
    if (digState !== 'locked' && digState !== 'digging') return;
    if (digState === 'locked') setDigState('digging');

    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);
    Vibration.vibrate(30);

    const newCount = digCount + 1;
    setDigCount(newCount);
    const progress = newCount / maxDigs;
    setDigProgress(progress);

    if (newCount >= maxDigs) {
      await completeDig();
    }
  };

  const completeDig = async () => {
    setDigState('complete');
    if (!trackedNymph) return;

    const deviation = Math.random() * 25; // simulated AR precision
    const data = await apiService.digNymph(
      trackedNymph.id, trackedNymph.lat, trackedNymph.lng, deviation
    );

    if (data?.success) {
      setResult({ success: true, nymph: trackedNymph, coin: data.coin_reward || trackedNymph.estimated_value, exp: data.exp_reward || trackedNymph.quality * 10 });
      addGold(data.coin_reward || Math.floor(trackedNymph.estimated_value));
      addExp(data.exp_reward || trackedNymph.quality * 10);
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
      Vibration.vibrate([100, 50, 200, 100, 300]);
      setTrackedNymph(null);
    } else {
      setResult({ success: false, reason: data?.fail_reason || '偏差太大，知了猴溜走了', rate: Math.round((data?.success_rate || 0) * 100) });
      Haptics.notificationAsync(Haptics.NotificationFeedbackType.Error);
    }
  };

  const reset = () => { setDigState('idle'); setResult(null); setDigProgress(0); setDigCount(0); };

  const scene = getSceneData();
  const nymph = trackedNymph;

  return (
    <View style={[st.container, { backgroundColor: scene.bg }]}>
      {/* Header */}
      <View style={st.header}>
        <Text style={st.headerTitle}>⛏️ 挖掘知了猴</Text>
        <Text style={st.weatherNote}>{getGroundVisual()} {scene.label}</Text>
      </View>

      {!nymph && digState === 'idle' && (
        <View style={st.emptyState}>
          <Text style={st.emptyIcon}>🗺️</Text>
          <Text style={st.emptyText}>请先在地图页面选择一只知了猴进行追踪</Text>
          <Text style={st.emptyHint}>追踪后自动进入挖掘模式</Text>
        </View>
      )}

      {nymph && (
        <>
          {/* Nymph info */}
          <View style={st.nymphInfo}>
            <Text style={st.speciesTag}>{nymph.species_name}</Text>
            <Text style={st.qualityStars}>{'★'.repeat(nymph.quality)}{'☆'.repeat(5 - nymph.quality)}</Text>
            <Text style={st.depthText}>📐 深度 {nymph.depth_cm}cm · 📏 {nymph.size_cm}cm</Text>

            {/* Depth visual */}
            <View style={st.depthBar}>
              <Text style={st.depthLabel}>地面</Text>
              <View style={st.depthTrack}>
                <View style={[st.depthFill, { height: `${nymph.depth_cm / 50 * 100}%` }]} />
              </View>
              <Text style={st.depthLabel}>{nymph.depth_cm}cm</Text>
            </View>
          </View>

          {/* X-mark area */}
          <TouchableOpacity style={st.xMarkArea} onPress={digState === 'locked' || digState === 'digging' ? performDig : startScanning} activeOpacity={0.8}>
            <Animated.Text style={[
              st.xMark,
              { opacity: xMarkLocked ? 1 : pulseAnim,
                color: xMarkLocked ? COLORS.gold : COLORS.danger,
                textShadowColor: xMarkLocked ? COLORS.gold : COLORS.danger,
              }
            ]}>
              ✖
            </Animated.Text>
            <Text style={st.xMarkHint}>
              {digState === 'idle' && '点击开始扫描地面'}
              {digState === 'scanning' && '正在扫描...移动手机对准地面'}
              {digState === 'locked' && '🔒 X标记已锁定! 点击挖掘!'}
              {digState === 'digging' && `挖掘中... ${digCount}/${maxDigs}`}
            </Text>
          </TouchableOpacity>

          {/* Progress bar */}
          {digState === 'digging' && (
            <View style={st.progressContainer}>
              <View style={[st.progressFill, { width: `${digProgress * 100}%` }]} />
            </View>
          )}

          {/* Dig button */}
          {(digState === 'locked' || digState === 'digging') && (
            <TouchableOpacity style={st.digBtn} onPress={performDig}>
              <Text style={st.digBtnText}>🪒 挖!</Text>
            </TouchableOpacity>
          )}

          {digState === 'idle' && (
            <TouchableOpacity style={st.digBtn} onPress={startScanning}>
              <Text style={st.digBtnText}>🔍 开始扫描</Text>
            </TouchableOpacity>
          )}

          {/* Result */}
          {digState === 'complete' && result && (
            <View style={[st.resultCard, result.success ? st.resultSuccess : st.resultFail]}>
              <Text style={st.resultTitle}>{result.success ? '🎉 挖到了!' : '😢 没挖到'}</Text>
              {result.success ? (
                <>
                  <Text style={st.resultSpecies}>{result.nymph?.species_name}</Text>
                  <Text style={st.resultReward}>💰 +¥{result.coin}  ✨ +{result.exp}EXP</Text>
                </>
              ) : (
                <Text style={st.resultReason}>{result.reason} (成功率 {result.rate}%)</Text>
              )}
              <TouchableOpacity style={st.againBtn} onPress={reset}>
                <Text style={st.againBtnText}>继续寻找</Text>
              </TouchableOpacity>
            </View>
          )}
        </>
      )}

      {currentTutorial && (
        <TutorialTip message={currentTutorial.message} onDismiss={() => dismissTutorial(currentTutorial.key)} />
      )}
    </View>
  );
}

const st = StyleSheet.create({
  container: { flex: 1, padding: 16, paddingTop: 50 },
  header: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 },
  headerTitle: { fontSize: 20, fontWeight: 'bold', color: COLORS.text },
  weatherNote: { fontSize: 12, color: COLORS.textDim },

  emptyState: { flex: 1, justifyContent: 'center', alignItems: 'center', gap: 12 },
  emptyIcon: { fontSize: 48 }, emptyText: { color: COLORS.textDim, fontSize: 15, textAlign: 'center' },
  emptyHint: { color: COLORS.textDim, fontSize: 12 },

  nymphInfo: {
    backgroundColor: COLORS.bgCard, borderRadius: 12, padding: 16,
    borderWidth: 1, borderColor: COLORS.border, alignItems: 'center', marginBottom: 16,
  },
  speciesTag: {
    backgroundColor: 'rgba(76,175,80,0.15)', paddingHorizontal: 16, paddingVertical: 6,
    borderRadius: 20, color: COLORS.primary, fontSize: 16, fontWeight: 'bold',
    borderWidth: 1, borderColor: COLORS.primaryDark,
  },
  qualityStars: { fontSize: 18, color: COLORS.gold, marginTop: 6, letterSpacing: 2 },
  depthText: { color: COLORS.textDim, fontSize: 13, marginTop: 4 },
  depthBar: { flexDirection: 'row', alignItems: 'center', gap: 8, marginTop: 10 },
  depthLabel: { color: COLORS.textDim, fontSize: 11 },
  depthTrack: { flex: 1, height: 30, backgroundColor: COLORS.border, borderRadius: 4, overflow: 'hidden' },
  depthFill: { backgroundColor: COLORS.gold, position: 'absolute', bottom: 0, left: 0, right: 0 },

  xMarkArea: { flex: 1, justifyContent: 'center', alignItems: 'center' },
  xMark: {
    fontSize: 100, textShadowOpacity: 0.3, textShadowRadius: 20, textShadowOffset: { width: 0, height: 0 },
  },
  xMarkHint: { color: COLORS.textDim, fontSize: 14, marginTop: 12, textAlign: 'center' },

  progressContainer: {
    height: 8, backgroundColor: COLORS.border, borderRadius: 4, overflow: 'hidden', marginBottom: 12,
  },
  progressFill: {
    height: '100%', backgroundColor: COLORS.primary, borderRadius: 4,
  },

  digBtn: {
    backgroundColor: COLORS.primaryDark, borderRadius: 16, paddingVertical: 16, alignItems: 'center',
    marginBottom: 20, shadowColor: COLORS.primary, shadowOpacity: 0.3, shadowRadius: 8,
  },
  digBtnText: { color: COLORS.white, fontSize: 20, fontWeight: 'bold' },

  resultCard: {
    position: 'absolute', bottom: 100, left: 20, right: 20,
    borderRadius: 16, padding: 24, alignItems: 'center',
  },
  resultSuccess: { backgroundColor: 'rgba(76,175,80,0.15)', borderWidth: 1, borderColor: COLORS.primary },
  resultFail: { backgroundColor: 'rgba(229,57,53,0.15)', borderWidth: 1, borderColor: COLORS.danger },
  resultTitle: { fontSize: 22, fontWeight: 'bold', color: COLORS.text, marginBottom: 8 },
  resultSpecies: { fontSize: 18, color: COLORS.gold, marginBottom: 4 },
  resultReward: { fontSize: 16, color: COLORS.gold, fontWeight: 'bold' },
  resultReason: { fontSize: 14, color: COLORS.textDim, textAlign: 'center' },
  againBtn: { backgroundColor: COLORS.primary, borderRadius: 12, paddingVertical: 12, paddingHorizontal: 30, marginTop: 12 },
  againBtnText: { color: COLORS.white, fontSize: 16, fontWeight: 'bold' },
});
