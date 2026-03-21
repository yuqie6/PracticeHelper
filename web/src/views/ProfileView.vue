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
      class="neo-panel-hero profile-stage"
      :class="
        isReturningUser ? 'bg-[var(--neo-blue)]' : 'bg-[var(--neo-yellow)]'
      "
    >
      <div class="profile-stage-copy">
        <p class="neo-kicker bg-white">{{ t('profile.hero.kicker') }}</p>
        <h1 class="profile-stage-title">
          {{
            isReturningUser
              ? t('profile.hero.returningTitle')
              : t('profile.hero.newUserTitle')
          }}
        </h1>
        <p class="profile-stage-note">
          {{
            isReturningUser
              ? t('profile.hero.returningDescription', {
                  role: form.target_role || '—',
                  company: form.target_company_type || '—',
                  stage: form.current_stage || '—',
                })
              : t('profile.hero.newUserDescription')
          }}
        </p>
      </div>

      <div class="profile-stage-stats">
        <article
          v-if="dashboard?.days_until_deadline != null"
          class="profile-stage-stat"
        >
          <span>{{ dashboard.days_until_deadline }}</span>
          <small>{{ t('common.daysRemainingLabel') }}</small>
        </article>
        <article class="profile-stage-stat">
          <span>{{ techStacks.length }}</span>
          <small>{{ t('profile.fields.techStacks') }}</small>
        </article>
        <article class="profile-stage-stat">
          <span>{{ projects.length }}</span>
          <small>{{ t('profile.fields.linkedProjects') }}</small>
        </article>
        <article class="profile-stage-stat">
          <span>{{ systemWeaknesses.length }}</span>
          <small>{{ t('profile.fields.systemWeaknesses') }}</small>
        </article>
      </div>
    </header>

    <NoticePanel
      v-if="saveError"
      tone="error"
      :title="t('profile.saveErrorTitle')"
      :message="saveError"
    />

    <form class="profile-shell" @submit.prevent="submit">
      <div class="profile-main">
        <section class="neo-panel profile-section">
          <div class="profile-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-red)]">A</p>
              <h2 class="profile-section-title">
                {{ t('profile.sections.directionTitle') }}
              </h2>
            </div>
            <p class="neo-note profile-section-note">
              {{ t('profile.sections.directionHint') }}
            </p>
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
            <span class="text-sm font-bold">
              {{ t('profile.fields.targetCompanyType') }}
            </span>
            <div class="profile-choice-grid">
              <button
                v-for="option in companyPresets"
                :key="option.value"
                type="button"
                class="profile-choice"
                :class="
                  form.target_company_type === option.value
                    ? 'profile-choice-active'
                    : ''
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
        </section>

        <section class="neo-panel profile-section">
          <div class="profile-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-blue)]">B</p>
              <h2 class="profile-section-title">
                {{ t('profile.sections.stageTitle') }}
              </h2>
            </div>
            <p class="neo-note profile-section-note">
              {{ t('profile.sections.stageHint') }}
            </p>
          </div>

          <div class="space-y-2">
            <span class="text-sm font-bold">{{
              t('profile.fields.currentStage')
            }}</span>
            <div class="profile-choice-grid">
              <button
                v-for="option in stagePresets"
                :key="option.value"
                type="button"
                class="profile-choice"
                :class="
                  form.current_stage === option.value
                    ? 'profile-choice-active'
                    : ''
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
            <span class="text-sm font-bold">
              {{ t('profile.fields.applicationDeadline') }}
            </span>
            <input
              v-model="form.application_deadline"
              type="date"
              class="neo-input"
            />
            <p v-if="!form.application_deadline" class="neo-note">
              {{ t('profile.deadlineHint') }}
            </p>
          </label>
        </section>

        <section class="neo-panel profile-section">
          <div class="profile-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-green)]">C</p>
              <h2 class="profile-section-title">
                {{ t('profile.sections.techTitle') }}
              </h2>
            </div>
            <p class="neo-note profile-section-note">
              {{ t('profile.sections.techHint') }}
            </p>
          </div>

          <div class="profile-tech-grid">
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
          </div>
        </section>
      </div>

      <aside class="profile-side">
        <section
          v-if="savedSummary"
          class="neo-panel profile-side-panel bg-[var(--neo-green)]"
        >
          <p class="neo-kicker bg-white">{{ t('profile.saveSuccess') }}</p>
          <h2 class="profile-section-title">{{ t('profile.saveSuccess') }}</h2>
          <p class="neo-note">{{ savedSummary }}</p>
          <div class="profile-side-actions">
            <RouterLink to="/train" class="neo-button-dark w-full">
              {{ t('common.start') }}
            </RouterLink>
            <RouterLink to="/" class="neo-button w-full bg-white">
              {{ t('app.nav.home') }}
            </RouterLink>
          </div>
        </section>

        <section class="neo-panel profile-side-panel">
          <p class="neo-kicker bg-[var(--neo-yellow)]">
            {{ t('profile.fields.systemWeaknesses') }}
          </p>
          <h2 class="profile-section-title">
            {{ t('profile.fields.linkedProjects') }}
          </h2>

          <div v-if="systemWeaknesses.length" class="profile-badge-cloud">
            <span
              v-for="w in systemWeaknesses"
              :key="w.id"
              class="profile-badge"
            >
              {{ w.label }}
              <span class="text-xs font-bold opacity-60">
                {{ w.severity.toFixed(1) }}
              </span>
            </span>
          </div>
          <p v-else class="neo-note">{{ t('profile.noSystemWeaknesses') }}</p>

          <div v-if="projects.length" class="profile-badge-cloud">
            <RouterLink
              v-for="p in projects"
              :key="p.id"
              :to="`/projects`"
              class="profile-link-chip"
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
        </section>

        <section class="neo-panel profile-side-panel">
          <p class="neo-kicker bg-[var(--neo-blue)]">
            {{ t('profile.hero.kicker') }}
          </p>
          <h2 class="profile-section-title">{{ form.target_role || '—' }}</h2>
          <p class="neo-note">
            {{
              t('profile.hero.returningDescription', {
                role: form.target_role || '—',
                company: form.target_company_type || '—',
                stage: form.current_stage || '—',
              })
            }}
          </p>
          <div class="profile-side-actions">
            <button
              type="submit"
              class="neo-button-dark w-full"
              :disabled="isSaving"
            >
              {{
                isSaving
                  ? t('common.saving')
                  : isReturningUser
                    ? t('profile.saveAction')
                    : t('profile.saveAndTrain')
              }}
            </button>
          </div>
        </section>
      </aside>
    </form>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, reactive, ref, watchEffect } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink, useRoute, useRouter } from 'vue-router';

