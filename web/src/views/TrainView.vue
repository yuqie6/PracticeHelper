<template>
  <section class="neo-page train-page space-y-6 xl:space-y-8">
    <header class="neo-panel-hero train-stage bg-[var(--neo-red)] text-black">
      <div class="train-stage-copy">
        <p class="neo-kicker bg-white">{{ t('train.hero.kicker') }}</p>
        <h1 class="train-stage-title">
          {{ t('train.hero.title') }}
        </h1>
        <p class="train-stage-note">
          {{ t('train.hero.description') }}
        </p>

        <div class="train-stage-stats neo-stagger-list">
          <article class="train-stage-stat">
            <span>{{ projectCount }}</span>
            <small>{{ t('projects.listTitle') }}</small>
          </article>
          <article class="train-stage-stat">
            <span>{{ readyJobTargetCount }}</span>
            <small>{{ t('jobs.listTitle') }}</small>
          </article>
          <article class="train-stage-stat">
            <span>{{ promptSetCount }}</span>
            <small>{{ t('train.fields.promptSet') }}</small>
          </article>
        </div>
      </div>

      <div class="train-stage-side">
        <article v-if="currentSession" class="train-stage-context">
          <p class="neo-kicker bg-white">
            {{ t('home.currentSession.kicker') }}
          </p>
          <h2 class="text-xl font-black">{{ t('train.resumeTitle') }}</h2>
          <p class="neo-note">
            {{
              t('train.resumeDescription', {
                name: formatSessionName(t, currentSession),
                status: formatStatusLabel(t, currentSession.status),
              })
            }}
          </p>
          <RouterLink
            :to="buildSessionTarget(currentSession)"
            class="neo-button-dark w-full"
          >
            {{ t('common.resume') }}
          </RouterLink>
        </article>

        <article
          v-else-if="onboardingMode"
          class="train-stage-context bg-[color:var(--neo-yellow)]"
        >
          <p class="neo-kicker bg-white">{{ t('train.onboarding.kicker') }}</p>
          <h2 class="text-xl font-black">{{ t('train.onboarding.title') }}</h2>
          <p class="neo-note">{{ t('train.onboarding.description') }}</p>
        </article>

        <article v-else class="train-stage-context">
          <p class="neo-kicker bg-white">{{ t('train.fields.jobTarget') }}</p>
          <h2 class="text-xl font-black">
            {{
              activeJobTarget
                ? activeJobTarget.title
                : t('train.genericJobTargetOption')
            }}
          </h2>
          <p class="neo-note">
            {{
              activeJobTarget
                ? describeJobTargetStatus(
                    t,
                    'trainFallback',
                    activeJobTarget.latest_analysis_status,
                    {
                      name: activeJobTarget.title,
                    },
                  )
                : t('common.noRecommendation')
            }}
          </p>
        </article>
      </div>
    </header>

    <ProgressPanel
      v-if="isStarting"
      :kicker="t('session.processingKicker')"
      :title="t('progress.createSession.title')"
      :description="t('progress.createSession.description')"
      :steps="createSessionSteps"
      :active-index="createSessionStepIndex"
    />

    <StreamTracePanel
      v-if="isStarting && streamSections.length"
      :kicker="t('session.processingKicker')"
      :title="t('progress.createSession.title')"
      :description="t('progress.createSession.description')"
      :reasoning-title="t('session.reasoningTitle')"
      :content-title="t('session.contentTitle')"
      :sections="streamSections"
    />

    <NoticePanel
      v-if="startError"
      tone="error"
      :title="t('train.startErrorTitle')"
      :message="startError"
    />

    <div class="train-shell">
      <form
        class="neo-panel train-form-panel neo-stagger-list"
        @submit.prevent="submit"
      >
        <section class="train-form-section">
          <div class="train-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-yellow)]">
                {{ t('train.fields.mode') }}
              </p>
              <h2 class="train-section-title">{{ t('train.hero.title') }}</h2>
            </div>
          </div>

          <div class="train-grid">
            <label class="space-y-2">
              <span class="neo-subheading">{{ t('train.fields.mode') }}</span>
              <select v-model="form.mode" class="neo-select">
                <option value="basics">
                  {{ formatModeLabel(t, 'basics') }}
                </option>
                <option value="project">
                  {{ formatModeLabel(t, 'project') }}
                </option>
              </select>
            </label>
            <label class="space-y-2">
              <span class="neo-subheading">
                {{ t('train.fields.intensity') }}
              </span>
              <select v-model="form.intensity" class="neo-select">
                <option value="auto">
                  {{ formatIntensityLabel(t, 'auto') }}
                </option>
                <option value="light">
                  {{ formatIntensityLabel(t, 'light') }}
                </option>
                <option value="standard">
                  {{ formatIntensityLabel(t, 'standard') }}
                </option>
                <option value="pressure">
                  {{ formatIntensityLabel(t, 'pressure') }}
                </option>
              </select>
            </label>
            <label class="space-y-2">
              <span class="neo-subheading">
                {{ t('train.fields.maxTurns') }}
              </span>
              <select v-model.number="form.max_turns" class="neo-select">
                <option :value="2">2</option>
                <option :value="3">3</option>
                <option :value="4">4</option>
                <option :value="5">5</option>
              </select>
            </label>
          </div>
        </section>

        <section class="train-form-section">
          <div class="train-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-blue)]">
                {{
                  form.mode === 'basics'
                    ? t('train.fields.topic')
                    : t('train.fields.project')
                }}
              </p>
              <h2 class="train-section-title">{{ trainFocusTitle }}</h2>
            </div>
            <p class="neo-note train-section-note">{{ trainFocusHint }}</p>
          </div>

          <label v-if="form.mode === 'basics'" class="space-y-2">
            <span class="neo-subheading">{{ t('train.fields.topic') }}</span>
            <select v-model="form.topic" class="neo-select">
              <option value="mixed">{{ formatTopicLabel(t, 'mixed') }}</option>
              <option value="go">{{ formatTopicLabel(t, 'go') }}</option>
              <option value="redis">{{ formatTopicLabel(t, 'redis') }}</option>
              <option value="kafka">{{ formatTopicLabel(t, 'kafka') }}</option>
              <option value="mysql">{{ formatTopicLabel(t, 'mysql') }}</option>
              <option value="system_design">
                {{ formatTopicLabel(t, 'system_design') }}
              </option>
              <option value="distributed">
                {{ formatTopicLabel(t, 'distributed') }}
              </option>
              <option value="network">
                {{ formatTopicLabel(t, 'network') }}
              </option>
              <option value="microservice">
                {{ formatTopicLabel(t, 'microservice') }}
              </option>
              <option value="os">{{ formatTopicLabel(t, 'os') }}</option>
              <option value="docker_k8s">
                {{ formatTopicLabel(t, 'docker_k8s') }}
              </option>
            </select>
          </label>

          <label v-else class="space-y-2">
            <span class="neo-subheading">{{ t('train.fields.project') }}</span>
            <select v-model="form.project_id" class="neo-select">
              <option disabled value="">{{ t('train.chooseProject') }}</option>
              <option
                v-for="project in projects ?? []"
                :key="project.id"
                :value="project.id"
              >
                {{ project.name }}
              </option>
            </select>
          </label>
        </section>

        <section class="train-form-section">
          <div class="train-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-green)]">
                {{ t('train.fields.jobTarget') }}
              </p>
              <h2 class="train-section-title">
                {{ t('train.fields.jobTarget') }}
              </h2>
            </div>
          </div>

          <label class="space-y-2">
            <span class="neo-subheading">{{
              t('train.fields.jobTarget')
            }}</span>
            <select
              v-model="form.job_target_id"
              class="neo-select"
              @change="markJobTargetSelectionTouched"
            >
              <option value="">{{ t('train.genericJobTargetOption') }}</option>
              <option
                v-for="jobTarget in jobTargets ?? []"
                :key="jobTarget.id"
                :value="jobTarget.id"
              >
                {{ jobTarget.title }}
              </option>
            </select>
          </label>

          <p
            v-if="jobTargetBlockedReason"
            class="neo-note text-[var(--neo-red)]"
          >
            {{ jobTargetBlockedReason }}
          </p>
          <p
            v-else-if="selectedJobTargetHint"
            class="neo-note text-[var(--neo-green)]"
          >
            {{ selectedJobTargetHint }}
          </p>
          <p
            v-else-if="activeJobTargetFallbackNotice"
            class="neo-note text-[var(--neo-red)]"
          >
            {{ activeJobTargetFallbackNotice }}
          </p>
        </section>

        <section class="train-form-section">
          <div class="train-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-yellow)]">
                {{ t('train.fields.promptSet') }}
              </p>
              <h2 class="train-section-title">
                {{ t('train.fields.promptSet') }}
              </h2>
            </div>
          </div>

          <label class="space-y-2">
            <span class="neo-subheading">{{
              t('train.fields.promptSet')
            }}</span>
            <select v-model="form.prompt_set_id" class="neo-select">
              <option
                v-for="promptSet in promptSets ?? []"
                :key="promptSet.id"
                :value="promptSet.id"
              >
                {{ formatPromptSetLabel(promptSet) }}
              </option>
            </select>
          </label>

          <p v-if="selectedPromptSet" class="neo-note">
            {{ selectedPromptSet.description }}
          </p>
        </section>

        <div class="train-form-actions">
          <button
            type="submit"
            class="neo-button-dark w-full sm:w-auto"
            :disabled="isStarting || Boolean(jobTargetBlockedReason)"
          >
            {{ isStarting ? t('common.starting') : t('train.startAction') }}
          </button>
        </div>
      </form>

      <aside class="train-side neo-stagger-list">
        <section class="neo-panel train-side-panel">
          <p class="neo-kicker bg-[var(--neo-blue)]">
            {{
              form.mode === 'basics'
                ? t('train.fields.topic')
                : t('train.fields.project')
            }}
          </p>
          <h2 class="train-section-title">{{ trainFocusTitle }}</h2>
          <p class="neo-note">{{ trainFocusHint }}</p>
        </section>

        <section class="neo-panel train-side-panel">
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('train.fields.jobTarget') }}
          </p>
          <h2 class="train-section-title">
            {{
              selectedJobTarget
                ? selectedJobTarget.title
                : activeJobTarget
                  ? activeJobTarget.title
                  : t('train.genericJobTargetOption')
            }}
          </h2>
          <p class="neo-note">
            {{
              jobTargetBlockedReason ||
              selectedJobTargetHint ||
              activeJobTargetFallbackNotice ||
              t('common.noRecommendation')
            }}
          </p>
        </section>

        <section class="neo-panel train-side-panel">
          <p class="neo-kicker bg-[var(--neo-yellow)]">
            {{ t('train.fields.promptSet') }}
          </p>
          <h2 class="train-section-title">
            {{
              selectedPromptSet
                ? formatPromptSetLabel(selectedPromptSet)
                : t('train.fields.promptSet')
            }}
          </h2>
          <p class="neo-note">
            {{
              selectedPromptSet?.description ||
              t('progress.createSession.description')
            }}
          </p>
        </section>
      </aside>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery } from '@tanstack/vue-query';
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink, useRoute, useRouter } from 'vue-router';

