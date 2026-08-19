using System;
using System.Collections;
using CicadaHunt.Core;
using CicadaHunt.Models;
using CicadaHunt.Network;
using UnityEngine;

namespace CicadaHunt.Mode1_Dig
{
    /// <summary>
    /// L3/L4 Digging Controller: handles the AR-assisted digging interaction.
    ///
    /// State machine:
    ///   SCANNING → AIMING → DIGGING → UNEARTHED
    ///
    /// Players swipe on screen to "dig" at the marked X position.
    /// The AR precision of their swipe determines success rate.
    /// </summary>
    public class DigController : MonoBehaviour
    {
        [Header("Digging Parameters")]
        [SerializeField] private float _swipeThreshold = 50f;        // Min swipe distance (pixels)
        [SerializeField] private float _maxDeviationCm = 30f;        // Max precision deviation
        [SerializeField] private int _baseDigsRequired = 6;          // Digs to complete with bare hands
        [SerializeField] private float _xMarkUncertaintyCm = 30f;    // GPS uncertainty range

        [Header("Visual References")]
        [SerializeField] private GameObject _xMarkPrefab;
        [SerializeField] private ParticleSystem _dirtParticles;
        [SerializeField] private ParticleSystem _unearthedParticles;
        [SerializeField] private ProgressBarUI _digProgressBar;

        [Header("Audio")]
        [SerializeField] private AudioSource _digAudioSource;
        [SerializeField] private AudioClip[] _digSounds;
        [SerializeField] private AudioClip _unearthedSound;
        [SerializeField] private AudioClip _missedSound;

        // State
        private DigState _state = DigState.Idle;
        private NymphData _targetNymph;
        private Vector2 _swipeStartPos;
        private int _digsPerformed;
        private float _digProgress;
        private float _xMarkDeviationCm;
        private Vector3 _xMarkWorldPos;

        // X-mark oscillation
        private float _xMarkOscillationTime;
        private bool _xMarkLocked;

        // Tool stats
        private Models.ToolStatsData _currentTool;

        // Events
        public event Action<DigState> OnStateChanged;
        public event Action<float> OnProgressChanged;       // 0-1
        public event Action<int> OnDigPerformed;            // dig count
        public event Action<DigResult> OnDigComplete;

        public DigState CurrentState => _state;
        public float DigProgress => _digProgress;
        public float DeviationCm => _xMarkDeviationCm;

        private enum DigState
        {
            Idle,
            Scanning,    // Looking for X mark with phone
            Aiming,      // X mark locked, ready to dig
            Digging,     // Swiping to dig
            Unearthed,   // Nymph found!
            Failed       // Missed
        }

        private void Start()
        {
            LoadToolStats();
        }

        private void Update()
        {
            switch (_state)
            {
                case DigState.Scanning:
                    UpdateScanning();
                    break;
                case DigState.Digging:
                    HandleDigInput();
                    break;
            }
        }

        // ================================================================
        // Public API
        // ================================================================

        /// <summary>
        /// Begin the dig workflow for a specific nymph target.
        /// </summary>
        public void StartDigging(NymphData target)
        {
            _targetNymph = target;
            _digsPerformed = 0;
            _digProgress = 0f;
            _xMarkLocked = false;
            _xMarkDeviationCm = _xMarkUncertaintyCm;

            TransitionTo(DigState.Scanning);
            Debug.Log($"[DigController] Starting dig for: {target.species_name}");
        }

        /// <summary>
        /// Cancel the current digging attempt.
        /// </summary>
        public void CancelDigging()
        {
            TransitionTo(DigState.Idle);
            _targetNymph = null;
        }

        // ================================================================
        // State Handlers
        // ================================================================

        private void UpdateScanning()
        {
            // Simulate the X-mark converging as the player aims the phone
            // In real AR: use ARRaycast to find ground intersection
            // For now: progressively reduce uncertainty

            _xMarkOscillationTime += Time.deltaTime;

            if (!_xMarkLocked)
            {
                // X-mark oscillates with decreasing amplitude as player "aims"
                _xMarkDeviationCm = _xMarkUncertaintyCm * (0.3f + 0.7f * Mathf.Abs(Mathf.Sin(_xMarkOscillationTime * 0.5f)));

                // Auto-lock after ~3 seconds of "scanning"
                if (_xMarkOscillationTime > 3f && _xMarkDeviationCm < 5f)
                {
                    LockXMark();
                }
            }
        }

        private void LockXMark()
        {
            _xMarkLocked = true;
            _xMarkDeviationCm = UnityEngine.Random.Range(0f, 3f); // Final uncertainty: 0-3cm

            // Haptic feedback
            Handheld.Vibrate();

            // Spawn X mark visual
            if (_xMarkPrefab != null)
            {
                var xMark = Instantiate(_xMarkPrefab);
                xMark.transform.position = _xMarkWorldPos;
            }

            TransitionTo(DigState.Aiming);

            EventBus.Instance.Publish(new XMarkLockedEvent
            {
                NymphID = _targetNymph.id,
                DepthCm = _targetNymph.depth_cm,
                WorldPosition = _xMarkWorldPos,
            });
        }

