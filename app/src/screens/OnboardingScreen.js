import React, { useState, useRef } from 'react';
import {
  View, Text, StyleSheet, Dimensions, TouchableOpacity,
  ScrollView, Animated,
} from 'react-native';
import { useGame } from '../hooks/useGame';
import { useWeather } from '../hooks/useWeather';
import { COLORS, ONBOARDING_STEPS } from '../utils/constants';

const { width } = Dimensions.get('window');

export default function OnboardingScreen() {
  const { completeOnboarding } = useGame();
  const { getTreeVisual } = useWeather();
  const [step, setStep] = useState(0);
  const fadeAnim = useRef(new Animated.Value(1)).current;
  const scrollRef = useRef(null);

  const goNext = () => {
    if (step < ONBOARDING_STEPS.length - 1) {
      Animated.sequence([
        Animated.timing(fadeAnim, { toValue: 0, duration: 200, useNativeDriver: true }),
        Animated.timing(fadeAnim, { toValue: 1, duration: 200, useNativeDriver: true }),
      ]).start();
      setStep(s => s + 1);
    } else {
      completeOnboarding();
    }
  };

  const s = ONBOARDING_STEPS[step];
  const isLast = step === ONBOARDING_STEPS.length - 1;

  const stepIcons = ['🦗', '⛏️', '🥅', '🌿'];

  return (
    <View style={styles.container}>
      <ScrollView contentContainerStyle={styles.content} ref={scrollRef}>
        {/* Progress dots */}
        <View style={styles.dots}>
          {ONBOARDING_STEPS.map((_, i) => (
            <View key={i} style={[styles.dot, i === step && styles.dotActive]} />
          ))}
        </View>

        {/* Step content */}
        <Animated.View style={[styles.card, { opacity: fadeAnim }]}>
          {/* Icon */}
          <Text style={styles.bigIcon}>{stepIcons[step]}</Text>

          {/* Weather-adaptive tree visual */}
          <View style={[styles.treeBox, { backgroundColor: COLORS.bgSurface }]}>
            <Text style={styles.treeVisual}>{getTreeVisual()}</Text>
            <Text style={styles.treeLabel}>当前环境</Text>
          </View>

          {/* Title */}
          <Text style={styles.title}>{s.title}</Text>

          {/* Description */}
          <Text style={styles.description}>{s.description}</Text>

          {/* Fun fact */}
          {step === 0 && (
            <View style={styles.factBox}>
              <Text style={styles.factTitle}>💡 知了冷知识</Text>
              <Text style={styles.factText}>
                北美有一种"周期蝉"，在地下生活整整17年才出土！它们选择质数年份大规模出现，以此躲避天敌的同步演化。2024年北美出现了200年一遇的双重出土奇观——超过1万亿只蝉同时出现！
              </Text>
            </View>
          )}
        </Animated.View>
      </ScrollView>

      {/* Bottom buttons */}
      <View style={styles.bottom}>
        {!isLast && (
          <TouchableOpacity style={styles.skipBtn} onPress={completeOnboarding}>
            <Text style={styles.skipText}>跳过</Text>
          </TouchableOpacity>
        )}
        <TouchableOpacity style={styles.nextBtn} onPress={goNext}>
          <Text style={styles.nextText}>
            {isLast ? '🎮 开始游戏！' : `下一步 (${step + 1}/${ONBOARDING_STEPS.length})`}
          </Text>
        </TouchableOpacity>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1, backgroundColor: COLORS.bg,
  },
  content: {
    flexGrow: 1, padding: 20, paddingTop: 60, alignItems: 'center',
  },
  dots: {
    flexDirection: 'row', gap: 8, marginBottom: 24,
  },
  dot: {
    width: 8, height: 8, borderRadius: 4, backgroundColor: COLORS.border,
  },
  dotActive: {
    backgroundColor: COLORS.primary, width: 24,
  },
  card: {
    backgroundColor: COLORS.bgCard, borderRadius: 16, padding: 24,
    borderWidth: 1, borderColor: COLORS.border, width: '100%', maxWidth: 360,
    alignItems: 'center',
  },
  bigIcon: { fontSize: 64, marginBottom: 12 },
  treeBox: {
    paddingHorizontal: 20, paddingVertical: 10, borderRadius: 12,
    marginBottom: 16, alignItems: 'center',
  },
  treeVisual: { fontSize: 40 },
  treeLabel: { fontSize: 12, color: COLORS.textDim, marginTop: 4 },
  title: {
    fontSize: 22, fontWeight: 'bold', color: COLORS.text,
    marginBottom: 12, textAlign: 'center',
  },
  description: {
    fontSize: 15, color: COLORS.textDim, lineHeight: 24,
    textAlign: 'center', marginBottom: 16,
  },
  factBox: {
    backgroundColor: 'rgba(76,175,80,0.1)', borderRadius: 12,
    padding: 16, borderWidth: 1, borderColor: COLORS.primaryDark,
    width: '100%',
  },
  factTitle: { fontSize: 14, fontWeight: 'bold', color: COLORS.primary, marginBottom: 6 },
  factText: { fontSize: 13, color: COLORS.textDim, lineHeight: 20 },
  bottom: {
    flexDirection: 'row', padding: 16, gap: 12,
    borderTopWidth: 1, borderTopColor: COLORS.border,
  },
  skipBtn: {
    paddingVertical: 14, paddingHorizontal: 20,
  },
  skipText: { color: COLORS.textDim, fontSize: 15 },
  nextBtn: {
    flex: 1, backgroundColor: COLORS.primary, borderRadius: 12,
    paddingVertical: 14, alignItems: 'center',
  },
  nextText: { color: COLORS.white, fontSize: 16, fontWeight: 'bold' },
});
