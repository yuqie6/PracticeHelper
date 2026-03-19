<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-green)]">
      <p class="neo-kicker bg-white">{{ t('projects.hero.kicker') }}</p>
      <p class="text-base font-semibold">
        {{ t('projects.hero.description') }}
      </p>
    </header>

    <form class="neo-panel flex flex-col gap-4 md:flex-row" @submit.prevent="submitImport">
      <input
        v-model="repoUrl"
        class="neo-input flex-1"
        :placeholder="t('projects.importPlaceholder')"
      />
      <button type="submit" class="neo-button-red" :disabled="isImporting">
        {{ isImporting ? t('common.starting') : t('projects.importAction') }}
      </button>
    </form>

    <NoticePanel
      v-if="importError"
      tone="error"
      :title="t('projects.importErrorTitle')"
      :message="importError"
    />

    <div class="neo-grid lg:grid-cols-[0.8fr_1.2fr]">
      <div class="neo-panel space-y-3">
        <p class="neo-kicker bg-[var(--neo-yellow)]">{{ t('projects.listTitle') }}</p>
        <button
          v-for="project in projects"
          :key="project.id"
          type="button"
          class="w-full border-2 border-black bg-white px-4 py-3 text-left shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] md:border-4"
          :class="{ 'bg-[var(--neo-yellow)]': selectedProjectId === project.id }"
          @click="selectProject(project.id)"
        >
          <p class="text-sm font-black uppercase">
            {{ formatImportStatusLabel(t, project.import_status) }}
          </p>
          <p class="text-lg font-bold">{{ project.name }}</p>
          <p class="mt-1 text-sm font-semibold">{{ project.repo_url }}</p>
        </button>
        <p v-if="!projects.length" class="neo-note">{{ t('projects.emptyList') }}</p>
      </div>

      <div v-if="selectedProject" class="neo-panel">
        <p class="neo-kicker bg-[var(--neo-red)]">{{ t('projects.editorTitle') }}</p>
        <form class="space-y-4" @submit.prevent="submitUpdate">
          <label class="space-y-2">
            <span class="neo-subheading">{{ t('projects.fields.name') }}</span>
            <input v-model="editor.name" class="neo-input" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">{{ t('projects.fields.summary') }}</span>
            <textarea v-model="editor.summary" class="neo-textarea" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">{{ t('projects.fields.techStack') }}</span>
            <input v-model="editor.tech_stack" class="neo-input" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">{{ t('projects.fields.highlights') }}</span>
            <textarea v-model="editor.highlights" class="neo-textarea" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">{{ t('projects.fields.challenges') }}</span>
            <textarea v-model="editor.challenges" class="neo-textarea" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">{{ t('projects.fields.tradeoffs') }}</span>
            <textarea v-model="editor.tradeoffs" class="neo-textarea" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">{{ t('projects.fields.ownership') }}</span>
            <textarea v-model="editor.ownership_points" class="neo-textarea" />
          </label>

          <label class="space-y-2">
            <span class="neo-subheading">{{ t('projects.fields.followups') }}</span>
            <textarea v-model="editor.followup_points" class="neo-textarea" />
          </label>

          <button class="neo-button-dark" type="submit" :disabled="isUpdating">
            {{ isUpdating ? t('common.saving') : t('projects.saveAction') }}
          </button>
        </form>
      </div>
    </div>

    <NoticePanel
      v-if="saveError"
      tone="error"
      :title="t('projects.saveErrorTitle')"
      :message="saveError"
    />
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { importProject, listProjects, updateProject, type ProjectProfile } from '../api/client';
import NoticePanel from '../components/NoticePanel.vue';
import { formatImportStatusLabel } from '../lib/labels';

const queryClient = useQueryClient();
const repoUrl = ref('');
const selectedProjectId = ref('');
const importError = ref('');
const saveError = ref('');
const { t } = useI18n();

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
    // 页面交互默认是“列表有数据就始终绑定一个当前编辑对象”，
    // 这里只在还没有用户选择时自动选首项，避免 refetch 把正在编辑的项目切走。
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
  // 编辑器用逗号/换行文本框承载数组字段，目的是降低手工维护成本，
  // 所以这里需要把后端结构化数据稳定映射回用户可直接粘贴编辑的文本格式。
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
    importError.value = '';
    selectedProjectId.value = project.id;
    await queryClient.invalidateQueries({ queryKey: ['projects'] });
  },
  onError: (error) => {
    importError.value = error instanceof Error ? error.message : t('common.requestFailed');
  },
});

const updateMutation = useMutation({
  mutationFn: ({ projectId, payload }: { projectId: string; payload: Partial<ProjectProfile> }) =>
    updateProject(projectId, payload),
  onSuccess: async () => {
    saveError.value = '';
    await queryClient.invalidateQueries({ queryKey: ['projects'] });
  },
  onError: (error) => {
    saveError.value = error instanceof Error ? error.message : t('common.requestFailed');
  },
});

const isImporting = computed(() => importMutation.isPending.value);
const isUpdating = computed(() => updateMutation.isPending.value);

function splitLines(value: string): string[] {
  // 保存时同时接受换行和中英文逗号，主要是兼容用户从 README、飞书或简历里直接粘贴内容，
  // 而不是要求前端把每类字段拆成更重的多控件编辑器。
  return value
    .split(/[\n,，]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function selectProject(projectId: string) {
  selectedProjectId.value = projectId;
}

function submitImport() {
  importError.value = '';
  mutationGuard(repoUrl.value, () => importMutation.mutate(repoUrl.value));
}

function submitUpdate() {
  if (!selectedProject.value) {
    return;
  }

  saveError.value = '';
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

function mutationGuard(value: string, action: () => void) {
  if (!value.trim()) {
    return;
  }

  action();
}
</script>
