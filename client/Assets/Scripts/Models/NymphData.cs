using System;
using UnityEngine;

namespace CicadaHunt.Models
{
    /// <summary>
    /// Client-side representation of a cicada nymph (知了猴).
    /// Mirrors the server NymphSpawn model.
    /// </summary>
    [Serializable]
    public class NymphData
    {
        public string id;
        public double lat;
        public double lng;
        public float depth_cm;

        public string species;
        public string species_name;
        public float size_cm;
        public float weight_g;
        public int quality;
        public bool is_rare;
        public float estimated_value;

        public string status;

        // Client-side computed fields
        public float DistanceM { get; set; }
        public float SignalStrength { get; set; }
        public Vector2 LatLng => new Vector2((float)lat, (float)lng);

        /// <summary>Display label for the nymph marker on the map.</summary>
        public string DisplayLabel
        {
            get
            {
                if (DistanceM < 20f) return $"{species_name} ★{quality}";
                if (DistanceM < 50f) return species_name;
                return "";
            }
        }

        /// <summary>Color based on rarity for UI display.</summary>
        public Color RarityColor => quality switch
        {
            5 => new Color(1f, 0.84f, 0f),    // legendary gold
            4 => new Color(0.6f, 0.2f, 1f),   // epic purple
            3 => new Color(0.2f, 0.5f, 1f),   // rare blue
            2 => new Color(0.3f, 0.8f, 0.3f), // uncommon green
            _ => new Color(0.8f, 0.8f, 0.8f), // common white
        };

        public bool IsActive => status == "active";
    }

    /// <summary>
    /// Server response wrapper for the nymph query endpoint.
    /// </summary>
    [Serializable]
    public class NymphQueryResponse
    {
        public NymphData[] nymphs;
        public CellDensityInfo[] density_info;
        public int total_in_area;
    }

    [Serializable]
    public class CellDensityInfo
    {
        public string h3_cell_lv9;
        public float curr_density;
        public float tree_score;
        public float soil_score;
        public bool is_hotspot;
        public float recommend_idx;
    }

    /// <summary>
    /// Server response for a digging action.
    /// </summary>
    [Serializable]
    public class DigResponse
    {
        public bool success;
        public NymphData nymph;
        public string fail_reason;
        public string fail_reason_code;
        public float success_rate;
        public long coin_reward;
        public long exp_reward;
        public string new_tool_unlocked;
    }

    /// <summary>
    /// Request body for the digging endpoint.
    /// </summary>
    [Serializable]
    public class DigRequest
    {
        public double lat;
        public double lng;
        public double distance_m;
        public double deviation_cm;
        public double angle_deg;
        public string tool_used;
    }
}
