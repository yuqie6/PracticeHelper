<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-blue)]">
      <p class="neo-kicker bg-white">Profile Builder</p>
      <h2 class="neo-heading">先让系统认识你，再决定该往哪打。</h2>
      <p class="mt-3 text-base font-semibold">
        这里记录目标岗位、阶段、技术栈和主讲项目，后面所有推荐和追问都会以这份画像为底。
      </p>
    </header>

    <form class="neo-panel space-y-4" @submit.prevent="submit">
      <div class="neo-grid md:grid-cols-2">
        <label class="space-y-2">
          <span class="neo-subheading">目标岗位</span>
          <input v-model="form.target_role" class="neo-input" placeholder="Go 后端 / Agent 工程师" />
        </label>
        <label class="space-y-2">
          <span class="neo-subheading">目标公司类型</span>
          <input v-model="form.target_company_type" class="neo-input" placeholder="AI 应用 / 中厂 / 创业团队" />
        </label>
      </div>

      <div class="neo-grid md:grid-cols-2">
        <label class="space-y-2">
          <span class="neo-subheading">当前阶段</span>
          <input v-model="form.current_stage" class="neo-input" placeholder="大三 / 实习前 / 秋招前" />
        </label>
        <label class="space-y-2">
          <span class="neo-subheading">目标投递时间</span>
          <input v-model="form.application_deadline" type="date" class="neo-input" />
        </label>
      </div>

      <label class="space-y-2">
        <span class="neo-subheading">技术栈</span>
        <input v-model="techStacksRaw" class="neo-input" placeholder="Go, Redis, Kafka, LangGraph" />
      </label>

      <label class="space-y-2">
        <span class="neo-subheading">主讲项目</span>
        <input v-model="projectsRaw" class="neo-input" placeholder="Mirror, SneakerFlash, OfferPilot" />
      </label>

      <label class="space-y-2">
        <span class="neo-subheading">自我感知弱项</span>
        <textarea
          v-model="weaknessesRaw"
          class="neo-textarea"
          placeholder="例如：项目 trade-off 讲不硬、Kafka 幂等一追问就虚"
        />
      </label>

      <div class="flex flex-wrap items-center gap-3">
        <button type="submit" class="neo-button-dark" :disabled="isSaving">
          {{ isSaving ? '保存中...' : '保存画像' }}
        </button>
        <span class="neo-note">{{ successMessage }}</span>
      </div>
    </form>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, reactive, ref, watchEffect } from 'vue';

import { getProfile, saveProfile, type UserProfile } from '../api/client';

const queryClient = useQueryClient();
const successMessage = ref('');

const form = reactive({
  target_role: '',
  target_company_type: '',
  current_stage: '',
  application_deadline: '',
});

const techStacksRaw = ref('');
const projectsRaw = ref('');
const weaknessesRaw = ref('');

const { data } = useQuery({
  queryKey: ['profile'],
  queryFn: getProfile,
});

watchEffect(() => {
  const profile = data.value;
  if (!profile) {
    return;
  }
  applyProfile(profile);
});

const mutation = useMutation({
  mutationFn: saveProfile,
  onSuccess: async (profile) => {
    queryClient.setQueryData(['profile'], profile);
    await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
    successMessage.value = '画像已更新，后续推荐会立刻用上。';
  },
});

const isSaving = computed(() => mutation.isPending.value);

function applyProfile(profile: UserProfile) {
  form.target_role = profile.target_role;
  form.target_company_type = profile.target_company_type;
  form.current_stage = profile.current_stage;
  form.application_deadline = profile.application_deadline?.slice(0, 10) ?? '';
  techStacksRaw.value = profile.tech_stacks.join(', ');
  projectsRaw.value = profile.primary_projects.join(', ');
  weaknessesRaw.value = profile.self_reported_weaknesses.join(', ');
}

function splitCsv(input: string): string[] {
  return input
    .split(/[,，\n]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function submit() {
  successMessage.value = '';
  mutation.mutate({
    ...form,
    tech_stacks: splitCsv(techStacksRaw.value),
    primary_projects: splitCsv(projectsRaw.value),
    self_reported_weaknesses: splitCsv(weaknessesRaw.value),
  });
}
</script>
