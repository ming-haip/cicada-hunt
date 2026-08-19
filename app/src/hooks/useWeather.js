import React, { createContext, useContext, useState, useEffect } from 'react';
import { COLORS, WEATHER_SCENES } from '../utils/constants';

const WeatherContext = createContext(null);

export function WeatherProvider({ children }) {
  const [weather, setWeather] = useState({
    scene: 'sunny',
    temp: 25,
    humidity: 60,
    isRaining: false,
    isNight: false,
    windSpeed: 2,
    season: 'summer',
  });

  useEffect(() => {
    updateWeather();
    const interval = setInterval(updateWeather, 60000); // every minute
    return () => clearInterval(interval);
  }, []);

  function updateWeather() {
    const now = new Date();
    const hour = now.getHours();
    const month = now.getMonth() + 1;
    const isNight = hour < 6 || hour >= 20;

    // Determine season
    let season;
    if (month >= 3 && month <= 5) season = 'spring';
    else if (month >= 6 && month <= 8) season = 'summer';
    else if (month >= 9 && month <= 11) season = 'autumn';
    else season = 'winter';

    // Determine base scene
    let scene = 'sunny';
    if (isNight) scene = 'night';
    else if (season === 'winter') scene = 'snowy';

    // In production: call OpenWeather API for real weather data
    // For now: time/season-based defaults

    setWeather({
      scene,
      temp: isNight ? 18 : 28,
      humidity: 60,
      isRaining: false,
      isNight,
      windSpeed: 2,
      season,
    });
  }

  const getSceneData = () => WEATHER_SCENES[weather.scene] || WEATHER_SCENES.sunny;

  // Weather effects on gameplay
  const getNymphBonus = () => {
    let bonus = 1.0;
    if (weather.isNight) bonus *= 2.0; // night bonus
    if (weather.isRaining) bonus *= 1.3; // rain bonus
    if (weather.season === 'summer') bonus *= 1.5; // summer peak
    if (weather.season === 'winter') bonus *= 0.2; // winter dormancy
    return bonus;
  };

  const getTreeVisual = () => {
    const map = {
      sunny: '🌳', cloudy: '🌥️🌳', rainy: '🌧️🌳',
      night: '🌙🌳', snowy: '❄️🌲',
    };
    return map[weather.scene] || '🌳';
  };

  const getGroundVisual = () => {
    const map = {
      sunny: '🟫', cloudy: '🟫', rainy: '💧🟫',
      night: '⬛', snowy: '⬜',
    };
    return map[weather.scene] || '🟫';
  };

  return (
    <WeatherContext.Provider value={{
      weather, getSceneData, getNymphBonus,
      getTreeVisual, getGroundVisual, updateWeather,
    }}>
      {children}
    </WeatherContext.Provider>
  );
}

export function useWeather() {
  const ctx = useContext(WeatherContext);
  if (!ctx) throw new Error('useWeather must be inside WeatherProvider');
  return ctx;
}
