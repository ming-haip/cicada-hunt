import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import AsyncStorage from '@react-native-async-storage/async-storage';

const GameContext = createContext(null);

export function GameProvider({ children }) {
  const [showOnboarding, setShowOnboarding] = useState(true);
  const [player, setPlayer] = useState({ level: 1, gold: 100, diamonds: 5, exp: 0 });
  const [dailyStats, setDailyStats] = useState({ digs: 0, limit: 50 });
  const [trackedNymph, setTrackedNymph] = useState(null);
  const [tutorialsSeen, setTutorialsSeen] = useState({});
  const [currentTutorial, setCurrentTutorial] = useState(null);

  useEffect(() => {
    AsyncStorage.getItem('onboardingComplete').then(v => {
      if (v === 'true') setShowOnboarding(false);
    });
    AsyncStorage.getItem('tutorialsSeen').then(v => {
      if (v) setTutorialsSeen(JSON.parse(v));
    });
  }, []);

  const completeOnboarding = useCallback(async () => {
    setShowOnboarding(false);
    await AsyncStorage.setItem('onboardingComplete', 'true');
  }, []);

  const addGold = useCallback((amount) => {
    setPlayer(p => ({ ...p, gold: p.gold + amount }));
  }, []);

  const addExp = useCallback((amount) => {
    setPlayer(p => {
      const newExp = p.exp + amount;
      const newLevel = Math.floor(newExp / 100) + 1;
      return { ...p, exp: newExp, level: newLevel };
    });
  }, []);

  const showTutorial = useCallback((key, message) => {
    if (tutorialsSeen[key]) return;
    setCurrentTutorial({ key, message });
  }, [tutorialsSeen]);

  const dismissTutorial = useCallback(async (key) => {
    setCurrentTutorial(null);
    if (key) {
      const updated = { ...tutorialsSeen, [key]: true };
      setTutorialsSeen(updated);
      await AsyncStorage.setItem('tutorialsSeen', JSON.stringify(updated));
    }
  }, [tutorialsSeen]);

  return (
    <GameContext.Provider value={{
      showOnboarding, completeOnboarding,
      player, addGold, addExp, setPlayer,
      dailyStats, setDailyStats,
      trackedNymph, setTrackedNymph,
      currentTutorial, showTutorial, dismissTutorial,
    }}>
      {children}
    </GameContext.Provider>
  );
}

export function useGame() {
  const ctx = useContext(GameContext);
  if (!ctx) throw new Error('useGame must be inside GameProvider');
  return ctx;
}
