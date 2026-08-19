import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { useWeather } from '../hooks/useWeather';

/**
 * WeatherBackground — renders weather-adaptive environment visuals.
 *
 * Displays different scene elements based on current weather conditions:
 * - Sunny: bright green trees, brown ground
 * - Rainy: dark clouds, wet ground with raindrops
 * - Night: dark sky with stars, moonlit trees
 * - Snowy: white ground, bare trees
 * - Cloudy: overcast canopy, muted colors
 */
export default function WeatherBackground({ children }) {
  const { weather, getTreeVisual, getGroundVisual, getSceneData } = useWeather();
  const scene = getSceneData();

  // Generate weather-specific environment elements
  const renderEnvironment = () => {
    const elements = [];

    // Trees (size varies by season)
    const treeCount = weather.season === 'winter' ? 5 : 8;
    for (let i = 0; i < treeCount; i++) {
      const x = 10 + (i * 40) % 80;
      const y = 5 + (i * 25) % 40;
      elements.push(
        <Text key={`tree_${i}`} style={[ss.envTree, { left: `${x}%`, top: `${y}%` }]}>
          {getTreeVisual()}
        </Text>
      );
    }

    // Ground pattern
    for (let i = 0; i < 6; i++) {
      const x = i * 18;
      elements.push(
        <Text key={`gnd_${i}`} style={[ss.envGround, { left: `${x}%`, bottom: 0 }]}>
          {getGroundVisual()}
        </Text>
      );
    }

    // Sky elements
    if (weather.isNight) {
      for (let i = 0; i < 8; i++) {
        elements.push(
          <Text key={`star_${i}`} style={[ss.star, {
            left: `${10 + i * 11}%`, top: `${3 + (i % 3) * 6}%`,
            opacity: 0.3 + (i % 3) * 0.3,
          }]}>✦</Text>
        );
      }
      elements.push(<Text key="moon" style={ss.moon}>🌙</Text>);
    }

    if (weather.isRaining) {
      for (let i = 0; i < 12; i++) {
        elements.push(
          <Text key={`rain_${i}`} style={[ss.rain, {
            left: `${5 + i * 8}%`, top: `${-5 + (i % 4) * 8}%`,
          }]}>💧</Text>
        );
      }
    }

    if (weather.scene === 'snowy') {
      for (let i = 0; i < 10; i++) {
        elements.push(
          <Text key={`snow_${i}`} style={[ss.snow, {
            left: `${5 + i * 10}%`, top: `${(i % 5) * 12}%`,
          }]}>❄️</Text>
        );
      }
    }

    return elements;
  };

  return (
    <View style={[ss.container, { backgroundColor: scene.bg }]}>
      {/* Environment layer */}
      <View style={ss.environment}>
        {renderEnvironment()}
      </View>

      {/* Season indicator */}
      <View style={ss.seasonTag}>
        <Text style={ss.seasonIcon}>{scene.icon}</Text>
        <Text style={ss.seasonText}>{scene.label}</Text>
        <Text style={ss.seasonTemp}>{weather.temp}°C</Text>
      </View>

      {/* Content */}
      <View style={ss.content}>
        {children}
      </View>
    </View>
  );
}

const ss = StyleSheet.create({
  container: { flex: 1, position: 'relative' },
  environment: { position: 'absolute', top: 0, left: 0, right: 0, bottom: 0, opacity: 0.3, overflow: 'hidden' },
  envTree: { position: 'absolute', fontSize: 30, opacity: 0.4 },
  envGround: { position: 'absolute', fontSize: 18, opacity: 0.3 },
  star: { position: 'absolute', fontSize: 10, color: '#ffc' },
  moon: { position: 'absolute', fontSize: 20, top: '8%', right: '15%' },
  rain: { position: 'absolute', fontSize: 14, opacity: 0.5 },
  snow: { position: 'absolute', fontSize: 12, opacity: 0.6 },
  seasonTag: {
    position: 'absolute', top: 52, right: 12,
    flexDirection: 'row', alignItems: 'center', gap: 4,
    backgroundColor: 'rgba(0,0,0,0.4)', paddingHorizontal: 10, paddingVertical: 4,
    borderRadius: 12, zIndex: 5,
  },
  seasonIcon: { fontSize: 14 },
  seasonText: { color: '#fff', fontSize: 11, fontWeight: '600' },
  seasonTemp: { color: '#ffc107', fontSize: 11 },
  content: { flex: 1 },
});
