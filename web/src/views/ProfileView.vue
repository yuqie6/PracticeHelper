<template>
  <section class="neo-page space-y-6">
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

    <!-- Summary header for returning users -->
    <header v-if="isReturningUser" class="neo-panel bg-[var(--neo-blue)]">
      <p class="neo-kicker bg-white">{{ t('profile.hero.kicker') }}</p>
      <h2 class="text-xl font-black">{{ t('profile.hero.returningTitle') }}</h2>
      <p class="mt-1 text-base font-semibold">
        {{
          t('profile.hero.returningDescription', {
            role: form.target_role || '—',
            company: form.target_company_type || '—',
            stage: form.current_stage || '—',
          })
        }}
      </p>
      <div class="mt-3 flex flex-wrap gap-3">
        <span
          v-if="dashboard?.days_until_deadline != null"
          class="neo-badge bg-white"
        >
          {{
            t('profile.summaryStats.deadline', {
              days: dashboard.days_until_deadline,
            })
          }}
        </span>
        <span v-if="techStacks.length" class="neo-badge bg-white">
          {{
            t('profile.summaryStats.techCount', { count: techStacks.length })
          }}
        </span>
        <span class="neo-badge bg-white">
          {{
            t('profile.summaryStats.sessionCount', {
              count: dashboard?.recent_sessions?.length ?? 0,
            })
          }}
        </span>
      </div>
    </header>

    <!-- Guided header for new users -->
    <header v-else-if="!isLoading" class="neo-panel bg-[var(--neo-yellow)]">
      <p class="neo-kicker bg-white">{{ t('profile.hero.kicker') }}</p>
      <h2 class="text-xl font-black">{{ t('profile.hero.newUserTitle') }}</h2>
      <p class="mt-1 text-base font-semibold">
        {{ t('profile.hero.newUserDescription') }}
      </p>
    </header>

    <!-- Save success summary card -->
    <div v-if="savedSummary" class="neo-panel space-y-3 bg-[var(--neo-green)]">
      <p class="text-lg font-black">{{ t('profile.saveSuccess') }}</p>
      <p class="text-base font-semibold">{{ savedSummary }}</p>
      <div class="flex flex-wrap gap-3">
        <RouterLink to="/train" class="neo-button-dark">{{
          t('common.start')
        }}</RouterLink>
        <RouterLink to="/" class="neo-button bg-white">{{
          t('app.nav.home')
        }}</RouterLink>
      </div>
    </div>

    <form class="space-y-6" @submit.prevent="submit">
      <!-- Section A: Direction -->
      <div class="neo-panel space-y-4">
        <div>
          <p class="neo-kicker bg-[var(--neo-red)]">A</p>
          <h3 class="text-lg font-black">
            {{ t('profile.sections.directionTitle') }}
          </h3>
          <p class="neo-note mt-1">{{ t('profile.sections.directionHint') }}</p>
        </div>

        <label class="space-y-2">
          <span class="text-sm font-bold">{{
            t('profile.fields.targetRole')
          }}</span>
          <input
            v-model="form.target_role"
            class="neo-input"
            :class="{
              '!border-[var(--neo-red)]': validationErrors.target_role,
            }"
            :placeholder="t('profile.placeholders.targetRole')"
          />
          <p
            v-if="validationErrors.target_role"
            class="text-xs font-bold text-[var(--neo-red)]"
          >
            {{ t('profile.validation.targetRoleRequired') }}
          </p>
        </label>

        <div class="space-y-2">
          <span class="text-sm font-bold">{{
            t('profile.fields.targetCompanyType')
          }}</span>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="option in companyPresets"
              :key="option.value"
              type="button"
              class="border-2 border-black px-3 py-1.5 text-sm font-bold md:border-4"
              :class="
                form.target_company_type === option.value
                  ? 'bg-[var(--neo-yellow)]'
                  : 'bg-white'
              "
              @click="form.target_company_type = option.value"
            >
              {{ option.label }}
            </button>
          </div>
          <input
            v-if="isCustomCompanyType"
            v-model="form.target_company_type"
            class="neo-input mt-2"
            placeholder=""
          />
        </div>
      </div>

      <!-- Section B: Stage -->
      <div class="neo-panel space-y-4">
        <div>
          <p class="neo-kicker bg-[var(--neo-blue)]">B</p>
          <h3 class="text-lg font-black">
            {{ t('profile.sections.stageTitle') }}
          </h3>
          <p class="neo-note mt-1">{{ t('profile.sections.stageHint') }}</p>
        </div>

        <div class="space-y-2">
          <span class="text-sm font-bold">{{
            t('profile.fields.currentStage')
          }}</span>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="option in stagePresets"
              :key="option.value"
              type="button"
              class="border-2 border-black px-3 py-1.5 text-sm font-bold md:border-4"
              :class="
                form.current_stage === option.value
                  ? 'bg-[var(--neo-yellow)]'
                  : 'bg-white'
              "
              @click="form.current_stage = option.value"
            >
              {{ option.label }}
            </button>
          </div>
          <input
            v-if="isCustomStage"
            v-model="form.current_stage"
            class="neo-input mt-2"
            placeholder=""
          />
        </div>

        <label class="space-y-2">
          <span class="text-sm font-bold">{{
            t('profile.fields.applicationDeadline')
          }}</span>
          <input
            v-model="form.application_deadline"
            type="date"
            class="neo-input"
          />
          <p v-if="!form.application_deadline" class="neo-note">
            {{ t('profile.deadlineHint') }}
          </p>
        </label>
      </div>

      <!-- Section C: Tech -->
      <div class="neo-panel space-y-4">
        <div>
          <p class="neo-kicker bg-[var(--neo-green)]">C</p>
          <h3 class="text-lg font-black">
            {{ t('profile.sections.techTitle') }}
          </h3>
          <p class="neo-note mt-1">{{ t('profile.sections.techHint') }}</p>
        </div>

        <div class="space-y-2">
          <span class="text-sm font-bold">{{
            t('profile.fields.techStacks')
          }}</span>
          <TagInput
            v-model="techStacks"
            :placeholder="t('profile.placeholders.techStacks')"
            :suggestions="techSuggestions"
          />
        </div>

        <div class="space-y-2">
          <span class="text-sm font-bold">{{
            t('profile.fields.weaknesses')
          }}</span>
          <TagInput
            v-model="weaknesses"
            :placeholder="t('profile.placeholders.weaknesses')"
          />
        </div>

        <!-- System-tracked weaknesses -->
        <div v-if="systemWeaknesses.length" class="space-y-2">
          <span class="text-sm font-bold">{{
            t('profile.fields.systemWeaknesses')
          }}</span>
          <div class="flex flex-wrap gap-2">
            <span
              v-for="w in systemWeaknesses"
              :key="w.id"
              class="inline-flex items-center gap-1 border-2 border-black bg-[var(--neo-paper)] px-2 py-0.5 text-sm font-semibold"
            >
              {{ w.label }}
              <span class="text-xs font-bold opacity-60">{{
                w.severity.toFixed(1)
              }}</span>
            </span>
          </div>
        </div>
        <p v-else class="neo-note">{{ t('profile.noSystemWeaknesses') }}</p>

        <!-- Linked projects -->
        <div class="space-y-2">
          <span class="text-sm font-bold">{{
            t('profile.fields.linkedProjects')
          }}</span>
          <div v-if="projects.length" class="flex flex-wrap gap-2">
            <RouterLink
              v-for="p in projects"
              :key="p.id"
              :to="`/projects`"
              class="inline-flex border-2 border-black bg-white px-2 py-0.5 text-sm font-bold hover:bg-[var(--neo-yellow)]"
            >
              {{ p.name }}
            </RouterLink>
          </div>
          <p v-else class="neo-note">
            {{ t('profile.noProjects') }}
            <RouterLink to="/projects" class="font-black underline">
              {{ t('profile.goImportProject') }}
            </RouterLink>
          </p>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <button type="submit" class="neo-button-dark" :disabled="isSaving">
          {{
            isSaving
              ? t('common.saving')
              : isReturningUser
                ? t('profile.saveAction')
                : t('profile.saveAndTrain')
          }}
        </button>
      </div>
    </form>

    <NoticePanel
      v-if="saveError"
      tone="error"
      :title="t('profile.saveErrorTitle')"
      :message="saveError"
    />
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, reactive, ref, watchEffect } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink, useRouter } from 'vue-router';

