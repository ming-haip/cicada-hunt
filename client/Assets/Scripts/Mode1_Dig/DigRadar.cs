using System;
using System.Collections;
using System.Collections.Generic;
using CicadaHunt.Core;
using CicadaHunt.Models;
using CicadaHunt.Network;
using UnityEngine;

namespace CicadaHunt.Mode1_Dig
{
    /// <summary>
    /// L1 Map Radar mode: displays nymph hotspots and markers on a 2D map.
    /// Handles the top-down map view with heatmap overlay and nymph markers.
    /// </summary>
    public class DigRadar : MonoBehaviour
    {
        [Header("Radar Settings")]
        [SerializeField] private float _queryRadiusM = 200f;
        [SerializeField] private int _maxResults = 30;
        [SerializeField] private float _refreshInterval = 5f;

        [Header("Map References")]
        [SerializeField] private MapController _mapController;

        [Header("Marker Prefabs")]
        [SerializeField] private GameObject _nymphMarkerPrefab;
        [SerializeField] private GameObject _hotspotIndicatorPrefab;

        // Internal state
        private List<NymphData> _activeNymphs = new();
        private List<GameObject> _spawnedMarkers = new();
        private ObjectPool<GameObject> _markerPool;
        private Coroutine _pollCoroutine;

        // Events
        public event Action<NymphData> OnNymphSelected;
        public event Action<NymphData> OnNymphEnteredProximity; // < 50m

        private void Start()
        {
            _markerPool = new ObjectPool<GameObject>(
                () => Instantiate(_nymphMarkerPrefab),
                marker => marker.SetActive(true),
                marker => marker.SetActive(false),
                30
            );

            _pollCoroutine = StartCoroutine(PollNymphsRoutine());
        }

        private void OnEnable()
        {
            if (_pollCoroutine == null)
                _pollCoroutine = StartCoroutine(PollNymphsRoutine());
        }

        private void OnDisable()
        {
            if (_pollCoroutine != null)
            {
                StopCoroutine(_pollCoroutine);
                _pollCoroutine = null;
            }
        }

        /// <summary>
        /// Periodically poll the server for nearby nymph locations.
        /// </summary>
        private IEnumerator PollNymphsRoutine()
        {
            while (true)
            {
                if (GPS.Instance != null && GPS.Instance.IsReady)
                {
                    var lat = GPS.Instance.Latitude;
                    var lng = GPS.Instance.Longitude;

                    yield return APIClient.Instance.QueryNearbyNymphs(
                        lat, lng, _queryRadiusM, _maxResults,
                        OnNymphsReceived,
                        error => Debug.LogWarning($"[DigRadar] Query failed: {error}")
                    );
                }

                yield return new WaitForSeconds(_refreshInterval);
            }
        }

        private void OnNymphsReceived(NymphQueryResponse response)
        {
            _activeNymphs.Clear();

            if (response?.nymphs == null) return;

            foreach (var nymph in response.nymphs)
            {
                if (!nymph.IsActive) continue;

                // Calculate distance from player
                var playerLatLng = new Vector2(
                    (float)GPS.Instance.Latitude,
                    (float)GPS.Instance.Longitude
                );
                nymph.DistanceM = HaversineDistance(playerLatLng, nymph.LatLng);

                _activeNymphs.Add(nymph);

                // Check proximity
                if (nymph.DistanceM < 50f)
                {
                    OnNymphEnteredProximity?.Invoke(nymph);
                }
            }

            // Sort by distance
            _activeNymphs.Sort((a, b) => a.DistanceM.CompareTo(b.DistanceM));

            // Update map markers
            UpdateMarkers();

            // Update density info display
            if (response.density_info != null && response.density_info.Length > 0)
            {
                _mapController?.UpdateDensityOverlay(response.density_info);
            }
        }

        /// <summary>
        /// Refresh map markers based on current nymph data.
        /// </summary>
        private void UpdateMarkers()
        {
            // Return all markers to pool
            foreach (var marker in _spawnedMarkers)
            {
                _markerPool.Return(marker);
            }
            _spawnedMarkers.Clear();

            // Spawn markers for nearby nymphs (LOD: closer = more detail)
            foreach (var nymph in _activeNymphs)
            {
                var marker = _markerPool.Get();
                marker.transform.SetParent(_mapController.MarkerContainer, false);

                var markerComp = marker.GetComponent<NymphMarker>();
                if (markerComp != null)
                {
                    markerComp.SetNymph(nymph);
                    markerComp.OnTapped += () => OnNymphSelected?.Invoke(nymph);
                }

                // Position on the map
                var mapPos = _mapController.GeoToMapPosition(nymph.lat, nymph.lng);
                marker.transform.localPosition = mapPos;

                _spawnedMarkers.Add(marker);
            }
        }

        /// <summary>
        /// Haversine distance between two lat/lng points in meters.
        /// </summary>
        public static float HaversineDistance(Vector2 a, Vector2 b)
        {
            const float R = 6371000f;
            var dLat = (b.x - a.x) * Mathf.Deg2Rad;
            var dLng = (b.y - a.y) * Mathf.Deg2Rad;
            var sinLat = Mathf.Sin(dLat / 2f);
            var sinLng = Mathf.Sin(dLng / 2f);
            var h = sinLat * sinLat +
                    Mathf.Cos(a.x * Mathf.Deg2Rad) * Mathf.Cos(b.x * Mathf.Deg2Rad) *
                    sinLng * sinLng;
            return R * 2f * Mathf.Atan2(Mathf.Sqrt(h), Mathf.Sqrt(1f - h));
        }
    }

    /// <summary>
    /// GPS singleton providing device location.
    /// In Unity, this should be backed by Input.location or a platform-specific plugin.
    /// </summary>
    public class GPS : MonoBehaviour
    {
        public static GPS Instance { get; private set; }
        public double Latitude { get; private set; } = 39.9042;
        public double Longitude { get; private set; } = 116.4074;
        public bool IsReady { get; private set; }

        private void Awake()
        {
            if (Instance != null) { Destroy(gameObject); return; }
            Instance = this;
            DontDestroyOnLoad(gameObject);
        }

        private IEnumerator Start()
        {
            // Request location permission
            if (!Input.location.isEnabledByUser)
            {
                Debug.LogWarning("[GPS] Location services not enabled");
                IsReady = true; // Use default Beijing coords for development
                yield break;
            }

            Input.location.Start(1f, 1f); // 1m accuracy

            int maxWait = 20;
            while (Input.location.status == LocationServiceStatus.Initializing && maxWait > 0)
            {
                yield return new WaitForSeconds(1f);
                maxWait--;
            }

            if (Input.location.status == LocationServiceStatus.Running)
            {
                var loc = Input.location.lastData;
                Latitude = loc.latitude;
                Longitude = loc.longitude;
                IsReady = true;
                Debug.Log($"[GPS] Location acquired: {Latitude}, {Longitude}");
            }
            else
            {
                Debug.LogWarning("[GPS] Failed to get location, using defaults");
                IsReady = true;
            }
        }

        private void Update()
        {
            if (Input.location.status == LocationServiceStatus.Running)
            {
                var loc = Input.location.lastData;
                Latitude = loc.latitude;
                Longitude = loc.longitude;
            }
        }

        private void OnDestroy()
        {
            Input.location.Stop();
        }
    }
}
