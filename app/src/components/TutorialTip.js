import React, { useEffect, useRef } from 'react';
import { View, Text, StyleSheet, TouchableOpacity, Animated } from 'react-native';
import { COLORS } from '../utils/constants';

export default function TutorialTip({ message, onDismiss }) {
  const opacity = useRef(new Animated.Value(0)).current;
  const translateY = useRef(new Animated.Value(20)).current;

  useEffect(() => {
    Animated.parallel([
      Animated.timing(opacity, { toValue: 1, duration: 300, useNativeDriver: true }),
      Animated.timing(translateY, { toValue: 0, duration: 300, useNativeDriver: true }),
    ]).start();
  }, []);

  const handleDismiss = () => {
    Animated.parallel([
      Animated.timing(opacity, { toValue: 0, duration: 200, useNativeDriver: true }),
      Animated.timing(translateY, { toValue: -20, duration: 200, useNativeDriver: true }),
    ]).start(() => onDismiss());
  };

  return (
    <Animated.View style={[st.container, { opacity, transform: [{ translateY }] }]}>
      <TouchableOpacity style={st.content} onPress={handleDismiss} activeOpacity={0.9}>
        <View style={st.header}>
          <Text style={st.icon}>💡</Text>
          <Text style={st.title}>新手提示</Text>
        </View>
        <Text style={st.message}>{message}</Text>
        <Text style={st.dismissHint}>点击关闭</Text>
      </TouchableOpacity>
    </Animated.View>
  );
}

const st = StyleSheet.create({
  container: {
    position: 'absolute', bottom: 30, left: 16, right: 16,
    zIndex: 100,
  },
  content: {
    backgroundColor: 'rgba(22,38,22,0.97)', borderRadius: 16,
    padding: 18, borderWidth: 1, borderColor: COLORS.primary,
    shadowColor: COLORS.primary, shadowOpacity: 0.2, shadowRadius: 12,
  },
  header: { flexDirection: 'row', alignItems: 'center', gap: 8, marginBottom: 8 },
  icon: { fontSize: 18 },
  title: { fontSize: 14, fontWeight: 'bold', color: COLORS.primary },
  message: { fontSize: 14, color: COLORS.text, lineHeight: 22 },
  dismissHint: { fontSize: 11, color: COLORS.textDim, textAlign: 'right', marginTop: 10 },
});