import {
  ApiError,
  createSessionStream,
  getDashboard,
  listJobTargets,
  listPromptSets,
  listProjects,
  type PromptSetSummary,
  type StreamEvent,
} from '../api/client';
import NoticePanel from '../components/NoticePanel.vue';
import ProgressPanel from '../components/ProgressPanel.vue';
import StreamTracePanel from '../components/StreamTracePanel.vue';
import {
  describeJobTargetStatus,
  isJobTargetReady,
} from '../lib/jobTargetStatus';
import {
  formatIntensityLabel,
  formatModeLabel,
  formatStatusLabel,
  formatTopicLabel,
} from '../lib/labels';
import { buildSessionTarget, formatSessionName } from '../lib/sessionSummary';
import { appendStreamEvent, type StreamSection } from '../lib/streaming';
import { useProgressSteps } from '../lib/useProgressSteps';

const router = useRouter();
const route = useRoute();
const { t, tm } = useI18n();
const onboardingMode = computed(() => route.query.onboarding === '1');

const form = reactive({
  mode: 'basics' as 'basics' | 'project',
  topic: 'go',
  project_id: '',
  job_target_id: '',
  prompt_set_id: '',
  intensity: 'auto',
  max_turns: 2,
});
const jobTargetSelectionTouched = ref(false);
const streamSections = ref<StreamSection[]>([]);
const streamEvents = ref<StreamEvent[]>([]);
const startError = ref('');

