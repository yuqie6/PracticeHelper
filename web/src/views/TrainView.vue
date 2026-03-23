<template>
  <section class="neo-page train-page space-y-6 xl:space-y-8">
    <TrainStageHero
      :onboarding-mode="onboardingMode"
      :project-count="projectCount"
      :ready-job-target-count="readyJobTargetCount"
      :prompt-set-count="promptSetCount"
      :fallback-job-target-title="fallbackJobTargetTitle"
      :fallback-job-target-description="fallbackJobTargetDescription"
      :current-session="currentSessionCard"
    />

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

    <NoticePanel
      v-if="promptPreferencesError"
      tone="error"
      :title="t('promptOverlay.saveErrorTitle')"
      :message="promptPreferencesError"
    />

    <div class="train-shell">
      <TrainConfigForm
        :form="form"
        :projects="projects ?? []"
        :job-targets="jobTargets ?? []"
        :prompt-sets="promptSets ?? []"
        :selected-prompt-set="selectedPromptSet"
        :focus-title="trainFocusTitle"
        :focus-hint="trainFocusHint"
        :job-target-blocked-reason="jobTargetBlockedReason"
        :selected-job-target-hint="selectedJobTargetHint"
        :active-job-target-fallback-notice="activeJobTargetFallbackNotice"
        :is-starting="isStarting"
        :is-saving-prompt-preferences="isSavingPromptPreferences"
        :format-prompt-set-label="formatPromptSetLabel"
        @submit="submit"
        @touch-job-target="markJobTargetSelectionTouched"
        @save-prompt-preferences="savePromptPreferencesFromForm"
      />

      <TrainContextSummary
        :focus-kicker="
          form.mode === 'basics'
            ? t('train.fields.topic')
            : t('train.fields.project')
        "
        :focus-title="trainFocusTitle"
        :focus-hint="trainFocusHint"
        :job-target-title="contextJobTargetTitle"
        :job-target-description="contextJobTargetDescription"
        :prompt-set-title="contextPromptSetTitle"
        :prompt-set-description="contextPromptSetDescription"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery } from '@tanstack/vue-query';
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import {
  ApiError,
  createSessionStream,
  getDashboard,
  getPromptPreferences,
  listJobTargets,
  listPromptSets,
  listProjects,
  savePromptPreferences,
  type PromptSetSummary,
  type StreamEvent,
} from '../api/client';
import NoticePanel from '../components/NoticePanel.vue';
import ProgressPanel from '../components/ProgressPanel.vue';
import StreamTracePanel from '../components/StreamTracePanel.vue';
import TrainConfigForm from '../components/TrainConfigForm.vue';
import TrainContextSummary from '../components/TrainContextSummary.vue';
import TrainStageHero from '../components/TrainStageHero.vue';
import { resolveApiErrorMessage } from '../lib/apiErrors';
import {
  describeJobTargetStatus,
  isJobTargetReady,
} from '../lib/jobTargetStatus';
import { formatStatusLabel, formatTopicLabel } from '../lib/labels';
import {
  emptyPromptOverlayForm,
  formatPromptOverlaySummary,
  normalizePromptOverlay,
  toPromptOverlayFormState,
} from '../lib/promptOverlay';
import { buildSessionTarget, formatSessionName } from '../lib/sessionSummary';
import { appendStreamEvent, type StreamSection } from '../lib/streaming';
import { useProgressSteps } from '../lib/useProgressSteps';
import { useToast } from '../lib/useToast';

const router = useRouter();
const route = useRoute();
const { t, tm } = useI18n();
const { show: showToast } = useToast();
const onboardingMode = computed(() => route.query.onboarding === '1');

const form = reactive({
  mode: 'basics' as 'basics' | 'project',
  topic: 'go',
  project_id: '',
  job_target_id: '',
  prompt_set_id: '',
  intensity: 'auto',
  max_turns: 2,
  prompt_overlay: emptyPromptOverlayForm(),
});
const jobTargetSelectionTouched = ref(false);
const promptPreferencesHydrated = ref(false);
const streamSections = ref<StreamSection[]>([]);
const streamEvents = ref<StreamEvent[]>([]);
const startError = ref('');
const promptPreferencesError = ref('');

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

