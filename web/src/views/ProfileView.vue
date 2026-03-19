<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-blue)]">
      <p class="neo-kicker bg-white">{{ t('profile.hero.kicker') }}</p>
      <h2 class="neo-heading">{{ t('profile.hero.title') }}</h2>
      <p class="mt-3 text-base font-semibold">
        {{ t('profile.hero.description') }}
      </p>
    </header>

    <form class="neo-panel space-y-4" @submit.prevent="submit">
      <div class="neo-grid md:grid-cols-2">
        <label class="space-y-2">
          <span class="neo-subheading">{{ t('profile.fields.targetRole') }}</span>
          <input
            v-model="form.target_role"
            class="neo-input"
            :placeholder="t('profile.placeholders.targetRole')"
          />
        </label>
        <label class="space-y-2">
          <span class="neo-subheading">{{ t('profile.fields.targetCompanyType') }}</span>
          <input
            v-model="form.target_company_type"
            class="neo-input"
            :placeholder="t('profile.placeholders.targetCompanyType')"
          />
        </label>
      </div>

      <div class="neo-grid md:grid-cols-2">
        <label class="space-y-2">
          <span class="neo-subheading">{{ t('profile.fields.currentStage') }}</span>
          <input
            v-model="form.current_stage"
            class="neo-input"
            :placeholder="t('profile.placeholders.currentStage')"
          />
        </label>
        <label class="space-y-2">
          <span class="neo-subheading">{{ t('profile.fields.applicationDeadline') }}</span>
          <input v-model="form.application_deadline" type="date" class="neo-input" />
        </label>
      </div>

      <label class="space-y-2">
        <span class="neo-subheading">{{ t('profile.fields.techStacks') }}</span>
        <input
          v-model="techStacksRaw"
          class="neo-input"
          :placeholder="t('profile.placeholders.techStacks')"
        />
      </label>

      <label class="space-y-2">
        <span class="neo-subheading">{{ t('profile.fields.primaryProjects') }}</span>
        <input
          v-model="projectsRaw"
          class="neo-input"
          :placeholder="t('profile.placeholders.primaryProjects')"
        />
      </label>

      <label class="space-y-2">
        <span class="neo-subheading">{{ t('profile.fields.weaknesses') }}</span>
        <textarea
          v-model="weaknessesRaw"
          class="neo-textarea"
          :placeholder="t('profile.placeholders.weaknesses')"
        />
      </label>

      <div class="flex flex-wrap items-center gap-3">
        <button type="submit" class="neo-button-dark" :disabled="isSaving">
          {{ isSaving ? t('common.saving') : t('profile.saveAction') }}
        </button>
        <span class="neo-note">{{ successMessage }}</span>
      </div>
    </form>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, reactive, ref, watchEffect } from 'vue';
import { useI18n } from 'vue-i18n';

import { getProfile, saveProfile, type UserProfile } from '../api/client';

const queryClient = useQueryClient();
const successMessage = ref('');
const { t } = useI18n();

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
    successMessage.value = t('profile.saveSuccess');
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
