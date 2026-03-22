<template>
  <section class="neo-page projects-page space-y-6 xl:space-y-8">
    <header class="neo-panel-hero projects-stage bg-[var(--neo-green)]">
      <div class="projects-stage-copy">
        <p class="neo-kicker bg-white">{{ t('projects.hero.kicker') }}</p>
        <h1 class="projects-stage-title">
          {{ t('projects.hero.title') }}
        </h1>
        <p class="projects-stage-note">
          {{ t('projects.hero.description') }}
        </p>

        <form class="projects-import-bar" @submit.prevent="submitImport">
          <input
            v-model="repoUrl"
            class="neo-input flex-1"
            :placeholder="t('projects.importPlaceholder')"
          />
          <button
            type="submit"
            class="neo-button-red w-full md:w-auto"
            :disabled="isImporting"
          >
            {{
              isImporting ? t('common.starting') : t('projects.importAction')
            }}
          </button>
        </form>
      </div>

      <div class="projects-stage-side">
        <article class="projects-stage-stat">
          <span>{{ projects.length }}</span>
          <small>{{ t('projects.listTitle') }}</small>
        </article>
        <article class="projects-stage-stat">
          <span>{{ activeImportCount }}</span>
          <small>{{ t('projects.jobsTitle') }}</small>
        </article>

        <div v-if="onboardingMode" class="projects-stage-onboarding">
          <p class="neo-kicker bg-white">
            {{ t('projects.onboarding.kicker') }}
          </p>
          <h2 class="text-xl font-black">
            {{ t('projects.onboarding.title') }}
          </h2>
          <p class="neo-note">
            {{
              projects.length
                ? t('projects.onboarding.readyDescription')
                : t('projects.onboarding.description')
            }}
          </p>
          <div class="flex flex-col gap-3">
            <RouterLink to="/train?onboarding=1" class="neo-button-dark w-full">
              {{
                projects.length
                  ? t('projects.onboarding.continueAction')
                  : t('projects.onboarding.skipAction')
              }}
            </RouterLink>
            <RouterLink
              to="/profile?onboarding=1"
              class="neo-button w-full bg-white"
            >
              {{ t('projects.onboarding.backAction') }}
            </RouterLink>
          </div>
        </div>
      </div>
    </header>

    <NoticePanel
      v-if="importError"
      tone="error"
      :title="t('projects.importErrorTitle')"
      :message="importError"
    />

    <NoticePanel
      v-if="retryError"
      tone="error"
      :title="t('projects.retryErrorTitle')"
      :message="retryError"
    />

    <NoticePanel
      v-if="saveError"
      tone="error"
      :title="t('projects.saveErrorTitle')"
      :message="saveError"
    />

    <div class="projects-shell">
      <aside class="projects-feed">
        <section class="neo-panel projects-jobs-panel">
          <div class="projects-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-blue)]">
                {{ t('projects.jobsTitle') }}
              </p>
              <h2 class="projects-section-title">
                {{ t('projects.jobsTitle') }}
              </h2>
            </div>
            <span class="neo-badge bg-white">
              {{ importJobs.length }}
            </span>
          </div>

          <div
            v-if="importJobs.length"
            class="projects-job-list neo-stagger-list"
          >
            <article
              v-for="job in importJobs"
              :key="job.id"
              class="projects-job-row"
            >
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div class="space-y-1">
                  <p class="text-sm font-black uppercase tracking-[0.08em]">
                    {{ formatImportJobStatusLabel(t, job.status) }}
                  </p>
                  <p class="text-base font-semibold">{{ job.message }}</p>
                </div>
                <span class="neo-badge bg-[var(--neo-yellow)]">
                  {{ formatImportJobStageLabel(t, job.stage) }}
                </span>
              </div>

              <div
                class="space-y-1 break-all text-sm font-semibold text-black/80"
              >
                <p>{{ t('projects.jobRepo') }}: {{ job.repo_url }}</p>
                <p v-if="job.error_message">{{ job.error_message }}</p>
                <p v-if="job.project_name">
                  {{ t('projects.jobResult') }}: {{ job.project_name }}
                </p>
              </div>

              <div class="projects-job-actions">
                <button
                  v-if="job.project_id"
                  type="button"
                  class="neo-button-dark w-full"
                  @click="selectProject(job.project_id, { revealDetail: true })"
                >
                  {{ t('projects.openProject') }}
                </button>
                <button
                  v-if="job.status === 'failed'"
                  type="button"
                  class="neo-button-red w-full"
                  :disabled="isRetrying"
                  @click="retryJob(job.id)"
                >
                  {{
                    isRetrying
                      ? t('common.starting')
                      : t('projects.retryAction')
                  }}
                </button>
              </div>
            </article>
          </div>
          <p v-else class="neo-note">{{ t('projects.jobsEmpty') }}</p>
        </section>
      </aside>

      <div class="projects-workspace">
        <section class="neo-panel projects-list-panel">
          <div class="projects-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-yellow)]">
                {{ t('projects.listTitle') }}
              </p>
              <h2 class="projects-section-title">
                {{ t('projects.listTitle') }}
              </h2>
            </div>
            <span class="neo-badge bg-white">{{ projects.length }}</span>
          </div>

          <div v-if="projects.length" class="projects-list neo-stagger-list">
            <button
              v-for="project in projects"
              :key="project.id"
              type="button"
              class="projects-list-row"
              :class="{
                'projects-list-row-active': selectedProjectId === project.id,
              }"
              @click="selectProject(project.id, { revealDetail: true })"
            >
              <div class="flex items-start justify-between gap-3">
                <div class="space-y-1">
                  <p class="text-sm font-black uppercase">
                    {{ formatImportStatusLabel(t, project.import_status) }}
                  </p>
                  <p class="text-lg font-black">{{ project.name }}</p>
                </div>
                <span class="neo-badge bg-white">
                  {{ formatImportStatusLabel(t, project.import_status) }}
                </span>
              </div>
              <p class="break-all text-sm font-semibold text-black/80">
                {{ project.repo_url }}
              </p>
            </button>
          </div>
          <p v-else class="neo-note">
            {{ t('projects.emptyList') }}
          </p>
        </section>

        <section
          v-if="selectedProject"
          ref="projectDetailRef"
          class="neo-panel projects-editor-panel"
        >
          <div class="projects-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-red)]">
                {{ t('projects.editorTitle') }}
              </p>
              <h2 class="projects-section-title">
                {{ selectedProject.name }}
              </h2>
            </div>
            <span class="neo-badge bg-white">
              {{ formatImportStatusLabel(t, selectedProject.import_status) }}
            </span>
          </div>

          <p class="neo-note projects-editor-summary">
            {{
              selectedProject.summary ||
              selectedProject.repo_url ||
              t('projects.hero.description')
            }}
          </p>

          <form class="projects-editor-form" @submit.prevent="submitUpdate">
            <div class="projects-editor-grid">
              <label class="space-y-2">
                <span class="neo-subheading">{{
                  t('projects.fields.name')
                }}</span>
                <input v-model="editor.name" class="neo-input" />
              </label>

              <label class="space-y-2">
                <span class="neo-subheading">
                  {{ t('projects.fields.techStack') }}
                </span>
                <input v-model="editor.tech_stack" class="neo-input" />
              </label>
            </div>

            <label class="space-y-2">
              <span class="neo-subheading">{{
                t('projects.fields.summary')
              }}</span>
              <textarea v-model="editor.summary" class="neo-textarea" />
            </label>

            <div class="projects-editor-grid">
              <label class="space-y-2">
                <span class="neo-subheading">
                  {{ t('projects.fields.highlights') }}
                </span>
                <textarea v-model="editor.highlights" class="neo-textarea" />
              </label>

              <label class="space-y-2">
                <span class="neo-subheading">
                  {{ t('projects.fields.challenges') }}
                </span>
                <textarea v-model="editor.challenges" class="neo-textarea" />
              </label>

              <label class="space-y-2">
                <span class="neo-subheading">
                  {{ t('projects.fields.tradeoffs') }}
                </span>
                <textarea v-model="editor.tradeoffs" class="neo-textarea" />
              </label>

              <label class="space-y-2">
                <span class="neo-subheading">
                  {{ t('projects.fields.ownership') }}
                </span>
                <textarea
                  v-model="editor.ownership_points"
                  class="neo-textarea"
                />
              </label>
            </div>

            <label class="space-y-2">
              <span class="neo-subheading">
                {{ t('projects.fields.followups') }}
              </span>
              <textarea v-model="editor.followup_points" class="neo-textarea" />
            </label>

            <button
              class="neo-button-dark w-full sm:w-auto"
              type="submit"
              :disabled="isUpdating"
            >
              {{ isUpdating ? t('common.saving') : t('projects.saveAction') }}
            </button>
          </form>
        </section>

        <section v-else class="neo-panel projects-empty-panel">
          <p class="neo-kicker bg-[var(--neo-red)]">
            {{ t('projects.editorTitle') }}
          </p>
          <h2 class="projects-section-title">
            {{ t('projects.listTitle') }}
          </h2>
          <p class="neo-note">
            {{ t('projects.emptyList') }}
          </p>
        </section>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, nextTick, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink, useRoute, useRouter } from 'vue-router';