const { data: promptPreferences } = useQuery({
  queryKey: ['prompt-preferences'],
  queryFn: getPromptPreferences,
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
const currentSessionCard = computed(() => {
  const session = currentSession.value;
  if (!session) {
    return null;
  }

  return {
    description: t('train.resumeDescription', {
      name: formatSessionName(t, session),
      status: formatStatusLabel(t, session.status),
    }),
    href: buildSessionTarget(session),
  };
});
const fallbackJobTargetTitle = computed(() =>
  activeJobTarget.value
    ? activeJobTarget.value.title
    : t('train.genericJobTargetOption'),
);
const fallbackJobTargetDescription = computed(() =>
  activeJobTarget.value
    ? describeJobTargetStatus(
        t,
        'trainFallback',
        activeJobTarget.value.latest_analysis_status,
        {
          name: activeJobTarget.value.title,
        },
      )
    : t('common.noRecommendation'),
);
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
const contextJobTargetTitle = computed(() =>
  selectedJobTarget.value
    ? selectedJobTarget.value.title
    : activeJobTarget.value
      ? activeJobTarget.value.title
      : t('train.genericJobTargetOption'),
);
const contextJobTargetDescription = computed(
  () =>
    jobTargetBlockedReason.value ||
    selectedJobTargetHint.value ||
    activeJobTargetFallbackNotice.value ||
    t('common.noRecommendation'),
);
const contextPromptSetTitle = computed(() =>
  selectedPromptSet.value
    ? formatPromptSetLabel(selectedPromptSet.value)
    : t('train.fields.promptSet'),
);
const promptOverlaySummary = computed(() =>
  formatPromptOverlaySummary(t, normalizePromptOverlay(form.prompt_overlay)),
);
const contextPromptSetDescription = computed(() => {
  const baseDescription =
    selectedPromptSet.value?.description ||
    t('progress.createSession.description');
  if (!promptOverlaySummary.value) {
    return baseDescription;
  }
  return `${baseDescription} · ${promptOverlaySummary.value}`;
});

const mutation = useMutation({
  mutationFn: (payload: {
    mode: 'basics' | 'project';
    topic?: string;
    project_id?: string;
    job_target_id?: string;
    prompt_set_id?: string;
    prompt_overlay?: ReturnType<typeof normalizePromptOverlay>;
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

const savePromptPreferencesMutation = useMutation({
  mutationFn: () => {
    promptPreferencesError.value = '';
    return savePromptPreferences(
      normalizePromptOverlay(form.prompt_overlay) ?? {},
    );
  },
  onSuccess: () => {
    promptPreferencesError.value = '';
    showToast(t('promptOverlay.saveSuccess'), 'success');
  },
  onError: (error) => {
    promptPreferencesError.value = resolveStartErrorMessage(error);
  },
});

const isStarting = computed(() => mutation.isPending.value);
const isSavingPromptPreferences = computed(
  () => savePromptPreferencesMutation.isPending.value,
);
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

watch(
  promptPreferences,
  (value) => {
    if (promptPreferencesHydrated.value || value == null) {
      return;
    }

    Object.assign(form.prompt_overlay, toPromptOverlayFormState(value));
    promptPreferencesHydrated.value = true;
  },
  { immediate: true },
);

function submit() {
  const promptOverlay = normalizePromptOverlay(form.prompt_overlay);
  mutation.mutate({
    mode: form.mode,
    topic: form.mode === 'basics' ? form.topic : undefined,
    project_id: form.mode === 'project' ? form.project_id : undefined,
    job_target_id: form.job_target_id || undefined,
    prompt_set_id: form.prompt_set_id || undefined,
    prompt_overlay: promptOverlay,
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

  if (error instanceof ApiError) {
    return resolveApiErrorMessage(t, error);
  }

  return error instanceof Error ? error.message : t('common.requestFailed');
}

function markJobTargetSelectionTouched() {
  jobTargetSelectionTouched.value = true;
}

function savePromptPreferencesFromForm() {
  if (isSavingPromptPreferences.value) {
    return;
  }
  savePromptPreferencesMutation.mutate();
}

function formatPromptSetLabel(item: PromptSetSummary): string {
  return `${item.label} · ${item.status}`;
}
</script>

<style scoped>
.train-page {
  position: relative;
}

.train-shell {
  display: grid;
  gap: 1rem;
}

@media (min-width: 1280px) {
  .train-shell {
    align-items: start;
    grid-template-columns: minmax(0, 1fr) minmax(18rem, 21rem);
  }
}
</style>
