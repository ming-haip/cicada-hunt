using System;
using System.Collections.Generic;
using UnityEngine;

namespace CicadaHunt
{
    /// <summary>
    /// Generic object pool for Unity GameObjects.
    /// Reduces GC pressure and instantiation overhead for frequently spawned objects.
    /// </summary>
    public class ObjectPool<T> where T : Component
    {
        private readonly Func<T> _createFunc;
        private readonly Action<T> _onGet;
        private readonly Action<T> _onReturn;
        private readonly Stack<T> _pool;
        private readonly int _defaultCapacity;

        public ObjectPool(
            Func<T> createFunc,
            Action<T> onGet = null,
            Action<T> onReturn = null,
            int defaultCapacity = 10)
        {
            _createFunc = createFunc ?? throw new ArgumentNullException(nameof(createFunc));
            _onGet = onGet;
            _onReturn = onReturn;
            _defaultCapacity = defaultCapacity;
            _pool = new Stack<T>(defaultCapacity);

            // Pre-warm the pool
            for (int i = 0; i < defaultCapacity; i++)
            {
                var obj = _createFunc();
                obj.gameObject.SetActive(false);
                _pool.Push(obj);
            }
        }

        /// <summary>
        /// Get an object from the pool or create a new one if the pool is empty.
        /// </summary>
        public T Get()
        {
            T obj;
            if (_pool.Count > 0)
            {
                obj = _pool.Pop();
            }
            else
            {
                obj = _createFunc();
            }

            _onGet?.Invoke(obj);
            return obj;
        }

        /// <summary>
        /// Return an object to the pool for reuse.
        /// </summary>
        public void Return(T obj)
        {
            if (obj == null) return;

            _onReturn?.Invoke(obj);
            _pool.Push(obj);
        }

        /// <summary>
        /// Number of available objects in the pool.
        /// </summary>
        public int AvailableCount => _pool.Count;
    }

    /// <summary>
    /// Non-generic variant for GameObjects (used in DigRadar).
    /// </summary>
    public class ObjectPool
    {
        private readonly Func<GameObject> _createFunc;
        private readonly Action<GameObject> _onGet;
        private readonly Action<GameObject> _onReturn;
        private readonly Stack<GameObject> _pool;

        public ObjectPool(
            Func<GameObject> createFunc,
            Action<GameObject> onGet = null,
            Action<GameObject> onReturn = null,
            int defaultCapacity = 10)
        {
            _createFunc = createFunc;
            _onGet = onGet;
            _onReturn = onReturn;
            _pool = new Stack<GameObject>(defaultCapacity);

            for (int i = 0; i < defaultCapacity; i++)
            {
                var obj = _createFunc();
                obj.SetActive(false);
                _pool.Push(obj);
            }
        }

        public GameObject Get()
        {
            var obj = _pool.Count > 0 ? _pool.Pop() : _createFunc();
            _onGet?.Invoke(obj);
            return obj;
        }

        public void Return(GameObject obj)
        {
            if (obj == null) return;
            _onReturn?.Invoke(obj);
            _pool.Push(obj);
        }

        public int AvailableCount => _pool.Count;
    }

    /// <summary>
    /// Simple progress bar for dig progress visualization.
    /// </summary>
    public class ProgressBarUI : MonoBehaviour
    {
        [SerializeField] private RectTransform _fillRect;
        [SerializeField] private GameObject _container;
        [SerializeField] private float _maxWidth = 300f;

        public void SetProgress(float progress)
        {
            if (_fillRect != null)
            {
                var size = _fillRect.sizeDelta;
                size.x = Mathf.Lerp(0, _maxWidth, Mathf.Clamp01(progress));
                _fillRect.sizeDelta = size;
            }
        }

        public void Show()
        {
            if (_container != null) _container.SetActive(true);
        }

        public void Hide()
        {
            SetProgress(0f);
            if (_container != null) _container.SetActive(false);
        }
    }

    /// <summary>
    /// Nymph marker on the map view.
    /// </summary>
    public class NymphMarker : MonoBehaviour
    {
        [SerializeField] private SpriteRenderer _icon;
        [SerializeField] private TMPro.TextMeshPro _label; // or use UnityEngine.UI.Text for world-space

        public event Action OnTapped;

        public void SetNymph(Models.NymphData nymph)
        {
            if (_icon != null)
            {
                _icon.color = nymph.RarityColor;

                // Scale based on distance (LOD)
                float scale = Mathf.Lerp(0.5f, 1.5f, 1f - Mathf.Clamp01(nymph.DistanceM / 100f));
                transform.localScale = Vector3.one * scale;
            }

            if (_label != null)
            {
                _label.text = nymph.DisplayLabel;
            }
        }

        private void OnMouseDown()
        {
            OnTapped?.Invoke();
        }
    }

    /// <summary>
    /// Stub for MapController — manages the map view and coordinate conversions.
    /// </summary>
    public class MapController : MonoBehaviour
    {
        public Transform MarkerContainer;
        [SerializeField] private float _mapScale = 100f; // pixels per meter (at current zoom)

        public Vector3 GeoToMapPosition(double lat, double lng)
        {
            // Convert GPS coordinates to map local position
            // Uses a reference point (player position) as origin
            if (GPS.Instance == null) return Vector3.zero;

            float dLat = (float)(lat - GPS.Instance.Latitude) * 111320f;
            float dLng = (float)(lng - GPS.Instance.Longitude) * 111320f *
                         Mathf.Cos((float)GPS.Instance.Latitude * Mathf.Deg2Rad);

            return new Vector3(dLng * _mapScale, dLat * _mapScale, 0f);
        }

        public void UpdateDensityOverlay(Models.CellDensityInfo[] densityInfo)
        {
            // Update the heatmap overlay on the map
        }
    }
}
