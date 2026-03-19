<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-red)] text-black">
      <p class="neo-kicker bg-white">Question Trainer</p>
      <h2 class="neo-heading">训练模式先选准，再让系统狠狠干你最虚的地方。</h2>
      <p class="mt-3 text-base font-semibold">
        八股训练适合打单点，项目训练适合把真实项目讲硬。v0 默认每轮主问题 + 1 次追问。
      </p>
    </header>

    <form class="neo-panel space-y-4" @submit.prevent="submit">
      <div class="neo-grid md:grid-cols-2">
        <label class="space-y-2">
          <span class="neo-subheading">训练模式</span>
          <select v-model="form.mode" class="neo-select">
            <option value="basics">八股训练</option>
            <option value="project">项目训练</option>
          </select>
        </label>
        <label class="space-y-2">
          <span class="neo-subheading">强度</span>
          <select v-model="form.intensity" class="neo-select">
            <option value="light">轻刷</option>
            <option value="standard">标准</option>
            <option value="pressure">压迫训练</option>
          </select>
        </label>
      </div>

      <label v-if="form.mode === 'basics'" class="space-y-2">
        <span class="neo-subheading">主题</span>
        <select v-model="form.topic" class="neo-select">
          <option value="go">Go</option>
          <option value="redis">Redis</option>
          <option value="kafka">Kafka</option>
        </select>
      </label>

      <label v-else class="space-y-2">
        <span class="neo-subheading">选择项目</span>
        <select v-model="form.project_id" class="neo-select">
          <option disabled value="">请选择项目</option>
          <option v-for="project in projects ?? []" :key="project.id" :value="project.id">
            {{ project.name }}
          </option>
        </select>
      </label>

      <button type="submit" class="neo-button-dark" :disabled="isStarting">
        {{ isStarting ? '启动中...' : '开始这一轮训练' }}
      </button>
    </form>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery } from '@tanstack/vue-query';
import { computed, reactive } from 'vue';
import { useRouter } from 'vue-router';

import { createSession, listProjects } from '../api/client';

const router = useRouter();

const form = reactive({
  mode: 'basics' as 'basics' | 'project',
  topic: 'go',
  project_id: '',
  intensity: 'standard',
});

const { data: projects } = useQuery({
  queryKey: ['projects'],
  queryFn: listProjects,
});

const mutation = useMutation({
  mutationFn: createSession,
  onSuccess: async (session) => {
    await router.push(`/sessions/${session.id}`);
  },
});

const isStarting = computed(() => mutation.isPending.value);

function submit() {
  mutation.mutate({
    mode: form.mode,
    topic: form.mode === 'basics' ? form.topic : undefined,
    project_id: form.mode === 'project' ? form.project_id : undefined,
    intensity: form.intensity,
  });
}
</script>
