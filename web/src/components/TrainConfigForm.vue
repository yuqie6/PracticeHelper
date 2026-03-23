<template>
  <form
    class="neo-panel train-form-panel neo-stagger-list"
    @submit.prevent="emit('submit')"
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
          <h2 class="train-section-title">{{ focusTitle }}</h2>
        </div>
        <p class="neo-note train-section-note">{{ focusHint }}</p>
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
            v-for="project in projects"
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
        <span class="neo-subheading">{{ t('train.fields.jobTarget') }}</span>
        <select
          v-model="form.job_target_id"
          class="neo-select"
          @change="emit('touch-job-target')"
        >
          <option value="">{{ t('train.genericJobTargetOption') }}</option>
          <option
            v-for="jobTarget in jobTargets"
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
        <span class="neo-subheading">{{ t('train.fields.promptSet') }}</span>
        <select v-model="form.prompt_set_id" class="neo-select">
          <option
            v-for="promptSet in promptSets"
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

    <section class="train-form-section">
      <div class="train-section-head">
        <div class="space-y-2">
          <p class="neo-kicker bg-[var(--neo-blue)]">
            {{ t('train.fields.promptStyle') }}
          </p>
          <h2 class="train-section-title">
            {{ t('promptOverlay.sectionTitle') }}
          </h2>
        </div>
        <p class="neo-note train-section-note">
          {{ t('promptOverlay.description') }}
        </p>
      </div>

      <div class="train-grid">
        <label class="space-y-2">
          <span class="neo-subheading">{{
            t('promptOverlay.fields.tone')
          }}</span>
          <select v-model="form.prompt_overlay.tone" class="neo-select">
            <option value="">{{ t('promptOverlay.useDefaultOption') }}</option>
            <option
              v-for="tone in promptOverlayTones"
              :key="tone"
              :value="tone"
            >
              {{ t(`promptOverlay.tone.${tone}`) }}
            </option>
          </select>
        </label>

        <label class="space-y-2">
          <span class="neo-subheading">
            {{ t('promptOverlay.fields.detailLevel') }}
          </span>
          <select v-model="form.prompt_overlay.detail_level" class="neo-select">
            <option value="">{{ t('promptOverlay.useDefaultOption') }}</option>
            <option
              v-for="detailLevel in promptOverlayDetailLevels"
              :key="detailLevel"
              :value="detailLevel"
            >
              {{ t(`promptOverlay.detailLevel.${detailLevel}`) }}
            </option>
          </select>
        </label>

        <label class="space-y-2">
          <span class="neo-subheading">
            {{ t('promptOverlay.fields.followupIntensity') }}
          </span>
          <select
            v-model="form.prompt_overlay.followup_intensity"
            class="neo-select"
          >
            <option value="">{{ t('promptOverlay.useDefaultOption') }}</option>
            <option
              v-for="followupIntensity in promptOverlayFollowupIntensities"
              :key="followupIntensity"
              :value="followupIntensity"
            >
              {{ t(`promptOverlay.followupIntensity.${followupIntensity}`) }}
            </option>
          </select>
        </label>

        <label class="space-y-2">
          <span class="neo-subheading">
            {{ t('promptOverlay.fields.answerLanguage') }}
          </span>
          <select
            v-model="form.prompt_overlay.answer_language"
            class="neo-select"
          >
            <option value="">{{ t('promptOverlay.useDefaultOption') }}</option>
            <option
              v-for="answerLanguage in promptOverlayLanguages"
              :key="answerLanguage"
              :value="answerLanguage"
            >
              {{ t(`promptOverlay.answerLanguage.${answerLanguage}`) }}
            </option>
          </select>
        </label>
      </div>

      <div class="space-y-3">
        <div class="space-y-1">
          <p class="neo-subheading">
            {{ t('promptOverlay.fields.focusTags') }}
          </p>
          <p class="neo-note">{{ t('promptOverlay.focusTagsHint') }}</p>
        </div>
        <div class="train-focus-grid">
          <label
            v-for="tag in promptOverlayFocusTags"
            :key="tag"
            class="train-focus-tag"
            :class="{ 'train-focus-tag-disabled': isFocusTagDisabled(tag) }"
          >
            <input
              class="neo-checkbox"
              type="checkbox"
              :checked="isFocusTagChecked(tag)"
              :disabled="isFocusTagDisabled(tag)"
              @change="toggleFocusTag(tag)"
            />
            <span>{{ t(`promptOverlay.focusTags.${tag}`) }}</span>
          </label>
        </div>
      </div>

      <label class="space-y-2">
        <span class="neo-subheading">
          {{ t('promptOverlay.fields.customInstruction') }}
        </span>
        <textarea
          v-model="form.prompt_overlay.custom_instruction"
          class="neo-textarea min-h-28"
          maxlength="280"
          :placeholder="t('promptOverlay.customInstructionPlaceholder')"
        />
        <span class="neo-note">
          {{
            t('promptOverlay.customInstructionCounter', {
              count: form.prompt_overlay.custom_instruction.length,
            })
          }}
        </span>
      </label>

      <div class="train-form-inline-actions">
        <button
          type="button"
          class="neo-button w-full sm:w-auto"
          :disabled="isSavingPromptPreferences"
          @click="emit('save-prompt-preferences')"
        >
          {{
            isSavingPromptPreferences
              ? t('common.saving')
              : t('promptOverlay.saveDefaultAction')
          }}
        </button>
      </div>
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
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { PromptSetSummary } from '../api/client';
import {
  formatIntensityLabel,
  formatModeLabel,
  formatTopicLabel,
} from '../lib/labels';
import {
  PROMPT_OVERLAY_DETAIL_LEVELS,
  PROMPT_OVERLAY_FOCUS_TAGS,
  PROMPT_OVERLAY_FOLLOWUP_INTENSITIES,
  PROMPT_OVERLAY_LANGUAGES,
  PROMPT_OVERLAY_TONES,
} from '../lib/promptOverlay';