const { data: projects } = useQuery({
  queryKey: ['projects'],
  queryFn: listProjects,
});

const { data: jobTargets } = useQuery({
  queryKey: ['job-targets'],
  queryFn: listJobTargets,
});

const { data: dashboard } = useQuery({
  queryKey: ['dashboard'],
  queryFn: getDashboard,
});

const { data: promptSets } = useQuery({
  queryKey: ['prompt-sets'],
  queryFn: listPromptSets,
});

const currentSession = computed(() => dashboard.value?.current_session ?? null);
const activeJobTarget = computed(
  () => dashboard.value?.active_job_target ?? null,
);
const selectedProject = computed(
  () =>
    (projects.value ?? []).find((project) => project.id === form.project_id) ??
    null,
);
const selectedJobTarget = computed(
  () =>
    (jobTargets.value ?? []).find(
      (target) => target.id === form.job_target_id,
    ) ?? null,
);
const selectedPromptSet = computed(
  () =>
    (promptSets.value ?? []).find((item) => item.id === form.prompt_set_id) ??
    null,
);
const jobTargetBlockedReason = computed(() => {
  if (!selectedJobTarget.value) {
    return '';
  }
  return isJobTargetReady(selectedJobTarget.value.latest_analysis_status)
    ? ''
    : describeJobTargetStatus(
        t,
        'trainSelection',
        selectedJobTarget.value.latest_analysis_status,
      );
});
const selectedJobTargetHint = computed(() => {
  if (!selectedJobTarget.value) {
    return '';
  }
  return isJobTargetReady(selectedJobTarget.value.latest_analysis_status)
    ? describeJobTargetStatus(
        t,
        'trainSelection',
        selectedJobTarget.value.latest_analysis_status,
      )
    : '';
});
const activeJobTargetFallbackNotice = computed(() => {
  if (
    route.query.job_target_id ||
    !activeJobTarget.value ||
    form.job_target_id ||
    dashboard.value?.recommendation_scope === 'job_target'
  ) {
    return '';
  }
  return describeJobTargetStatus(
    t,
    'trainFallback',
    activeJobTarget.value.latest_analysis_status,
    {
      name: activeJobTarget.value.title,
    },
  );
});
const projectCount = computed(() => (projects.value ?? []).length);
const readyJobTargetCount = computed(
  () =>
    (jobTargets.value ?? []).filter((target) =>
      isJobTargetReady(target.latest_analysis_status),
    ).length,
);
const promptSetCount = computed(() => (promptSets.value ?? []).length);
const trainFocusTitle = computed(() =>
  form.mode === 'basics'
    ? formatTopicLabel(t, form.topic)
    : selectedProject.value?.name || t('train.chooseProject'),
);
const trainFocusHint = computed(() => {
  if (form.mode === 'basics') {
    return t('train.hero.description');
  }

  return selectedProject.value?.summary || t('projects.emptyList');
});

