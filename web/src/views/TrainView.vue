<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-red)] text-black">
      <p class="neo-kicker bg-white">{{ t('train.hero.kicker') }}</p>
      <h2 class="neo-heading">{{ t('train.hero.title') }}</h2>
      <p class="mt-3 text-base font-semibold">
        {{ t('train.hero.description') }}
      </p>
    </header>

    <div v-if="currentSession" class="neo-panel bg-[var(--neo-yellow)]">
      <p class="neo-kicker bg-white">{{ t('home.currentSession.kicker') }}</p>
      <div class="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h3 class="text-xl font-black uppercase tracking-[0.06em]">
            {{ t('train.resumeTitle') }}
          </h3>
          <p class="mt-2 text-base font-semibold">
            {{ t('train.resumeDescription', { name: formatSessionName(currentSession), status: formatStatusLabel(t, currentSession.status) }) }}
          </p>
        </div>
        <RouterLink :to="buildSessionTarget(currentSession)" class="neo-button-dark">
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
      v-if="isStarting && (streamReasoning.length || streamContent)"
      :kicker="t('session.processingKicker')"
      :title="t('progress.createSession.title')"
      :description="t('progress.createSession.description')"
      :reasoning-title="t('session.reasoningTitle')"
      :content-title="t('session.contentTitle')"
      :reasoning="streamReasoning"
      :content="streamContent"
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
            <option value="light">{{ formatIntensityLabel(t, 'light') }}</option>
            <option value="standard">{{ formatIntensityLabel(t, 'standard') }}</option>
            <option value="pressure">{{ formatIntensityLabel(t, 'pressure') }}</option>
          </select>
        </label>
      </div>

      <label v-if="form.mode === 'basics'" class="space-y-2">
        <span class="neo-subheading">{{ t('train.fields.topic') }}</span>
        <select v-model="form.topic" class="neo-select">
          <option value="go">{{ formatTopicLabel(t, 'go') }}</option>
          <option value="redis">{{ formatTopicLabel(t, 'redis') }}</option>
          <option value="kafka">{{ formatTopicLabel(t, 'kafka') }}</option>
        </select>
      </label>

      <label v-else class="space-y-2">
        <span class="neo-subheading">{{ t('train.fields.project') }}</span>
        <select v-model="form.project_id" class="neo-select">
          <option disabled value="">{{ t('train.chooseProject') }}</option>
          <option v-for="project in projects ?? []" :key="project.id" :value="project.id">
            {{ project.name }}
          </option>
        </select>
      </label>

      <button type="submit" class="neo-button-dark" :disabled="isStarting">
        {{ isStarting ? t('common.starting') : t('train.startAction') }}
      </button>
    </form>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery } from '@tanstack/vue-query';
import { computed, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink, useRouter } from 'vue-router';

import {
  createSessionStream,
  getDashboard,
  listProjects,
  type StreamEvent,
  type TrainingSessionSummary,
} from '../api/client';
import ProgressPanel from '../components/ProgressPanel.vue';
import StreamTracePanel from '../components/StreamTracePanel.vue';
import {
  formatIntensityLabel,
  formatModeLabel,
  formatStatusLabel,
  formatTopicLabel,
} from '../lib/labels';
import { useProgressSteps } from '../lib/useProgressSteps';

const router = useRouter();
const { t, tm } = useI18n();

const form = reactive({
  mode: 'basics' as 'basics' | 'project',
  topic: 'go',
  project_id: '',
  intensity: 'standard',
});
const streamReasoning = ref<string[]>([]);
const streamContent = ref('');

const { data: projects } = useQuery({
  queryKey: ['projects'],
  queryFn: listProjects,
});

const { data: dashboard } = useQuery({
  queryKey: ['dashboard'],
  queryFn: getDashboard,
});

const currentSession = computed(() => dashboard.value?.current_session ?? null);

const mutation = useMutation({
  mutationFn: (payload: {
    mode: 'basics' | 'project';
    topic?: string;
    project_id?: string;
    intensity: string;
  }) => {
    streamReasoning.value = [];
    streamContent.value = '';
    return createSessionStream(payload, handleStreamEvent);
  },
  onSuccess: async (session) => {
    await router.push(`/sessions/${session.id}`);
  },
});

const isStarting = computed(() => mutation.isPending.value);
const createSessionSteps = computed(() => tm('progress.createSession.steps') as string[]);
const createSessionProgressSteps = computed(() => [
  { afterMs: 0, label: createSessionSteps.value[0] },
  { afterMs: 1200, label: createSessionSteps.value[1] },
  { afterMs: 2600, label: createSessionSteps.value[2] },
]);
const { activeIndex: createSessionStepIndex } = useProgressSteps(
  isStarting,
  createSessionProgressSteps,
);

function submit() {
  mutation.mutate({
    mode: form.mode,
    topic: form.mode === 'basics' ? form.topic : undefined,
    project_id: form.mode === 'project' ? form.project_id : undefined,
    intensity: form.intensity,
  });
}

function handleStreamEvent(event: StreamEvent) {
  if (event.type === 'reasoning' && event.text) {
    streamReasoning.value = [...streamReasoning.value, event.text];
  }

  if (event.type === 'content' && event.text) {
    streamContent.value += event.text;
  }
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
</script>
