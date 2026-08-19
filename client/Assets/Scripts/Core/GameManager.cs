using System;
using UnityEngine;

namespace CicadaHunt.Core
{
    /// <summary>
    /// Central game state manager. Persists as a singleton across scenes.
    /// Manages game mode switching, player state, and global events.
    /// </summary>
    public class GameManager : MonoBehaviour
    {
        public static GameManager Instance { get; private set; }

        [Header("Game State")]
        [SerializeField] private GameMode _currentMode = GameMode.Map;

        [Header("Player Info")]
        [SerializeField] private string _playerID;
        [SerializeField] private string _playerName = "Player";
        [SerializeField] private int _level = 1;
        [SerializeField] private long _goldCoins = 100;
        [SerializeField] private int _diamonds = 5;

        // Equipment
        public string CurrentShovelID { get; private set; } = "bare_hand";
        public string CurrentNetID { get; private set; } = "basic_net";
        public string CurrentRadarID { get; private set; } = "basic_radar";

        // Daily stats
        public int TodayDigs { get; set; }
        public int TodayCatches { get; set; }
        public int DailyDigLimit { get; private set; } = 50;

        // Properties
        public string PlayerID => _playerID;
        public string PlayerName => _playerName;
        public int Level => _level;
        public long GoldCoins => _goldCoins;
        public int Diamonds => _diamonds;
        public GameMode CurrentMode => _currentMode;

        // Events
        public event Action<GameMode> OnModeChanged;
        public event Action<long> OnGoldChanged;
        public event Action<int> OnLevelUp;

        private void Awake()
        {
            if (Instance != null && Instance != this)
            {
                Destroy(gameObject);
                return;
            }
            Instance = this;
            DontDestroyOnLoad(gameObject);

            // Initialize player ID if not set
            if (string.IsNullOrEmpty(_playerID))
            {
                _playerID = SystemInfo.deviceUniqueIdentifier;
            }
        }

        private void Start()
        {
            LoadPlayerPrefs();
            EventBus.Instance.Subscribe<DigSuccessEvent>(OnDigSuccess);
            EventBus.Instance.Subscribe<CatchSuccessEvent>(OnCatchSuccess);
        }

        private void OnDestroy()
        {
            if (EventBus.Instance != null)
            {
                EventBus.Instance.Unsubscribe<DigSuccessEvent>(OnDigSuccess);
                EventBus.Instance.Unsubscribe<CatchSuccessEvent>(OnCatchSuccess);
            }
        }

        /// <summary>
        /// Switch between game modes (Map, DigMode, CatchMode, ARScan).
        /// </summary>
        public void SwitchMode(GameMode newMode)
        {
            if (_currentMode == newMode) return;

            var oldMode = _currentMode;
            _currentMode = newMode;
            OnModeChanged?.Invoke(newMode);
            Debug.Log($"[GameManager] Mode: {oldMode} → {newMode}");
        }

        /// <summary>
        /// Add gold coins to the player's balance.
        /// </summary>
        public void AddGold(long amount)
        {
            _goldCoins += amount;
            OnGoldChanged?.Invoke(_goldCoins);
            SavePlayerPrefs();
        }

        /// <summary>
        /// Spend gold coins. Returns false if insufficient funds.
        /// </summary>
        public bool SpendGold(long amount)
        {
            if (_goldCoins < amount) return false;
            _goldCoins -= amount;
            OnGoldChanged?.Invoke(_goldCoins);
            SavePlayerPrefs();
            return true;
        }

        /// <summary>
        /// Add experience points and handle level-ups.
        /// </summary>
        public void AddExp(long exp)
        {
            // Simple level formula: level * 100 XP per level
            _level += (int)(exp / 100);
            OnLevelUp?.Invoke(_level);
            SavePlayerPrefs();
        }

        private void OnDigSuccess(DigSuccessEvent evt)
        {
            TodayDigs++;
            AddGold(evt.CoinReward);
            AddExp(evt.ExpReward);
        }

        private void OnCatchSuccess(CatchSuccessEvent evt)
        {
            TodayCatches++;
            AddGold(evt.CoinReward);
            AddExp(evt.ExpReward);
        }

        private void LoadPlayerPrefs()
        {
            _level = PlayerPrefs.GetInt("player_level", 1);
            _goldCoins = PlayerPrefs.GetInt("gold_coins", 100) ;
        }

        private void SavePlayerPrefs()
        {
            PlayerPrefs.SetInt("player_level", _level);
            PlayerPrefs.SetInt("gold_coins", (int)_goldCoins);
            PlayerPrefs.Save();
        }
    }

    public enum GameMode
    {
        Map,        // 2D map view
        DigMode,    // AR digging for nymphs
        CatchMode,  // AR cicada catching
        ARScan,     // AR scanning for targets
        Menu        // Menu/inventory
    }

    public struct DigSuccessEvent
    {
        public string NymphID;
        public string SpeciesName;
        public int Quality;
        public long CoinReward;
        public long ExpReward;
    }

    public struct CatchSuccessEvent
    {
        public string CicadaID;
        public string SpeciesName;
        public int Rarity;
        public long CoinReward;
        public long ExpReward;
    }
}
