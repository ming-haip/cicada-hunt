import React, { useState, useEffect, useCallback } from 'react';
import { StatusBar } from 'expo-status-bar';
import { NavigationContainer } from '@react-navigation/native';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { View, Text, StyleSheet } from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';

import MapScreen from './src/screens/MapScreen';
import DigScreen from './src/screens/DigScreen';
import CatchScreen from './src/screens/CatchScreen';
import ProfileScreen from './src/screens/ProfileScreen';
import OnboardingScreen from './src/screens/OnboardingScreen';
import { GameProvider, useGame } from './src/hooks/useGame';
import { WeatherProvider } from './src/hooks/useWeather';
import { apiService } from './src/services/api';
import { COLORS } from './src/utils/constants';

const Tab = createBottomTabNavigator();

function TabIcon({ name, focused }) {
  const icons = {
    Map: focused ? '🗺️' : '🗺',
    Dig: focused ? '⛏️' : '⛏',
    Catch: focused ? '🥅' : '🥅',
    Profile: focused ? '👤' : '👤',
  };
  return (
    <View style={styles.tabIcon}>
      <Text style={[styles.tabEmoji, focused && styles.tabEmojiActive]}>
        {icons[name] || '🦗'}
      </Text>
    </View>
  );
}

function AppContent() {
  const { showOnboarding } = useGame();

  if (showOnboarding) {
    return <OnboardingScreen />;
  }

  return (
    <Tab.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarIcon: ({ focused }) => <TabIcon name={route.name} focused={focused} />,
        tabBarActiveTintColor: COLORS.primary,
        tabBarInactiveTintColor: COLORS.textDim,
        tabBarStyle: styles.tabBar,
        tabBarLabelStyle: styles.tabLabel,
      })}
    >
      <Tab.Screen name="Map" component={MapScreen} options={{ tabBarLabel: '地图' }} />
      <Tab.Screen name="Dig" component={DigScreen} options={{ tabBarLabel: '挖掘' }} />
      <Tab.Screen name="Catch" component={CatchScreen} options={{ tabBarLabel: '捕蝉' }} />
      <Tab.Screen name="Profile" component={ProfileScreen} options={{ tabBarLabel: '我的' }} />
    </Tab.Navigator>
  );
}

export default function App() {
  return (
    <WeatherProvider>
      <GameProvider>
        <NavigationContainer>
          <StatusBar style="light" />
          <AppContent />
        </NavigationContainer>
      </GameProvider>
    </WeatherProvider>
  );
}

const styles = StyleSheet.create({
  tabBar: {
    backgroundColor: '#0d1b0df0',
    borderTopColor: '#2a402a',
    borderTopWidth: 1,
    height: 70,
    paddingBottom: 8,
    paddingTop: 4,
  },
  tabLabel: {
    fontSize: 11,
    fontWeight: '600',
  },
  tabIcon: {
    alignItems: 'center',
    justifyContent: 'center',
  },
  tabEmoji: {
    fontSize: 22,
    opacity: 0.5,
  },
  tabEmojiActive: {
    opacity: 1,
    fontSize: 26,
  },
});