const mutation = useMutation({
  mutationFn: (payload: {
    mode: 'basics' | 'project';
    topic?: string;
    project_id?: string;
    job_target_id?: string;
    prompt_set_id?: string;
    ignore_active_job_target?: boolean;
    intensity: string;
    max_turns?: number;
  }) => {
    streamSections.value = [];
    streamEvents.value = [];
    startError.value = '';
    return createSessionStream(payload, handleStreamEvent);
  },
  onSuccess: async (session) => {
    await router.push(`/sessions/${session.id}`);
  },
  onError: (error) => {
    startError.value = resolveStartErrorMessage(error);
  },
});

const isStarting = computed(() => mutation.isPending.value);
const createSessionSteps = computed(
  () => tm('progress.createSession.steps') as string[],
);
const createSessionProgressSteps = computed(() => [
  {
    label: createSessionSteps.value[0] ?? '',
    signals: [{ type: 'phase' as const, value: 'prepare_context' }],
  },
  {
    label: createSessionSteps.value[1] ?? '',
    signals: [{ type: 'phase' as const, value: 'call_model' }],
  },
  {
    label: createSessionSteps.value[2] ?? '',
    signals: [{ type: 'phase' as const, value: 'parse_result' }],
  },
]);
const { activeIndex: createSessionStepIndex } = useProgressSteps(
  isStarting,
  createSessionProgressSteps,
  streamEvents,
);

watch(
  () => route.query,
  () => {
    jobTargetSelectionTouched.value = false;
    const mode = route.query.mode;
    const topic = route.query.topic;
    const projectId = route.query.project_id;
    const jobTargetId = route.query.job_target_id;
    const promptSetId = route.query.prompt_set_id;

    if (mode === 'basics' || mode === 'project') {
      form.mode = mode;
    }
    if (typeof topic === 'string' && topic) {
      form.topic = topic;
    }
    // 推荐下一轮可能只指定 project 模式，不指定 project_id。
    // 这里要主动清空 project_id，避免误用页面里残留的旧项目选择。
    if (typeof projectId === 'string') {
      form.project_id = projectId;
    } else if (form.mode === 'project' || mode === 'basics') {
      form.project_id = '';
    }
    if (typeof jobTargetId === 'string') {
      form.job_target_id = jobTargetId;
    } else {
      form.job_target_id = '';
    }
    if (typeof promptSetId === 'string' && promptSetId) {
      form.prompt_set_id = promptSetId;
    }
  },
  { immediate: true },
);

