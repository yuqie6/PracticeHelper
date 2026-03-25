<template>
  <section class="neo-page profile-page space-y-6 xl:space-y-8">
    <NoticePanel
      v-if="loadError"
      tone="error"
      :title="t('profile.loadErrorTitle')"
      :message="loadError"
    />
    <button
      v-if="loadError"
      type="button"
      class="neo-button-dark"
      @click="refetch()"
    >
      {{ t('common.retry') }}
    </button>

    <header
      v-if="!isLoading"
    >
      <ProfileStageHero
        :is-returning-user="isReturningUser"
        :target-role="form.target_role"
        :company-type="form.target_company_type"
        :current-stage="form.current_stage"
        :stats="profileStageStats"
      />
    </header>

    <NoticePanel
      v-if="saveError"
      tone="error"
      :title="t('profile.saveErrorTitle')"
      :message="saveError"
    />

    <form class="profile-shell" @submit.prevent="submit">
      <ProfileFormSections
        :form="form"
        :validation-errors="validationErrors"
        :company-presets="companyPresets"
        :stage-presets="stagePresets"
        :is-custom-company-type="isCustomCompanyType"
        :is-custom-stage="isCustomStage"
        :tech-stacks="techStacks"
        :weaknesses="weaknesses"
        :tech-suggestions="techSuggestions"
        @update:field="updateFormField"
        @update:tech-stacks="techStacks = $event"
        @update:weaknesses="weaknesses = $event"
      />

      <ProfileSideSummary
        :saved-summary="savedSummary"
        :system-weaknesses="systemWeaknesses"
        :projects="projects"
        :target-role="form.target_role"
        :company-type="form.target_company_type"
        :current-stage="form.current_stage"
        :is-saving="isSaving"
        :is-returning-user="isReturningUser"
      />
    </form>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, reactive, ref, watchEffect } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import {
  getDashboard,
  getProfile,
  listProjects,
  saveProfile,
  type ProjectProfile,
  type WeaknessTag,
} from '../api/client';
import NoticePanel from '../components/NoticePanel.vue';
import ProfileFormSections from '../components/ProfileFormSections.vue';
import ProfileSideSummary from '../components/ProfileSideSummary.vue';
import ProfileStageHero from '../components/ProfileStageHero.vue';
import { buildOnboardingTarget } from '../lib/onboarding';

const queryClient = useQueryClient();
const router = useRouter();
const route = useRoute();
const savedSummary = ref('');
const saveError = ref('');
const startTrainingAfterSave = ref(false);
const validationErrors = reactive({ target_role: false });
const { t } = useI18n();

const form = reactive({
  target_role: '',
  target_company_type: '',
  current_stage: '',
  application_deadline: '',
});

const techStacks = ref<string[]>([]);
const weaknesses = ref<string[]>([]);

const techSuggestions = [
  'Go',
  'Redis',
  'Kafka',
  'MySQL',
  'Docker',
  'Kubernetes',
  'LangGraph',
  'Python',
  'gRPC',
  'Elasticsearch',
];

const companyPresets = computed(() => [
  {
    value: t('profile.presets.companyType.ai'),
    label: t('profile.presets.companyType.ai'),
  },
  {
    value: t('profile.presets.companyType.bigTech'),
    label: t('profile.presets.companyType.bigTech'),
  },
  {
    value: t('profile.presets.companyType.startup'),
    label: t('profile.presets.companyType.startup'),
  },
  {
    value: t('profile.presets.companyType.midsize'),
    label: t('profile.presets.companyType.midsize'),
  },
  {
    value: t('profile.presets.companyType.other'),
    label: t('profile.presets.companyType.other'),
  },
]);

const stagePresets = computed(() => [
  {
    value: t('profile.presets.stage.campus'),
    label: t('profile.presets.stage.campus'),
  },
  {
    value: t('profile.presets.stage.newGrad'),
    label: t('profile.presets.stage.newGrad'),
  },
  {
    value: t('profile.presets.stage.preIntern'),
    label: t('profile.presets.stage.preIntern'),
  },
  {
    value: t('profile.presets.stage.jobSwitch'),
    label: t('profile.presets.stage.jobSwitch'),
  },
  {
    value: t('profile.presets.stage.other'),
    label: t('profile.presets.stage.other'),
  },
]);

const isCustomCompanyType = computed(() => {
  const val = form.target_company_type;
  return Boolean(
    val === t('profile.presets.companyType.other') ||
    (val && !companyPresets.value.some((p) => p.value === val))
  );
});

