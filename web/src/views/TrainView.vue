<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-red)] text-black">
      <p class="neo-kicker bg-white">{{ t('train.hero.kicker') }}</p>
      <h2 class="neo-heading">{{ t('train.hero.title') }}</h2>
      <p class="mt-3 text-base font-semibold">
        {{ t('train.hero.description') }}
      </p>
    </header>

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
import { computed, reactive } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { createSession, listProjects } from '../api/client';
import { formatIntensityLabel, formatModeLabel, formatTopicLabel } from '../lib/labels';

const router = useRouter();
const { t } = useI18n();

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
