using System;
using System.Collections.Generic;
using CicadaHunt.Core;
using UnityEngine;

namespace CicadaHunt.Mode2_Catch
{
    /// <summary>
    /// Cicada Radar UI controller — the 2D radar display for tracking flying cicadas.
    /// Renders a rotating radar sweep, signal blips, and locked-target indicators.
    /// </summary>
    public class CicadaRadarUI : MonoBehaviour
    {
        [Header("Radar Display")]
        [SerializeField] private RectTransform _radarContainer;
        [SerializeField] private float _displayRadius = 150f;
        [SerializeField] private float _maxRadarRange = 200f;

        [Header("Scan Animation")]
        [SerializeField] private float _scanPeriod = 4f;        // Seconds per full rotation
        [SerializeField] private float _scanBeamWidth = 15f;    // Degrees
        [SerializeField] private Transform _scanLine;

        [Header("Signal Blip Prefab")]
        [SerializeField] private GameObject _blipPrefab;

        [Header("Locked Target UI")]
        [SerializeField] private CicadaInfoPanel _infoPanel;

        private List<CicadaSignal> _activeSignals = new();
        private List<RadarBlip> _spawnedBlips = new();
        private string _lockedCicadaID;
        private float _scanAngle;

        public string LockedCicadaID => _lockedCicadaID;
        public event Action<CicadaSignal> OnSignalLocked;
        public event Action OnLockReleased;

        private void Update()
        {
            // Rotate scan line
            _scanAngle = (Time.time % _scanPeriod) / _scanPeriod * 360f;

            if (_scanLine != null)
            {
                _scanLine.rotation = Quaternion.Euler(0, 0, -_scanAngle);
            }

            // Update blip positions
            UpdateBlips();
        }

        /// <summary>
        /// Update the radar display with fresh signal data from the server.
        /// </summary>
        public void UpdateSignals(List<CicadaSignal> signals)
        {
            _activeSignals = signals;

            // Spawn/recycle blips
            while (_spawnedBlips.Count < signals.Count)
            {
                var blipGO = Instantiate(_blipPrefab, _radarContainer);
                _spawnedBlips.Add(blipGO.GetComponent<RadarBlip>());
            }

            // Hide excess blips
            for (int i = 0; i < _spawnedBlips.Count; i++)
            {
                if (i < signals.Count)
                {
                    _spawnedBlips[i].gameObject.SetActive(true);
                    _spawnedBlips[i].SetSignal(signals[i]);
                }
                else
                {
                    _spawnedBlips[i].gameObject.SetActive(false);
                }
            }
        }

        /// <summary>
        /// Lock onto a specific cicada signal for tracking.
        /// </summary>
        public void LockSignal(string cicadaID)
        {
            _lockedCicadaID = cicadaID;

            var signal = _activeSignals.Find(s => s.CicadaID == cicadaID);
            if (signal != null)
            {
                _infoPanel?.Show(signal);
                OnSignalLocked?.Invoke(signal);
            }
        }

        /// <summary>
        /// Release the current lock.
        /// </summary>
        public void ReleaseLock()
        {
            _lockedCicadaID = null;
            _infoPanel?.Hide();
            OnLockReleased?.Invoke();
        }

        private void UpdateBlips()
        {
            foreach (var blip in _spawnedBlips)
            {
                if (!blip.gameObject.activeSelf) continue;
                blip.UpdatePosition(_scanAngle, _scanBeamWidth);
            }
        }

        /// <summary>
        /// Convert a real-world bearing and distance to radar screen coordinates.
        /// </summary>
        public Vector2 SignalToRadarPos(float bearing, float distanceM)
        {
            float radius = (distanceM / _maxRadarRange) * _displayRadius;
            float angleRad = (bearing - 90f) * Mathf.Deg2Rad; // North-up

            return new Vector2(
                Mathf.Cos(angleRad) * radius,
                Mathf.Sin(angleRad) * radius
            );
        }
    }