import {
  importProject,
  listImportJobs,
  listProjects,
  retryImportJob,
  updateProject,
  type ProjectImportJob,
  type ProjectProfile,
} from '../api/client';
import NoticePanel from '../components/NoticePanel.vue';
import {
  formatImportJobStageLabel,
  formatImportJobStatusLabel,
  formatImportStatusLabel,
} from '../lib/labels';

const queryClient = useQueryClient();
const route = useRoute();
const router = useRouter();
const repoUrl = ref('');
const selectedProjectId = ref('');
const importError = ref('');
const retryError = ref('');
const saveError = ref('');
const { t } = useI18n();
const onboardingMode = computed(() => route.query.onboarding === '1');
const projectDetailRef = ref<HTMLElement | null>(null);

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

const { data: importJobsData } = useQuery({
  queryKey: ['import-jobs'],
  queryFn: listImportJobs,
  refetchInterval: (query) => {
    const jobs = (query.state.data as ProjectImportJob[] | undefined) ?? [];
    return jobs.some((job) => ['queued', 'running'].includes(job.status))
      ? 3000
      : false;
  },
});

const importJobs = computed(() => importJobsData.value ?? []);
const activeImportCount = computed(
  () =>
    importJobs.value.filter((job) => ['queued', 'running'].includes(job.status))
      .length,
);

