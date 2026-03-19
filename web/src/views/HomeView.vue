<template>
  <section class="neo-page space-y-6">
    <div class="neo-grid md:grid-cols-[1.35fr_0.65fr]">
      <div class="neo-panel bg-[var(--neo-yellow)]">
        <p class="neo-kicker bg-white">今日训练建议</p>
        <h2 class="neo-heading">别刷随机题，直接打你现在最虚的地方。</h2>
        <p class="mt-4 max-w-3xl text-base font-semibold leading-7">
          {{ dashboard?.today_focus ?? '先完成画像初始化，然后立刻开第一轮训练。' }}
        </p>
        <div class="mt-6 flex flex-wrap gap-3">
          <RouterLink to="/train" class="neo-button-red">开始训练</RouterLink>
          <RouterLink to="/projects" class="neo-button bg-white">管理项目资产</RouterLink>
        </div>
      </div>

      <div class="neo-panel bg-[var(--neo-blue)]">
        <p class="neo-kicker bg-white">投递节奏</p>
        <div class="space-y-3">
          <p class="text-6xl font-black">{{ dashboard?.days_until_deadline ?? '--' }}</p>
          <p class="text-sm font-bold uppercase tracking-[0.08em]">距离目标投递还剩天数</p>
          <p class="neo-note">
            {{ dashboard?.recommended_track ?? '先完成画像后，系统会给你下一步专项建议。' }}
          </p>
        </div>
      </div>
    </div>

    <div class="neo-grid md:grid-cols-2 xl:grid-cols-4">
      <StatCard
        kicker="Top 5"
        kicker-class="bg-[var(--neo-red)]"
        title="薄弱点"
        :description="weaknessSummary"
      />
      <StatCard
        kicker="History"
        kicker-class="bg-[var(--neo-green)]"
        title="最近训练"
        :description="sessionSummary"
      />
      <StatCard
        kicker="Track"
        kicker-class="bg-[var(--neo-yellow)]"
        title="推荐专项"
        :description="dashboard?.recommended_track ?? '暂无推荐，先做第一轮训练。'"
      />
      <StatCard
        kicker="Profile"
        kicker-class="bg-[var(--neo-blue)]"
        title="目标画像"
        :description="profileSummary"
      />
    </div>

    <div class="neo-grid lg:grid-cols-[0.9fr_1.1fr]">
      <div class="neo-panel">
        <p class="neo-kicker bg-[var(--neo-red)]">弱项热区</p>
        <ul class="space-y-3">
          <li
            v-for="item in dashboard?.weaknesses ?? []"
            :key="item.id"
            class="flex items-center justify-between border-2 border-black bg-white px-3 py-3 md:border-4"
          >
            <div>
              <p class="text-sm font-black uppercase">{{ item.kind }}</p>
              <p class="text-lg font-bold">{{ item.label }}</p>
            </div>
            <span class="neo-badge bg-[var(--neo-yellow)]">严重度 {{ item.severity.toFixed(2) }}</span>
          </li>
          <li v-if="!dashboard?.weaknesses.length" class="neo-note">
            还没有历史弱项，先开始第一轮训练。
          </li>
        </ul>
      </div>

      <div class="neo-panel">
        <p class="neo-kicker bg-[var(--neo-green)]">训练记录</p>
        <ul class="space-y-3">
          <li
            v-for="session in dashboard?.recent_sessions ?? []"
            :key="session.id"
            class="border-2 border-black bg-white p-4 md:border-4"
          >
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div>
                <p class="text-sm font-black uppercase">{{ session.mode }}</p>
                <p class="text-lg font-bold">
                  {{ session.project_name || session.topic || '未命名训练' }}
                </p>
              </div>
              <span class="neo-badge bg-[var(--neo-blue)]">{{ session.total_score.toFixed(1) }}</span>
            </div>
          </li>
          <li v-if="!dashboard?.recent_sessions.length" class="neo-note">
            首页现在是空态，一旦完成第一轮训练，这里就会开始积累痕迹。
          </li>
        </ul>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { computed } from 'vue';
import { RouterLink } from 'vue-router';

import { getDashboard } from '../api/client';
import StatCard from '../components/StatCard.vue';
import { describeProfile, describeSession, describeWeakness } from '../lib/dashboard';

const { data } = useQuery({
  queryKey: ['dashboard'],
  queryFn: getDashboard,
});

const dashboard = computed(() => data.value ?? null);
const weaknessSummary = computed(() => describeWeakness(dashboard.value));
const sessionSummary = computed(() => describeSession(dashboard.value));
const profileSummary = computed(() => describeProfile(dashboard.value));
</script>
