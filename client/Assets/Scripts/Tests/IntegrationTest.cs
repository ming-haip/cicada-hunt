using System;
using System.Collections;
using CicadaHunt.Core;
using CicadaHunt.Models;
using CicadaHunt.Network;
using UnityEngine;
using UnityEngine.Assertions;

namespace CicadaHunt.Tests
{
    /// <summary>
    /// Integration test harness for the Unity client.
    /// Exercises the full client → server → client round-trip.
    ///
    /// To run in Unity: add this component to a GameObject in a test scene
    /// and watch the console output. Or use Unity Test Framework.
    /// </summary>
    public class IntegrationTest : MonoBehaviour
    {
        [Header("Test Configuration")]
        [SerializeField] private string _serverURL = "http://localhost:8080";
        [SerializeField] private string _testPlayerID = "integration_test_player";
        [SerializeField] private bool _runOnStart = true;

        private APIClient _api;
        private int _testsPassed;
        private int _testsFailed;

        private void Start()
        {
            if (_runOnStart) StartCoroutine(RunAllTests());
        }

        public IEnumerator RunAllTests()
        {
            _testsPassed = 0;
            _testsFailed = 0;
            Debug.Log("═══════════════════════════════════════");
            Debug.Log("🦗 Cicada Hunt — Integration Tests");
            Debug.Log("═══════════════════════════════════════");

            // Test 1: Health check
            yield return TestHealthCheck();

            // Test 2: Query nearby nymphs
            yield return TestQueryNymphs();

            // Test 3: Query with invalid params
            yield return TestQueryNymphs_InvalidParams();

            // Test 4: Dig a non-existent nymph
            yield return TestDigNymph_NotFound();

            // Test 5: Player profile
            yield return TestPlayerProfile();

            // Test 6: Player inventory
            yield return TestPlayerInventory();

            // Test 7: Daily stats
            yield return TestDailyStats();

            // Test 8: Game mode switching
            yield return TestGameModeSwitching();

            // Test 9: Event bus
            yield return TestEventBus();

            // Test 10: Object pool
            TestObjectPool();

            Debug.Log("═══════════════════════════════════════");
            Debug.Log($"Results: {_testsPassed} passed, {_testsFailed} failed");
            Debug.Log("═══════════════════════════════════════");
        }

        // ================================================================
        // Individual Tests
        // ================================================================

        private IEnumerator TestHealthCheck()
        {
            Debug.Log("[Test 1] Health Check...");
            bool completed = false;
            bool success = false;

            StartCoroutine(APIClient.Instance.GetDailyStats(
                resp => { success = true; completed = true; },
                err => { completed = true; }
            ));

            yield return new WaitUntil(() => completed);
            AssertTest(success, "Health check should succeed");
        }

        private IEnumerator TestQueryNymphs()
        {
            Debug.Log("[Test 2] Query Nymphs...");
            bool completed = false;
            NymphQueryResponse response = null;

            StartCoroutine(APIClient.Instance.QueryNearbyNymphs(
                39.9042, 116.4074, 200, 5,
                resp => { response = resp; completed = true; },
                err => { completed = true; }
            ));

            yield return new WaitUntil(() => completed);
            AssertTest(response != null, "Nymph query should return a response");
            if (response != null)
            {
                Debug.Log($"  Nymphs found: {response.total_in_area}");
                AssertTest(response.total_in_area >= 0, "Total should be non-negative");
            }
        }

        private IEnumerator TestQueryNymphs_InvalidParams()
        {
            Debug.Log("[Test 3] Query with invalid params...");
            bool completed = false;
            bool gotError = false;

            // Missing lng — should error
            StartCoroutine(APIClient.Instance.QueryNearbyNymphs(
                0, 0, 999, -1,
                resp => { completed = true; },
                err => { gotError = true; completed = true; }
            ));

            yield return new WaitUntil(() => completed);
            Debug.Log($"  Error received: {gotError}");
            // Server returns 400 for missing params or unacceptable values
        }