watch(
  () =>
    importJobs.value
      .map((job) => `${job.id}:${job.status}:${job.project_id}`)
      .join('|'),
  async () => {
    await queryClient.invalidateQueries({ queryKey: ['projects'] });
  },
);

watch(
  projects,
  (items) => {
    // 页面交互默认是“列表有数据就始终绑定一个当前编辑对象”，
    // 这里只在还没有用户选择时自动选首项，避免 refetch 把正在编辑的项目切走。
    if (!selectedProjectId.value && items.length > 0) {
      selectProject(items[0].id, { revealDetail: false });
    }
  },
  { immediate: true },
);

const selectedProject = computed(
  () =>
    projects.value.find((project) => project.id === selectedProjectId.value) ??
    null,
);

watch(
  selectedProject,
  (project) => {
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
  },
  { immediate: true },
);

const importMutation = useMutation({
  mutationFn: importProject,
  onSuccess: async (project) => {
    repoUrl.value = '';
    importError.value = '';
    await queryClient.invalidateQueries({ queryKey: ['import-jobs'] });
    await queryClient.invalidateQueries({ queryKey: ['projects'] });

    if (onboardingMode.value && project.project_id) {
      await router.push(
        `/train?onboarding=1&mode=project&project_id=${encodeURIComponent(project.project_id)}`,
      );
      return;
    }

    if (project.project_id) {
      await selectProject(project.project_id, { revealDetail: true });
    }
  },
  onError: (error) => {
    importError.value =
      error instanceof Error ? error.message : t('common.requestFailed');
  },
});

const retryMutation = useMutation({
  mutationFn: retryImportJob,
  onSuccess: async () => {
    retryError.value = '';
    await queryClient.invalidateQueries({ queryKey: ['import-jobs'] });
  },
  onError: (error) => {
    retryError.value =
      error instanceof Error ? error.message : t('common.requestFailed');
  },
});

const updateMutation = useMutation({
  mutationFn: ({
    projectId,
    payload,
  }: {
    projectId: string;
    payload: Partial<ProjectProfile>;
  }) => updateProject(projectId, payload),
  onSuccess: async () => {
    saveError.value = '';
    await queryClient.invalidateQueries({ queryKey: ['projects'] });
  },
  onError: (error) => {
    saveError.value =
      error instanceof Error ? error.message : t('common.requestFailed');
  },
});

const isImporting = computed(() => importMutation.isPending.value);
const isRetrying = computed(() => retryMutation.isPending.value);
const isUpdating = computed(() => updateMutation.isPending.value);

