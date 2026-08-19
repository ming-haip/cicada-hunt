export const COLORS = {
  bg: '#0d1b0d',
  bgCard: '#152615',
  bgSurface: '#1a301a',
  primary: '#4caf50',
  primaryDark: '#2e7d32',
  gold: '#ffc107',
  danger: '#e53935',
  text: '#e8e8e8',
  textDim: '#88a088',
  border: '#2a402a',
  white: '#ffffff',
};

export const RARITY_COLORS = {
  1: '#888888',
  2: '#4caf50',
  3: '#2196f3',
  4: '#9c27b0',
  5: '#ffc107',
};

export const WEATHER_SCENES = {
  sunny: { bg: '#1a2a0d', icon: '☀️', label: '晴天', treeImg: 'tree_sunny' },
  cloudy: { bg: '#1a221a', icon: '☁️', label: '多云', treeImg: 'tree_cloudy' },
  rainy: { bg: '#0d1a1a', icon: '🌧️', label: '下雨', treeImg: 'tree_rainy' },
  night: { bg: '#0a0f1a', icon: '🌙', label: '夜晚', treeImg: 'tree_night' },
  snowy: { bg: '#1a1a1a', icon: '❄️', label: '下雪', treeImg: 'tree_snowy' },
};

export const ONBOARDING_STEPS = [
  {
    title: '欢迎来到抓知了猴 🦗',
    description: '知了是世界上寿命最长的昆虫之一，它们在地下潜伏2-17年，然后在夏夜破土而出、蜕壳羽化，在阳光下歌唱短短60天后便完成生命轮回。\n\n"数年蛰伏，一夏高歌"——这就是知了的传奇一生。',
    image: 'onboard_intro',
  },
  {
    title: '⛏️ 挖掘知了猴',
    description: '知了猴（若虫）潜伏在树根附近的泥土中。\n\n打开地图查看周边分布 → 走向目标 → 用铲子挖出！\n\n💡 提示：杨树、柳树、榆树下的泥土里最多知了猴。夜晚和雨后是挖掘的最佳时机。',
    image: 'onboard_dig',
  },
  {
    title: '🥅 捕捉蝉',
    description: '知了猴蜕壳后变成会飞的蝉。\n\n使用蝉雷达扫描周边 → 悄悄接近树枝上的蝉 → 挥网捕捉！\n\n💡 提示：从蝉的背后悄悄靠近，快速挥网！惊动它就会飞走。',
    image: 'onboard_catch',
  },
  {
    title: '🌿 准备好了吗？',
    description: '走到户外，在大自然中寻找知了的踪迹吧！\n\n真实天气和环境会影响知了猴的出现。\n每只知了猴都有不同的品种和品质，\n收集图鉴，成为知了大师！',
    image: 'onboard_ready',
  },
];

export const TUTORIAL_TIPS = {
  firstMapOpen: '🗺️ 这是地图视图，绿色区域表示知了猴密度高。点击知了猴标记可以查看详情并追踪。',
  firstTrack: '🎯 你正在追踪这只知了猴！底部的信号条会随着你靠近而增强，当信号变红时就可以挖掘了！',
  firstDig: '⛏️ 将手机对准地面，等待X标记锁定（变金色），然后快速滑动屏幕来挖掘！',
  firstCatch: '🥅 慢慢靠近树枝上的蝉，从它的背后悄悄接近。进入范围后点击挥网按钮！',
  nightMode: '🌙 现在是夜晚！知了猴在夜间出土更活跃。打开手机手电筒可以扩大探测范围。',
  rainyDay: '🌧️ 下雨了！雨后的泥土松软湿润，知了猴更容易出土。出现率+30%！',
  winterMode: '❄️ 冬季知了猴都在地下深处冬眠，很难找到。等到夏天再来吧！',
};
