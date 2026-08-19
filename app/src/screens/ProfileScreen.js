import React, { useState, useEffect } from 'react';
import {
  View, Text, StyleSheet, ScrollView, TouchableOpacity, RefreshControl,
} from 'react-native';
import { useGame } from '../hooks/useGame';
import { apiService } from '../services/api';
import { COLORS } from '../utils/constants';

export default function ProfileScreen() {
  const { player, dailyStats } = useGame();
  const [inventory, setInventory] = useState({});
  const [refreshing, setRefreshing] = useState(false);

  const loadData = async () => {
    setRefreshing(true);
    const [profile, stats, inv] = await Promise.all([
      apiService.getProfile(),
      apiService.getDailyStats(),
      apiService.getInventory(),
    ]);
    if (inv?.tools) setInventory(inv.tools);
    setRefreshing(false);
  };

  useEffect(() => { loadData(); }, []);

  return (
    <View style={ps.container}>
      <ScrollView
        contentContainerStyle={ps.scroll}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={loadData} tintColor={COLORS.primary} />}
      >
        <Text style={ps.headerTitle}>👤 我的</Text>

        {/* Profile card */}
        <View style={ps.card}>
          <Text style={ps.avatar}>🦗</Text>
          <Text style={ps.name}>Player</Text>
          <View style={ps.statRow}>
            <View style={ps.stat}><Text style={ps.statVal}>Lv.{player.level}</Text></View>
            <View style={ps.stat}><Text style={[ps.statVal, { color: COLORS.gold }]}>💰 {player.gold}</Text></View>
            <View style={ps.stat}><Text style={ps.statVal}>💎 {player.diamonds}</Text></View>
          </View>
          <View style={ps.statRow}>
            <Text style={ps.statSmall}>EXP: {player.exp}</Text>
          </View>
        </View>

        {/* Daily stats */}
        <View style={ps.card}>
          <Text style={ps.sectionTitle}>📊 今日统计</Text>
          <View style={ps.statGrid}>
            <View style={ps.statItem}>
              <Text style={ps.statNumber}>{dailyStats.digs}</Text>
              <Text style={ps.statLabel}>挖掘次数</Text>
            </View>
            <View style={ps.statItem}>
              <Text style={ps.statNumber}>{dailyStats.limit - dailyStats.digs}</Text>
              <Text style={ps.statLabel}>剩余次数</Text>
            </View>
          </View>
        </View>

        {/* Equipment */}
        <View style={ps.card}>
          <Text style={ps.sectionTitle}>🔧 装备</Text>
          {Object.entries(inventory).length === 0 && (
            <Text style={ps.emptyText}>加载中...</Text>
          )}
          {Object.entries(inventory).map(([id, tool]) => (
            <View key={id} style={ps.toolRow}>
              <View style={{ flex: 1 }}>
                <Text style={ps.toolName}>
                  {tool.type === 'shovel' ? '🪒' : tool.type === 'net' ? '🥅' : '📡'} {tool.name}
                </Text>
                <Text style={ps.toolDesc}>{tool.description}</Text>
              </View>
              <Text style={ps.toolLevel}>Lv.{tool.level}</Text>
            </View>
          ))}
        </View>

        {/* Knowledge section */}
        <View style={ps.card}>
          <Text style={ps.sectionTitle}>📚 知了百科</Text>
          <View style={ps.knowledgeItem}>
            <Text style={ps.knowledgeTitle}>🦗 知了的一生</Text>
            <Text style={ps.knowledgeText}>
              蝉属于不完全变态昆虫，一生经历三个阶段：卵 → 若虫（地下2-17年）→ 成虫（地上60-70天）。
              雌蝉将卵产在嫩枝中，孵化后的若虫落入土中，以吸食树根汁液为生。
              在地下蜕皮4-5次后，夏夜破土而出，爬上树干完成最后一次蜕壳，羽化为成虫。
            </Text>
          </View>
          <View style={ps.knowledgeItem}>
            <Text style={ps.knowledgeTitle}>🎵 蝉鸣的奥秘</Text>
            <Text style={ps.knowledgeText}>
              只有雄蝉才会鸣叫！它们的腹部有特殊的"鸣肌"结构，每秒伸缩约1万次，
              通过鼓膜共鸣发出响亮的声音。蝉鸣有三种：集合声、求偶声和惊飞声。
            </Text>
          </View>
        </View>

        {/* App info */}
        <View style={ps.card}>
          <Text style={ps.sectionTitle}>ℹ️ 关于</Text>
          <Text style={ps.knowledgeText}>
            抓知了猴 v1.0.0{'\n'}
            一款融合AR与户外探索的AI游戏{'\n'}
            在真实世界中体验知了的奇妙一生{'\n\n'}
            "数年蛰伏，一夏高歌"
          </Text>
        </View>
      </ScrollView>
    </View>
  );
}

const ps = StyleSheet.create({
  container: { flex: 1, backgroundColor: COLORS.bg },
  scroll: { padding: 16, paddingTop: 50, paddingBottom: 40 },
  headerTitle: { fontSize: 20, fontWeight: 'bold', color: COLORS.text, marginBottom: 16 },
  card: {
    backgroundColor: COLORS.bgCard, borderRadius: 12, padding: 16,
    borderWidth: 1, borderColor: COLORS.border, marginBottom: 12,
  },
  avatar: { fontSize: 48, textAlign: 'center' },
  name: { fontSize: 18, fontWeight: 'bold', color: COLORS.text, textAlign: 'center', marginVertical: 6 },
  statRow: { flexDirection: 'row', justifyContent: 'center', gap: 12, marginTop: 4 },
  stat: { backgroundColor: COLORS.bgSurface, paddingHorizontal: 12, paddingVertical: 4, borderRadius: 20 },
  statVal: { color: COLORS.text, fontSize: 13, fontWeight: '600' },
  statSmall: { color: COLORS.textDim, fontSize: 12 },
  sectionTitle: { fontSize: 16, fontWeight: 'bold', color: COLORS.text, marginBottom: 12 },
  statGrid: { flexDirection: 'row', gap: 16 },
  statItem: {
    flex: 1, backgroundColor: COLORS.bgSurface, borderRadius: 10, padding: 12,
    alignItems: 'center',
  },
  statNumber: { fontSize: 24, fontWeight: 'bold', color: COLORS.primary },
  statLabel: { fontSize: 12, color: COLORS.textDim, marginTop: 4 },
  toolRow: {
    flexDirection: 'row', alignItems: 'center', paddingVertical: 10,
    borderBottomWidth: 1, borderBottomColor: COLORS.border,
  },
  toolName: { fontSize: 14, fontWeight: '600', color: COLORS.text },
  toolDesc: { fontSize: 12, color: COLORS.textDim, marginTop: 2 },
  toolLevel: { color: COLORS.gold, fontSize: 12, fontWeight: 'bold' },
  emptyText: { color: COLORS.textDim, fontSize: 13, textAlign: 'center', paddingVertical: 20 },
  knowledgeItem: { marginBottom: 12 },
  knowledgeTitle: { fontSize: 14, fontWeight: 'bold', color: COLORS.primary, marginBottom: 4 },
  knowledgeText: { fontSize: 13, color: COLORS.textDim, lineHeight: 20 },
});