watch(
  () => dashboard.value?.active_job_target,
  (active) => {
    if (jobTargetSelectionTouched.value || route.query.job_target_id) {
      return;
    }
    if (active?.latest_analysis_status === 'succeeded') {
      form.job_target_id = active.id;
      return;
    }
    form.job_target_id = '';
  },
  { immediate: true },
);

watch(
  promptSets,
  (items) => {
    const available = items ?? [];
    if (available.length === 0) {
      return;
    }

    const requested = route.query.prompt_set_id;
    if (typeof requested === 'string' && requested) {
      if (available.some((item) => item.id === requested)) {
        form.prompt_set_id = requested;
        return;
      }
    }

    if (available.some((item) => item.id === form.prompt_set_id)) {
      return;
    }

    form.prompt_set_id =
      available.find((item) => item.is_default)?.id ?? available[0].id;
  },
  { immediate: true },
);

function submit() {
  mutation.mutate({
    mode: form.mode,
    topic: form.mode === 'basics' ? form.topic : undefined,
    project_id: form.mode === 'project' ? form.project_id : undefined,
    job_target_id: form.job_target_id || undefined,
    prompt_set_id: form.prompt_set_id || undefined,
    ignore_active_job_target:
      !form.job_target_id && Boolean(activeJobTarget.value),
    intensity: form.intensity,
    max_turns: form.max_turns,
  });
}

function handleStreamEvent(event: StreamEvent) {
  streamEvents.value = [...streamEvents.value, event];
  streamSections.value = appendStreamEvent(streamSections.value, event);
}

function resolveStartErrorMessage(error: unknown): string {
  if (error instanceof ApiError && error.code === 'job_target_not_ready') {
    if (selectedJobTarget.value) {
      return describeJobTargetStatus(
        t,
        'trainSelection',
        selectedJobTarget.value.latest_analysis_status,
      );
    }
    if (activeJobTarget.value) {
      return describeJobTargetStatus(
        t,
        'trainFallback',
        activeJobTarget.value.latest_analysis_status,
        {
          name: activeJobTarget.value.title,
        },
      );
    }
    return t('train.jobTargetUnavailable');
  }

  return error instanceof Error ? error.message : t('common.requestFailed');
}

function markJobTargetSelectionTouched() {
  jobTargetSelectionTouched.value = true;
}

function formatPromptSetLabel(item: PromptSetSummary): string {
  return `${item.label} · ${item.status}`;
}
</script>

<style scoped>
.train-page {
  position: relative;
}

.train-stage {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--neo-red) 82%, white) 0%,
    color-mix(in srgb, var(--neo-red) 58%, var(--neo-yellow)) 100%
  );
}

.train-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.train-stage-copy,
.train-stage-side {
  position: relative;
  z-index: 1;
}

.train-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.train-stage-title {
  font-size: clamp(2.1rem, 6vw, 4.8rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 0.95;
  margin: 0;
  max-width: 11ch;
  text-transform: uppercase;
}

.train-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 38rem;
}

.train-stage-stats {
  display: grid;
  gap: 0.75rem;
}

.train-stage-stat,
.train-stage-context {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.train-stage-stat span {
  font-size: clamp(2.4rem, 8vw, 4rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.train-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.train-stage-side {
  display: grid;
  gap: 1rem;
}

.train-shell {
  display: grid;
  gap: 1rem;
}

.train-form-panel,
.train-side-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.train-form-section {
  border-top: 1px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: grid;
  gap: 1rem;
  padding-top: 1rem;
}

.train-form-section:first-child {
  border-top: 0;
  padding-top: 0;
}

.train-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.train-section-title {
  font-size: 1.35rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.train-section-note {
  line-height: 1.7;
  margin: 0;
  max-width: 24rem;
}

.train-grid {
  display: grid;
  gap: 1rem;
}

.train-form-actions {
  border-top: 1px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
  margin-top: 0.5rem;
  padding-top: 1rem;
}

.train-side {
  display: grid;
  gap: 1rem;
}

@media (min-width: 768px) {
  .train-stage-stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .train-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .train-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.2fr) minmax(19rem, 0.8fr);
  }

  .train-shell {
    align-items: start;
    grid-template-columns: minmax(0, 1fr) minmax(18rem, 21rem);
  }

  .train-side {
    position: sticky;
    top: 1.5rem;
  }
}
</style>