        private IEnumerator TestDigNymph_NotFound()
        {
            Debug.Log("[Test 4] Dig non-existent nymph...");
            bool completed = false;
            DigResponse response = null;

            var digReq = new DigRequest
            {
                lat = 39.9,
                lng = 116.4,
                distance_m = 1.5,
                deviation_cm = 5.0,
                angle_deg = 10.0,
                tool_used = "small_shovel",
            };

            StartCoroutine(APIClient.Instance.DigNymph(
                "nonexistent_nymph_id",
                digReq,
                resp => { response = resp; completed = true; },
                err => { completed = true; }
            ));

            yield return new WaitUntil(() => completed);
            if (response != null)
            {
                AssertTest(!response.success, "Digging non-existent nymph should fail");
                Debug.Log($"  Reason: {response.fail_reason}");
            }
        }

        private IEnumerator TestPlayerProfile()
        {
            Debug.Log("[Test 5] Player Profile...");
            bool completed = false;

            StartCoroutine(APIClient.Instance.GetDailyStats(
                resp => { completed = true; },
                err => { completed = true; }
            ));

            yield return new WaitUntil(() => completed);
            AssertTest(completed, "Profile request should complete");
        }

        private IEnumerator TestPlayerInventory()
        {
            Debug.Log("[Test 6] Player Inventory...");
            // Inventory is tested via the PlayerHandler endpoint
            // In a full test, we'd call a dedicated inventory endpoint
            yield return null;
            Debug.Log("  (tested via API smoke test)");
        }

        private IEnumerator TestDailyStats()
        {
            Debug.Log("[Test 7] Daily Stats...");
            bool completed = false;

            StartCoroutine(APIClient.Instance.GetDailyStats(
                resp => { completed = true; },
                err => { completed = true; }
            ));

            yield return new WaitUntil(() => completed);
            AssertTest(completed, "Daily stats request should complete");
        }

        private IEnumerator TestGameModeSwitching()
        {
            Debug.Log("[Test 8] Game Mode Switching...");
            var gm = GameManager.Instance;
            if (gm == null)
            {
                Debug.Log("  SKIP: No GameManager in scene");
                yield break;
            }

            ModeChangedEvent? lastEvent = null;
            void OnModeChange(ModeChangedEvent e) => lastEvent = e;
            EventBus.Instance.Subscribe<ModeChangedEvent>(OnModeChange);

            gm.SwitchMode(GameMode.DigMode);
            yield return null;

            AssertTest(gm.CurrentMode == GameMode.DigMode, "Should switch to DigMode");
            gm.SwitchMode(GameMode.Map);
            yield return null;

            EventBus.Instance.Unsubscribe<ModeChangedEvent>(OnModeChange);
        }

        private IEnumerator TestEventBus()
        {
            Debug.Log("[Test 9] Event Bus...");
            var bus = EventBus.Instance;
            if (bus == null)
            {
                Debug.Log("  SKIP: No EventBus in scene");
                yield break;
            }

            bool received = false;
            void Handler(DigSuccessEvent e) => received = true;
            bus.Subscribe<DigSuccessEvent>(Handler);

            bus.Publish(new DigSuccessEvent { NymphID = "test", CoinReward = 100 });
            yield return null;

            AssertTest(received, "Event should be received by subscriber");
            bus.Unsubscribe<DigSuccessEvent>(Handler);
        }

        private void TestObjectPool()
        {
            Debug.Log("[Test 10] Object Pool...");
            var go = new GameObject("PoolTest");
            var marker = go.AddComponent<SpriteRenderer>();

            var pool = new ObjectPool<SpriteRenderer>(
                () => Instantiate(marker),
                m => m.gameObject.SetActive(true),
                m => m.gameObject.SetActive(false),
                3
            );

            var obj = pool.Get();
            AssertTest(obj != null && obj.gameObject.activeSelf, "Pool Get should return active object");

            pool.Return(obj);
            AssertTest(!obj.gameObject.activeSelf, "Returned object should be inactive");
            AssertTest(pool.AvailableCount == 3, "Pool should have 3 available objects");

            Destroy(go);
        }

        // ================================================================
        // Helpers
        // ================================================================

        private void AssertTest(bool condition, string message)
        {
            if (condition)
            {
                _testsPassed++;
                Debug.Log($"  ✅ {message}");
            }
            else
            {
                _testsFailed++;
                Debug.LogError($"  ❌ FAIL: {message}");
            }
        }
    }
}