import {
  getDashboard,
  getProfile,
  listProjects,
  saveProfile,
  type ProjectProfile,
  type WeaknessTag,
} from '../api/client';
import NoticePanel from '../components/NoticePanel.vue';
import TagInput from '../components/TagInput.vue';

const queryClient = useQueryClient();
const router = useRouter();
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
  return (
    val === t('profile.presets.companyType.other') ||
    (val && !companyPresets.value.some((p) => p.value === val))
  );
});

const isCustomStage = computed(() => {
  const val = form.current_stage;
  return (
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

const { data: projectsData } = useQuery({
  queryKey: ['projects'],
  queryFn: listProjects,
});
const projects = computed<ProjectProfile[]>(() => projectsData.value ?? []);

const loadError = computed(() =>
  error.value instanceof Error ? error.value.message : '',
);

const isReturningUser = computed(() => hasMeaningfulProfile(data.value));

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
    queryClient.setQueryData(['profile'], profile);
    await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
    saveError.value = '';

    const parts = [
      profile.target_role,
      profile.target_company_type,
      profile.current_stage,
    ].filter(Boolean);
    savedSummary.value = parts.join(' · ');

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

function submit() {
  validationErrors.target_role = !form.target_role.trim();
  if (validationErrors.target_role) return;

  startTrainingAfterSave.value = !isReturningUser.value;
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
