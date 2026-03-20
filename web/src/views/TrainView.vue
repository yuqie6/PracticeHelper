<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-red)] text-black">
      <p class="neo-kicker bg-white">{{ t('train.hero.kicker') }}</p>
      <p class="text-base font-semibold">
        {{ t('train.hero.description') }}
      </p>
    </header>

    <div v-if="currentSession" class="neo-panel bg-[var(--neo-yellow)]">
      <p class="neo-kicker bg-white">{{ t('home.currentSession.kicker') }}</p>
      <div
        class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between"
      >
        <div>
          <h3 class="text-xl font-black uppercase tracking-[0.06em]">
            {{ t('train.resumeTitle') }}
          </h3>
          <p class="mt-2 text-base font-semibold">
            {{
              t('train.resumeDescription', {
                name: formatSessionName(currentSession),
                status: formatStatusLabel(t, currentSession.status),
              })
            }}
          </p>
        </div>
        <RouterLink
          :to="buildSessionTarget(currentSession)"
          class="neo-button-dark"
        >
          {{ t('common.resume') }}
        </RouterLink>
      </div>
    </div>

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

    <form class="neo-panel space-y-4" @submit.prevent="submit">
      <div class="neo-grid md:grid-cols-2">
        <label class="space-y-2">
          <span class="neo-subheading">{{ t('train.fields.mode') }}</span>
          <select v-model="form.mode" class="neo-select">
            <option value="basics">{{ formatModeLabel(t, 'basics') }}</option>
            <option value="project">{{ formatModeLabel(t, 'project') }}</option>
          </select>
        </label>
        <label class="space-y-2">
          <span class="neo-subheading">{{ t('train.fields.intensity') }}</span>
          <select v-model="form.intensity" class="neo-select">
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
          <span class="neo-subheading">{{ t('train.fields.maxTurns') }}</span>
          <select v-model.number="form.max_turns" class="neo-select">
            <option :value="2">2</option>
            <option :value="3">3</option>
            <option :value="4">4</option>
            <option :value="5">5</option>
          </select>
        </label>
      </div>

      <label v-if="form.mode === 'basics'" class="space-y-2">
        <span class="neo-subheading">{{ t('train.fields.topic') }}</span>
        <select v-model="form.topic" class="neo-select">
          <option value="go">{{ formatTopicLabel(t, 'go') }}</option>
          <option value="redis">{{ formatTopicLabel(t, 'redis') }}</option>
          <option value="kafka">{{ formatTopicLabel(t, 'kafka') }}</option>
          <option value="mysql">{{ formatTopicLabel(t, 'mysql') }}</option>
          <option value="system_design">{{ formatTopicLabel(t, 'system_design') }}</option>
          <option value="distributed">{{ formatTopicLabel(t, 'distributed') }}</option>
          <option value="network">{{ formatTopicLabel(t, 'network') }}</option>
          <option value="os">{{ formatTopicLabel(t, 'os') }}</option>
          <option value="microservice">{{ formatTopicLabel(t, 'microservice') }}</option>
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

      <label class="space-y-2">
        <span class="neo-subheading">{{ t('train.fields.jobTarget') }}</span>
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

      <p v-if="jobTargetBlockedReason" class="neo-note text-[var(--neo-red)]">
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

      <button
        type="submit"
        class="neo-button-dark"
        :disabled="isStarting || Boolean(jobTargetBlockedReason)"
      >
        {{ isStarting ? t('common.starting') : t('train.startAction') }}
      </button>
    </form>
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
  listProjects,
  type StreamEvent,
  type TrainingSessionSummary,
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
import { appendStreamEvent, type StreamSection } from '../lib/streaming';
import { useProgressSteps } from '../lib/useProgressSteps';

const router = useRouter();
const route = useRoute();
const { t, tm } = useI18n();

const form = reactive({
  mode: 'basics' as 'basics' | 'project',
  topic: 'go',
  project_id: '',
  job_target_id: '',
  intensity: 'standard',
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

const currentSession = computed(() => dashboard.value?.current_session ?? null);
const activeJobTarget = computed(
  () => dashboard.value?.active_job_target ?? null,
);
const selectedJobTarget = computed(
  () =>
    (jobTargets.value ?? []).find(
      (target) => target.id === form.job_target_id,
    ) ?? null,
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

const mutation = useMutation({
  mutationFn: (payload: {
    mode: 'basics' | 'project';
    topic?: string;
    project_id?: string;
    job_target_id?: string;
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

function submit() {
  mutation.mutate({
    mode: form.mode,
    topic: form.mode === 'basics' ? form.topic : undefined,
    project_id: form.mode === 'project' ? form.project_id : undefined,
    job_target_id: form.job_target_id || undefined,
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

function formatSessionName(session: TrainingSessionSummary): string {
  if (session.project_name) {
    return session.project_name;
  }

  if (session.topic) {
    return formatTopicLabel(t, session.topic);
  }

  return formatModeLabel(t, session.mode);
}

function buildSessionTarget(session: TrainingSessionSummary): string {
  if (session.status === 'completed' && session.review_id) {
    return `/reviews/${session.review_id}`;
  }

  return `/sessions/${session.id}`;
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
</script>
