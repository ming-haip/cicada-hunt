using System;
using System.Collections.Generic;
using UnityEngine;

namespace CicadaHunt.Core
{
    /// <summary>
    /// Lightweight publish-subscribe event bus for decoupled communication.
    /// </summary>
    public class EventBus : MonoBehaviour
    {
        public static EventBus Instance { get; private set; }

        private readonly Dictionary<Type, Delegate> _handlers = new();

        private void Awake()
        {
            if (Instance != null && Instance != this)
            {
                Destroy(gameObject);
                return;
            }
            Instance = this;
            DontDestroyOnLoad(gameObject);
        }

        /// <summary>
        /// Subscribe to events of type T.
        /// </summary>
        public void Subscribe<T>(Action<T> handler)
        {
            var type = typeof(T);
            if (_handlers.TryGetValue(type, out var existing))
            {
                _handlers[type] = Delegate.Combine(existing, handler);
            }
            else
            {
                _handlers[type] = handler;
            }
        }

        /// <summary>
        /// Unsubscribe from events of type T.
        /// </summary>
        public void Unsubscribe<T>(Action<T> handler)
        {
            var type = typeof(T);
            if (_handlers.TryGetValue(type, out var existing))
            {
                var newDelegate = Delegate.Remove(existing, handler);
                if (newDelegate == null)
                {
                    _handlers.Remove(type);
                }
                else
                {
                    _handlers[type] = newDelegate;
                }
            }
        }

        /// <summary>
        /// Publish an event to all subscribers of type T.
        /// </summary>
        public void Publish<T>(T eventData)
        {
            var type = typeof(T);
            if (_handlers.TryGetValue(type, out var handler))
            {
                (handler as Action<T>)?.Invoke(eventData);
            }
        }
    }

    // ================================================================
    // Game Events
    // ================================================================

    /// <summary>Fired when the player enters proximity range of a nymph.</summary>
    public struct NymphProximityEvent
    {
        public string NymphID;
        public float DistanceM;
        public float SignalStrength;
    }

    /// <summary>Fired when the AR X-mark locks onto a nymph position.</summary>
    public struct XMarkLockedEvent
    {
        public string NymphID;
        public float DepthCm;
        public Vector3 WorldPosition;
    }

    /// <summary>Fired when a digging attempt completes (success or fail).</summary>
    public struct DigAttemptEvent
    {
        public string NymphID;
        public bool Success;
        public string FailReason;
        public float SuccessRate;
        public int DigsPerformed;
    }

    /// <summary>Fired when the game mode changes.</summary>
    public struct ModeChangedEvent
    {
        public GameMode PreviousMode;
        public GameMode NewMode;
    }

    /// <summary>Fired when a cicada is detected on radar.</summary>
    public struct CicadaDetectedEvent
    {
        public string CicadaID;
        public float Bearing;
        public float DistanceM;
        public float SignalStrength;
        public string SpeciesName;
    }

    /// <summary>Fired when a cicada changes behavior state.</summary>
    public struct CicadaStateChangedEvent
    {
        public string CicadaID;
        public string PreviousState;
        public string NewState;
    }

    /// <summary>Fired when the player swings the net.</summary>
    public struct NetSwingEvent
    {
        public Vector3 NetPosition;
        public Vector3 SwingDirection;
        public float SwingSpeed;
    }

    /// <summary>Fired when AR ground surface type is classified.</summary>
    public struct GroundSurfaceEvent
    {
        public string SurfaceType;
        public bool IsDiggable;
        public float DigDifficulty;
    }
}