type TrainFormState = {
  mode: 'basics' | 'project';
  topic: string;
  project_id: string;
  job_target_id: string;
  prompt_set_id: string;
  intensity: string;
  max_turns: number;
  prompt_overlay: {
    tone: string;
    detail_level: string;
    followup_intensity: string;
    answer_language: string;
    focus_tags: string[];
    custom_instruction: string;
  };
};

const props = defineProps<{
  form: TrainFormState;
  projects: Array<{ id: string; name: string }>;
  jobTargets: Array<{ id: string; title: string }>;
  promptSets: PromptSetSummary[];
  selectedPromptSet: PromptSetSummary | null;
  focusTitle: string;
  focusHint: string;
  jobTargetBlockedReason: string;
  selectedJobTargetHint: string;
  activeJobTargetFallbackNotice: string;
  isStarting: boolean;
  isSavingPromptPreferences: boolean;
  formatPromptSetLabel: (item: PromptSetSummary) => string;
}>();

const emit = defineEmits<{
  (event: 'submit'): void;
  (event: 'touch-job-target'): void;
  (event: 'save-prompt-preferences'): void;
}>();

const { t } = useI18n();
const promptOverlayTones = PROMPT_OVERLAY_TONES;
const promptOverlayDetailLevels = PROMPT_OVERLAY_DETAIL_LEVELS;
const promptOverlayFollowupIntensities = PROMPT_OVERLAY_FOLLOWUP_INTENSITIES;
const promptOverlayLanguages = PROMPT_OVERLAY_LANGUAGES;
const promptOverlayFocusTags = PROMPT_OVERLAY_FOCUS_TAGS;

function isFocusTagChecked(tag: string): boolean {
  return props.form.prompt_overlay.focus_tags.includes(tag);
}

function isFocusTagDisabled(tag: string): boolean {
  return (
    !isFocusTagChecked(tag) && props.form.prompt_overlay.focus_tags.length >= 2
  );
}

function toggleFocusTag(tag: string) {
  if (isFocusTagChecked(tag)) {
    props.form.prompt_overlay.focus_tags =
      props.form.prompt_overlay.focus_tags.filter((item) => item !== tag);
    return;
  }
  if (props.form.prompt_overlay.focus_tags.length >= 2) {
    return;
  }
  props.form.prompt_overlay.focus_tags = [
    ...props.form.prompt_overlay.focus_tags,
    tag,
  ];
}
</script>

<style scoped>
.train-form-panel {
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

.train-form-inline-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.train-focus-grid {
  display: grid;
  gap: 0.75rem;
}

.train-focus-tag {
  align-items: center;
  background: color-mix(in srgb, var(--neo-surface) 92%, transparent);
  border: 2px solid var(--neo-border);
  display: flex;
  gap: 0.75rem;
  min-height: 3.25rem;
  padding: 0.8rem 0.9rem;
}

.train-focus-tag-disabled {
  opacity: 0.5;
}

@media (min-width: 768px) {
  .train-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .train-focus-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