function splitLines(value: string): string[] {
  // 保存时同时接受换行和中英文逗号，主要是兼容用户从 README、飞书或简历里直接粘贴内容，
  // 而不是要求前端把每类字段拆成更重的多控件编辑器。
  return value
    .split(/[\n,，]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

async function selectProject(
  projectId: string,
  options: { revealDetail?: boolean } = {},
) {
  selectedProjectId.value = projectId;
  if (!options.revealDetail || !shouldRevealProjectDetail()) {
    return;
  }

  await nextTick();
  projectDetailRef.value?.scrollIntoView({
    behavior: prefersReducedMotion() ? 'auto' : 'smooth',
    block: 'start',
  });
}

function submitImport() {
  importError.value = '';
  mutationGuard(repoUrl.value, () => importMutation.mutate(repoUrl.value));
}

function retryJob(jobId: string) {
  retryError.value = '';
  retryMutation.mutate(jobId);
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

function shouldRevealProjectDetail(): boolean {
  if (
    typeof window === 'undefined' ||
    typeof window.matchMedia !== 'function'
  ) {
    return false;
  }

  return window.matchMedia('(max-width: 1279px)').matches;
}

function prefersReducedMotion(): boolean {
  if (
    typeof window === 'undefined' ||
    typeof window.matchMedia !== 'function'
  ) {
    return false;
  }

  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}
</script>

<style scoped>
.projects-page {
  position: relative;
}

.projects-stage {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--neo-green) 84%, white) 0%,
    color-mix(in srgb, var(--neo-green) 58%, var(--neo-yellow)) 100%
  );
}

.projects-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.projects-stage-copy,
.projects-stage-side {
  position: relative;
  z-index: 1;
}

.projects-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.projects-stage-title {
  font-size: clamp(2.1rem, 6vw, 4.8rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 0.95;
  margin: 0;
  max-width: 12ch;
  text-transform: uppercase;
}

.projects-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 36rem;
}

.projects-import-bar {
  display: grid;
  gap: 0.75rem;
  margin-top: 0.5rem;
}

.projects-stage-side {
  display: grid;
  gap: 1rem;
}

.projects-stage-stat,
.projects-stage-onboarding {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.projects-stage-stat span {
  font-size: clamp(2.5rem, 8vw, 4rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.projects-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.projects-shell {
  display: grid;
  gap: 1rem;
}

.projects-feed {
  min-width: 0;
}

.projects-jobs-panel,
.projects-list-panel,
.projects-editor-panel,
.projects-empty-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.projects-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.projects-section-title {
  font-size: 1.4rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.projects-job-list,
.projects-list {
  display: grid;
  gap: 0.85rem;
}

.projects-job-row,
.projects-list-row {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.85rem;
  padding: 1rem;
  transition:
    transform var(--motion-duration-base) var(--motion-ease-standard),
    box-shadow var(--motion-duration-base) var(--motion-ease-standard),
    background-color var(--motion-duration-fast) var(--motion-ease-soft);
}

.projects-job-row:hover,
.projects-list-row:hover {
  box-shadow: 8px 8px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  transform: translate(var(--motion-lift-md), var(--motion-lift-md));
}

.projects-list-row-active {
  background: color-mix(in srgb, var(--neo-yellow) 72%, white);
}

.projects-job-actions {
  display: grid;
  gap: 0.65rem;
}

.projects-workspace {
  display: grid;
  gap: 1rem;
}

.projects-editor-summary {
  line-height: 1.7;
  margin: 0;
  max-width: 54rem;
}

.projects-editor-form {
  display: grid;
  gap: 1rem;
}

.projects-editor-grid {
  display: grid;
  gap: 1rem;
}

@media (min-width: 768px) {
  .projects-import-bar {
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .projects-stage-side {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .projects-editor-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .projects-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.2fr) minmax(20rem, 0.8fr);
  }

  .projects-shell {
    align-items: start;
    grid-template-columns: minmax(18rem, 22rem) minmax(0, 1fr);
  }

  .projects-jobs-panel {
    position: sticky;
    top: 1.5rem;
  }

  .projects-workspace {
    grid-template-columns: minmax(18rem, 20rem) minmax(0, 1fr);
  }
}

@media (prefers-reduced-motion: reduce) {
  .projects-job-row,
  .projects-list-row {
    transition: none;
  }

  .projects-job-row:hover,
  .projects-list-row:hover {
    box-shadow: inherit;
    transform: none;
  }
}
</style>