const isCustomStage = computed(() => {
  const val = form.current_stage;
  return Boolean(
    val === t('profile.presets.stage.other') ||
    (val && !stagePresets.value.some((p) => p.value === val))
  );
});

const { data, error, isLoading, refetch } = useQuery({
  queryKey: ['profile'],
  queryFn: getProfile,
});

const { data: dashboardData } = useQuery({
  queryKey: ['dashboard'],
  queryFn: getDashboard,
});
const dashboard = computed(() => dashboardData.value ?? null);
const systemWeaknesses = computed<WeaknessTag[]>(
  () => dashboard.value?.weaknesses ?? [],
);
const profileStageStats = computed(() => {
  const stats = [];
  if (dashboard.value?.days_until_deadline != null) {
    stats.push({
      value: dashboard.value.days_until_deadline,
      label: t('common.daysRemainingLabel'),
    });
  }
  stats.push(
    { value: techStacks.value.length, label: t('profile.fields.techStacks') },
    { value: projects.value.length, label: t('profile.fields.linkedProjects') },
    {
      value: systemWeaknesses.value.length,
      label: t('profile.fields.systemWeaknesses'),
    },
  );
  return stats;
});

const { data: projectsData } = useQuery({
  queryKey: ['projects'],
  queryFn: listProjects,
});
const projects = computed<ProjectProfile[]>(() => projectsData.value ?? []);

const loadError = computed(() =>
  error.value instanceof Error ? error.value.message : '',
);

const isReturningUser = computed(() => hasMeaningfulProfile(data.value));
const onboardingMode = computed(() => route.query.onboarding === '1');

watchEffect(() => {
  const profile = data.value;
  if (!profile) return;
  form.target_role = profile.target_role;
  form.target_company_type = profile.target_company_type;
  form.current_stage = profile.current_stage;
  form.application_deadline = profile.application_deadline?.slice(0, 10) ?? '';
  techStacks.value = profile.tech_stacks ?? [];
  weaknesses.value = profile.self_reported_weaknesses ?? [];
});

const mutation = useMutation({
  mutationFn: saveProfile,
  onSuccess: async (profile) => {
    // profile 自己可以直接回填缓存，但 dashboard 含有聚合统计和派生字段，仍要回源刷新。
    queryClient.setQueryData(['profile'], profile);
    await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
    saveError.value = '';

    const parts = [
      profile.target_role,
      profile.target_company_type,
      profile.current_stage,
    ].filter(Boolean);
    savedSummary.value = parts.join(' · ');

    if (onboardingMode.value) {
      await router.push(
        projects.value.length > 0
          ? buildOnboardingTarget('train')
          : buildOnboardingTarget('projects'),
      );
      return;
    }

    if (startTrainingAfterSave.value) {
      await router.push('/train');
    }
  },
  onError: (err) => {
    saveError.value =
      err instanceof Error ? err.message : t('common.requestFailed');
  },
});

const isSaving = computed(() => mutation.isPending.value);

function updateFormField(payload: { field: string; value: string }) {
  if (payload.field === 'target_role') {
    form.target_role = payload.value;
    return;
  }
  if (payload.field === 'target_company_type') {
    form.target_company_type = payload.value;
    return;
  }
  if (payload.field === 'current_stage') {
    form.current_stage = payload.value;
    return;
  }
  if (payload.field === 'application_deadline') {
    form.application_deadline = payload.value;
  }
}

function submit() {
  validationErrors.target_role = !form.target_role.trim();
  if (validationErrors.target_role) return;

  startTrainingAfterSave.value =
    !isReturningUser.value && !onboardingMode.value;
  savedSummary.value = '';
  saveError.value = '';
  mutation.mutate({
    ...form,
    tech_stacks: techStacks.value,
    primary_projects: data.value?.primary_projects ?? [],
    self_reported_weaknesses: weaknesses.value,
  });
}

function hasMeaningfulProfile(
  profile: typeof data.value | null | undefined,
): boolean {
  if (!profile) {
    return false;
  }
  return Boolean(
    profile.target_role ||
    profile.target_company_type ||
    profile.current_stage ||
    profile.application_deadline ||
    profile.tech_stacks?.length ||
    profile.primary_projects?.length ||
    profile.self_reported_weaknesses?.length,
  );
}
</script>

<style scoped>
.profile-page {
  position: relative;
}

.profile-shell {
  display: grid;
  gap: 1rem;
}

@media (min-width: 1280px) {
  .profile-shell {
    align-items: start;
    grid-template-columns: minmax(0, 1fr) minmax(18rem, 22rem);
  }
}
</style>