    /// <summary>
    /// Represents a single cicada signal on the radar.
    /// </summary>
    [Serializable]
    public class CicadaSignal
    {
        public string CicadaID;
        public string SpeciesName;
        public float Bearing;       // degrees from north
        public float DistanceM;
        public float AltitudeM;
        public float SignalStrength; // 0-1
        public string CurrentState;  // "resting", "alert", "flying", etc.
        public int Rarity;
        public float EstimatedValue;
        public bool InBeam;
        public bool IsFading;
    }

    /// <summary>
    /// Individual radar blip — a dot on the radar display.
    /// </summary>
    public class RadarBlip : MonoBehaviour
    {
        [SerializeField] private RectTransform _dot;
        [SerializeField] private float _blipSize = 8f;

        private CicadaSignal _signal;

        public void SetSignal(CicadaSignal signal)
        {
            _signal = signal;
            UpdateVisual();
        }

        public void UpdatePosition(float scanAngle, float beamWidth)
        {
            // Check if in scan beam
            float angleDiff = Mathf.DeltaAngle(scanAngle, _signal.Bearing);
            _signal.InBeam = Mathf.Abs(angleDiff) <= beamWidth / 2f;

            // Position the blip
            var radar = GetComponentInParent<CicadaRadarUI>();
            if (radar == null) return;

            var pos = radar.SignalToRadarPos(_signal.Bearing, _signal.DistanceM);
            (_transform as RectTransform).anchoredPosition = pos;

            UpdateVisual();
        }

        private void UpdateVisual()
        {
            if (_dot == null) return;

            // Size based on signal strength
            float size = Mathf.Lerp(3f, _blipSize, _signal.SignalStrength);
            _dot.sizeDelta = Vector2.one * size;

            // Color based on state
            _dot.GetComponent<UnityEngine.UI.Image>().color = _signal.CurrentState switch
            {
                "resting_singing" => new Color(0, 1f, 0.3f, _signal.InBeam ? 1f : 0.3f),
                "resting"         => new Color(0.2f, 0.8f, 0.2f, _signal.InBeam ? 0.9f : 0.3f),
                "alert"           => Color.yellow,
                "flying"          => new Color(1f, 0.6f, 0f),
                "startled"        => Color.red,
                _                 => Color.white,
            };

            // Flicker if fading
            if (_signal.IsFading)
            {
                float flicker = Mathf.Sin(Time.time * 8f) * 0.5f + 0.5f;
                var c = _dot.GetComponent<UnityEngine.UI.Image>().color;
                c.a *= flicker;
                _dot.GetComponent<UnityEngine.UI.Image>().color = c;
            }
        }
    }

    /// <summary>
    /// Info panel showing details about a locked cicada target.
    /// </summary>
    public class CicadaInfoPanel : MonoBehaviour
    {
        [SerializeField] private TMPro.TextMeshProUGUI _speciesLabel;
        [SerializeField] private TMPro.TextMeshProUGUI _distanceLabel;
        [SerializeField] private TMPro.TextMeshProUGUI _stateLabel;
        [SerializeField] private TMPro.TextMeshProUGUI _valueLabel;
        [SerializeField] private GameObject _container;

        public void Show(CicadaSignal signal)
        {
            if (_container != null) _container.SetActive(true);

            if (_speciesLabel != null) _speciesLabel.text = signal.SpeciesName;
            if (_distanceLabel != null) _distanceLabel.text = $"{signal.DistanceM:F0}m";
            if (_stateLabel != null) _stateLabel.text = GetStateDisplay(signal.CurrentState);
            if (_valueLabel != null) _valueLabel.text = $"¥{signal.EstimatedValue:F0}";
        }

        public void Hide()
        {
            if (_container != null) _container.SetActive(false);
        }

        private string GetStateDisplay(string state) => state switch
        {
            "resting_singing" => "🟢 栖息鸣叫中",
            "resting"         => "🟢 栖息中",
            "alert"           => "🟡 警觉",
            "flying"          => "🟠 飞行中",
            "startled"        => "🔴 受惊逃跑",
            _                 => state,
        };
    }
}
