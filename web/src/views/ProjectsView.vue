<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-green)]">
      <p class="neo-kicker bg-white">Project Interviewer</p>
      <h2 class="neo-heading">把 GitHub 仓库压成一份能拿去面试的项目资产。</h2>
      <p class="mt-3 text-base font-semibold">
        v0 先走仓库导入 + 手工修正。系统帮你扫材料，你负责把真正值钱的亮点和坑补硬。
      </p>
    </header>

    <form class="neo-panel flex flex-col gap-4 md:flex-row" @submit.prevent="submitImport">
      <input
        v-model="repoUrl"
        class="neo-input flex-1"
        placeholder="https://github.com/yourname/your-project"
      />
      <button type="submit" class="neo-button-red" :disabled="isImporting">
        {{ isImporting ? '导入中...' : '导入仓库' }}
      </button>
    </form>

    <div class="neo-grid lg:grid-cols-[0.8fr_1.2fr]">
      <div class="neo-panel space-y-3">
        <p class="neo-kicker bg-[var(--neo-yellow)]">项目列表</p>
        <button
          v-for="project in projects"
          :key="project.id"
          type="button"
          class="w-full border-2 border-black bg-white px-4 py-3 text-left shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] md:border-4"
          :class="{ 'bg-[var(--neo-yellow)]': selectedProjectId === project.id }"
          @click="selectProject(project.id)"
        >
          <p class="text-sm font-black uppercase">{{ project.import_status }}</p>
          <p class="text-lg font-bold">{{ project.name }}</p>
          <p class="mt-1 text-sm font-semibold">{{ project.repo_url }}</p>
        </button>
        <p v-if="!projects.length" class="neo-note">先导入一个公开 GitHub 仓库。</p>
      </div>

      <div class="neo-panel" v-if="selectedProject">
        <p class="neo-kicker bg-[var(--neo-red)]">项目画像编辑</p>
        <form class="space-y-4" @submit.prevent="submitUpdate">
          <label class="space-y-2">
            <span class="neo-subheading">项目名</span>
            <input v-model="editor.name" class="neo-input" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">项目摘要</span>
            <textarea v-model="editor.summary" class="neo-textarea" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">技术栈</span>
            <input v-model="editor.tech_stack" class="neo-input" placeholder="Go, Gin, SQLite" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">亮点</span>
            <textarea v-model="editor.highlights" class="neo-textarea" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">难点</span>
            <textarea v-model="editor.challenges" class="neo-textarea" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">trade-off</span>
            <textarea v-model="editor.tradeoffs" class="neo-textarea" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">ownership 点</span>
            <textarea v-model="editor.ownership_points" class="neo-textarea" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">可追问点</span>
            <textarea v-model="editor.followup_points" class="neo-textarea" />
          </label>

          <button class="neo-button-dark" type="submit" :disabled="isUpdating">
            {{ isUpdating ? '保存中...' : '保存项目画像' }}
          </button>
        </form>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, reactive, ref, watch } from 'vue';

import { importProject, listProjects, updateProject, type ProjectProfile } from '../api/client';

const queryClient = useQueryClient();
const repoUrl = ref('');
const selectedProjectId = ref('');

const editor = reactive({
  name: '',
  summary: '',
  tech_stack: '',
  highlights: '',
  challenges: '',
  tradeoffs: '',
  ownership_points: '',
  followup_points: '',
});

const { data } = useQuery({
  queryKey: ['projects'],
  queryFn: listProjects,
});

const projects = computed(() => data.value ?? []);

watch(
  projects,
  (items) => {
    if (!selectedProjectId.value && items.length > 0) {
      selectProject(items[0].id);
    }
  },
  { immediate: true },
);

const selectedProject = computed(() =>
  projects.value.find((project) => project.id === selectedProjectId.value) ?? null,
);

watch(selectedProject, (project) => {
  if (!project) {
    return;
  }
  editor.name = project.name;
  editor.summary = project.summary;
  editor.tech_stack = project.tech_stack.join(', ');
  editor.highlights = project.highlights.join('\n');
  editor.challenges = project.challenges.join('\n');
  editor.tradeoffs = project.tradeoffs.join('\n');
  editor.ownership_points = project.ownership_points.join('\n');
  editor.followup_points = project.followup_points.join('\n');
});

const importMutation = useMutation({
  mutationFn: importProject,
  onSuccess: async (project) => {
    repoUrl.value = '';
    selectedProjectId.value = project.id;
    await queryClient.invalidateQueries({ queryKey: ['projects'] });
  },
});

const updateMutation = useMutation({
  mutationFn: ({ projectId, payload }: { projectId: string; payload: Partial<ProjectProfile> }) =>
    updateProject(projectId, payload),
  onSuccess: async () => {
    await queryClient.invalidateQueries({ queryKey: ['projects'] });
  },
});

const isImporting = computed(() => importMutation.isPending.value);
const isUpdating = computed(() => updateMutation.isPending.value);

function splitLines(value: string): string[] {
  return value
    .split(/[\n,，]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function selectProject(projectId: string) {
  selectedProjectId.value = projectId;
}

function submitImport() {
  importMutation.mutate(repoUrl.value);
}

function submitUpdate() {
  if (!selectedProject.value) {
    return;
  }
  updateMutation.mutate({
    projectId: selectedProject.value.id,
    payload: {
      name: editor.name,
      summary: editor.summary,
      tech_stack: splitLines(editor.tech_stack),
      highlights: splitLines(editor.highlights),
      challenges: splitLines(editor.challenges),
      tradeoffs: splitLines(editor.tradeoffs),
      ownership_points: splitLines(editor.ownership_points),
      followup_points: splitLines(editor.followup_points),
    },
  });
}
</script>