import {
  getDashboard,
  getProfile,
  listProjects,
  saveProfile,
  type ProjectProfile,
  type WeaknessTag,
} from '../api/client';
import NoticePanel from '../components/NoticePanel.vue';
import { buildOnboardingTarget } from '../lib/onboarding';
import TagInput from '../components/TagInput.vue';

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

.profile-stage {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
}

.profile-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.profile-stage-copy,
.profile-stage-stats {
  position: relative;
  z-index: 1;
}

.profile-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.profile-stage-title {
  font-size: clamp(2.1rem, 6vw, 4.6rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 0.95;
  margin: 0;
  max-width: 12ch;
  text-transform: uppercase;
}

.profile-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 38rem;
}

.profile-stage-stats {
  display: grid;
  gap: 0.75rem;
}

.profile-stage-stat {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.profile-stage-stat span {
  font-size: clamp(2.4rem, 8vw, 4rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.profile-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.profile-shell {
  display: grid;
  gap: 1rem;
}

.profile-main,
.profile-side {
  display: grid;
  gap: 1rem;
}

.profile-section,
.profile-side-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.profile-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.profile-section-title {
  font-size: 1.35rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.profile-section-note {
  line-height: 1.7;
  margin: 0;
  max-width: 24rem;
}

.profile-choice-grid,
.profile-badge-cloud,
.profile-side-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.profile-choice,
.profile-badge,
.profile-link-chip {
  align-items: center;
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: inline-flex;
  gap: 0.5rem;
  min-height: 2.75rem;
  padding: 0.65rem 0.9rem;
}

.profile-choice {
  font-size: 0.95rem;
  font-weight: 700;
}

.profile-choice-active {
  background: color-mix(in srgb, var(--neo-yellow) 72%, white);
}

.profile-tech-grid {
  display: grid;
  gap: 1rem;
}

@media (min-width: 768px) {
  .profile-stage-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .profile-tech-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .profile-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.1fr) minmax(18rem, 0.9fr);
  }

  .profile-shell {
    align-items: start;
    grid-template-columns: minmax(0, 1fr) minmax(18rem, 22rem);
  }

  .profile-side {
    position: sticky;
    top: 1.5rem;
  }
}
</style>
