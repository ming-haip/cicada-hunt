using System;
using System.Collections;
using System.Text;
using CicadaHunt.Models;
using UnityEngine;
using UnityEngine.Networking;

namespace CicadaHunt.Network
{
    /// <summary>
    /// HTTP client for communicating with the Cicada Hunt game server.
    /// </summary>
    public class APIClient : MonoBehaviour
    {
        public static APIClient Instance { get; private set; }

        [Header("Server Configuration")]
        [SerializeField] private string _baseURL = "http://localhost:8080";
        [SerializeField] private string _apiVersion = "v1";
        [SerializeField] private float _requestTimeout = 10f;

        private string _playerID;

        private string APIBase => $"{_baseURL}/api/{_apiVersion}";

        private void Awake()
        {
            if (Instance != null && Instance != this)
            {
                Destroy(gameObject);
                return;
            }
            Instance = this;
            DontDestroyOnLoad(gameObject);

            _playerID = SystemInfo.deviceUniqueIdentifier;
        }

        // ================================================================
        // Nymph Endpoints
        // ================================================================

        /// <summary>
        /// Query nearby nymphs around a geographic position.
        /// GET /api/v1/nymphs?lat=X&lng=Y&radius=R&limit=L
        /// </summary>
        public IEnumerator QueryNearbyNymphs(
            double lat, double lng, float radiusM, int limit,
            Action<NymphQueryResponse> onSuccess, Action<string> onError)
        {
            var url = $"{APIBase}/nymphs?lat={lat}&lng={lng}&radius={radiusM}&limit={limit}";

            using var request = CreateGetRequest(url);
            yield return request.SendWebRequest();

            if (request.result == UnityWebRequest.Result.Success)
            {
                var response = JsonUtility.FromJson<NymphQueryResponse>(request.downloadHandler.text);
                onSuccess?.Invoke(response);
            }
            else
            {
                onError?.Invoke(request.error);
            }
        }

        /// <summary>
        /// Dig a specific nymph.
        /// POST /api/v1/nymphs/{nymphID}/dig
        /// </summary>
        public IEnumerator DigNymph(
            string nymphID, DigRequest digReq,
            Action<DigResponse> onSuccess, Action<string> onError)
        {
            var url = $"{APIBase}/nymphs/{nymphID}/dig";
            var json = JsonUtility.ToJson(digReq);

            using var request = CreatePostRequest(url, json);
            yield return request.SendWebRequest();

            if (request.result == UnityWebRequest.Result.Success)
            {
                var response = JsonUtility.FromJson<DigResponse>(request.downloadHandler.text);
                onSuccess?.Invoke(response);
            }
            else
            {
                onError?.Invoke(request.error);
            }
        }

        // ================================================================
        // Player Endpoints
        // ================================================================

        /// <summary>
        /// Fetch the player's profile and daily stats.
        /// GET /api/v1/player/daily-stats
        /// </summary>
        public IEnumerator GetDailyStats(
            Action<string> onSuccess, Action<string> onError)
        {
            var url = $"{APIBase}/player/daily-stats";

            using var request = CreateGetRequest(url);
            yield return request.SendWebRequest();

            if (request.result == UnityWebRequest.Result.Success)
            {
                onSuccess?.Invoke(request.downloadHandler.text);
            }
            else
            {
                onError?.Invoke(request.error);
            }
        }

        // ================================================================
        // Helpers
        // ================================================================

        private UnityWebRequest CreateGetRequest(string url)
        {
            var request = UnityWebRequest.Get(url);
            request.timeout = (int)_requestTimeout;
            request.SetRequestHeader("X-Player-ID", _playerID);
            request.SetRequestHeader("X-Client-Version", Application.version);
            request.SetRequestHeader("Content-Type", "application/json");
            return request;
        }

        private UnityWebRequest CreatePostRequest(string url, string jsonBody)
        {
            var request = new UnityWebRequest(url, "POST");
            var bodyRaw = Encoding.UTF8.GetBytes(jsonBody);
            request.uploadHandler = new UploadHandlerRaw(bodyRaw);
            request.downloadHandler = new DownloadHandlerBuffer();
            request.timeout = (int)_requestTimeout;
            request.SetRequestHeader("X-Player-ID", _playerID);
            request.SetRequestHeader("X-Client-Version", Application.version);
            request.SetRequestHeader("Content-Type", "application/json");
            return request;
        }
    }
}
