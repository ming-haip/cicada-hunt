using System;
using CicadaHunt.Core;
using CicadaHunt.Models;
using UnityEngine;

namespace CicadaHunt.Mode1_Dig
{
    /// <summary>
    /// L2 Proximity Detection mode: provides multi-sensory feedback (haptic, audio, visual)
    /// as the player approaches a nymph location. Simulates a metal detector's progressive signal.
    /// </summary>
    public class ProximityDetector : MonoBehaviour
    {
        [Header("Detection Parameters")]
        [SerializeField] private float _maxDetectRange = 50f;  // Start detecting at 50m
        [SerializeField] private float _closeRange = 10f;       // Strong signal < 10m
        [SerializeField] private float _veryCloseRange = 3f;    // Very strong < 3m

        [Header("Haptic Feedback")]
        [SerializeField] private float _hapticMinInterval = 2.0f;   // Seconds between pulses @ 50m
        [SerializeField] private float _hapticMaxInterval = 0.15f;   // Seconds between pulses @ 1m
        [SerializeField] private bool _useHaptics = true;

        [Header("Audio Feedback")]
        [SerializeField] private AudioSource _sonarSource;
        [SerializeField] private AnimationCurve _pitchCurve;  // Signal → pitch mapping
        [SerializeField] private AnimationCurve _volumeCurve; // Signal → volume mapping

        [Header("Visual Feedback")]
        [SerializeField] private SignalMeterUI _signalMeter;

        // State
        private NymphData _trackedNymph;
        private float _currentSignal;
        private float _previousSignal;
        private float _lastHapticTime;
        private bool _isTracking;

        // Events
        public event Action<float> OnSignalChanged;   // signal 0-1
        public event Action OnCloseRangeReached;       // < 3m — ready for AR scan

        /// <summary>
        /// Start tracking a specific nymph.
        /// </summary>
        public void StartTracking(NymphData nymph)
        {
            _trackedNymph = nymph;
            _isTracking = true;
            _currentSignal = 0f;
            _previousSignal = 0f;
            Debug.Log($"[ProximityDetector] Tracking nymph: {nymph.species_name} ({nymph.DistanceM:F0}m)");
        }

        /// <summary>
        /// Stop tracking the current nymph.
        /// </summary>
        public void StopTracking()
        {
            _isTracking = false;
            _trackedNymph = null;
            _signalMeter?.SetValue(0f);

            if (_sonarSource != null && _sonarSource.isPlaying)
                _sonarSource.Stop();
        }

        private void Update()
        {
            if (!_isTracking || _trackedNymph == null) return;

            // Calculate distance from player to nymph
            if (GPS.Instance == null || !GPS.Instance.IsReady) return;

            var playerPos = new Vector2((float)GPS.Instance.Latitude, (float)GPS.Instance.Longitude);
            var nymphPos = _trackedNymph.LatLng;

            float distanceM = DigRadar.HaversineDistance(playerPos, nymphPos);
            _trackedNymph.DistanceM = distanceM;

            // Calculate signal strength
            _previousSignal = _currentSignal;
            _currentSignal = CalculateSignalStrength(distanceM, _trackedNymph.depth_cm);
            _trackedNymph.SignalStrength = _currentSignal;

            // === Haptic Feedback ===
            UpdateHapticFeedback();

            // === Audio Feedback ===
            UpdateAudioFeedback();

            // === Visual Feedback ===
            UpdateVisualFeedback();

            // === Proximity Events ===
            OnSignalChanged?.Invoke(_currentSignal);

            if (distanceM < _veryCloseRange)
            {
                OnCloseRangeReached?.Invoke();
            }
        }

        /// <summary>
        /// Calculate signal strength from distance and depth.
        /// Returns 0-1 where 1 = strongest signal.
        /// </summary>
        private float CalculateSignalStrength(float distanceM, float depthCm)
        {
            if (distanceM > _maxDetectRange) return 0f;

            // Distance attenuation (exponential)
            float distSignal;
            if (distanceM <= 5f)
            {
                distSignal = 0.85f + (5f - distanceM) / 5f * 0.15f; // 0.85 → 1.0
            }
            else if (distanceM <= 10f)
            {
                distSignal = 0.7f + (10f - distanceM) / 5f * 0.15f; // 0.7 → 0.85
            }
            else if (distanceM <= 30f)
            {
                distSignal = 0.3f + (30f - distanceM) / 20f * 0.4f; // 0.3 → 0.7
            }
            else
            {
                distSignal = 0.1f + (50f - distanceM) / 20f * 0.2f; // 0.1 → 0.3
            }

            // Depth attenuation (deeper = harder to detect)
            float depthSignal = 1.0f - Mathf.Clamp((depthCm - 5f) / 45f * 0.6f, 0f, 0.6f);

            // Perlin noise for realistic uncertainty
            float noise = 1f + (Mathf.PerlinNoise(distanceM * 0.5f, Time.time * 0.3f) - 0.5f) * 0.15f;

            return Mathf.Clamp01(distSignal * depthSignal * noise);
        }

        private void UpdateHapticFeedback()
        {
            if (!_useHaptics || _currentSignal < 0.05f) return;

            float interval = Mathf.Lerp(_hapticMinInterval, _hapticMaxInterval, _currentSignal);

            if (Time.time - _lastHapticTime > interval)
            {
                // Stronger pulse when signal is increasing (player moving toward target)
                float strength = _currentSignal > _previousSignal
                    ? _currentSignal * 1.2f
                    : _currentSignal * 0.8f;

                Handheld.Vibrate();
                _lastHapticTime = Time.time;
            }
        }

        private void UpdateAudioFeedback()
        {
            if (_sonarSource == null || _currentSignal < 0.05f) return;

            float pitch = _pitchCurve?.Evaluate(_currentSignal) ?? Mathf.Lerp(0.5f, 2.0f, _currentSignal);
            float volume = _volumeCurve?.Evaluate(_currentSignal) ?? Mathf.Lerp(0.1f, 0.8f, _currentSignal);

            _sonarSource.pitch = pitch;
            _sonarSource.volume = volume;

            if (!_sonarSource.isPlaying)
            {
                _sonarSource.Play();
            }
        }

        private void UpdateVisualFeedback()
        {
            _signalMeter?.SetValue(_currentSignal);
        }
    }

    /// <summary>
    /// Simple signal meter UI that displays proximity signal as a bar.
    /// </summary>
    public class SignalMeterUI : MonoBehaviour
    {
        [SerializeField] private RectTransform _fillBar;
        [SerializeField] private float _maxWidth = 200f;
        [SerializeField] private UnityEngine.UI.Image _fillImage;

        public void SetValue(float signal)
        {
            if (_fillBar != null)
            {
                var size = _fillBar.sizeDelta;
                size.x = Mathf.Lerp(0, _maxWidth, signal);
                _fillBar.sizeDelta = size;
            }

            if (_fillImage != null)
            {
                _fillImage.color = signal switch
                {
                    < 0.2f => Color.gray,
                    < 0.5f => Color.yellow,
                    < 0.8f => new Color(1f, 0.5f, 0f), // orange
                    _      => Color.red,
                };
            }
        }
    }
}
