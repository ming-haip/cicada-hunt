using System;
using System.Collections;
using CicadaHunt.Core;
using UnityEngine;

namespace CicadaHunt.Mode2_Catch
{
    /// <summary>
    /// Controls the cicada catching interaction in AR mode.
    ///
    /// State flow:
    ///   TRACKING → SNEAKING → SWINGING → RESULT
    ///
    /// The player must:
    /// 1. Track the cicada to its tree
    /// 2. Sneak within net range without alerting it
    /// 3. Swing the net at the right moment
    /// </summary>
    public class CatchController : MonoBehaviour
    {
        [Header("Catch Parameters")]
        [SerializeField] private float _netReachM = 2.0f;
        [SerializeField] private float _netConeAngle = 30f;
        [SerializeField] private float _maxSwingSpeed = 10f;

        [Header("AR References")]
        [SerializeField] private Camera _arCamera;
        [SerializeField] private GameObject _netVisualPrefab;
        [SerializeField] private GameObject _cicadaVisualPrefab;

        [Header("Feedback")]
        [SerializeField] private AudioSource _audioSource;
        [SerializeField] private AudioClip _swingSound;
        [SerializeField] private AudioClip _catchSound;
        [SerializeField] private AudioClip _missSound;
        [SerializeField] private AudioClip _startleSound;

        // State
        private CatchState _state = CatchState.Idle;
        private CicadaSignal _targetCicada;
        private GameObject _netVisual;
        private float _approachSpeed;
        private float _lastSwingTime;

        // Events
        public event Action<CatchState> OnStateChanged;
        public event Action<CatchResult> OnCatchComplete;

        public enum CatchState
        {
            Idle,
            Tracking,    // Following radar to cicada location
            Sneaking,    // In range, approaching carefully
            Swinging,    // Net swing animation playing
            Result,      // Hit or miss shown
        }

        public struct CatchResult
        {
            public bool Success;
            public string CicadaID;
            public string SpeciesName;
            public float SuccessRate;
            public string FailReason;
            public bool CicadaEvaded;
        }

        /// <summary>
        /// Begin tracking a cicada for capture.
        /// </summary>
        public void StartTracking(CicadaSignal signal)
        {
            _targetCicada = signal;
            TransitionTo(CatchState.Tracking);
            Debug.Log($"[CatchController] Tracking: {signal.SpeciesName} at {signal.DistanceM:F0}m");
        }

        /// <summary>
        /// Called when the player enters the sneak range (< 8m).
        /// The approach speed affects whether the cicada detects the player.
        /// </summary>
        public void UpdateSneakApproach(float distanceM, float playerSpeedMS)
        {
            if (_state != CatchState.Sneaking && _state != CatchState.Tracking) return;

            _approachSpeed = playerSpeedMS;

            // Enter sneak mode when close enough
            if (distanceM <= _netReachM * 2f && _state == CatchState.Tracking)
            {
                TransitionTo(CatchState.Sneaking);
            }

            // Calculate detection risk based on speed and angle
            float detectionRisk = CalculateDetectionRisk(distanceM, playerSpeedMS);
            EventBus.Instance.Publish(new CicadaProximityEvent
            {
                CicadaID = _targetCicada.CicadaID,
                DistanceM = distanceM,
                DetectionRisk = detectionRisk,
            });
        }

        /// <summary>
        /// Player swings the net. Called from UI button or gesture recognizer.
        /// </summary>
        public void SwingNet()
        {
            if (_state != CatchState.Sneaking) return;

            // Prevent rapid-fire swinging
            if (Time.time - _lastSwingTime < 0.5f) return;
            _lastSwingTime = Time.time;

            TransitionTo(CatchState.Swinging);

            // Play swing animation + sound
            if (_audioSource != null && _swingSound != null)
                _audioSource.PlayOneShot(_swingSound);

            // Calculate swing parameters
            var swingData = new NetSwingData
            {
                NetPosition = _arCamera.transform.position + _arCamera.transform.forward * 1.5f,
                Direction = _arCamera.transform.forward,
                Speed = Mathf.Lerp(3f, _maxSwingSpeed, 0.7f), // TODO: use actual gesture speed
                ConeAngle = _netConeAngle,
                MaxReachM = _netReachM,
            };

            EventBus.Instance.Publish(new NetSwingEvent
            {
                NetPosition = swingData.NetPosition,
                SwingDirection = swingData.Direction,
                SwingSpeed = swingData.Speed,
            });

            // Server will evaluate the catch; for now simulate locally
            StartCoroutine(EvaluateCatchRoutine(swingData));
        }

        private IEnumerator EvaluateCatchRoutine(NetSwingData swingData)
        {
            // Wait for "server" response (simulated)
            yield return new WaitForSeconds(0.3f);

            // Mock evaluation for client-side preview
            // Real evaluation happens server-side
            bool success = UnityEngine.Random.value < 0.65f;

            var result = new CatchResult
            {
                Success = success,
                CicadaID = _targetCicada?.CicadaID ?? "",
                SpeciesName = _targetCicada?.SpeciesName ?? "",
                SuccessRate = 0.65f,
                FailReason = success ? "" : "蝉躲开了！",
                CicadaEvaded = !success,
            };

            if (success)
            {
                if (_audioSource != null && _catchSound != null)
                    _audioSource.PlayOneShot(_catchSound);

                EventBus.Instance.Publish(new CatchSuccessEvent
                {
                    CicadaID = result.CicadaID,
                    SpeciesName = result.SpeciesName,
                    Rarity = _targetCicada?.Rarity ?? 1,
                    CoinReward = (long)(_targetCicada?.EstimatedValue ?? 5),
                    ExpReward = 20,
                });
            }
            else
            {
                if (_audioSource != null && _missSound != null)
                    _audioSource.PlayOneShot(_missSound);

                if (_targetCicada?.CurrentState == "alert")
                {
                    if (_audioSource != null && _startleSound != null)
                        _audioSource.PlayOneShot(_startleSound);
                }
            }

            OnCatchComplete?.Invoke(result);
            TransitionTo(CatchState.Result);
        }

        private float CalculateDetectionRisk(float distanceM, float playerSpeedMS)
        {
            float risk = 0f;

            // Distance: closer = higher risk
            risk += Mathf.Clamp01(1f - (distanceM / 8f)) * 0.5f;

            // Speed: faster = higher risk
            if (playerSpeedMS > 2f)
                risk += 0.3f;
            if (playerSpeedMS > 4f)
                risk += 0.3f;

            return Mathf.Clamp01(risk);
        }

        private void TransitionTo(CatchState newState)
        {
            _state = newState;
            OnStateChanged?.Invoke(newState);
        }

        private struct NetSwingData
        {
            public Vector3 NetPosition;
            public Vector3 Direction;
            public float Speed;
            public float ConeAngle;
            public float MaxReachM;
        }
    }

    /// <summary>
    /// Event fired when the player approaches a cicada in sneak mode.
    /// </summary>
    public struct CicadaProximityEvent
    {
        public string CicadaID;
        public float DistanceM;
        public float DetectionRisk; // 0-1, triggers cicada AI state change
    }
}