        private void HandleDigInput()
        {
            // Touch/swipe input for digging
            if (Input.touchCount > 0)
            {
                var touch = Input.GetTouch(0);

                switch (touch.phase)
                {
                    case TouchPhase.Began:
                        _swipeStartPos = touch.position;
                        break;

                    case TouchPhase.Moved:
                        float swipeDist = Vector2.Distance(_swipeStartPos, touch.position);
                        if (swipeDist >= _swipeThreshold)
                        {
                            PerformDig();
                            _swipeStartPos = touch.position; // Reset for continuous digging
                        }
                        break;
                }
            }

            // Mouse input (editor testing)
#if UNITY_EDITOR
            if (Input.GetMouseButtonDown(0))
            {
                _swipeStartPos = Input.mousePosition;
            }
            if (Input.GetMouseButton(0))
            {
                float dist = Vector2.Distance(_swipeStartPos, Input.mousePosition);
                if (dist >= _swipeThreshold)
                {
                    PerformDig();
                    _swipeStartPos = Input.mousePosition;
                }
            }
#endif
        }

        private void PerformDig()
        {
            _digsPerformed++;

            // Calculate effective digs needed based on tool
            int effectiveDigs = Mathf.RoundToInt(_baseDigsRequired / _currentTool.Efficiency);
            _digProgress = Mathf.Clamp01((float)_digsPerformed / effectiveDigs);

            // Feedback
            PlayDigFeedback();

            OnDigPerformed?.Invoke(_digsPerformed);
            OnProgressChanged?.Invoke(_digProgress);

            // Check completion
            if (_digProgress >= 1.0f)
            {
                StartCoroutine(CompleteDigging());
            }
        }

        private void PlayDigFeedback()
        {
            // Haptic
            Handheld.Vibrate();

            // Audio: cycle through dig sounds
            if (_digAudioSource != null && _digSounds.Length > 0)
            {
                var clip = _digSounds[_digsPerformed % _digSounds.Length];
                _digAudioSource.PlayOneShot(clip);
            }

            // Particles
            if (_dirtParticles != null)
            {
                _dirtParticles.Emit(5 + _digsPerformed * 2);
            }
        }

        private IEnumerator CompleteDigging()
        {
            TransitionTo(DigState.Unearthed);

            // Send dig request to server
            var digReq = new DigRequest
            {
                lat = GPS.Instance.Latitude,
                lng = GPS.Instance.Longitude,
                distance_m = _targetNymph.DistanceM,
                deviation_cm = _xMarkDeviationCm,
                angle_deg = UnityEngine.Random.Range(0f, 15f), // simulated
                tool_used = GameManager.Instance.CurrentShovelID,
            };

            bool serverResponded = false;
            DigResponse response = null;

            yield return APIClient.Instance.DigNymph(
                _targetNymph.id, digReq,
                resp => { response = resp; serverResponded = true; },
                error =>
                {
                    Debug.LogError($"[DigController] Server error: {error}");
                    serverResponded = true;
                }
            );

            // Wait for response (with timeout)
            float timeout = 5f;
            while (!serverResponded && timeout > 0f)
            {
                yield return new WaitForSeconds(0.1f);
                timeout -= 0.1f;
            }

            if (response != null && response.success)
            {
                // Success!
                if (_unearthedParticles != null)
                {
                    var main = _unearthedParticles.main;
                    main.startColor = _targetNymph.RarityColor;
                    _unearthedParticles.Play();
                }

                if (_unearthedSound != null)
                    _digAudioSource.PlayOneShot(_unearthedSound);

                EventBus.Instance.Publish(new DigSuccessEvent
                {
                    NymphID = _targetNymph.id,
                    SpeciesName = _targetNymph.species_name,
                    Quality = _targetNymph.quality,
                    CoinReward = response.coin_reward,
                    ExpReward = response.exp_reward,
                });

                OnDigComplete?.Invoke(new DigResult
                {
                    Success = true,
                    Nymph = _targetNymph,
                    CoinReward = response.coin_reward,
                    ExpReward = response.exp_reward,
                });
            }
            else
            {
                // Failed
                if (_missedSound != null)
                    _digAudioSource.PlayOneShot(_missedSound);

                EventBus.Instance.Publish(new DigAttemptEvent
                {
                    NymphID = _targetNymph.id,
                    Success = false,
                    FailReason = response?.fail_reason ?? "Network timeout",
                    SuccessRate = response?.success_rate ?? 0f,
                    DigsPerformed = _digsPerformed,
                });

                OnDigComplete?.Invoke(new DigResult
                {
                    Success = false,
                    FailReason = response?.fail_reason ?? "Network timeout",
                });

                TransitionTo(DigState.Failed);
            }
        }

        private void TransitionTo(DigState newState)
        {
            var oldState = _state;
            _state = newState;
            OnStateChanged?.Invoke(newState);

            if (newState == DigState.Digging)
            {
                _digProgressBar?.Show();
            }

            if (newState == DigState.Idle || newState == DigState.Unearthed || newState == DigState.Failed)
            {
                _digProgressBar?.Hide();
            }
        }

        private void LoadToolStats()
        {
            var toolName = GameManager.Instance?.CurrentShovelID ?? "bare_hand";

            // Default tool stats (should match server's ToolStats)
            _currentTool = toolName switch
            {
                "small_shovel" => new Models.ToolStatsData { Name = "小铲子", Efficiency = 1.5f, Accuracy = 0.80f },
                "pro_shovel" => new Models.ToolStatsData { Name = "专业挖掘铲", Efficiency = 2.2f, Accuracy = 0.95f },
                _ => new Models.ToolStatsData { Name = "徒手", Efficiency = 1.0f, Accuracy = 0.60f },
            };
        }
    }

    // Local models for the dig controller
    namespace Models
    {
        [Serializable]
        public class ToolStatsData
        {
            public string Name;
            public float Efficiency = 1.0f;
            public float Accuracy = 0.6f;
        }
    }

    public struct DigResult
    {
        public bool Success;
        public NymphData Nymph;
        public string FailReason;
        public long CoinReward;
        public long ExpReward;
    }
}
