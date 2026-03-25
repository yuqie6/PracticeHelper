<template>
  <div class="profile-main neo-stagger-list">
    <section class="neo-panel profile-section">
      <div class="profile-section-head">
        <div class="space-y-2">
          <p class="neo-kicker bg-[var(--neo-red)]">A</p>
          <h2 class="profile-section-title">{{ t('profile.sections.directionTitle') }}</h2>
        </div>
        <p class="neo-note profile-section-note">
          {{ t('profile.sections.directionHint') }}
        </p>
      </div>

      <label class="space-y-2">
        <span class="text-sm font-bold">{{ t('profile.fields.targetRole') }}</span>
        <input
          :value="form.target_role"
          class="neo-input"
          :class="{ '!border-[var(--neo-red)]': validationErrors.target_role }"
          :placeholder="t('profile.placeholders.targetRole')"
          @input="emitField('target_role', $event)"
        />
        <p
          v-if="validationErrors.target_role"
          class="text-xs font-bold text-[var(--neo-red)]"
        >
          {{ t('profile.validation.targetRoleRequired') }}
        </p>
      </label>

      <div class="space-y-2">
        <span class="text-sm font-bold">{{ t('profile.fields.targetCompanyType') }}</span>
        <div class="profile-choice-grid">
          <button
            v-for="option in companyPresets"
            :key="option.value"
            type="button"
            class="profile-choice"
            :class="form.target_company_type === option.value ? 'profile-choice-active' : ''"
            @click="emit('update:field', { field: 'target_company_type', value: option.value })"
          >
            {{ option.label }}
          </button>
        </div>
        <input
          v-if="isCustomCompanyType"
          :value="form.target_company_type"
          class="neo-input mt-2"
          placeholder=""
          @input="emitField('target_company_type', $event)"
        />
      </div>
    </section>

    <section class="neo-panel profile-section">
      <div class="profile-section-head">
        <div class="space-y-2">
          <p class="neo-kicker bg-[var(--neo-blue)]">B</p>
          <h2 class="profile-section-title">{{ t('profile.sections.stageTitle') }}</h2>
        </div>
        <p class="neo-note profile-section-note">
          {{ t('profile.sections.stageHint') }}
        </p>
      </div>

      <div class="space-y-2">
        <span class="text-sm font-bold">{{ t('profile.fields.currentStage') }}</span>
        <div class="profile-choice-grid">
          <button
            v-for="option in stagePresets"
            :key="option.value"
            type="button"
            class="profile-choice"
            :class="form.current_stage === option.value ? 'profile-choice-active' : ''"
            @click="emit('update:field', { field: 'current_stage', value: option.value })"
          >
            {{ option.label }}
          </button>
        </div>
        <input
          v-if="isCustomStage"
          :value="form.current_stage"
          class="neo-input mt-2"
          placeholder=""
          @input="emitField('current_stage', $event)"
        />
      </div>

      <label class="space-y-2">
        <span class="text-sm font-bold">
          {{ t('profile.fields.applicationDeadline') }}
        </span>
        <input
          :value="form.application_deadline"
          type="date"
          class="neo-input"
          @input="emitField('application_deadline', $event)"
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
          <h2 class="profile-section-title">{{ t('profile.sections.techTitle') }}</h2>
        </div>
        <p class="neo-note profile-section-note">
          {{ t('profile.sections.techHint') }}
        </p>
      </div>

      <div class="profile-tech-grid">
        <div class="space-y-2">
          <span class="text-sm font-bold">{{ t('profile.fields.techStacks') }}</span>
          <TagInput
            :model-value="techStacks"
            :placeholder="t('profile.placeholders.techStacks')"
            :suggestions="techSuggestions"
            @update:model-value="emit('update:techStacks', $event)"
          />
        </div>

        <div class="space-y-2">
          <span class="text-sm font-bold">{{ t('profile.fields.weaknesses') }}</span>
          <TagInput
            :model-value="weaknesses"
            :placeholder="t('profile.placeholders.weaknesses')"
            @update:model-value="emit('update:weaknesses', $event)"
          />
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import TagInput from './TagInput.vue';

defineProps<{
  form: {
    target_role: string;
    target_company_type: string;
    current_stage: string;
    application_deadline: string;
  };
  validationErrors: {
    target_role: boolean;
  };
  companyPresets: Array<{ value: string; label: string }>;
  stagePresets: Array<{ value: string; label: string }>;
  isCustomCompanyType: boolean;
  isCustomStage: boolean;
  techStacks: string[];
  weaknesses: string[];
  techSuggestions: string[];
}>();

const emit = defineEmits<{
  (event: 'update:field', payload: { field: string; value: string }): void;
  (event: 'update:techStacks', value: string[]): void;
  (event: 'update:weaknesses', value: string[]): void;
}>();

const { t } = useI18n();

function emitField(field: string, event: Event) {
  emit('update:field', {
    field,
    value: (event.target as HTMLInputElement).value,
  });
}
</script>

<style scoped>
.profile-main {
  display: grid;
  gap: 1rem;
}

.profile-section {
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

.profile-choice-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.profile-choice {
  align-items: center;
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: inline-flex;
  gap: 0.5rem;
  min-height: 2.75rem;
  padding: 0.65rem 0.9rem;
  font-size: 0.95rem;
  font-weight: 700;
  transition:
    transform var(--motion-duration-base) var(--motion-ease-standard),
    box-shadow var(--motion-duration-base) var(--motion-ease-standard),
    background-color var(--motion-duration-fast) var(--motion-ease-soft);
}

.profile-choice:hover {
  box-shadow: 5px 5px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  transform: translate(var(--motion-lift-sm), var(--motion-lift-sm));
}

.profile-choice-active {
  background: color-mix(in srgb, var(--neo-yellow) 72%, white);
}

.profile-tech-grid {
  display: grid;
  gap: 1rem;
}

@media (min-width: 768px) {
  .profile-tech-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
