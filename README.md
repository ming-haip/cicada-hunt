# 方向1：AR 挖掘交互层技术方案

> **前置依赖**：[玩法一生成系统](抓知了猴_玩法一_生成系统技术方案.md) — 知了猴点位已生成，需在此基础上实现挖掘交互
> **文档定位**：从"看到知了猴标记"到"成功挖出"的完整前端交互技术方案
> **文档版本**：v0.1

---

## 目录

1. [交互流程全览](#1-交互流程全览)
2. [L1 地图雷达层](#2-l1-地图雷达层)
3. [L2 近场探测层](#3-l2-近场探测层)
4. [L3 AR 热力扫描层](#4-l3-ar-热力扫描层)
5. [L4 精确挖掘层](#5-l4-精确挖掘层)
6. [挖掘判定与命中算法](#6-挖掘判定与命中算法)
7. [AR 核心技术实现](#7-ar-核心技术实现)
8. [动效与音效系统](#8-动效与音效系统)
9. [多机型适配](#9-多机型适配)

---
# 1. 交互流程全览

### 1.1 四级渐进式交互

知了猴挖掘采用**从宏观到微观**的四级渐进交互，每拉近一级距离，信息密度和交互精度提升一个量级：

```
距离范围      交互层级              视图模式        核心交互
───────────────────────────────────────────────────────────
 200m - 50m   L1 地图雷达层         俯视地图         热点色块 + 标记点
  50m - 10m   L2 近场探测层         混合视图         信号强度条 + 震动引导
  10m - 2m    L3 AR 热力扫描层      AR 实景          热力波纹 + 方位指示
   2m - 0m    L4 精确挖掘层         AR 俯视地面       X标记 + 铲子挖掘
```

### 1.2 全链路状态机

```
                    ┌──────────────┐
    应用启动 ─────→ │  IDLE        │
                    │  待机状态     │
                    └──────┬───────┘
                           │ 进入玩法一
                           ↓
                    ┌──────────────┐
             ┌────→ │  MAP_RADAR   │ ← 距离 > 50m
             │      │  地图雷达模式  │
             │      └──────┬───────┘
             │             │ 选中目标 + 距离 < 50m
             │             ↓
             │      ┌──────────────┐
             │      │  TRACKING    │ ← 10m < 距离 < 50m
             │      │  追踪导航模式  │
             │      └──────┬───────┘
             │             │ 到达 10m 范围
             │             ↓
             │      ┌──────────────┐
             │      │  AR_SCAN     │ ← 2m < 距离 < 10m
             │      │  AR热力扫描   │
             │      └──────┬───────┘
             │             │ 定位X点 + 进入2m内
             │             ↓
             │      ┌──────────────┐
             │      │  DIGGING     │ ← 距离 < 2m
             │      │  挖掘中       │
             │      └──────┬───────┘
             │             │ 挖掘完成
             │             ↓
             │      ┌──────────────┐
             │      │  REWARD      │
             │      │  收获展示     │
             │      └──────────────┘
             │             │ 继续寻找
             └─────────────┘
```

---

## 2. L1 地图雷达层

### 2.1 功能定位

玩家在较远距离（50-200m）时，通过**俯视地图**了解周边知了猴分布情况。

### 2.2 视觉设计

```
┌──────────────────────────────────────────────┐
│         🗺️  地图视图                          │
│                                              │
│    ┌──────────────────────────────────┐      │
│    │                  🟡               │      │
│    │        🟢        🟡🟡            │      │
│    │     🟢🟢🟢      🟡🟡🟡🟡         │      │
│    │  🟢🟢🟢🟢🟢     🟡🟡🟡          │      │
│    │     🟢🟢🟢       🟡              │      │
│    │        🟢                         │      │
│    └──────────────────────────────────┘      │
│                                              │
│   图例：🟢 高密度区  🟡 中密度区  ⚪ 低密度区   │
│         📍 知了猴个体点位                      │
│         🔵 我的位置                           │
│                                              │
│   ┌────────────────────────────────────┐     │
│   │ 📊 当前区域：朝阳公园南侧             │     │
│   │ 🌳 树木：杨树为主                    │     │
│   │ 🦗 估测密度：85只/km²               │     │
│   │ ⭐ 推荐指数：★★★★★ (极佳挖掘点)      │     │
│   └────────────────────────────────────┘     │
└──────────────────────────────────────────────┘
```

### 2.3 热力图渲染技术

#### 瓦片方案

```
热力图作为独立瓦片图层叠加到地图上：

┌────────────────────────────────────┐
│         地图渲染管线                │
│                                    │
│   地图底图 (高德/Mapbox)            │
│        ↓                           │
│   + 热力图瓦片 (自建服务)           │
│        ↓                           │
│   + 知了猴点位图层 (Marker)         │
│        ↓                           │
│   + UI 叠加层 (控件/状态栏)         │
│        ↓                           │
│   最终渲染                          │
└────────────────────────────────────┘
```

#### 热力图瓦片服务

```go
// GET /tiles/heatmap/{z}/{x}/{y}.png
// 返回256×256的PNG热力图瓦片

func HeatmapTileHandler(w http.ResponseWriter, r *http.Request) {
    z, x, y := parseTileCoords(r)
    
    // 1. 计算该瓦片覆盖的 H3 Cell 集合
    tileBounds := tileToLatLngBounds(z, x, y)
    cells := h3.PolyfillFromRect(tileBounds, Level7)
    
    // 2. 并发获取各 Cell 密度
    densities := batchGetCellDensities(cells)
    
    // 3. 渲染为热力图 PNG
    img := renderHeatmapTile(256, 256, tileBounds, densities, HeatmapConfig{
        ColorScheme: "green-yellow-red",  // 绿→黄→红 色阶
        MinDensity:  0,
        MaxDensity:  200,
        BlurRadius:  8,  // 高斯模糊半径（像素）
        Opacity:     0.6,
    })
    
    // 4. 设置缓存头 (CDN缓存1小时)
    w.Header().Set("Cache-Control", "public, max-age=3600")
    w.Header().Set("Content-Type", "image/png")
    png.Encode(w, img)
}
```

#### 热力图渲染参数

```go
type HeatmapConfig struct {
    // 色阶定义
    ColorStops: []ColorStop{
        {Value: 0.0,  Color: RGBA{0, 0, 0, 0}},        // 0密度 → 完全透明
        {Value: 0.1,  Color: RGBA{100, 200, 100, 100}}, // 低密度 → 浅绿
        {Value: 0.3,  Color: RGBA{50, 200, 50, 150}},   // 中低密度 → 绿色
        {Value: 0.5,  Color: RGBA{200, 200, 50, 180}},  // 中等密度 → 黄色
        {Value: 0.7,  Color: RGBA{220, 150, 30, 200}},  // 中高密度 → 橙色
        {Value: 1.0,  Color: RGBA{220, 50, 30, 220}},   // 高密度 → 红
    },
    
    // 径向渐变叠加 (模拟树根周围的聚集效应)
    RadialGradient: {
        Enabled:   true,
        Radius:    40,  // 热力扩散半径（像素）
        InnerAlpha: 1.0,
        OuterAlpha: 0.0,
    },
}
```

### 2.4 点位标注

```csharp
// Unity 端：在地图上放置知了猴标记
public class NymphMarkerController : MonoBehaviour
{
    [SerializeField] private MapView _mapView;
    
    // 对象池：知了猴标记
    private ObjectPool<Marker> _markerPool = new(initialSize: 30);
    
    public void UpdateMarkers(List<NymphInfo> nymphs, float currentZoom)
    {
        // 1. 回收所有旧标记
        _markerPool.ReturnAll();
        
        // 2. 按距离排序，近的优先渲染
        nymphs.Sort((a, b) => a.Distance.CompareTo(b.Distance));
        
        // 3. LOD：根据缩放级别决定显示层级
        int maxShow = currentZoom switch
        {
            < 14 => 0,     // 缩太小不显示个体
            < 15 => 10,    // 最多10个
            < 16 => 20,
            _    => 30,
        };
        
        // 4. 放置标记
        foreach (var nymph in nymphs.Take(maxShow))
        {
            var marker = _markerPool.Get();
            
            // 远距离：小圆点
            // 中距离：知了猴图标
            // 近距离：图标+品种名
            marker.SetVisual(GetVisualLevel(nymph.Distance));
            marker.SetPosition(nymph.Lat, nymph.Lng);
            marker.OnClick = () => OnMarkerClicked(nymph);
        }
    }
    
    private MarkerVisual GetVisualLevel(float distance) => distance switch
    {
        > 100  => MarkerVisual.TinyDot,        // 4px 橙色点
        > 30   => MarkerVisual.SmallDot,       // 8px 图标
        > 10   => MarkerVisual.Icon,           // 图标+光环
        _      => MarkerVisual.IconWithLabel,  // 图标+品种名+距离
    };
}
```

---

## 3. L2 近场探测层

### 3.1 功能定位

玩家进入 10-50m 范围后，切换到**追踪导航模式**。这一层的核心体验是"感知靠近"——类似金属探测器的渐进式反馈。

### 3.2 信号强度模型

```csharp
// 信号强度 = 基于距离的倒数衰减 + 深度衰减 + 随机噪声
public class ProximitySignalCalculator
{
    // 信号强度范围 [0, 1]
    public float CalculateSignal(float distanceM, float depthCm, float timeOfDay, float weatherBonus)
    {
        if (distanceM > 50f) return 0f;
        
        // 1. 距离衰减：在10-50m区间指数变化
        //    50m → 0.1, 10m → 0.7, 5m → 0.85
        float distSignal = distanceM switch
        {
            <= 5f  => 0.85f + (5f - distanceM) / 5f * 0.15f,  // 5m内 0.85→1.0
            <= 10f => 0.7f + (10f - distanceM) / 5f * 0.15f,  // 5-10m 0.7→0.85
            <= 30f => 0.3f + (30f - distanceM) / 20f * 0.4f,  // 10-30m 0.3→0.7
            _      => 0.1f + (50f - distanceM) / 20f * 0.2f,  // 30-50m 0.1→0.3
        };
        
        // 2. 深度衰减：越深信号越弱
        //    5cm → 1.0, 50cm → 0.4
        float depthSignal = 1.0f - (depthCm - 5f) / 45f * 0.6f;
        depthSignal = Mathf.Clamp(depthSignal, 0.4f, 1.0f);
        
        // 3. 时间加成：夜间信号感知更强（知了猴更活跃）
        float timeBonus = timeOfDay switch
        {
            >= 20f or <= 5f => 1.2f,  // 夜间
            >= 18f and < 20f => 1.1f, // 黄昏
            _ => 1.0f,                 // 白天
        };
        
        // 4. 加入微噪声模拟真实探测的不确定性
        float noise = 1.0f + (Mathf.PerlinNoise(distanceM * 0.5f, Time.time * 0.3f) - 0.5f) * 0.15f;
        
        return Mathf.Clamp(distSignal * depthSignal * timeBonus * weatherBonus * noise, 0f, 1f);
    }
}
```

### 3.3 多感官反馈系统

```csharp
public class ProximityFeedbackController : MonoBehaviour
{
    [Header("触觉反馈")]
    [SerializeField] private float _hapticMinInterval = 2.0f;  // 信号=0.1时的震动间隔
    [SerializeField] private float _hapticMaxInterval = 0.15f; // 信号=1.0时的震动间隔
    
    [Header("音频反馈")]
    [SerializeField] private AudioSource _sonarAudio;
    [SerializeField] private AnimationCurve _pitchCurve;       // 信号→音高映射
    
    [Header("视觉反馈")]
    [SerializeField] private SignalMeterUI _signalMeter;
    [SerializeField] private Light _phoneFlashlight;           // 手电筒频闪
    
    private float _currentSignal = 0f;
    
    public void UpdateSignal(float signal, NymphTarget target)
    {
        _currentSignal = signal;
        
        // === 触觉：震动频率随信号增强 ===
        float interval = Mathf.Lerp(_hapticMinInterval, _hapticMaxInterval, signal);
        if (Time.time - _lastHapticTime > interval && signal > 0.05f)
        {
            // 使用不同的震动强度区分"方向正确"和"方向偏离"
            float hapticStrength = signal > _previousSignal 
                ? signal * 1.2f   // 信号增强 → 更强震动（走对了）
                : signal * 0.8f;  // 信号减弱 → 较弱震动（走偏了）
            
            Handheld.Vibrate(); // 或使用更精细的 Haptic Engine API
            _lastHapticTime = Time.time;
        }
        
        // === 音频：类似声纳的脉冲音 ===
        float pitch = _pitchCurve.Evaluate(signal);  // 0→200Hz, 1→800Hz
        _sonarAudio.pitch = pitch;
        if (!_sonarAudio.isPlaying && signal > 0.05f)
        {
            _sonarAudio.PlayOneShot(_sonarClip); // 播放一次声纳脉冲
        }
        
        // === 视觉：信号表 ===
        _signalMeter.SetValue(signal);
        _signalMeter.SetColor(signal switch
        {
            < 0.2f => Color.gray,
            < 0.5f => Color.yellow,
            < 0.8f => Color.orange,
            _      => Color.red,
        });
        
        // === 手电筒频闪（夜间模式）===
        if (_isNightMode && signal > 0.6f)
        {
            float flashFreq = Mathf.Lerp(1f, 8f, (signal - 0.6f) / 0.4f);
            _phoneFlashlight.enabled = (Time.time % (1f / flashFreq)) < (0.5f / flashFreq);
        }
        
        _previousSignal = signal;
    }
}
```

### 3.4 信号表 UI 组件

```
┌─────────────────────────────────┐
│                                 │
│     ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓    │  ← 信号条（20段）
│     ███████████████░░░░░░░    │     填充率 = 信号强度
│                                 │
│     📶 信号强度: 72%            │
│     📏 距离预估: 约8.5m         │
│     📐 深度预估: 约20cm         │
│     ⬆️ 继续向前走...            │
│                                 │
└─────────────────────────────────┘
```

---

## 4. L3 AR 热力扫描层

### 4.1 功能定位

玩家进入 2-10m 范围后，**举起手机进入 AR 模式**。摄像头拍摄真实地面，叠加**地下热力信号可视化**。

这是整个体验中最"科幻感"的一层——用户感觉自己的手机变成了能透视地下的扫描仪。

### 4.2 AR 场景构成

```
        真实世界                           AR 叠加层
    ────────────────                ──────────────────
                                   
      🌳 树干                        🟢 树干识别框
        ││                                ││
      ┌─┘└──────────┐              ┌─────┘└────────────┐
      │   地面区域    │              │  🌊 热力波纹动画    │
      │  🟢🟢🟢🟢   │              │  (红→橙→黄渐变色)  │
      │ 🟢🔴🔴🟢   │              │                    │
      │  🟢🔴🟢    │              │  ✖ X标记点          │
      │   🟢🟢     │              │  ← 目标位置         │
      └────────────┘              └────────────────────┘
                                   
      📱 手机屏幕 = 两者合成
```

### 4.3 热力波纹生成算法

#### 在 GPU 上实时生成波纹

```glsl
// heat_ripple.shader — 热力波纹 Shader
// 在目标点周围生成向外扩散的同心波纹

Shader "CicadaHunt/HeatRipple"
{
    Properties
    {
        _MainTex ("Base (RGB)", 2D) = "white" {}
        _RippleSpeed ("Ripple Speed", Float) = 1.5
        _RippleSpacing ("Ripple Spacing", Float) = 0.8
        _RippleThickness ("Ripple Thickness", Float) = 0.15
        _MaxRadius ("Max Radius", Float) = 3.0
        _HeatIntensity ("Heat Intensity", Float) = 0.7
        _CenterUV ("Center UV", Vector) = (0.5, 0.5, 0, 0)
    }
    
    SubShader
    {
        Pass
        {
            CGPROGRAM
            #pragma vertex vert
            #pragma fragment frag
            
            float4 _CenterUV;       // 目标点在屏幕空间的位置
            float _RippleSpeed;
            float _RippleSpacing;
            float _RippleThickness;
            float _MaxRadius;
            float _HeatIntensity;
            
            fixed4 frag(v2f i) : SV_Target
            {
                // 1. 计算当前像素到目标点的距离
                float2 delta = i.uv - _CenterUV.xy;
                float dist = length(delta);
                
                // 2. 同心波纹：用 sin 产生间距递增的波纹
                //   波纹周期随距离增大（模拟真实波纹衰减）
                float ripple = sin(dist * _RippleSpacing * 6.28 - _Time.y * _RippleSpeed);
                ripple = abs(ripple);  // 全波整流 → 只有"峰"没有"谷"
                
                // 3. 波纹厚度：用 smoothstep 收窄
                ripple = smoothstep(1.0 - _RippleThickness, 1.0, ripple);
                
                // 4. 距离衰减：超过MaxRadius的波纹不显示
                float distAtten = 1.0 - smoothstep(0, _MaxRadius, dist);
                
                // 5. 径向渐变：中心最亮
                float radialGrad = 1.0 - smoothstep(0, _MaxRadius, dist);
                
                // 6. 组合热力颜色
                float intensity = (ripple * 0.6 + radialGrad * 0.4) * distAtten * _HeatIntensity;
                
                // 7. 热力色阶：黑→红→橙→黄
                fixed3 heatColor = lerp(
                    fixed3(0, 0, 0),        // 冷区=透明
                    fixed3(1, 0.1, 0),      // 红
                    saturate(intensity * 2)
                );
                heatColor = lerp(heatColor, fixed3(1, 0.6, 0), saturate((intensity - 0.3) * 3));
                heatColor = lerp(heatColor, fixed3(1, 0.9, 0.2), saturate((intensity - 0.6) * 4));
                
                return fixed4(heatColor, intensity * 0.8); // alpha用于混合
            }
            ENDCG
        }
    }
}
```

#### C# 控制脚本

```csharp
public class ARHeatRippleController : MonoBehaviour
{
    [SerializeField] private Material _heatRippleMaterial;
    [SerializeField] private float _maxRadius = 3.0f;
    
    private Camera _arCamera;
    private List<NymphARTarget> _activeTargets = new();
    
    void Update()
    {
        foreach (var target in _activeTargets)
        {
            // 1. 将知了猴世界坐标投影到屏幕
            Vector3 screenPos = _arCamera.WorldToScreenPoint(target.WorldPosition);
            
            // 2. 检查是否在屏幕内
            if (screenPos.z < 0 || screenPos.x < 0 || screenPos.x > Screen.width 
                || screenPos.y < 0 || screenPos.y > Screen.height)
            {
                target.IsOnScreen = false;
                continue;
            }
            
            target.IsOnScreen = true;
            Vector2 screenUV = new Vector2(
                screenPos.x / Screen.width,
                screenPos.y / Screen.height
            );
            
            // 3. 调整波纹强度（随距离衰减）
            float distance = Vector3.Distance(_arCamera.transform.position, target.WorldPosition);
            float intensity = Mathf.Lerp(1.0f, 0.1f, distance / 10f);
            
            // 4. 更新 Shader 参数
            // 实际渲染时用 Graphics.Blit 或 CommandBuffer 为每个目标画一个波纹
            target.RippleIntensity = intensity;
            target.ScreenUV = screenUV;
        }
    }
    
    // 使用 Graphics.DrawMeshInstanced 高效渲染多个波纹
    void RenderRipples()
    {
        // 为屏幕上的每个目标渲染一个热力波纹Quad
        var visibleTargets = _activeTargets.Where(t => t.IsOnScreen).ToList();
        
        Matrix4x4[] matrices = new Matrix4x4[visibleTargets.Count];
        Vector4[] centers = new Vector4[visibleTargets.Count];
        
        for (int i = 0; i < visibleTargets.Count; i++)
        {
            // 每个波纹Quad放置在屏幕空间对应位置
            Vector3 worldPos = visibleTargets[i].WorldPosition;
            Quaternion rot = Quaternion.LookRotation(
                worldPos - _arCamera.transform.position, 
                Vector3.up
            );
            float scale = visibleTargets[i].RippleIntensity * _maxRadius * 2;
            matrices[i] = Matrix4x4.TRS(worldPos, rot, new Vector3(scale, scale, 1));
            centers[i] = new Vector4(0.5f, 0.5f, 0, 0); // Quad中心
        }
        
        MaterialPropertyBlock props = new MaterialPropertyBlock();
        props.SetVectorArray("_CenterUV", centers);
        
        Graphics.DrawMeshInstanced(
            _rippleQuadMesh,    // 一个简单的四边形
            0,
            _heatRippleMaterial,
            matrices,
            visibleTargets.Count,
            props
        );
    }
}
```

### 4.4 方位指示器

当目标不在屏幕内时，在屏幕边缘显示方向箭头：

```csharp
public class AROffscreenIndicator : MonoBehaviour
{
    public RectTransform ArrowPrefab;
    private Camera _arCamera;
    private RectTransform _canvasRect;
    
    void Update()
    {
        foreach (var target in _activeTargets)
        {
            if (target.IsOnScreen) 
            {
                target.Arrow?.gameObject.SetActive(false);
                continue;
            }
            
            // 1. 计算目标在相机空间的方向
            Vector3 dirToTarget = (target.WorldPosition - _arCamera.transform.position).normalized;
            Vector3 camForward = _arCamera.transform.forward;
            
            // 2. 计算屏幕边缘交点
            Vector3 screenCenter = new Vector3(Screen.width / 2, Screen.height / 2, 0);
            Vector3 screenDir = _arCamera.WorldToScreenPoint(
                _arCamera.transform.position + dirToTarget * 10f
            ) - screenCenter;
            
            // 3. 将箭头限制在屏幕边缘
            Vector2 clampedPos = ClampToScreenEdge(screenCenter, screenDir.normalized);
            
            target.Arrow.anchoredPosition = clampedPos;
            
            // 4. 旋转箭头指向目标方向
            float angle = Mathf.Atan2(screenDir.y, screenDir.x) * Mathf.Rad2Deg;
            target.Arrow.rotation = Quaternion.Euler(0, 0, angle - 90);
            
            // 5. 显示距离
            target.ArrowDistanceText.text = $"{target.Distance:F0}m";
        }
    }
    
    // 将方向向量映射到屏幕边缘
    Vector2 ClampToScreenEdge(Vector3 screenCenter, Vector3 dir)
    {
        float halfW = Screen.width / 2f - 40f;  // 留40px边距
        float halfH = Screen.height / 2f - 40f;
        
        float tx = dir.x == 0 ? float.MaxValue : halfW / Mathf.Abs(dir.x);
        float ty = dir.y == 0 ? float.MaxValue : halfH / Mathf.Abs(dir.y);
        float t = Mathf.Min(tx, ty);
        
        return new Vector2(dir.x * t, dir.y * t);
    }
}
```

---

## 5. L4 精确挖掘层

### 5.1 功能定位

玩家到达距离<2m的精确位置后，进入**AR俯视挖掘模式**。这是整个交互的终点——玩家对准地面进行挖掘操作。

### 5.2 精确标记定位

```csharp
public class ARPreciseMarker : MonoBehaviour
{
    // 从生成系统获得的知了猴GPS坐标
    private Vector2 _nymphLatLng;
    private float _nymphDepth; // cm
    
    // 定位到世界坐标
    private Vector3 _nymphWorldPos;
    
    // GPS误差模拟：实际位置有 ±30cm 的误差（增加真实感）
    private Vector2 _gpsOffset;
    
    void Start()
    {
        // 真实GPS有1-3m误差，但游戏中将其收窄到±30cm
        // 引入误差让玩家需要在附近"搜寻"一下，增加交互感
        _gpsOffset = new Vector2(
            Random.Range(-0.3f, 0.3f),
            Random.Range(-0.3f, 0.3f)
        );
        
        // 使用 Google Geospatial API 或 ARKit/ARCore 锚点
        // 将GPS坐标转换为Unity世界坐标
        _nymphWorldPos = GeospatialToWorld(_nymphLatLng + _gpsOffset);
    }
}
```

### 5.3 挖掘交互设计

#### 交互状态机

```
                手机对准地面
                     ↓
    ┌──────────────────────────────────┐
    │  SCANNING — 扫描定位              │
    │  · X标记在地面浮动（±30cm不确定） │
    │  · 玩家移动手机"扫描"地面         │
    │  · 越靠近真实位置，X标记越稳定     │
    │  · X标记收缩至确定位置后自动锁定   │
    └───────────────┬──────────────────┘
                    │ X标记锁定
                    ↓
    ┌──────────────────────────────────┐
    │  AIMING — 瞄准确认               │
    │  · X标记变为红色+脉冲             │
    │  · 显示深度指示（"深约20cm"）      │
    │  · 提示：点击铲子按钮开始挖掘      │
    └───────────────┬──────────────────┘
                    │ 点击铲子
                    ↓
    ┌──────────────────────────────────┐
    │  DIGGING — 挖掘中                │
    │  · 屏幕显示铲子+进度环            │
    │  · 每次滑动画圈 = 挖一铲          │
    │  · 进度条逐步填满                 │
    │  · 粒子效果：泥土飞溅             │
    └───────────────┬──────────────────┘
                    │ 进度100%
                    ↓
    ┌──────────────────────────────────┐
    │  UNEARTHED — 出土!              │
    │  · 地面裂开动画                   │
    │  · 知了猴3D模型从土中升起          │
    │  · 品种鉴定动画                   │
    └──────────────────────────────────┘
```

### 5.4 挖掘手势识别

```csharp
public class DigGestureRecognizer : MonoBehaviour
{
    [Header("挖掘参数")]
    [SerializeField] private int _totalDigsRequired = 6;  // 基础挖掘次数
    [SerializeField] private float _swipeThreshold = 50f; // 最小滑动距离（像素）
    
    private int _currentDigs = 0;
    private float _progress = 0f;
    
    // 挖掘动作：在屏幕上画圈拖动
    void Update()
    {
        if (_state != DigState.DIGGING) return;
        
        if (Input.touchCount > 0)
        {
            Touch touch = Input.GetTouch(0);
            
            switch (touch.phase)
            {
                case TouchPhase.Began:
                    _swipeStartPos = touch.position;
                    break;
                    
                case TouchPhase.Moved:
                    float swipeDist = Vector2.Distance(_swipeStartPos, touch.position);
                    
                    // 滑过阈值 → 计一铲
                    if (swipeDist >= _swipeThreshold)
                    {
                        PerformDig();
                        _swipeStartPos = touch.position; // 重置起点，支持连续挖掘
                    }
                    break;
            }
        }
        
        // 也支持鼠标操作（编辑器中测试）
        #if UNITY_EDITOR
        if (Input.GetMouseButtonDown(0))
        {
            _swipeStartPos = Input.mousePosition;
        }
        if (Input.GetMouseButton(0))
        {
            float swipeDist = Vector2.Distance(_swipeStartPos, Input.mousePosition);
            if (swipeDist >= _swipeThreshold)
            {
                PerformDig();
                _swipeStartPos = Input.mousePosition;
            }
        }
        #endif
    }
    
    void PerformDig()
    {
        _currentDigs++;
        
        // 实际需要的次数受工具效率影响
        float efficiency = _inventory.GetCurrentTool().Efficiency;
        _progress += 1f / (_totalDigsRequired / efficiency);
        
        // 触发反馈
        _hapticController.TriggerDigHaptic();
        _audioController.PlayDigSound(_currentDigs);
        _vfxController.SpawnDirtParticles(_nymphWorldPos);
        
        // 更新UI
        _digUI.SetProgress(_progress);
        
        if (_progress >= 1.0f)
        {
            OnDiggingComplete();
        }
    }
}
```

### 5.5 深度可视化

```csharp
// AR深度指示器：在X标记旁显示地下深度
public class DepthIndicator : MonoBehaviour
{
    public void RenderDepthGuide(float depthCm)
    {
        // 从X标记点向下延伸一条虚拟的深度标尺
        // 
        //     地面 ─────────────────
        //          │
        //          │ ← 深度标尺（半透明）
        //          │    标注"5cm", "10cm", "15cm"...
        //          │
        //    ═══════ ← 知了猴所在深度（脉冲光点）
        //          │
        //     深处 ──┘
        
        Vector3 groundPos = _nymphWorldPos;
        Vector3 nymphPos = groundPos + Vector3.down * (depthCm / 100f);
        
        // 画线渲染器
        _lineRenderer.SetPosition(0, groundPos);
        _lineRenderer.SetPosition(1, nymphPos);
        
        // 目标深度光点
        _depthMarker.transform.position = nymphPos;
        float pulse = 0.7f + Mathf.Sin(Time.time * 3f) * 0.3f;
        _depthMarker.material.SetFloat("_Alpha", pulse);
    }
}
```

---

## 6. 挖掘判定与命中算法

### 6.1 三级判定体系

```
玩家点击挖掘 → 进入判定管线

┌─────────────────────────────────────────────┐
│                                             │
│   L1: 距离判定 (Distance Check)              │
│   ────────────────────────────              │
│   玩家GPS位置 vs 知了猴GPS位置                │
│   有效距离：< 2m                             │
│   FAIL → "你离目标太远了，请靠近一些"          │
│                                             │
│   L2: 方位判定 (Direction Check)             │
│   ──────────────────────────────            │
│   手机摄像头朝向 vs 知了猴所在方向             │
│   有效角度：±45°（锥形）                      │
│   FAIL → "请将手机对准目标方向"               │
│                                             │
│   L3: 精度判定 (Precision Check)             │
│   ──────────────────────────────            │
│   AR中的X标记与手机指向的交叉点               │
│   有效偏差：< 15cm                           │
│   → 决定挖掘成功率(60%-95%)                  │
│                                             │
└─────────────────────────────────────────────┘
```

### 6.2 完整判定代码

```csharp
public class DigHitEvaluator
{
    // 三级判定，返回挖掘结果
    public DigResult EvaluateDig(
        Vector2 playerLatLng,
        Vector3 phoneForward,      // 手机摄像头朝向
        Vector3 phoneUp,           // 手机上方向
        Vector2 nymphLatLng,
        float nymphDepth,
        ToolStats tool)
    {
        var result = new DigResult();
        
        // ==========================================
        // L1: 距离判定
        // ==========================================
        float distanceM = HaversineDistance(playerLatLng, nymphLatLng);
        result.Distance = distanceM;
        
        if (distanceM > tool.MaxReachM)
        {
            result.Passed = false;
            result.FailReason = FailReason.TooFar;
            result.FeedbackMessage = $"距离目标还有 {distanceM:F1}m，再靠近一点！";
            return result;
        }
        
        // ==========================================
        // L2: 方位判定
        // ==========================================
        Vector3 dirToNymph = GeoToWorldDirection(playerLatLng, nymphLatLng);
        float angle = Vector3.Angle(phoneForward, dirToNymph);
        result.Angle = angle;
        
        if (angle > 45f)
        {
            result.Passed = false;
            result.FailReason = FailReason.WrongDirection;
            result.FeedbackMessage = angle switch
            {
                > 135f => "完全反了！转身看看？",
                > 90f  => "方向偏差较大，请向右/左转",
                _      => "请稍微调整手机方向...",
            };
            return result;
        }
        
        // ==========================================
        // L3: 精度判定 (AR 空间交叉)
        // ==========================================
        // 使用手机的AR射线与地面的交点
        Ray arRay = new Ray(phonePosition, phoneForward);
        Plane groundPlane = new Plane(Vector3.up, nymphWorldPos.y);
        
        float hitDistance;
        if (!groundPlane.Raycast(arRay, out hitDistance))
        {
            result.Passed = false;
            result.FailReason = FailReason.NotAimingAtGround;
            result.FeedbackMessage = "请将手机对准地面！";
            return result;
        }
        
        Vector3 arHitPoint = arRay.GetPoint(hitDistance);
        float deviationCm = Vector3.Distance(arHitPoint, nymphWorldPos) * 100f;
        result.DeviationCm = deviationCm;
        
        // 成功率由偏差距离决定
        float successRate = CalculateSuccessRate(deviationCm, tool);
        result.SuccessRate = successRate;
        
        result.Passed = Random.value < successRate;
        
        if (!result.Passed)
        {
            result.FailReason = FailReason.Missed;
            result.FeedbackMessage = deviationCm switch
            {
                < 10f => "差一点点！再试一次！",
                < 20f => "偏了一点，调整下位置...",
                _     => "没挖到，请对准X标记",
            };
        }
        
        return result;
    }
    
    // 成功率曲线
    float CalculateSuccessRate(float deviationCm, ToolStats tool)
    {
        // 偏差0cm → 基础成功率 = tool.Accuracy
        // 偏差15cm → 成功率 = tool.Accuracy * 0.3
        // 偏差30cm → 成功率 = 0.05 (几乎不中)
        
        float baseRate = tool.Accuracy;  // 工具基础精度 (0.6~0.95)
        float penalty = Mathf.Pow(deviationCm / 30f, 2); // 二次衰减
        return Mathf.Clamp(baseRate * (1f - penalty), 0.05f, 1f);
    }
}

public struct DigResult
{
    public bool Passed;
    public FailReason FailReason;
    public string FeedbackMessage;
    public float Distance;
    public float Angle;
    public float DeviationCm;
    public float SuccessRate;
}

public enum FailReason
{
    None,
    TooFar,             // 距离太远
    WrongDirection,     // 方向不对
    NotAimingAtGround,  // 没对准地面
    Missed,             // 偏差太大没挖到
}
```

### 6.3 工具效率计算

```csharp
// 不同工具的挖掘参数
public static class ToolDatabase
{
    public static Dictionary<string, ToolStats> Tools = new()
    {
        ["bare_hand"] = new ToolStats
        {
            Name = "徒手",
            MaxReachM = 1.0f,
            Accuracy = 0.60f,       // 基础成功率60%
            DigSpeed = 1.0f,        // 基础挖掘速度
            DigsRequired = 12,      // 需要挖12次
            DepthBonus = 0f,        // 无深度优势
        },
        ["small_shovel"] = new ToolStats
        {
            Name = "小铲子",
            MaxReachM = 1.5f,
            Accuracy = 0.80f,
            DigSpeed = 1.5f,
            DigsRequired = 8,
            DepthBonus = 0.1f,      // 深挖效率+10%
        },
        ["pro_shovel"] = new ToolStats
        {
            Name = "专业挖掘铲",
            MaxReachM = 2.0f,
            Accuracy = 0.95f,
            DigSpeed = 2.2f,
            DigsRequired = 5,
            DepthBonus = 0.3f,
        },
    };
}
```

---

## 7. AR 核心技术实现

### 7.1 地面检测与平面识别

```csharp
public class ARGroundDetector : MonoBehaviour
{
    [SerializeField] private ARPlaneManager _planeManager;
    [SerializeField] private float _minPlaneArea = 0.5f; // 最小平面面积 m²
    
    private ARPlane _groundPlane;
    
    void OnEnable()
    {
        _planeManager.planesChanged += OnPlanesChanged;
    }
    
    void OnPlanesChanged(ARPlanesChangedEventArgs args)
    {
        // 筛选地面平面（法线朝上的水平面）
        foreach (var plane in args.added)
        {
            if (IsGroundPlane(plane))
            {
                if (_groundPlane == null || plane.size.magnitude > _groundPlane.size.magnitude)
                {
                    _groundPlane = plane;
                }
            }
        }
        
        // 更新地面信息
        foreach (var plane in args.updated)
        {
            if (plane == _groundPlane)
            {
                OnGroundPlaneUpdated();
            }
        }
    }
    
    bool IsGroundPlane(ARPlane plane)
    {
        // 地面平面的法线应该指向上方（允许±20°偏差）
        float upAngle = Vector3.Angle(plane.normal, Vector3.up);
        return upAngle < 20f && plane.alignment == PlaneAlignment.Horizontal;
    }
    
    // 获取地面在给定屏幕点的世界坐标
    public bool RaycastGround(Vector2 screenPoint, out Vector3 worldPos)
    {
        Ray ray = _arCamera.ScreenPointToRay(screenPoint);
        
        // 1. 优先使用 ARPlane 的 Raycast
        var hits = new List<ARRaycastHit>();
        if (_arRaycastManager.Raycast(screenPoint, hits, TrackableType.PlaneWithinPolygon))
        {
            worldPos = hits[0].pose.position;
            return true;
        }
        
        // 2. 降级方案：使用已知地面平面的数学交线
        if (_groundPlane != null)
        {
            Plane plane = new Plane(_groundPlane.normal, _groundPlane.center);
            float enter;
            if (plane.Raycast(ray, out enter))
            {
                worldPos = ray.GetPoint(enter);
                return true;
            }
        }
        
        worldPos = Vector3.zero;
        return false;
    }
}
```

### 7.2 GPS → 世界坐标转换

```csharp
public class GeoToWorldConverter : MonoBehaviour
{
    [SerializeField] private float _referenceLat;
    [SerializeField] private float _referenceLng;
    [SerializeField] private float _referenceAlt;
    
    private Vector3 _referenceWorldPos;
    private bool _isCalibrated;
    
    // 校准：在已知GPS坐标的位置设定世界原点
    public void Calibrate(Vector2 latLng, float altitude)
    {
        _referenceLat = latLng.x;
        _referenceLng = latLng.y;
        _referenceAlt = altitude;
        _referenceWorldPos = transform.position;
        _isCalibrated = true;
    }
    
    // 将GPS坐标转为Unity世界坐标
    public Vector3 GeoToWorld(double lat, double lng, double alt = 0)
    {
        // 使用 ENU (East-North-Up) 坐标系转换
        
        // 1. 计算经纬度差异带来的位移（米）
        double dLat = (lat - _referenceLat) * 111320.0; // 1度纬度 ≈ 111320m
        double dLng = (lng - _referenceLng) * 111320.0 * Math.Cos(_referenceLat * Math.PI / 180.0);
        double dAlt = alt - _referenceAlt;
        
        // 2. 映射到Unity坐标 (ENU → Unity: X=East, Y=Up, Z=North)
        Vector3 offset = new Vector3(
            (float)dLng,   // X = 东
            (float)dAlt,   // Y = 上
            (float)dLat    // Z = 北
        );
        
        return _referenceWorldPos + offset;
    }
    
    // 方向向量: 从GPS_A指向GPS_B (用于方位判定)
    public Vector3 GeoToWorldDirection(Vector2 from, Vector2 to)
    {
        Vector3 posA = GeoToWorld(from.x, from.y);
        Vector3 posB = GeoToWorld(to.x, to.y);
        return (posB - posA).normalized;
    }
}
```

> **生产环境建议**：直接使用 **Google Geospatial API** 或 **Niantic Lightship** 的 VPS（Visual Positioning System），它们提供了更精准的 GPS→AR 映射，CM级定位精度，无需手动校准。

### 7.3 不稳定地面处理

真实户外场景中，地面不总是平坦的：

```csharp
public class RoughGroundHandler : MonoBehaviour
{
    // 当AR检测到的地面有坡度或凹凸时，调整知了猴的位置
    
    public Vector3 AdjustNymphPositionToTerrain(Vector3 idealPos)
    {
        // 1. 向下发射射线，找到实际地面
        RaycastHit hit;
        if (Physics.Raycast(idealPos + Vector3.up * 2f, Vector3.down, out hit, 5f, LayerMask.GetMask("ARPlane", "Ground")))
        {
            // 将知了猴放置在实际地面上
            return hit.point;
        }
        
        // 2. 如果找不到地面（如草丛中），使用AR平面估计
        if (_groundPlane != null)
        {
            Plane plane = new Plane(_groundPlane.normal, _groundPlane.center);
            float enter;
            Ray ray = new Ray(idealPos + Vector3.up * 2f, Vector3.down);
            if (plane.Raycast(ray, out enter))
            {
                return ray.GetPoint(enter);
            }
        }
        
        return idealPos; // fallback
    }
    
    // 检测地面是否适合挖掘
    public bool IsSurfaceDiggable(Vector3 position)
    {
        // 排除：水面、水泥地、岩石、太陡的坡
        
        // 1. 坡度检查
        if (_groundPlane != null)
        {
            float slope = Vector3.Angle(_groundPlane.normal, Vector3.up);
            if (slope > 30f) return false; // 太陡
        }
        
        // 2. 语义分割检查（如果设备支持）
        if (_semanticSegmentation != null)
        {
            string surfaceType = _semanticSegmentation.GetLabel(position);
            return surfaceType switch
            {
                "grass" or "dirt" or "soil" or "gravel" => true,
                "water" or "concrete" or "asphalt" or "building" => false,
                _ => true, // 未知表面默认允许
            };
        }
        
        return true;
    }
}
```

---

## 8. 动效与音效系统

### 8.1 挖掘粒子效果

```csharp
public class DigVFXController : MonoBehaviour
{
    [Header("粒子系统")]
    [SerializeField] private ParticleSystem _dirtParticles;
    [SerializeField] private ParticleSystem _dustParticles;
    [SerializeField] private ParticleSystem _unearthedParticles; // 出土金光
    
    [Header("材质动画")]
    [SerializeField] private Material _groundCrackMaterial;
    
    public void PlayDigVFX(int digNumber, Vector3 position, float depthCm, ToolStats tool)
    {
        // 1. 泥土飞溅
        _dirtParticles.transform.position = position;
        var dirtEmission = _dirtParticles.emission;
        dirtEmission.rateOverTime = 10 + digNumber * 2; // 越挖越多
        _dirtParticles.Play();
        
        // 2. 尘土扬起
        _dustParticles.transform.position = position + Vector3.up * 0.1f;
        var dustMain = _dustParticles.main;
        dustMain.startColor = new ParticleSystem.MinMaxGradient(
            new Color(0.4f, 0.3f, 0.2f, 0.6f), // 浅土色
            new Color(0.2f, 0.15f, 0.1f, 0.8f)  // 深土色
        );
        _dustParticles.Play();
        
        // 3. 地面裂痕扩散
        float crackProgress = (float)digNumber / tool.DigsRequired;
        _groundCrackMaterial.SetFloat("_CrackProgress", crackProgress);
        
        // 4. 工具音效
        PlayToolSFX(digNumber, tool);
    }
    
    public void PlayUnearthVFX(Vector3 position, NymphAttributes nymph)
    {
        // 出土特效:
        // 1. 地面裂开
        // 2. 知了猴3D模型升起
        // 3. 金光/粒子环绕
        // 4. 品质光环（白=普通, 蓝=稀有, 紫=罕见, 金=传说）
        
        _unearthedParticles.transform.position = position;
        var main = _unearthedParticles.main;
        main.startColor = nymph.Quality switch
        {
            5 => new Color(1f, 0.84f, 0f),    // 金
            4 => new Color(0.6f, 0.2f, 1f),   // 紫
            3 => new Color(0.2f, 0.5f, 1f),   // 蓝
            2 => new Color(0.3f, 0.8f, 0.3f), // 绿
            _ => new Color(0.8f, 0.8f, 0.8f), // 白
        };
        _unearthedParticles.Play();
    }
}
```

### 8.2 音效分级设计

```
场景状态              音效层                          说明
────────────────────────────────────────────────────────────
L1 地图雷达           环境底噪 + 低频嗡嗡              类似雷达扫描
L2 近场探测           声纳脉冲（间隔随距离缩短）        类似潜艇声纳
L3 AR热力扫描         电子啸叫声（音高随信号增强）      类似盖革计数器
L4 挖掘中             铲子掘土声 + 泥土沙沙声          拟真挖掘音效
  ├ 铲子接触硬土      沉闷的"咚"声
  ├ 铲子接触软土      清脆的"唰"声
  └ 接近知了猴        心跳加速音（逐渐变快变响）
L4 出土               短促上扬旋律 + 品质提示音         获得反馈
```

### 8.3 触觉 (Haptics) 设计

```csharp
public class DigHapticsController
{
    // iOS CoreHaptics / Android VibrationEffect 抽象
    public void PlayHapticPattern(DigPhase phase, float intensity)
    {
        switch (phase)
        {
            case DigPhase.Scanning:
                // 短暂的"滴"震动，随信号增强频率提高
                PlayTransientHaptic(intensity * 0.3f, 0.05f);
                break;
                
            case DigPhase.XMarkLocked:
                // 强烈的"锁定了"震动 — 一次重击
                PlayTransientHaptic(1.0f, 0.15f);
                break;
                
            case DigPhase.Digging:
                // 每次铲子入土 — 带有"撞击感"的震动
                // 使用 AHAP (Apple Haptic Audio Pattern) 定义铲子入土曲线
                PlayDigImpactHaptic(intensity);
                break;
                
            case DigPhase.NearingNymph:
                // 连续心跳震动：咚-咚-咚 越来越快
                float interval = Mathf.Lerp(0.8f, 0.25f, intensity);
                PlayHeartbeatHaptic(interval);
                break;
                
            case DigPhase.Unearth:
                // 先升后降的庆祝震动
                PlayRampUpHaptic(0.5f, 1.0f); // 0.5秒从0到满
                break;
        }
    }
}
```

---

## 9. 多机型适配

### 9.1 能力分级

```
级别      机型                                     AR能力
─────────────────────────────────────────────────────────────
Tier 1    iPhone 14/15 Pro+ (LiDAR)               完整AR体验
          · LiDAR 实时深度图
          · 精确地面检测
          · 语义分割

Tier 2    iPhone XR+ / Android ARCore 旗舰机       标准AR体验
          · 视觉惯性里程计(VIO)
          · AR平面检测
          · 基本地面识别

Tier 3    iPhone 8+ / Android ARCore 中端机         简化AR体验
          · 基础AR功能
          · 可能无平面检测

Tier 4    低端/无AR支持设备                         降级体验
          · 无AR模式
          · 仅地图+信号条模式
```

### 9.2 降级策略

```csharp
public class ARCapabilityManager : MonoBehaviour
{
    public enum DeviceTier { Tier1_LiDAR, Tier2_FullAR, Tier3_BasicAR, Tier4_NoAR }
    
    public DeviceTier CurrentTier { get; private set; }
    
    void Start()
    {
        CurrentTier = DetectDeviceTier();
        ConfigureARExperience();
    }
    
    DeviceTier DetectDeviceTier()
    {
        #if UNITY_IOS
        // 检测LiDAR
        if (SystemInfo.deviceModel.Contains("iPhone14") ||  // Pro系列
            SystemInfo.deviceModel.Contains("iPhone15") ||
            SystemInfo.deviceModel.Contains("iPhone16"))
        {
            return DeviceTier.Tier1_LiDAR;
        }
        return ARSession.state == ARSessionState.Ready ? DeviceTier.Tier2_FullAR : DeviceTier.Tier3_BasicAR;
        #endif
        
        #if UNITY_ANDROID
        // 检查 ARCore 支持级别
        return Google.AR.Core.ArCoreApk.Instance.IsAvailable() 
            ? DeviceTier.Tier2_FullAR 
            : DeviceTier.Tier4_NoAR;
        #endif
    }
    
    void ConfigureARExperience()
    {
        switch (CurrentTier)
        {
            case DeviceTier.Tier1_LiDAR:
                // 全功能：热力波纹 + 深度遮挡 + 地面语义分割
                _heatRippleController.EnableFullQuality();
                _arOcclusionManager.enabled = true;  // LiDAR遮挡
                _semanticSegmentation.Enable();
                break;
                
            case DeviceTier.Tier2_FullAR:
                // 标准AR：热力波纹 + 基础平面检测
                _heatRippleController.EnableStandardQuality();
                _arOcclusionManager.enabled = false; // 无硬件遮挡
                break;
                
            case DeviceTier.Tier3_BasicAR:
                // 简化AR：仅2D热力叠加（着色Quad跟随相机）
                _heatRippleController.EnableSimpleMode();
                break;
                
            case DeviceTier.Tier4_NoAR:
                // 降级模式：2D俯视地图上的"模拟挖掘"
                // 点击挖掘按钮 → 播放挖掘动画 → 判定
                _arSession.enabled = false;
                _2DDigModeController.Enable();
                break;
        }
    }
}
```

### 9.3 无AR降级挖掘

```csharp
// Tier 4 设备用纯2D模式替代AR
public class TwoDDigModeController : MonoBehaviour
{
    [SerializeField] private Transform _playerMarker;  // 地图上玩家位置
    [SerializeField] private Transform _nymphMarker;   // 地图上知了猴位置
    
    public void StartDigging()
    {
        // 玩家到达知了猴附近后
        float distance = Vector2.Distance(
            new Vector2(_playerMarker.position.x, _playerMarker.position.z),
            new Vector2(_nymphMarker.position.x, _nymphMarker.position.z)
        );
        
        if (distance < 2f)
        {
            // 切换到挖掘小游戏界面（纯UI，无AR）
            _digMiniGame.Show();
        }
    }
}

// 2D挖掘小游戏
public class TwoDDigMiniGame : MonoBehaviour
{
    // 类似"黄金矿工"式小游戏：
    // ┌────────────────────┐
    // │                    │
    // │     [铲子] ←左右→  │  ← 左右移动到X标记上方
    // │        ↓           │
    // │   ────X──── 地面   │
    // │   │  │  │         │
    // │   │  │  │ 深度标尺 │  ← 点击向下挖掘
    // │   │ 🦗 │         │
    // │   └──┴──┘         │
    // │                    │
    // │  挖掘进度 ████░░░   │
    // └────────────────────┘
    
    private float _xPosition;       // 铲子左右位置 [-1, 1]
    private float _nymphXPosition;  // 知了猴X位置（随机偏移）
    
    void Update()
    {
        // AD键或手指左右滑动 → 移动铲子
        _xPosition += Input.GetAxis("Horizontal") * Time.deltaTime * 3f;
        _xPosition = Mathf.Clamp(_xPosition, -1f, 1f);
        
        if (Input.GetButtonDown("Fire1")) // 点击挖掘
        {
            float deviation = Mathf.Abs(_xPosition - _nymphXPosition);
            DigAndEvaluate(deviation);
        }
    }
}
```

---

## 附录A：挖掘交互完整事件流

```
时间轴    事件                          客户端                          服务端
────────────────────────────────────────────────────────────────────────────
T+0s    进入L4 AR挖掘模式           加载地面检测+热力渲染              
T+1s    X标记搜索阶段开始           显示浮动X标记+方向引导             
T+5s    X标记锁定                   震动+音效反馈                    
T+6s    玩家开始挖掘                手势识别+粒子效果                
T+6-12s 挖掘6次（工具差异）         每次触发触觉+音效+泥土粒子         
T+12s   挖掘完成                    请求服务端验证 ──────────────→  验证通过
T+13s   出土动画                    知了猴3D模型升起+品质特效  ←──  返回属性
T+15s   收获展示                    品种卡片弹出                     
T+18s   完成                        更新背包+图鉴                    
```

## 附录B：AR锚点持久化

```
多人场景下，被挖掘后的知了猴需要对所有玩家标记为"已挖走"：

1. Google Cloud Anchors 或 Niantic Lightship 共享AR锚点
2. 玩家A挖掘成功 → 服务端标记consumed
3. 服务端推送更新给附近玩家B
4. 玩家B的地图上该标记自动消失或变为灰色"已挖"
```

---

> 📌 **下一篇**：[方向2：玩法二捕蝉雷达与蝉AI行为树](抓知了猴_方向2_捕蝉雷达与蝉AI技术方案.md)
